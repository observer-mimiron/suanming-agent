package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	einomodel "github.com/cloudwego/eino/components/model"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/llm"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/schemas"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// Executor 负责执行已批准的路由，驱动 manager-owned execution plan、
// prefill、bounded specialist runner 和 final guard 主链。
type Executor struct {
	reg                *tools.Registry
	toolRunner         *tools.ToolRunner
	flashChat          llm.Chat
	summarizerModel    einomodel.ToolCallingChatModel
	specialistRegistry *specialists.Registry
	builder            *AgentBuilder
	manager            *Manager
	llmModel           string
	historyLimit       int
	orchestrationGraph compose.Runnable[string, string] // 预编译 Graph
	cpStore            compose.CheckPointStore          // nil = Phase 1 模式不启用 Checkpoint
	router             intent.Router                    // semantic router，供 preflight/guidance_gate 用；nil 走 regex
}

type ExecutorConfig struct {
	LLMModel     string
	HistoryLimit int
	Router       intent.Router
	Builder      AgentBuilderConfig
}

// NewExecutor 创建运行时执行器。
// summarizerModel 用于 specialist 的 summarization 中间件压缩长对话历史，传 nil 则不启用压缩。
func NewExecutor(reg *tools.Registry, sr *specialists.Registry, model einomodel.ToolCallingChatModel, flashChat llm.Chat, summarizerModel einomodel.ToolCallingChatModel, cfg ExecutorConfig) (*Executor, error) {
	graph, err := buildOrchestrationGraph(nil)
	if err != nil {
		return nil, fmt.Errorf("compile orchestration graph: %w", err)
	}
	return &Executor{
		reg:                reg,
		toolRunner:         tools.NewToolRunner(reg),
		flashChat:          flashChat,
		summarizerModel:    summarizerModel,
		specialistRegistry: sr,
		builder:            NewAgentBuilder(model, reg, flashChat, summarizerModel, cfg.Builder),
		llmModel:           cfg.LLMModel,
		historyLimit:       cfg.HistoryLimit,
		manager:            NewManager(flashChat),
		orchestrationGraph: graph,
		router:             cfg.Router,
	}, nil
}

// SetCheckPointStore 注入 Checkpoint 存储并重新编译 Graph，启用中断-恢复能力。
// 传 nil 回退到 Phase 1 模式（不启用 Checkpoint）。
func (e *Executor) SetCheckPointStore(cpStore compose.CheckPointStore) error {
	e.cpStore = cpStore
	graph, err := buildOrchestrationGraph(cpStore)
	if err != nil {
		return fmt.Errorf("recompile orchestration graph with checkpoint: %w", err)
	}
	e.orchestrationGraph = graph
	return nil
}

// Execute 执行已批准的路由。
//
// 流程：
//  1. 更新路由快照
//  2. 注入 orchestrationState 到 ctx
//  3. 调用预编译 Graph（preflight → branch → {short_circuit | prefill → agent → guard}）
//
// preflight / guard 的 tracing span 在对应节点 Lambda 内创建，Execute 不再直接持有。
func (e *Executor) Execute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (turnType string, assistantText string, err error) {
	plan := e.manager.BuildExecutionPlan(st, route, message)
	route = plan.Route

	e.syncExecutionRoute(ctx, st, route, plan)

	// 将 supervisor 提取的 Profile 合并到会话状态，支持后续轮次复用
	if len(route.Slots.Profile) > 0 {
		st.MergeProfile(route.Slots.Profile)
	}

	// 构造本轮 specialist 运行所需的 SessionValues。
	vals := e.buildSessionValues(st, route)

	// 注入 init + runtime + result 到 ctx
	// Graph state（PreflightResult/Route）由 WithGenLocalState 管理，节点 Lambda 用 ProcessState 读写
	ctx = withOrchestrationInit(ctx, &orchestrationInit{
		St:      st,
		Route:   route,
		Plan:    plan,
		UserMsg: message,
		Vals:    vals,
	})
	ctx = withOrchestrationRuntime(ctx, &orchestrationRuntime{
		Sink:     sink,
		Executor: e,
		Router:   e.router,
	})
	ctx, result := withOrchestrationResult(ctx)

	// 生成 checkpoint ID（Phase 2 模式启用 Checkpoint 时用，上层调 Resume 时回传）
	var invokeOpts []compose.Option
	cpID := ""
	if e.cpStore != nil {
		cpID = fmt.Sprintf("%s-%d", st.SessionID, time.Now().UnixNano())
		invokeOpts = append(invokeOpts, compose.WithCheckPointID(cpID))
	}
	finalText, err := e.orchestrationGraph.Invoke(ctx, message, invokeOpts...)
	if err != nil {
		annotateRuntimeFailureTrace(ctx, err)
		// 检查是否为 InterruptError（Graph 在 agent 节点前中断）
		if info, ok := compose.ExtractInterruptInfo(err); ok {
			interruptID := ""
			if len(info.InterruptContexts) > 0 {
				interruptID = info.InterruptContexts[0].ID
			}
			e.manager.RecordInterrupt(st, route, cpID, interruptID, "solar_time_confirm")
			return "awaiting_confirm", finalText, &InterruptError{
				CheckPointID: cpID,
				InterruptID:  interruptID,
				Reason:       "solar_time_confirm",
			}
		}
		return "agent_error", finalText, err
	}

	// turnType 由 guardNode / emitShortCircuitNode 写入 result 容器
	finalRoute := route
	if result.PrimaryDomain != "" {
		finalRoute.PrimaryDomain = result.PrimaryDomain
	}
	finalResult := result.Specialist
	if strings.TrimSpace(finalResult.Summary) != "" {
		finalText = finalResult.Summary
	} else {
		if finalResult.Domain == "" {
			finalResult.Domain = firstNonEmpty(result.ReplyDomain, finalRoute.PrimaryDomain)
		}
		if strings.TrimSpace(finalResult.Summary) == "" {
			finalResult.Summary = finalText
		}
		finalText = e.manager.ComposeFinalReply(message, finalResult)
	}
	if strings.TrimSpace(finalResult.Domain) == "" {
		finalResult.Domain = firstNonEmpty(result.ReplyDomain, finalRoute.PrimaryDomain)
	}
	if strings.TrimSpace(finalResult.Summary) == "" {
		finalResult.Summary = finalText
	}
	storeFollowupArtifact(st, finalRoute, finalResult, finalText, message, result.TurnType)
	e.manager.FinishTurn(st, finalRoute, result.TurnType)
	return result.TurnType, finalText, nil
}

// InterruptError 表示 Graph 在 agent 节点前中断，等待用户确认后继续。
type InterruptError struct {
	CheckPointID string
	InterruptID  string
	Reason       string
}

func (e *InterruptError) Error() string {
	return fmt.Sprintf("graph interrupted: cpID=%s interruptID=%s reason=%s",
		e.CheckPointID, e.InterruptID, e.Reason)
}

// Resume 在 Checkpoint 中断后由用户回复触发，继续执行 Graph。
// 典型场景: prefill 后追问"出生时间是否为真太阳时"，用户回复后调用此方法。
//
// 参数:
//   - cpID: Execute 返回的 InterruptError.CheckPointID
//   - interruptID: Execute 返回的 InterruptError.InterruptID
//   - userMessage: 用户的回复文本（如"是的，真太阳时"）
func (e *Executor) Resume(ctx context.Context, sink EventSink, st *state.SessionState, cpID, interruptID, userMessage string) (string, string, error) {
	resumeRoute := policy.ApprovedRoute{
		PrimaryDomain: st.Routing.PrimaryDomain,
		TaskIntent:    st.Routing.TaskIntent,
		Slots: schemas.DecisionSlots{
			QuestionText:  userMessage,
			TimeScope:     st.Routing.TimeScope,
			TargetSubject: st.Routing.TargetSubject,
		},
	}

	plan := e.manager.BuildExecutionPlan(st, resumeRoute, userMessage)
	// 重建 vals（prefill 结果已在 session state，从 st 重建）
	vals := e.buildSessionValues(st, plan.Route)

	ctx = withOrchestrationInit(ctx, &orchestrationInit{
		St:      st,
		Plan:    plan,
		UserMsg: userMessage,
		Vals:    vals,
		// Route 不设——Resume 时 Graph state 从 Checkpoint 恢复，init.Route 被忽略
	})
	ctx = withOrchestrationRuntime(ctx, &orchestrationRuntime{
		Sink:     sink,
		Executor: e,
		Router:   e.router,
	})
	ctx, result := withOrchestrationResult(ctx)

	// 用 ResumeWithData 包装 ctx，携带 interruptID + 用户回复数据
	interruptID = e.manager.ResolveResumeInterruptID(st, cpID, interruptID)
	rCtx := compose.ResumeWithData(ctx, interruptID, userMessage)

	finalText, err := e.orchestrationGraph.Invoke(rCtx, userMessage, compose.WithCheckPointID(cpID))
	if err != nil {
		annotateRuntimeFailureTrace(ctx, err)
		// resume 后仍可能再次中断（多轮确认）
		if info, ok := compose.ExtractInterruptInfo(err); ok {
			newInterruptID := ""
			if len(info.InterruptContexts) > 0 {
				newInterruptID = info.InterruptContexts[0].ID
			}
			e.manager.RecordInterrupt(st, resumeRoute, cpID, newInterruptID, "solar_time_confirm")
			return "awaiting_confirm", finalText, &InterruptError{
				CheckPointID: cpID,
				InterruptID:  newInterruptID,
				Reason:       "solar_time_confirm",
			}
		}
		return "agent_error", finalText, err
	}

	if result.PrimaryDomain != "" {
		resumeRoute.PrimaryDomain = result.PrimaryDomain
	}
	finalResult := result.Specialist
	if strings.TrimSpace(finalResult.Summary) != "" {
		finalText = finalResult.Summary
	} else {
		if finalResult.Domain == "" {
			finalResult.Domain = firstNonEmpty(result.ReplyDomain, resumeRoute.PrimaryDomain)
		}
		if strings.TrimSpace(finalResult.Summary) == "" {
			finalResult.Summary = finalText
		}
		finalText = e.manager.ComposeFinalReply(userMessage, finalResult)
	}
	if strings.TrimSpace(finalResult.Domain) == "" {
		finalResult.Domain = firstNonEmpty(result.ReplyDomain, resumeRoute.PrimaryDomain)
	}
	if strings.TrimSpace(finalResult.Summary) == "" {
		finalResult.Summary = finalText
	}
	storeFollowupArtifact(st, resumeRoute, finalResult, finalText, userMessage, result.TurnType)
	e.manager.FinishTurn(st, resumeRoute, result.TurnType)
	return result.TurnType, finalText, nil
}

func (e *Executor) updateGuidanceState(st *state.SessionState, route policy.ApprovedRoute, message string, result preflightResult) {
	if st == nil {
		return
	}
	if result.ShortCircuit {
		// preflight 返回 GuidanceNext → executor 是唯一 mutation owner
		if result.GuidanceNext != nil {
			st.Guidance = result.GuidanceNext
		}
		return
	}

	// guided_fallback accepted → 清除 guidance
	if result.ForcedRoute != nil {
		st.Guidance = nil
		return
	}

	if shouldPreserveGuidanceOnExecution(route, st) {
		st.Guidance = policy.ReduceGuidance(policy.GuidanceReducerInput{
			Current: st.Guidance,
			Message: message,
			Profile: st.Profile,
		})
		return
	}

	st.Guidance = nil
}

func shouldPreserveGuidanceOnExecution(route policy.ApprovedRoute, st *state.SessionState) bool {
	if st == nil || st.Guidance == nil {
		return false
	}
	if st.IsProfileComplete() {
		return false
	}
	switch route.TaskIntent {
	case "collect_profile", "amend_profile":
		return true
	default:
		return false
	}
}

// prefill 按领域预执行可安全复用的确定性排盘工具链。所有领域的排盘由 Go 确定性执行，结果注入 vals 和 session state，LLM 不再接触排盘工具。
func (e *Executor) prefill(ctx context.Context, sink EventSink, st *state.SessionState, plan ExecutionPlan, vals map[string]any) {
	span := tracing.SpanFromContext(ctx, "prefill", tracing.KindChain)
	span.SetAttribute("domain", plan.Route.PrimaryDomain)
	executed := false
	defer func() {
		span.SetAttribute("executed", executed)
		span.End()
	}()

	artifacts := plan.RequiredArtifacts
	if len(artifacts) == 0 {
		artifacts = selectRequiredArtifacts(plan.Domains)
	}
	for _, artifact := range artifacts {
		switch artifact {
		case artifactBaziChart:
			if e.prefillBazi(ctx, sink, st, vals) {
				executed = true
			}
		case artifactQimenChart:
			if e.prefillQimen(ctx, sink, st, vals) {
				executed = true
			}
		case artifactZiweiChart:
			if e.prefillZiWei(ctx, sink, st, vals) {
				executed = true
			}
		}
	}
}

// prefillQimen 确定性执行奇门遁甲排盘，结果注入 vals 和 session state。
func (e *Executor) prefillQimen(ctx context.Context, sink EventSink, st *state.SessionState, vals map[string]any) bool {
	if st.QimenResult != nil {
		vals["qimen_result"] = st.QimenResult
		return true
	}

	params := buildToolParams(st.Profile)
	if params == nil {
		// 无出生资料时用当前时间排盘（时家奇门以问课时刻起局）
		now := time.Now()
		params = map[string]any{
			"year":   float64(now.Year()),
			"month":  float64(int(now.Month())),
			"day":    float64(now.Day()),
			"hour":   float64(now.Hour()),
			"minute": float64(now.Minute()),
		}
	}
	if result := e.callTool(ctx, "qimen_dunjia", params); result != nil {
		st.QimenResult = result
		vals["qimen_result"] = result
		if bj, err := json.Marshal(result); err == nil && sink != nil {
			emitChartFromToolResult(ctx, sink, "qimen_dunjia", string(bj))
		}
		return true
	}
	return false
}

// prefillZiWei 确定性执行紫微斗数排盘，结果注入 vals 和 session state。
func (e *Executor) prefillZiWei(ctx context.Context, sink EventSink, st *state.SessionState, vals map[string]any) bool {
	profile := st.Profile
	params := buildToolParams(profile)
	if params == nil {
		return false
	}

	if st.ZiWeiResult != nil {
		vals["ziwei_result"] = st.ZiWeiResult
	} else if result := e.callTool(ctx, "ziwei_calc", params); result != nil {
		st.ZiWeiResult = result
		vals["ziwei_result"] = result
		if bj, err := json.Marshal(result); err == nil && sink != nil {
			emitChartFromToolResult(ctx, sink, "ziwei_calc", string(bj))
		}
	}

	if st.ZiWeiResult != nil {
		// 预排当前流年 (ziwei_liunian)
		if _, ok := st.ZiWeiResult["liunian"]; !ok {
			// ZiWeiLiuNianTool 需要 year/month/day/hour（出生信息）+ target_year + age。
			// params 已含出生信息，复用并补充流年年份和虚岁年龄。
			liunianParams := map[string]any{
				"year":        params["year"],
				"month":       params["month"],
				"day":         params["day"],
				"hour":        params["hour"],
				"gender":      params["gender"],
				"target_year": float64(time.Now().Year()),
				"age":         float64(time.Now().Year() - int(params["year"].(float64)) + 1),
			}
			if result := e.callTool(ctx, "ziwei_liunian", liunianParams); result != nil {
				st.ZiWeiResult["liunian"] = result
				vals["ziwei_liunian"] = result
			}
		}
	}

	return st.ZiWeiResult != nil
}

func (e *Executor) prefillBazi(ctx context.Context, sink EventSink, st *state.SessionState, vals map[string]any) bool {
	profile := st.Profile
	baziParams := buildToolParams(profile)
	if baziParams == nil {
		return false
	}
	shouldEmitChart := false

	// 历史会话里可能缓存着旧的“晚子时整段进次日”命盘。
	// 一旦发现口径版本不是当前标准，就先丢弃旧盘，再按当前规则重排。
	if st.HasBaziResult() && !isCurrentBaziCalendarRule(st.BaziResult) {
		st.BaziResult = nil
		shouldEmitChart = true
	}

	if !st.HasBaziResult() {
		if result := e.callTool(ctx, "bazi_calc", baziParams); result != nil {
			st.BaziResult = result
			vals["bazi_result"] = result
			shouldEmitChart = true
		}
	}

	if st.BaziResult != nil {
		if existing, ok := st.BaziResult["yongshen"].(map[string]any); ok && len(existing) > 0 {
			vals["yongshen"] = existing
		} else if result := e.callTool(ctx, "yongshen", baziParams); result != nil {
			vals["yongshen"] = result
			st.BaziResult["yongshen"] = result
		}
	}

	if st.BaziResult != nil {
		if existing, ok := st.BaziResult["dayun_analyzed"].(map[string]any); ok && len(existing) > 0 {
			vals["dayun_analyzed"] = existing
		} else {
			dayunParams := map[string]any{"dayun": st.BaziResult["dayun"], "bazi_result": st.BaziResult}
			if result := e.callTool(ctx, "dayun_analyzer", dayunParams); result != nil {
				vals["dayun_analyzed"] = result
				st.BaziResult["dayun_analyzed"] = result
			}
		}
	}

	// 预排当前流年（bazi_liunian）：注入 target_year=time.Now().Year()，
	// 复用已就绪的 bazi_result（含 dayGan/pillars/dayun/birthday）。
	// 仿 prefillZiWei:322-341 模式，已有结果则跳过避免重复计算。
	if st.BaziResult != nil {
		if _, ok := st.BaziResult["liunian"].(map[string]any); !ok {
			liunianParams := map[string]any{
				"target_year": float64(time.Now().Year()),
				"bazi_result": st.BaziResult,
			}
			if result := e.callTool(ctx, "bazi_liunian", liunianParams); result != nil {
				st.BaziResult["liunian"] = result
				vals["liunian"] = result
			}
		}
	}

	if st.BaziResult != nil {
		vals["bazi_result"] = st.BaziResult
		if bj, err := json.Marshal(st.BaziResult); err == nil {
			vals["bazi_json"] = string(bj)
			if sink != nil && shouldEmitChart {
				// The frontend copies the chart payload directly, so emit only after
				// deterministic enrichment finishes to avoid losing geju/yongshen data.
				// follow-up 只复用已有命盘时不再重复发盘，避免前端重复插入同一 bazi-chart。
				emitChartFromToolResult(ctx, sink, "bazi_calc", string(bj))
			}
		}
	}

	return true
}

func (e *Executor) callTool(ctx context.Context, name string, params map[string]any) map[string]any {
	if e == nil || e.reg == nil {
		return nil
	}
	if e.toolRunner == nil {
		e.toolRunner = tools.NewToolRunner(e.reg)
	}
	result := e.toolRunner.Run(ctx, tools.ToolRunRequest{
		ToolName:       name,
		Params:         params,
		DecisionSource: "prefill",
	})
	if result.Status != tools.ToolRunStatusOK && result.Status != tools.ToolRunStatusFallback {
		if result.Error != nil {
			log.Printf("prefill: tool %s failed: %v", name, result.Error)
		}
		return nil
	}
	m, _ := result.Data.(map[string]any)
	return m
}

func buildToolParams(profile map[string]any) map[string]any {
	year := toFloat(profile["year"])
	month := toFloat(profile["month"])
	day := toFloat(profile["day"])
	hour := toFloat(profile["hour"])
	gender, _ := profile["gender"].(string)
	if year == 0 || month == 0 || day == 0 {
		return nil
	}
	params := map[string]any{"year": year, "month": month, "day": day, "hour": hour, "gender": gender}
	if minute, ok := profile["minute"]; ok {
		params["minute"] = toFloat(minute)
	}
	return params
}

func toFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case json.Number:
		f, _ := val.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	}
	return 0
}

// saveToolResult 将工具执行结果写回会话状态，供后续轮次复用。
func (e *Executor) saveToolResult(st *state.SessionState, toolName, resultJSON string) {
	// 只处理已知的排盘工具。specialist agent 的文本回答不是 JSON，跳过。
	switch toolName {
	case "bazi_calc", "qimen_dunjia", "ziwei_calc":
	default:
		return
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(resultJSON), &payload); err != nil || payload == nil {
		if err != nil {
			log.Printf("saveToolResult: json unmarshal %s failed: %v", toolName, err)
		}
		return
	}
	switch toolName {
	case "bazi_calc":
		st.BaziResult = payload
	case "qimen_dunjia":
		st.QimenResult = payload
	case "ziwei_calc":
		st.ZiWeiResult = payload
	}
}

func (e *Executor) buildSessionValues(st *state.SessionState, route policy.ApprovedRoute) map[string]any {
	vals := map[string]any{
		"profile": st.Profile,
		"domain":  route.PrimaryDomain,
	}
	if st.BaziResult != nil {
		vals["bazi_result"] = st.BaziResult
		if bj, err := json.Marshal(st.BaziResult); err == nil {
			vals["bazi_json"] = string(bj)
		}
	}
	if st.QimenResult != nil {
		vals["qimen_result"] = st.QimenResult
	}
	if st.ZiWeiResult != nil {
		vals["ziwei_result"] = st.ZiWeiResult
	}
	return vals
}

// updateRoutingSnapshot 将已批准的路由写回会话状态，供后续执行链复用。
func updateRoutingSnapshot(st *state.SessionState, route policy.ApprovedRoute) {
	// 同步 subject：TargetSubject 非空时更新，为空但有盘时默认"自己"
	if route.Slots.TargetSubject != "" {
		st.Subject = route.Slots.TargetSubject
	} else if st.Subject == "" && (st.HasBaziResult() || st.IsProfileComplete()) {
		st.Subject = "自己"
	}
	st.Routing = state.RoutingSnapshot{
		ConversationIntent:    route.ConversationIntent,
		PrimaryDomain:         route.PrimaryDomain,
		SecondaryDomains:      route.SecondaryDomains,
		TaskIntent:            route.TaskIntent,
		QimenMode:             route.PolicyHints.QimenMode,
		AwaitingClarification: route.NeedsClarification,
		Confidence:            route.Confidence,
		TimeScope:             route.Slots.TimeScope,
		TargetSubject:         route.Slots.TargetSubject,
	}
	st.LastInput = contracts.LastInputState{
		PreferredDomain:       route.PrimaryDomain,
		SecondaryDomains:      append([]string(nil), route.SecondaryDomains...),
		ExplicitMethod:        route.PrimaryDomain,
		ConsultMode:           route.TaskIntent,
		TimeScope:             route.Slots.TimeScope,
		TargetSubject:         route.Slots.TargetSubject,
		QuestionText:          route.Slots.QuestionText,
		GuidanceActive:        st.Guidance != nil,
		GuidanceDirectiveKind: guidanceDirectiveKind(st),
	}
}

func (e *Executor) syncExecutionRoute(ctx context.Context, st *state.SessionState, route policy.ApprovedRoute, plan ExecutionPlan) {
	updateRoutingSnapshot(st, route)
	st.Execution = plan.Snapshot
	if tr := tracing.TraceFromContext(ctx); tr != nil {
		tracing.SetLocalExecutionSnapshot(tr, plan.Snapshot)
	}
	if e != nil && e.manager != nil {
		e.manager.BeginTurn(st, route)
	}
	annotateApprovedRouteTrace(ctx, st, route)
}

func guidanceDirectiveKind(st *state.SessionState) string {
	if st == nil || st.Guidance == nil {
		return ""
	}
	return st.Guidance.DirectiveKind
}

// buildConversationMessages 从会话状态构建完整的输入消息列表。
// 若 RunningSummary 非空（对话超 30 轮后产生），作为 SystemMessage 注入消息列表开头，
// 让 supervisor 能看到早期对话的摘要，避免上下文断裂。
func (e *Executor) buildConversationMessages(st *state.SessionState, currentMessage string) []*schema.Message {
	limit := e.historyLimit
	if limit <= 0 {
		limit = len(st.RecentTurns)
	}

	msgs := make([]*schema.Message, 0, limit+2)

	if st.RunningSummary != "" {
		msgs = append(msgs, schema.SystemMessage("## 之前对话摘要（早期对话的压缩，供参考）\n\n"+st.RunningSummary))
	}

	start := 0
	if len(st.RecentTurns) > limit {
		start = len(st.RecentTurns) - limit
	}
	for i := start; i < len(st.RecentTurns); i++ {
		t := st.RecentTurns[i]
		if t.Role == "user" {
			msgs = append(msgs, schema.UserMessage(t.Content))
		} else {
			msgs = append(msgs, schema.AssistantMessage(t.Content, nil))
		}
	}
	msgs = append(msgs, schema.UserMessage(currentMessage))
	return msgs
}
