package runtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tools"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// Executor 负责执行已批准的路由，通过 ADK ChatModelAgent 动态调度工具。
type Executor struct {
	adapters      []einotool.BaseTool
	model         einomodel.ToolCallingChatModel
	llmModel      string
	promptBuilder *Builder
	baziSp        specialists.DomainHandler
	qimenSp       specialists.DomainHandler
	ziweiSp       specialists.DomainHandler
	historyLimit  int  // 传入 agent 的最近对话条数上限，0=不限制
}

// NewExecutor 创建运行时执行器。
func NewExecutor(reg *tools.Registry, model einomodel.ToolCallingChatModel, promptMode string) (*Executor, error) {
	adapters, err := buildAdapters(reg)
	if err != nil {
		return nil, fmt.Errorf("build tool adapters: %w", err)
	}
	return &Executor{
		adapters:      adapters,
		model:         model,
		promptBuilder: NewBuilder(promptMode),
	}, nil
}

// SetLLMModel 设置用于 LLM span 元数据的模型名称。
func (e *Executor) SetLLMModel(model string) { e.llmModel = model }

// SetSpecialists 注入八字、奇门和紫微斗数领域专家。
func (e *Executor) SetSpecialists(baziSp, qimenSp, ziweiSp specialists.DomainHandler) {
	e.baziSp = baziSp
	e.qimenSp = qimenSp
	e.ziweiSp = ziweiSp
}

// PromptBuilder 返回当前运行时使用的 prompt builder。
// SetHistoryLimit 设置传入 agent 的最近对话消息条数上限。
func (e *Executor) SetHistoryLimit(n int) { e.historyLimit = n }
func (e *Executor) PromptBuilder() *Builder { return e.promptBuilder }

// Execute 执行已批准的路由。
//
// 三条路径：
//  1. specialist 直接给出最终答复 → 短路由返回
//  2. supervisor 要求澄清 → 短路由返回
//  3. 主路径：ADK ChatModelAgent 执行，工具调用由模型动态决定
func (e *Executor) Execute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (turnType string, assistantText string, err error) {
	updateRoutingSnapshot(st, route)

	// 短路由 1: specialist 直接答复（资料不全需澄清）
	if final, summary := e.dispatchSpecialists(ctx, sink, st, route); final {
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": summary}})
		return "ask_missing_profile", summary, nil
	}

	// 短路由 2: supervisor 明确要求澄清
	if route.NeedsClarification {
		question := route.ClarificationQuestion
		if question == "" {
			question = "请确认一下您的需求，我再为您详细分析。"
		}
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": question}})
		return "clarification", question, nil
	}

	// 主路径: ADK agent run
	return e.runAgent(ctx, sink, st, route, message)
}

// runAgent 启动 ADK ChatModelAgent 执行并桥接事件到 SSE。
//
// 每次调用重建 agent（Instruction 随会话状态动态变化），
// 通过 agentEventBridge 消费事件流并返回最终文本。
// 工具执行结果通过 saveToolResult 回调写回 session state，供后续轮次复用。
func (e *Executor) runAgent(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (string, string, error) {
	instruction := e.promptBuilder.BuildAgentInstruction(st, route.PrimaryDomain)

	// 重建 agent（ChatModelAgent 创建是轻量操作，只构建配置对象）
	agent, err := NewRuntimeAgent(ctx, e.model, e.adapters, instruction)
	if err != nil {
		return "", "", fmt.Errorf("build runtime agent: %w", err)
	}

	// 注入追踪 span
	ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
		Name:       "adk_runtime_agent",
		Kind:       tracing.KindChain,
		Attributes: map[string]any{"model": e.llmModel},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// 将 profile 和已有命盘注入 SessionValues 供工具适配器读取
	vals := map[string]any{
		"profile": st.Profile,
		"domain":  route.PrimaryDomain,
	}
	if st.BaziResult != nil {
		vals["bazi_result"] = st.BaziResult
	}

	// 构建输入消息：从 RecentTurns 取历史对话 + 当前用户消息
	msgs := e.buildConversationMessages(st, message)

	iter := runner.Run(ctx, msgs, adk.WithSessionValues(vals))

	finalText, err := agentEventBridge(ctx, sink, iter, func(toolName, resultJSON string) {
		e.saveToolResult(st, toolName, resultJSON)
	})
	if err != nil {
		return "agent_error", finalText, err
	}

	return "agent_reading", finalText, nil
}

// saveToolResult 将工具执行结果写回会话状态，供后续轮次复用。
func (e *Executor) saveToolResult(st *state.SessionState, toolName, resultJSON string) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(resultJSON), &payload); err != nil || payload == nil {
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

// dispatchSpecialists 执行领域专家分发；若专家已经给出最终答复则直接返回。
//
// 与备份版本逻辑一致。只保留 specialist 用于资料校验和预检查，
// 实际的工具调度已交由 ADK agent 处理。
func (e *Executor) dispatchSpecialists(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute) (bool, string) {
	if e.baziSp == nil && e.qimenSp == nil {
		return false, ""
	}

	dispSpan := tracing.SpanFromContext(ctx, "domain_dispatch", tracing.KindChain)
	var spResult schemas.DomainResult
	var spErr error
	if primarySp := e.selectPrimarySpecialist(route.PrimaryDomain); primarySp != nil {
		spResult, spErr = primarySp.Run(ctx, st, route, specialistEventSink(sink))
		dispSpan.SetAttribute("primary_domain", spResult.Domain)
		dispSpan.SetAttribute("final", spResult.Final)
		if spErr != nil {
			dispSpan.SetAttribute("error", spErr.Error())
		}
	}
	dispSpan.End()

	if spResult.Final {
		return true, spResult.Summary
	}
	return false, ""
}

// selectPrimarySpecialist 根据主领域选择首个执行的专家。
//
// 与备份版本逻辑一致。
func (e *Executor) selectPrimarySpecialist(primaryDomain string) specialists.DomainHandler {
	primarySp := e.baziSp
	switch primaryDomain {
	case "qimen":
		if e.qimenSp != nil {
			primarySp = e.qimenSp
		}
	case "ziwei":
		if e.ziweiSp != nil {
			primarySp = e.ziweiSp
		}
	case "bazi":
		if e.baziSp == nil && e.qimenSp != nil {
			primarySp = e.qimenSp
		}
	default:
		if primarySp == nil {
			primarySp = e.qimenSp
		}
	}
	return primarySp
}

// updateRoutingSnapshot 将已批准的路由写回会话状态，供后续执行链复用。
func updateRoutingSnapshot(st *state.SessionState, route policy.ApprovedRoute) {
	st.Routing = state.RoutingSnapshot{
		ConversationIntent:    route.ConversationIntent,
		PrimaryDomain:         route.PrimaryDomain,
		SecondaryDomains:      route.SecondaryDomains,
		TaskIntent:            route.TaskIntent,
		AwaitingClarification: route.NeedsClarification,
		Confidence:            route.Confidence,
		TimeScope:             route.Slots.TimeScope,
		TargetSubject:         route.Slots.TargetSubject,
	}
}

func specialistEventSink(sink EventSink) specialists.EventSink {
	return func(ctx context.Context, evt specialists.Event) error {
		return sink.Emit(ctx, Event{Type: evt.Type, Data: evt.Data})
	}
}

// buildConversationMessages 从会话状态构建完整的输入消息列表。
// 包含历史对话（st.RecentTurns）和当前用户消息。
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
