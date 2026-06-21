package orchestrator

import (
	"context"
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

type recordingSink struct {
	events []Event
}

func (s *recordingSink) Emit(_ context.Context, evt Event) error {
	s.events = append(s.events, evt)
	return nil
}

func TestEmitTracePanels_SendsProcessAndDebugComponents(t *testing.T) {
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

	if len(sink.events) != 2 {
		t.Fatalf("events = %d, want 2", len(sink.events))
	}
	if sink.events[0].Type != "component" {
		t.Fatalf("event 0 type = %q, want component", sink.events[0].Type)
	}
	data0, ok := sink.events[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("event 0 data type = %T, want map[string]any", sink.events[0].Data)
	}
	if data0["type"] != "process-panel" {
		t.Fatalf("event 0 component type = %v, want process-panel", data0["type"])
	}
	data1, ok := sink.events[1].Data.(map[string]any)
	if !ok {
		t.Fatalf("event 1 data type = %T, want map[string]any", sink.events[1].Data)
	}
	if data1["type"] != "debug-trace" {
		t.Fatalf("event 1 component type = %v, want debug-trace", data1["type"])
	}
}
