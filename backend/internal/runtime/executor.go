// Package runtime contains the manager-owned execution flow.
//
// This file owns Executor, the runtime entrypoint that turns an approved route
// into a compiled graph invocation and applies the final session mutations.
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

// ExecutorConfig defines runtime wiring that is stable for the Executor lifetime.
// Router may be nil; preflight then falls back to deterministic regex guidance.
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

// Execute 执行已批准的路由，并返回本轮 turn type 与最终文本。
//
// 流程：
//  1. 解析资产焦点，合并 supervisor 新抽取的出生资料。
//  2. 由 Manager 生成 ExecutionPlan，并同步 route/debug snapshot。
//  3. 注入 graph init/runtime/result state 到 ctx。
//  4. 调用预编译 bounded Graph（preflight → decide → prefill/dispatch/aggregate）。
//  5. 在 Graph 返回后执行唯一 final guard，再保存 follow-up 资产。
func (e *Executor) Execute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (turnType string, assistantText string, err error) {
	turnContext := captureTurnContext(route)
	route = resolveArtifactFocus(st, route, message)
	// ProfileRevision is part of the artifact owner contract. Merge supervisor
	// extracted birth data before building the plan, otherwise prefill writes a
	// chart under a newer owner than the ArtifactRequirement expects.
	if len(route.Slots.Profile) > 0 {
		st.MergeProfile(route.Slots.Profile)
	}
	plan := e.manager.BuildExecutionPlanForTurn(st, route, message, turnContext)
	route = plan.Route

	e.syncExecutionRoute(ctx, st, route, plan)

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

	if result == nil || result.GraphState == nil {
		invariant := fmt.Errorf("orchestration graph returned no terminal state")
		annotateRuntimeFailureTrace(ctx, invariant)
		return "agent_error", "", classifyRuntimeFailure(route.PrimaryDomain, failureStageAgent, invariant)
	}
	if result.Failure.hasFailure() {
		graphErr := graphFailureError(result.Failure)
		if graphErr == nil {
			graphErr = fmt.Errorf("orchestration graph terminated with failure")
		}
		failure := &RuntimeFailure{
			Class:       firstNonEmpty(result.Failure.FailureClass, failureClassInvariantFailure),
			Stage:       firstNonEmpty(result.Failure.FailureStage, failureStageAgent),
			Domain:      firstNonEmpty(result.Failure.Domain, route.PrimaryDomain),
			Code:        firstNonEmpty(result.Failure.FailureCode, "ORCHESTRATION_FAILED"),
			Retryable:   result.Failure.Retryable,
			Degraded:    result.Failure.Degraded,
			UserVisible: true,
			Message:     firstNonEmpty(result.Failure.Message, "本轮执行未形成可安全展示的结果。"),
			Cause:       graphErr,
		}
		annotateRuntimeFailureTrace(ctx, failure)
		return "agent_error", "", failure
	}

	// Short-circuit 文本是前置澄清或资料提示，不需要 artifact guard；普通
	// specialist 结果必须在 Invoke 返回后统一通过 final guard。
	if result.TerminationReason == "short_circuit" {
		result.TurnType = firstNonEmpty(result.TurnType, "clarification")
		finalText = result.RawFinalText
		e.updateGuidanceState(st, route, message, result.GraphState.PreflightResult)
		if strings.TrimSpace(finalText) != "" {
			_ = emitEventWithTrace(ctx, sink, Event{Type: "text", Data: map[string]any{"content": finalText}}, map[string]any{"buffer_final": true, "turn_type": result.TurnType})
		}
		result.Specialist = specialists.Result{Domain: route.PrimaryDomain, Summary: finalText}
		storeFollowupArtifact(st, route, result.Specialist, finalText, message, result.TurnType)
		e.manager.FinishTurn(st, route, result.TurnType)
		return result.TurnType, finalText, nil
	}

	// Execute 只负责把 graph 原始结果经 final guard 后整理成统一的
	// follow-up 资产与 Manager 状态；Graph 内不发送最终 text。
	rawGraphText := firstNonEmpty(result.RawFinalText, finalText)
	// guided fallback 可能在 Graph 内重建计划；最终 guard 必须消费终态计划，
	// 否则前置和 dispatch 已切到新领域，guard 仍会按旧领域错误拦截结果。
	effectivePlan := result.GraphState.Plan
	guardedTurnType, guardedText := guardFinalAnswerWithPlan(ctx, effectivePlan, st, rawGraphText)
	result.TurnType = guardedTurnType
	finalText = guardedText
	if strings.TrimSpace(guardedText) != "" {
		_ = emitEventWithTrace(ctx, sink, Event{
			Type: "text",
			Data: map[string]any{"content": guardedText},
		}, map[string]any{"buffer_final": true, "turn_type": guardedTurnType})
	}

	finalRoute := route
	if result.PrimaryDomain != "" {
		finalRoute.PrimaryDomain = result.PrimaryDomain
	}
	finalResult := result.Specialist
	// Guard 已经是本轮唯一的用户可见文本边界；这里仅补齐 follow-up
	// 资产的结构化元数据，不能再次 compose，否则会出现发送文本与存档文本分叉。
	if finalResult.Domain == "" {
		finalResult.Domain = firstNonEmpty(result.ReplyDomain, finalRoute.PrimaryDomain)
	}
	finalResult.Summary = finalText
	if strings.TrimSpace(finalResult.Domain) == "" {
		finalResult.Domain = firstNonEmpty(result.ReplyDomain, finalRoute.PrimaryDomain)
	}
	storeFollowupArtifact(st, finalRoute, finalResult, finalText, message, result.TurnType)
	e.manager.FinishTurn(st, finalRoute, result.TurnType)
	return result.TurnType, finalText, nil
}

// updateGuidanceState 是 guidance session mutation 的唯一入口。
// preflight 只返回下一状态，避免 graph 节点在澄清、fallback、正常执行之间分散写 session。
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

// shouldPreserveGuidanceOnExecution keeps profile-collection guidance alive
// until the user supplies a complete profile or the route leaves profile collection.
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
			if e.prefillBaziForPlan(ctx, sink, st, plan, vals) {
				executed = true
			}
		case artifactQimenChart:
			if e.prefillQimen(ctx, sink, st, plan, vals) {
				executed = true
			}
		case artifactZiweiChart:
			if e.prefillZiWeiForPlan(ctx, sink, st, plan, vals) {
				executed = true
			}
		}
	}
}

// prefillQimen 确定性执行奇门遁甲排盘，结果注入 vals 和 session state。
func (e *Executor) prefillQimen(ctx context.Context, sink EventSink, st *state.SessionState, plan ExecutionPlan, vals map[string]any) bool {
	caseID := strings.TrimSpace(plan.TurnContext.CaseID)
	questionTime, ok := parseTurnTime(plan.TurnContext.QuestionTime)
	if st == nil || caseID == "" || !ok {
		return false
	}
	if chart := st.QimenChartForCase(caseID); qimenChartMatchesTurn(chart, plan.TurnContext) {
		vals["qimen_result"] = chart
		return true
	}

	// 奇门主链是问事盘：即使会话已有出生资料，也按本轮问课时刻起局。
	params := qimenQuestionTimeParams(questionTime)
	if result := e.callTool(ctx, "qimen_dunjia", params); result != nil {
		if questionTimeValue := questionTime.Format(time.RFC3339); stringValue(result["question_time"]) != questionTimeValue {
			result["question_time"] = questionTimeValue
		}
		result["case_id"] = caseID
		result["purpose"] = "event_question"
		result["owner_ref"] = map[string]any{"kind": "case", "id": caseID}
		result["time_source"] = "question_time"
		ref := st.StoreChartForOwner(state.AssetKindQimenCaseChart, state.AssetRef{Kind: "case", ID: caseID}, result, "qimen-go-v1")
		if ref.ID == "" {
			return false
		}
		vals["qimen_result"] = result
		if bj, err := json.Marshal(result); err == nil && sink != nil {
			emitChartFromToolResult(ctx, sink, "qimen_dunjia", string(bj))
		}
		return true
	}
	return false
}

// qimenChartMatchesTurn accepts only a chart whose runtime metadata binds it to
// the current Case and question time; legacy payloads without metadata are stale.
func qimenChartMatchesTurn(chart map[string]any, turn contracts.TurnContext) bool {
	if len(chart) == 0 || stringValue(chart["case_id"]) != strings.TrimSpace(turn.CaseID) {
		return false
	}
	owner, ok := chart["owner_ref"].(map[string]any)
	if !ok || stringValue(owner["kind"]) != "case" || stringValue(owner["id"]) != strings.TrimSpace(turn.CaseID) {
		return false
	}
	if stringValue(chart["purpose"]) != "event_question" || stringValue(chart["time_source"]) != "question_time" {
		return false
	}
	return stringValue(chart["question_time"]) == strings.TrimSpace(turn.QuestionTime) &&
		stringValue(chart["pan_schema"]) == "rotating_8" &&
		stringValue(chart["symbol_system"]) == "eight_gate_eight_god"
}

// qimenQuestionTimeParams builds the minimal deterministic params for a Qi Men
// event chart; birth profile fields must not leak into this question-time chart.
func qimenQuestionTimeParams(questionTime time.Time) map[string]any {
	return map[string]any{
		"question_time": questionTime.Format(time.RFC3339),
	}
}

// prefillZiWei 确定性执行紫微斗数排盘，结果注入 vals 和 session state。
func (e *Executor) prefillZiWei(ctx context.Context, sink EventSink, st *state.SessionState, vals map[string]any) bool {
	return e.prefillZiWeiForPlan(ctx, sink, st, ExecutionPlan{}, vals)
}

// prefillZiWeiForPlan executes Zi Wei prefill with the plan's dynamic target time.
func (e *Executor) prefillZiWeiForPlan(ctx context.Context, sink EventSink, st *state.SessionState, plan ExecutionPlan, vals map[string]any) bool {
	profile := st.ActiveProfile()
	params := buildToolParams(profile)
	if params == nil {
		return false
	}
	targetAt := prefillTargetTime(plan)

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
		if !targetAt.IsZero() {
			if _, ok := st.ZiWeiResult["liunian"]; !ok {
				// ZiWeiLiuNianTool 需要 year/month/day/hour（出生信息）+ target_year + age。
				// params 已含出生信息，复用并补充流年年份和虚岁年龄。
				liunianParams := map[string]any{
					"year":        params["year"],
					"month":       params["month"],
					"day":         params["day"],
					"hour":        params["hour"],
					"gender":      params["gender"],
					"target_year": float64(targetAt.Year()),
					"age":         float64(targetAt.Year() - int(toFloat(params["year"])) + 1),
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
	}

	return st.ZiWeiResult != nil
}

func (e *Executor) prefillBazi(ctx context.Context, sink EventSink, st *state.SessionState, vals map[string]any) bool {
	return e.prefillBaziForPlan(ctx, sink, st, ExecutionPlan{}, vals)
}

// prefillBaziForPlan executes Ba Zi prefill with the plan's dynamic target time.
func (e *Executor) prefillBaziForPlan(ctx context.Context, sink EventSink, st *state.SessionState, plan ExecutionPlan, vals map[string]any) bool {
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
		targetAt := prefillTargetTime(plan)
		if !targetAt.IsZero() && !hasCurrentBaziLiuNian(st.BaziResult["liunian"], targetAt) {
			liunianParams := map[string]any{
				"target_year":   float64(targetAt.Year()),
				"target_month":  float64(int(targetAt.Month())),
				"target_day":    float64(targetAt.Day()),
				"target_hour":   float64(targetAt.Hour()),
				"target_minute": float64(targetAt.Minute()),
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

// parseTurnTime parses the fixed per-turn time contract without consulting the system clock.
func parseTurnTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	value, err := time.Parse(time.RFC3339, raw)
	return value, err == nil
}

// prefillTargetTime applies the dynamic-target precedence. A zero time signals
// a missing contract; callers must degrade instead of consulting the system clock.
func prefillTargetTime(plan ExecutionPlan) time.Time {
	if targetAt, ok := parseTurnTime(plan.TurnContext.TargetAt); ok {
		return targetAt
	}
	if questionTime, ok := parseTurnTime(plan.TurnContext.QuestionTime); ok {
		return questionTime
	}
	return time.Time{}
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

// saveToolResult 将 specialist 允许调用的确定性结果写回会话状态。
// Qimen 排盘只允许由 prefill 按 Manager 的 Case 合同写入；这里即使收到旧式
// qimen_dunjia 回调也必须丢弃，不能在缺少 Case owner 时生成错误资产。
func (e *Executor) saveToolResult(st *state.SessionState, toolName, resultJSON string) {
	// specialist 默认只挂知识工具；这里只保留历史兼容的出生盘写入。
	switch toolName {
	case "bazi_calc", "ziwei_calc":
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
		e.manager.beginTurnForPlan(st, route, plan.TurnContext)
	}
	annotateApprovedRouteTrace(ctx, st, route)
}

// captureTurnContext captures the one runtime clock value shared by all
// deterministic preparation stages in a user turn.
func captureTurnContext(route policy.ApprovedRoute) contracts.TurnContext {
	now := time.Now().In(time.FixedZone("Asia/Shanghai", 8*60*60))
	questionTime := now.Format(time.RFC3339)
	return contracts.TurnContext{
		TurnID:              fmt.Sprintf("turn-%d", now.UnixNano()),
		QuestionTime:        questionTime,
		TargetAt:            questionTime,
		TemporalGranularity: temporalGranularityForRoute(route),
		Source:              "server_clock",
	}
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
