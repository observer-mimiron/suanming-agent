// Package orchestrator 实现命理咨询对话的会话主控。
//
// 负责管理用户对话的生命周期：加载会话状态、调用 supervisor 取得批准路由、
// 将执行委托给 runtime、维护上下文窗口、推送 SSE 事件并持久化状态。
// 所有公开方法在单会话内串行安全（由会话级锁保证），不同 sessionID 间可并发调用。
package orchestrator

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	appRuntime "github.com/wikiglobal/suanming-agent/internal/runtime"

	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// Orchestrator 是单个用户会话的中央协调器。
type Orchestrator struct {
	store      state.Store
	locker     state.Locker
	flash      llm.Chat
	tracer     tracing.Tracer
	runtime    *appRuntime.Executor
	supervisor RouteAdvisor
}

// New 使用给定的依赖创建 Orchestrator。
func New(executor *appRuntime.Executor, flashClient llm.Chat, store state.Store, locker state.Locker, tracer tracing.Tracer) *Orchestrator {
	return &Orchestrator{
		store:   store,
		locker:  locker,
		flash:   flashClient,
		tracer:  tracer,
		runtime: executor,
	}
}

// SetLLMModel 设置用于 LLM span 元数据的模型名称。
func (o *Orchestrator) SetLLMModel(model string) { o.runtime.SetLLMModel(model) }

// SetSupervisor 注入 supervisor 客户端用于阶段一路由。
func (o *Orchestrator) SetSupervisor(sv RouteAdvisor) { o.supervisor = sv }

// Run 处理会话中的一条用户消息。这是主入口点。
//
// 流程：
//  1. 获取会话锁 → 加载状态 → 启动跟踪。
//  2. 路由 + 执行：通过 supervisor 路由、策略门和领域执行器完成本轮处理。
//  3. 记录轮次 + 上下文窗口维护。
//  4. 跟踪摘要 + done 事件。
func (o *Orchestrator) Run(ctx context.Context, sink EventSink, sessionID, message string) error {
	unlock := o.locker.Lock(sessionID)
	defer unlock()

	ctx, trace := o.tracer.StartTrace(ctx, "chat.turn")
	defer trace.End()

	st := o.store.LoadOrCreate(sessionID)
	defer o.store.Save(st)

	if t := tracing.TraceFromContext(ctx); t != nil {
		t.SessionID = sessionID
		t.UserMessage = message
	}
	tracing.SetTraceAttributes(ctx, map[string]any{
		"session_id":           sessionID,
		"user_message_summary": summarizeTraceMessage(message),
	})

	if o.supervisor == nil || o.runtime == nil {
		return fmt.Errorf("orchestrator: supervisor not configured")
	}
	route, approveErr := o.supervisor.Approve(ctx, message, st)
	if approveErr != nil {
		log.Printf("orchestrator: supervisor 降级，使用 supervisor 返回的保守路由: %v", approveErr)
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "⚠️ 服务暂时降级，使用保守策略继续。如持续出现请稍后重试。",
		}})
	}
	turnType, assistantText, turnErr := o.runtime.Execute(ctx, sink, st, route, message)

	if turnErr == nil {
		o.recordTurnAndMaintainContext(ctx, st, message, assistantText)
	}

	if t := tracing.TraceFromContext(ctx); t != nil {
		t.TurnType = turnType
	}
	tracing.SetTraceAttribute(ctx, "turn_type", turnType)
	if turnErr != nil {
		trace.SetStatus("error")
	}

	sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
		"type":    "route-decision",
		"payload": st.Routing,
	}})
	o.emitTracePanels(ctx, sink, turnType)
	sink.Emit(ctx, Event{Type: "done", Data: map[string]any{}})
	return turnErr
}

// emitTracePanels 发送产品态 process panel 和显式 debug trace。
func (o *Orchestrator) emitTracePanels(ctx context.Context, sink EventSink, turnType string) {
	t := tracing.TraceFromContext(ctx)
	if t == nil {
		return
	}
	if t.TurnType == "" {
		t.TurnType = turnType
	}
	process := t.BuildProcessDigest()
	sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
		"type":    "process-panel",
		"payload": process,
	}})
	debug := t.BuildDebugDigest()
	sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
		"type":    "debug-trace",
		"payload": debug,
	}})

	// 统一执行链路树（与 debug-trace 共存，前端渐进升级）
	execTree := t.BuildExecutionTree()
	sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
		"type":    "execution-tree",
		"payload": execTree,
	}})
}

// recordTurnAndMaintainContext 记录对话轮次，裁剪窗口，溢出时更新滚动摘要。
func (o *Orchestrator) recordTurnAndMaintainContext(ctx context.Context, st *state.SessionState, userMsg, assistantMsg string) {
	if userMsg != "" {
		st.RecordTurn("user", userMsg)
	}
	if assistantMsg != "" {
		st.RecordTurn("assistant", assistantMsg)
	}
	overflow := st.TrimTurns()
	if len(overflow) == 0 {
		return
	}
	if o.flash == nil {
		st.RecentTurns = append(overflow, st.RecentTurns...)
		return
	}
	summary, ok := o.summarizeTurns(ctx, st.RunningSummary, overflow)
	if !ok {
		st.RecentTurns = append(overflow, st.RecentTurns...)
		return
	}
	st.RunningSummary = summary
}

func summarizeTraceMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if len(msg) <= 120 {
		return msg
	}
	return msg[:120]
}
