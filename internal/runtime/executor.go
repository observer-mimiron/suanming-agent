package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/adk"
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
	specialistRegistry *specialists.Registry
	builder            *AgentBuilder
	llmModel           string
	promptBuilder      *Builder
	historyLimit       int
	flashChat          llm.Chat
}

// NewExecutor 创建运行时执行器。
func NewExecutor(reg *tools.Registry, sr *specialists.Registry, model einomodel.ToolCallingChatModel, promptMode string) (*Executor, error) {

	return &Executor{
		reg:                reg,
		specialistRegistry: sr,
		builder:            NewAgentBuilder(model, reg),
		promptBuilder:      NewBuilder(promptMode),
	}, nil
}

// SetLLMModel 设置用于 LLM span 元数据的模型名称。
func (e *Executor) SetLLMModel(model string) { e.llmModel = model; e.builder.SetLLMModel(model) }

// PromptBuilder 返回当前运行时使用的 prompt builder。
func (e *Executor) PromptBuilder() *Builder { return e.promptBuilder }

// SetHistoryLimit 设置传入 agent 的最近对话消息条数上限。
func (e *Executor) SetHistoryLimit(n int) { e.historyLimit = n }

func (e *Executor) SetFlashChat(chat llm.Chat) { e.flashChat = chat }

// Execute 执行已批准的路由。
//
// 流程：
//  1. 更新路由快照
//  2. 确定性 preflight 检查
//  3. 短路返回（澄清、缺资料）
//  4. 主路径：构建 route-bound Supervisor Agent + AgentTool specialists 并执行
func (e *Executor) Execute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (turnType string, assistantText string, err error) {
	updateRoutingSnapshot(st, route)

	// 将 supervisor 提取的 Profile 合并到会话状态，支持后续轮次复用
	if len(route.Slots.Profile) > 0 {
		st.MergeProfile(route.Slots.Profile)
	}
	annotateApprovedRouteTrace(ctx, st, route)

	// 确定性 preflight
	preflightSpan := tracing.SpanFromContext(ctx, "preflight", tracing.KindChain)
	preflightSpan.SetAttribute("primary_domain", route.PrimaryDomain)
	preflightSpan.SetAttribute("task_intent", route.TaskIntent)
	result := preflight(st, route, message)
	preflightSpan.SetAttribute("short_circuit", result.ShortCircuit)
	if result.TurnType != "" {
		preflightSpan.SetAttribute("turn_type", result.TurnType)
	}
	preflightSpan.End()
	if result.ShortCircuit {
		e.updateGuidanceState(st, route, message, result)
		_ = emitEventWithTrace(ctx, sink, Event{Type: "text", Data: map[string]any{"content": result.Text}}, map[string]any{
			"turn_type": result.TurnType,
		})
		return result.TurnType, result.Text, nil
	}

	// ForcedRoute: guided_fallback accepted → use forced route for this turn
	if result.ForcedRoute != nil {
		route = *result.ForcedRoute
	}

	e.updateGuidanceState(st, route, message, result)

	// 主路径: AgentAsTool 执行
	return e.runAgentRoute(ctx, sink, st, route, message)
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
		// 旧 directive 兼容路径（Task 4 后删除）
		if result.GuidanceNext == nil && route.Directive != nil {
			st.Guidance = policy.ReduceGuidance(policy.GuidanceReducerInput{
				Current:   st.Guidance,
				Directive: route.Directive,
				Message:   message,
				Profile:   st.Profile,
			})
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

// runAgentRoute 根据 ApprovedRoute 动态构建 Supervisor Agent + AgentTool specialists，
// 启动 Runner 执行并通过 agentEventBridge 桥接事件到 SSE。
func (e *Executor) runAgentRoute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (string, string, error) {
	// SessionValues
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

	e.prefill(ctx, sink, st, route, vals)

	allConfigs := e.specialistRegistry.All()
	allowed := allowedSpecialists(route, allConfigs)
	supervisor, err := e.builder.BuildSupervisor(ctx, route, st, allowed)
	if err != nil {
		return "", "", fmt.Errorf("build supervisor agent: %w", err)
	}

	ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
		Name: "adk_supervisor_agent", Kind: tracing.KindChain,
		Attributes: map[string]any{"model": e.llmModel, "domain": route.PrimaryDomain},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: supervisor, EnableStreaming: true})
	msgs := e.buildConversationMessages(st, message)
	iter := runner.Run(ctx, msgs, adk.WithSessionValues(vals))
	finalText, err := agentEventBridge(ctx, sink, iter, func(toolName, resultJSON string) {
		e.saveToolResult(st, toolName, resultJSON)
	}, shouldBufferFinalAnswer(route))
	if err != nil {
		return "agent_error", finalText, err
	}
	turnType, guardedText := guardFinalAnswerWithTrace(ctx, route, st, finalText)
	if shouldBufferFinalAnswer(route) && guardedText != "" {
		_ = emitEventWithTrace(ctx, sink, Event{Type: "text", Data: map[string]any{"content": guardedText}}, map[string]any{
			"buffer_final": true,
			"turn_type":    turnType,
		})
	}
	return turnType, guardedText, nil
}

// prefill 按领域预执行可安全复用的确定性工具链。
// 当前只保留八字预填充；奇门/紫微必须走显式 AgentTool 路径，避免隐藏副作用和重复发图。
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
	}
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

	// knowledge_search + flash 压缩
	if e.flashChat != nil && st.BaziResult != nil {
		if existing, ok := st.BaziResult["knowledge_summary"].(string); ok && existing != "" {
			vals["knowledge_summary"] = existing
		} else {
			queries := buildKnowledgeQueries(st.BaziResult)
			var allPassages []string
			for i, q := range queries {
				if i >= 3 {
					break
				}
				if result := e.callTool(ctx, "knowledge_search", map[string]any{"query": q, "top_k": float64(3)}); result != nil {
					if passages, ok := result["passages"].([]interface{}); ok {
						for _, p := range passages {
							if pm, ok := p.(map[string]interface{}); ok {
								if c, ok := pm["content"].(string); ok {
									allPassages = append(allPassages, c)
								}
							}
						}
					}
				}
			}
			if len(allPassages) > 0 {
				if summary := e.summarizeKnowledge(ctx, allPassages); summary != "" {
					vals["knowledge_summary"] = summary
					st.BaziResult["knowledge_summary"] = summary
				}
			}
		}
	}
	// 标注 knowledge_summary 是盘面背景，不是针对当前问题的检索结果
	if vals["knowledge_summary"] != nil {
		vals["knowledge_summary_note"] = "以上 knowledge_summary 是根据八字盘面预检索的通用背景知识（非针对当前问题）。如果当前用户问题需要特定古籍引用、具体典籍解释或不同主题的内容，你必须调用 knowledge_search 做针对性检索。"
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

func (e *Executor) summarizeKnowledge(ctx context.Context, passages []string) string {
	if e.flashChat == nil || len(passages) == 0 {
		return ""
	}
	prompt := "将以下古籍原文提炼为关键命理要点，保留典籍出处，每条不超过80字。只输出要点，不要解释。"
	body := strings.Join(passages, "\n---\n")
	if len(body) > 6000 {
		body = body[:6000]
	}
	summary, _, err := e.flashChat.Generate(ctx, prompt, []llm.Message{{Role: "user", Content: body}})
	if err != nil {
		log.Printf("summarizeKnowledge: flash failed: %v", err)
		return ""
	}
	return summary
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

func buildKnowledgeQueries(baziResult map[string]any) []string {
	dayGan, _ := baziResult["dayGan"].(string)
	var monthZhi string
	if pillars, ok := baziResult["pillars"].([]interface{}); ok && len(pillars) >= 2 {
		if p, ok := pillars[1].(map[string]interface{}); ok {
			monthZhi, _ = p["branch"].(string)
		}
	}
	queries := []string{dayGan + " 日主 调候"}
	if monthZhi != "" {
		queries = append(queries, dayGan+" "+monthZhi+"月")
	}
	if dayGan != "" {
		if dayZhi, ok := baziResult["dayZhi"].(string); ok {
			queries = append(queries, dayGan+dayZhi+"日柱")
		}
	}
	return queries
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
