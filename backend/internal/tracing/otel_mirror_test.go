// This test file belongs to the trace projection layer.
// It verifies OpenTelemetry mirror behavior and protects the related contract from regressions.
// It projects runtime evidence; it must not change execution decisions.
package tracing

import (
	"context"
	"testing"
)

type fakeOTelBridge struct {
	root     *fakeOTelSpan
	children []*fakeOTelSpan
}

type fakeOTelSpan struct {
	name   string
	kind   SpanKind
	attrs  map[string]any
	status string
	ended  bool
}

func (b *fakeOTelBridge) StartRoot(_ context.Context, name string, kind SpanKind) (context.Context, otelSpanBridge) {
	b.root = &fakeOTelSpan{name: name, kind: kind, attrs: map[string]any{}}
	return context.Background(), b.root
}

func (b *fakeOTelBridge) StartChild(_ context.Context, name string, kind SpanKind) otelSpanBridge {
	span := &fakeOTelSpan{name: name, kind: kind, attrs: map[string]any{}}
	b.children = append(b.children, span)
	return span
}

func (s *fakeOTelSpan) SetAttribute(key string, value any) {
	s.attrs[key] = value
}

func (s *fakeOTelSpan) SetStatus(status string) {
	s.status = status
}

func (s *fakeOTelSpan) RecordError(err error) {
	if err != nil {
		s.attrs["error"] = err.Error()
	}
}

func (s *fakeOTelSpan) End() { s.ended = true }

func TestRealTracer_OTelMirrorReceivesRootAndChildSpans(t *testing.T) {
	bridge := &fakeOTelBridge{}
	rt := newRealTracerWithBridge(nil, bridge)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")

	SetTraceAttributes(ctx, map[string]any{
		"session_id":   "sess-otel",
		"turn_type":    "agent_reading",
		"input.value":  "帮我分析八字",
		"output.value": "这是最终回答",
	})

	sp := SpanFromContext(ctx, "prefill", KindChain)
	sp.SetAttribute("cache_hit", true)
	sp.SetStatus("degraded")
	sp.End()

	trace.End()

	if bridge.root == nil {
		t.Fatal("expected root OTel mirror span")
	}
	if bridge.root.name != "chat.turn" {
		t.Fatalf("root span name = %q, want chat.turn", bridge.root.name)
	}
	if got := bridge.root.attrs["session_id"]; got != "sess-otel" {
		t.Fatalf("root session_id = %v, want sess-otel", got)
	}
	if got := bridge.root.attrs["langfuse.session.id"]; got != "sess-otel" {
		t.Fatalf("root langfuse.session.id = %v, want sess-otel", got)
	}
	if got := bridge.root.attrs["turn_type"]; got != "agent_reading" {
		t.Fatalf("root turn_type = %v, want agent_reading", got)
	}
	if got := bridge.root.attrs["langfuse.trace.input"]; got != "帮我分析八字" {
		t.Fatalf("root langfuse.trace.input = %v, want 帮我分析八字", got)
	}
	if got := bridge.root.attrs["langfuse.trace.output"]; got != "这是最终回答" {
		t.Fatalf("root langfuse.trace.output = %v, want 这是最终回答", got)
	}
	if !bridge.root.ended {
		t.Fatal("expected root OTel span to end")
	}

	if len(bridge.children) != 1 {
		t.Fatalf("child span count = %d, want 1", len(bridge.children))
	}
	child := bridge.children[0]
	if child.name != "prefill" {
		t.Fatalf("child span name = %q, want prefill", child.name)
	}
	if got := child.attrs["cache_hit"]; got != true {
		t.Fatalf("child cache_hit = %v, want true", got)
	}
	if got := child.attrs["session_id"]; got != "sess-otel" {
		t.Fatalf("child session_id = %v, want sess-otel", got)
	}
	if got := child.attrs["langfuse.session.id"]; got != "sess-otel" {
		t.Fatalf("child langfuse.session.id = %v, want sess-otel", got)
	}
	if got := child.attrs["langfuse.trace.input"]; got != "帮我分析八字" {
		t.Fatalf("child langfuse.trace.input = %v, want 帮我分析八字", got)
	}
	if got := child.attrs["langfuse.trace.output"]; got != "这是最终回答" {
		t.Fatalf("child langfuse.trace.output = %v, want 这是最终回答", got)
	}
	if child.status != "degraded" {
		t.Fatalf("child status = %q, want degraded", child.status)
	}
	if !child.ended {
		t.Fatal("expected child OTel span to end")
	}
}

func TestExpandTraceAttributeAliases(t *testing.T) {
	attrs := expandTraceAttributeAliases("session_id", "sess-1")
	if got := attrs["session_id"]; got != "sess-1" {
		t.Fatalf("session_id = %v", got)
	}
	if got := attrs["langfuse.session.id"]; got != "sess-1" {
		t.Fatalf("langfuse.session.id = %v", got)
	}

	attrs = expandTraceAttributeAliases("user_id", "user-1")
	if got := attrs["langfuse.user.id"]; got != "user-1" {
		t.Fatalf("langfuse.user.id = %v", got)
	}
}
