package runtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	einomodel "github.com/cloudwego/eino/components/model"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	"github.com/wikiglobal/suanming-agent/internal/specialists/bazi"
	qimenSp "github.com/wikiglobal/suanming-agent/internal/specialists/qimen"
	"github.com/wikiglobal/suanming-agent/internal/specialists/ziwei"
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
}

// NewExecutor 创建运行时执行器。
func NewExecutor(reg *tools.Registry, model einomodel.ToolCallingChatModel, promptMode string) (*Executor, error) {
	sr := specialists.NewRegistry()
	bazi.Register(sr)
	qimenSp.Register(sr)
	ziwei.Register(sr)

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

// Execute 执行已批准的路由。
//
// 流程：
//  1. 更新路由快照
//  2. 确定性 preflight 检查
//  3. 短路返回（澄清、缺资料）
//  4. 主路径：构建 route-bound Supervisor Agent + AgentTool specialists 并执行
func (e *Executor) Execute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (turnType string, assistantText string, err error) {
	updateRoutingSnapshot(st, route)

	// 确定性 preflight
	result := preflight(st, route)
	if result.ShortCircuit {
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": result.Text}})
		return result.TurnType, result.Text, nil
	}

	// 主路径: AgentAsTool 执行
	return e.runAgentRoute(ctx, sink, st, route, message)
}

// runAgentRoute 根据 ApprovedRoute 动态构建 Supervisor Agent + AgentTool specialists，
// 启动 Runner 执行并通过 agentEventBridge 桥接事件到 SSE。
func (e *Executor) runAgentRoute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (string, string, error) {
	allConfigs := e.specialistRegistry.All()
	allowed := allowedSpecialists(route, allConfigs)

	// 构建 route-bound supervisor agent，只挂本轮允许的 AgentTool
	supervisor, err := e.builder.BuildSupervisor(ctx, route, allowed)
	if err != nil {
		return "", "", fmt.Errorf("build supervisor agent: %w", err)
	}

	// 注入追踪 span
	ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
		Name:       "adk_supervisor_agent",
		Kind:       tracing.KindChain,
		Attributes: map[string]any{"model": e.llmModel, "domain": route.PrimaryDomain},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           supervisor,
		EnableStreaming: true,
	})

	// SessionValues: 传入 profile、已有命盘、路由信息供 specialist agent 使用
	vals := map[string]any{
		"profile": st.Profile,
		"domain":  route.PrimaryDomain,
	}
	if st.BaziResult != nil {
		vals["bazi_result"] = st.BaziResult
	}
	if st.QimenResult != nil {
		vals["qimen_result"] = st.QimenResult
	}
	if st.ZiWeiResult != nil {
		vals["ziwei_result"] = st.ZiWeiResult
	}

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
