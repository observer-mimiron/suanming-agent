package tracing

import (
	"testing"
	"time"
)

func TestBuildDebugDigest_PreservesRawSpansIncludingSSEEmit(t *testing.T) {
	now := time.Now()
	tr := &TurnTrace{
		TraceID:   "trc_debug",
		TurnType:  "agent_reading",
		StartedAt: now,
		Status:    "ok",
		Spans: []TraceSpan{
			{SpanID: "root", Name: "chat.turn", Kind: KindAgent, Status: "ok", DurationMs: 100},
			{SpanID: "s1", ParentSpanID: "root", Name: "sse_emit", Kind: KindChain, Status: "ok", DurationMs: 1, Attributes: map[string]any{"event_type": "text"}},
		},
	}

	debug := tr.BuildDebugDigest()
	if len(debug.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(debug.Steps))
	}
	if debug.Steps[1].Name != "sse_emit" {
		t.Fatalf("step 1 name = %q, want sse_emit", debug.Steps[1].Name)
	}
	if debug.Steps[1].Meta["event_type"] != "text" {
		t.Fatalf("step 1 event_type = %v, want text", debug.Steps[1].Meta["event_type"])
	}
}
