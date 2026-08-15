// This test file belongs to the session orchestration layer.
// It verifies orchestrator trace projection behavior and protects the related contract from regressions.
// It owns turn lifecycle; domain reasoning stays in runtime and tools.
package orchestrator

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	appRuntime "github.com/observer-mimiron/suanming-agent/internal/runtime"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

type cancelableWaitLocker struct {
	waiting  chan struct{}
	release  chan struct{}
	unlocked chan struct{}
	started  sync.Once
	finished sync.Once
}

func newCancelableWaitLocker() *cancelableWaitLocker {
	return &cancelableWaitLocker{
		waiting:  make(chan struct{}),
		release:  make(chan struct{}),
		unlocked: make(chan struct{}),
	}
}

func (l *cancelableWaitLocker) Lock(string) func() {
	l.started.Do(func() { close(l.waiting) })
	<-l.release
	return func() { l.finished.Do(func() { close(l.unlocked) }) }
}

type recordingSink struct {
	events []Event
}

func (s *recordingSink) Emit(_ context.Context, evt Event) error {
	s.events = append(s.events, evt)
	return nil
}

type staticRouteAdvisor struct {
	route policy.ApprovedRoute
}

func (a staticRouteAdvisor) Approve(_ context.Context, _ string, _ *state.SessionState) (policy.ApprovedRoute, error) {
	return a.route, nil
}

func TestEmitTracePanels_SendsRunInspectionComponent(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()
	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	tr.TurnType = "agent_reading"
	tr.Status = "ok"
	root := tr.Spans[0]
	root.DurationMs = 1000
	tr.Spans = []tracing.TraceSpan{
		root,
		{SpanID: "s1", ParentSpanID: root.SpanID, Name: "supervisor_decision", Kind: tracing.KindChain, Status: "ok", DurationMs: 10},
		{SpanID: "s2", ParentSpanID: root.SpanID, Name: "sse_emit", Kind: tracing.KindChain, Status: "ok", DurationMs: 1, Attributes: map[string]any{"event_type": "text"}},
	}

	sink := &recordingSink{}
	orc := &Orchestrator{}
	orc.emitTracePanels(ctx, sink, "agent_reading")

	if len(sink.events) != 1 {
		t.Fatalf("events = %d, want 1", len(sink.events))
	}
	if sink.events[0].Type != "component" {
		t.Fatalf("event 0 type = %q, want component", sink.events[0].Type)
	}
	data0, ok := sink.events[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("event 0 data type = %T, want map[string]any", sink.events[0].Data)
	}
	if data0["type"] != "run-inspection" {
		t.Fatalf("event 0 component type = %v, want run-inspection", data0["type"])
	}
	payload, ok := data0["payload"].(tracing.RunInspection)
	if !ok {
		t.Fatalf("payload = %T, want tracing.RunInspection", data0["payload"])
	}
	if payload.TraceID == "" {
		t.Fatal("run inspection trace_id must be populated")
	}
}

func TestRun_EmitsConversationPromptAndDoneOnRuntimeFailure(t *testing.T) {
	exec, err := appRuntime.NewExecutor(tools.NewRegistry(), specialists.NewRegistry(), nil, nil, nil, appRuntime.ExecutorConfig{})
	if err != nil {
		t.Fatalf("NewExecutor() error = %v", err)
	}
	orc := New(exec, nil, state.NewPersistentStore(""), state.NewMemoryLocker(), tracing.NewRealTracer(nil))
	orc.SetSupervisor(staticRouteAdvisor{route: policy.ApprovedRoute{
		ConsultationKind: contracts.ConsultationKindEventQuestion,
		PrimaryDomain:    "qimen",
		TaskIntent:       "fortune_followup",
	}})
	sink := &recordingSink{}

	err = orc.Run(context.Background(), sink, "sess-runtime-error", "今天运气怎么样")
	if err == nil {
		t.Fatal("Run() error = nil, want runtime failure")
	}

	var textIndex, doneIndex = -1, -1
	for i, evt := range sink.events {
		switch evt.Type {
		case "text":
			textIndex = i
			data, ok := evt.Data.(map[string]any)
			if !ok {
				t.Fatalf("text event data = %T, want map", evt.Data)
			}
			if got := data["content"]; got != "本轮没有拿到必需的命盘或问事盘结果，无法继续解释。请稍后重试。" {
				t.Fatalf("conversation prompt = %v, want artifact-missing prompt", got)
			}
		case "done":
			doneIndex = i
		}
	}
	if textIndex < 0 {
		t.Fatalf("events missing conversation prompt: %+v", sink.events)
	}
	if doneIndex < 0 {
		t.Fatalf("events missing done: %+v", sink.events)
	}
	if textIndex > doneIndex {
		t.Fatalf("conversation prompt must be emitted before done, text=%d done=%d", textIndex, doneIndex)
	}
}

func TestRootTraceCanExposeLangfuseReadableInputOutput(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	tracing.SetTraceAttributes(ctx, map[string]any{
		"session_id":           "sess-root-io",
		"user_message_summary": "帮我分析八字",
		"input.value":          "帮我分析八字",
		"output.value":         "这是最终回答。",
	})

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	if got := tr.Attributes["input.value"]; got != "帮我分析八字" {
		t.Fatalf("input.value = %v, want 帮我分析八字", got)
	}
	if got := tr.Attributes["langfuse.trace.input"]; got != "帮我分析八字" {
		t.Fatalf("langfuse.trace.input = %v, want 帮我分析八字", got)
	}
	if got := tr.Attributes["output.value"]; got != "这是最终回答。" {
		t.Fatalf("output.value = %v, want 这是最终回答。", got)
	}
	if got := tr.Attributes["langfuse.trace.output"]; got != "这是最终回答。" {
		t.Fatalf("langfuse.trace.output = %v, want 这是最终回答。", got)
	}
}

func TestAcquireSessionLock_ReturnsPromptlyWhenContextCancelled(t *testing.T) {
	locker := newCancelableWaitLocker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)

	go func() {
		_, err := acquireSessionLock(ctx, locker, "same-session")
		result <- err
	}()
	select {
	case <-locker.waiting:
	case <-time.After(time.Second):
		t.Fatal("lock acquisition did not begin")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("acquireSessionLock() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("acquireSessionLock() did not return after cancellation")
	}
	close(locker.release)
	select {
	case <-locker.unlocked:
	case <-time.After(time.Second):
		t.Fatal("deferred lock waiter did not release the lock")
	}
}
