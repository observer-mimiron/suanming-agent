package orchestrator

import (
	"context"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/tracing"
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

	if len(sink.events) != 3 {
		t.Fatalf("events = %d, want 3", len(sink.events))
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
	data2, ok := sink.events[2].Data.(map[string]any)
	if !ok {
		t.Fatalf("event 2 data type = %T, want map[string]any", sink.events[2].Data)
	}
	if data2["type"] != "execution-tree" {
		t.Fatalf("event 2 component type = %v, want execution-tree", data2["type"])
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
