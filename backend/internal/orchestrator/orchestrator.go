// Package orchestrator 实现命理咨询对话的会话主控。
//
// 负责管理用户对话的生命周期：加载会话状态、调用 supervisor 取得批准路由、
// 将执行委托给 runtime、维护上下文窗口、推送 SSE 事件并持久化状态。
// 所有公开方法在单会话内串行安全（由会话级锁保证），不同 sessionID 间可并发调用。
package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/llm"
	appRuntime "github.com/observer-mimiron/suanming-agent/internal/runtime"

	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
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
	unlock, err := acquireSessionLock(ctx, o.locker, sessionID)
	if err != nil {
		return err
	}
	defer unlock()
	if err := ctx.Err(); err != nil {
		return err
	}

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
		"input.value":          message,
	})

	if o.supervisor == nil || o.runtime == nil {
		err := fmt.Errorf("orchestrator: supervisor not configured")
		emitRuntimeError(ctx, sink, err, "bootstrap")
		sink.Emit(ctx, Event{Type: "done", Data: map[string]any{}})
		return err
	}
	route, approveErr := o.supervisor.Approve(ctx, message, st)
	if approveErr != nil {
		log.Printf("orchestrator: supervisor 降级，使用 supervisor 返回的保守路由: %v", approveErr)
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "⚠️ 服务暂时降级，使用保守策略继续。如持续出现请稍后重试。",
		}})
	}
	turnType, assistantText, turnErr := o.runtime.Execute(ctx, sink, st, route, message)
	if strings.TrimSpace(assistantText) != "" {
		tracing.SetTraceAttribute(ctx, "output.value", assistantText)
	}

	if turnErr == nil {
		o.recordTurnAndMaintainContext(ctx, st, message, assistantText)
	}

	if t := tracing.TraceFromContext(ctx); t != nil {
		t.TurnType = turnType
	}
	tracing.SetTraceAttribute(ctx, "turn_type", turnType)
	if turnErr != nil {
		trace.SetStatus("error")
		if !errors.Is(turnErr, context.Canceled) && turnType != "awaiting_confirm" {
			emitRuntimeError(ctx, sink, turnErr, turnType)
		}
	}

	sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
		"type":    "route-decision",
		"payload": st.Routing,
	}})
	o.emitTracePanels(ctx, sink, turnType)
	sink.Emit(ctx, Event{Type: "done", Data: map[string]any{}})
	return turnErr
}

// acquireSessionLock 在保持既有 Locker 合同的前提下等待会话锁。
// Locker 本身不支持 context；请求取消时当前调用立即返回，随后获得锁的
// 后台等待者会自行释放锁，避免已取消请求继续进入对话执行链。
func acquireSessionLock(ctx context.Context, locker state.Locker, sessionID string) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	acquired := make(chan func())
	go func() {
		unlock := locker.Lock(sessionID)
		select {
		case acquired <- unlock:
		case <-ctx.Done():
			unlock()
		}
	}()

	select {
	case unlock := <-acquired:
		return unlock, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// emitRuntimeError 将可展示的执行失败投影为普通对话文本。
// 错误仍通过返回值和 trace 留给诊断，不能再以独立红色错误块打断对话。
func emitRuntimeError(ctx context.Context, sink EventSink, err error, stage string) {
	if sink == nil || err == nil {
		return
	}
	data := appRuntime.RuntimeFailureEventData(ctx, err, stage)
	sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": data["message"]}})
}

// emitTracePanels emits the single frontend diagnostic contract.
//
// RunInspection is the chat-page debugging surface. Older process/debug/tree
// projections were removed so troubleshooting has one source of truth.
func (o *Orchestrator) emitTracePanels(ctx context.Context, sink EventSink, turnType string) {
	t := tracing.TraceFromContext(ctx)
	if t == nil {
		return
	}
	if t.TurnType == "" {
		t.TurnType = turnType
	}
	inspection := t.BuildRunInspection()
	sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
		"type":    "run-inspection",
		"payload": inspection,
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

// summarizeTraceMessage keeps trace attributes compact while preserving the original prefix.
func summarizeTraceMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if len(msg) <= 120 {
		return msg
	}
	return msg[:120]
}
