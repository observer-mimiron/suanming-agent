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
	graph, err := buildOrchestrationGraph()
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

	// 将 supervisor 提取的 Profile 写入当前对象的新资料版本，支持后续轮次精确复用。
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

	finalText, err := e.orchestrationGraph.Invoke(ctx, message)
	if err != nil {
		annotateRuntimeFailureTrace(ctx, err)
		return "agent_error", finalText, classifyRuntimeFailure(route.PrimaryDomain, failureStageAgent, err)
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

	for _, requirement := range plan.Requirements {
		switch requirement.Kind {
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
	if chart := st.ActiveChart(state.AssetKindQimenChart); chart != nil {
		vals["qimen_result"] = chart
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
		st.StoreChart(state.AssetKindQimenChart, result, "qimen-go-v1")
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
	profile := st.ActiveProfile()
	params := buildToolParams(profile)
	if params == nil {
		return false
	}

	if chart := st.ActiveChart(state.AssetKindZiweiChart); chart != nil && isCurrentZiWeiSolarTime(chart) {
		vals["ziwei_result"] = chart
	} else if result := e.callTool(ctx, "ziwei_calc", params); result != nil {
		st.StoreChart(state.AssetKindZiweiChart, result, ziWeiMethodVersion())
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
			if minute, ok := params["minute"]; ok {
				liunianParams["minute"] = minute
			}
			if longitude, ok := params["longitude"]; ok {
				liunianParams["longitude"] = longitude
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
	profile := st.ActiveProfile()
	baziParams := buildToolParams(profile)
	if baziParams == nil {
		return false
	}
	shouldEmitChart := false

	// 历史会话里可能缓存着旧的“晚子时整段进次日”命盘。
	// 一旦发现口径版本不是当前标准，就先丢弃旧盘，再按当前规则重排。
	if st.HasBaziResult() && !isCurrentBaziCalendarRule(st.BaziResult) {
		st.InvalidateActiveChart(state.AssetKindBaziChart)
		shouldEmitChart = true
	}

	if !st.HasBaziResult() {
		if result := e.callTool(ctx, "bazi_calc", baziParams); result != nil {
			st.StoreChart(state.AssetKindBaziChart, result, "lunar-go")
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

	// 预排当前流年（bazi_liunian）：缓存只在同一自然日且结构完整时复用。
	// 空 map、旧日期或缺当前运选择信息都必须重算，否则动态报告会拿到
	// 已有九步大运但没有“当前大运”的自相矛盾输入。
	if st.BaziResult != nil {
		now := time.Now()
		if !hasCurrentBaziLiuNian(st.BaziResult["liunian"], now) {
			liunianParams := map[string]any{
				"target_year":   float64(now.Year()),
				"target_month":  float64(int(now.Month())),
				"target_day":    float64(now.Day()),
				"target_hour":   float64(now.Hour()),
				"target_minute": float64(now.Minute()),
				"bazi_result":   st.BaziResult,
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

// hasCurrentBaziLiuNian accepts only a same-day, structurally complete cache.
// The selected current luck period may legitimately be empty before the first
// luck boundary, so selection metadata rather than a non-empty ganZhi marks it
// as a valid calculation.
func hasCurrentBaziLiuNian(raw any, now time.Time) bool {
	liunian, ok := raw.(map[string]any)
	if !ok || len(liunian) == 0 {
		return false
	}
	targetAt := strings.TrimSpace(stringValue(liunian["liunian_target_at"]))
	if !strings.HasPrefix(targetAt, now.Format("2006-01-02")) {
		return false
	}
	if strings.TrimSpace(stringValue(liunian["liunian_ganzhi"])) == "" {
		return false
	}
	_, hasSelection := liunian["current_dayun_selection"]
	return hasSelection
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
	if longitude, ok := profile["longitude"]; ok {
		params["longitude"] = toFloat(longitude)
	} else if longitude, ok := longitudeForBirthplace(stringValue(profile["birthplace"])); ok {
		params["longitude"] = longitude
	}
	return params
}

func longitudeForBirthplace(birthplace string) (float64, bool) {
	// 城市级出生地只能给出近似经度；显式 longitude 总是优先，避免用城市中心点
	// 覆盖用户的精确地点。表只覆盖当前产品收集的常见城市，其他地点保持不修正。
	longitudes := map[string]float64{
		"北京": 116.4074,
		"上海": 121.4737,
		"广州": 113.2644,
		"深圳": 114.0579,
		"成都": 104.0665,
		"重庆": 106.5516,
		"武汉": 114.3054,
		"西安": 108.9398,
		"杭州": 120.1551,
		"南京": 118.7969,
		"天津": 117.2000,
		"香港": 114.1694,
	}
	longitude, ok := longitudes[strings.TrimSpace(birthplace)]
	return longitude, ok
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
		st.StoreChart(state.AssetKindBaziChart, payload, "lunar-go")
	case "qimen_dunjia":
		st.StoreChart(state.AssetKindQimenChart, payload, "qimen-go-v1")
	case "ziwei_calc":
		st.StoreChart(state.AssetKindZiweiChart, payload, ziWeiMethodVersion())
	}
}

func (e *Executor) buildSessionValues(st *state.SessionState, route policy.ApprovedRoute) map[string]any {
	vals := map[string]any{
		"profile": st.ActiveProfile(),
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
