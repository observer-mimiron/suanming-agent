package runtime

import (
	"context"
	"time"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	einomodel "github.com/cloudwego/eino/components/model"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tools"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// Executor 负责执行已批准的路由，通过 Supervisor Agent + AgentAsTool 调度领域专家。
type Executor struct {
	reg                *tools.Registry
	flashChat          llm.Chat
	summarizerModel    einomodel.ToolCallingChatModel
	specialistRegistry *specialists.Registry
	builder            *AgentBuilder
	llmModel           string
	historyLimit       int
	orchestrationGraph compose.Runnable[string, string] // 预编译 Graph
}

// NewExecutor 创建运行时执行器。
// summarizerModel 用于 specialist 的 summarization 中间件压缩长对话历史，传 nil 则不启用压缩。
func NewExecutor(reg *tools.Registry, sr *specialists.Registry, model einomodel.ToolCallingChatModel, flashChat llm.Chat, summarizerModel einomodel.ToolCallingChatModel) (*Executor, error) {
	graph, err := buildOrchestrationGraph()
	if err != nil {
		return nil, fmt.Errorf("compile orchestration graph: %w", err)
	}
	return &Executor{
		reg:                reg,
		summarizerModel:    summarizerModel,
		specialistRegistry: sr,
		builder:            NewAgentBuilder(model, reg, flashChat, summarizerModel),
		orchestrationGraph: graph,
	}, nil
}

// SetLLMModel 设置用于 LLM span 元数据的模型名称。
func (e *Executor) SetLLMModel(model string) { e.llmModel = model; e.builder.SetLLMModel(model) }


// SetHistoryLimit 设置传入 agent 的最近对话消息条数上限。
func (e *Executor) SetHistoryLimit(n int) { e.historyLimit = n }

// Execute 执行已批准的路由。
//
// 流程：
//  1. 更新路由快照
//  2. 注入 orchestrationState 到 ctx
//  3. 调用预编译 Graph（preflight → branch → {short_circuit | prefill → agent → guard}）
//
// preflight / guard 的 tracing span 在对应节点 Lambda 内创建，Execute 不再直接持有。
func (e *Executor) Execute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (turnType string, assistantText string, err error) {
	updateRoutingSnapshot(st, route)

	// 将 supervisor 提取的 Profile 合并到会话状态，支持后续轮次复用
	if len(route.Slots.Profile) > 0 {
		st.MergeProfile(route.Slots.Profile)
	}
	annotateApprovedRouteTrace(ctx, st, route)

	// 构造 per-request vals（原 runAgentRoute 的逻辑）
	vals := map[string]any{"profile": st.Profile, "domain": route.PrimaryDomain}
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

	// 注入 init + runtime + result 到 ctx
	// Graph state（PreflightResult/Route）由 WithGenLocalState 管理，节点 Lambda 用 ProcessState 读写
	ctx = withOrchestrationInit(ctx, &orchestrationInit{
		St:      st,
		Route:   route,
		UserMsg: message,
		Vals:    vals,
	})
	ctx = withOrchestrationRuntime(ctx, &orchestrationRuntime{
		Sink:     sink,
		Executor: e,
	})
	ctx, result := withOrchestrationResult(ctx)

	finalText, err := e.orchestrationGraph.Invoke(ctx, message)
	if err != nil {
		return "agent_error", finalText, err
	}

	// turnType 由 guardNode / emitShortCircuitNode 写入 result 容器
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
func (e *Executor) prefill(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, vals map[string]any) {
	span := tracing.SpanFromContext(ctx, "prefill", tracing.KindChain)
	span.SetAttribute("domain", route.PrimaryDomain)
	executed := false
	defer func() {
		span.SetAttribute("executed", executed)
		span.End()
	}()

	switch route.PrimaryDomain {
	case "bazi":
		executed = e.prefillBazi(ctx, sink, st, vals)
	case "qimen":
		executed = e.prefillQimen(ctx, sink, st, vals)
	case "ziwei":
		executed = e.prefillZiWei(ctx, sink, st, vals)
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
		// 无出生资料时用当前时间排盘
		params = map[string]any{}
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

	if !st.HasBaziResult() {
		if result := e.callTool(ctx, "bazi_calc", baziParams); result != nil {
			st.BaziResult = result
			vals["bazi_result"] = result
			if bj, err := json.Marshal(result); err == nil {
				vals["bazi_json"] = string(bj)
				if sink != nil {
					emitChartFromToolResult(ctx, sink, "bazi_calc", string(bj))
				}
			}
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



	return true
}

func (e *Executor) callTool(ctx context.Context, name string, params map[string]any) map[string]any {
	t, ok := e.reg.Get(name)
	if !ok {
		return nil
	}
	sp := tracing.SpanFromContext(ctx, name, tracing.KindTool)
	sp.SetAttribute("source", "prefill")
	defer sp.End()
	result, err := t.Execute(ctx, params)
	if err != nil || result == nil {
		if err != nil {
			log.Printf("prefill: tool %s failed: %v", name, err)
			sp.RecordError(err)
			sp.SetStatus("error")
		} else {
			sp.SetStatus("degraded")
		}
		return nil
	}
	m, _ := result.(map[string]any)
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
}

// buildConversationMessages 从会话状态构建完整的输入消息列表。
func (e *Executor) buildConversationMessages(st *state.SessionState, currentMessage string) []*schema.Message {
	limit := e.historyLimit
	if limit <= 0 {
		limit = len(st.RecentTurns)
	}

	msgs := make([]*schema.Message, 0, limit+1)

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
