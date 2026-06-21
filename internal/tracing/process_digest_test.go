package tracing

import (
	"testing"
	"time"
)

func TestBuildProcessDigest_GroupsRawSpansIntoUserPhases(t *testing.T) {
	now := time.Now()
	tr := &TurnTrace{
		TraceID:   "trc_process",
		TurnType:  "agent_reading",
		StartedAt: now,
		Status:    "ok",
		Spans: []TraceSpan{
			{SpanID: "root", Name: "chat.turn", Kind: KindAgent, Status: "ok", StartedAt: now, DurationMs: 1000},
			{SpanID: "s1", ParentSpanID: "root", Name: "supervisor_decision", Kind: KindChain, Status: "ok", DurationMs: 20},
			{SpanID: "s2", ParentSpanID: "root", Name: "policy_gate", Kind: KindChain, Status: "ok", DurationMs: 10},
			{SpanID: "s3", ParentSpanID: "root", Name: "knowledge_search", Kind: KindRetriever, Status: "ok", DurationMs: 50, Attributes: map[string]any{"hits": 2}},
			{SpanID: "s4", ParentSpanID: "root", Name: "llm_generate", Kind: KindLLM, Status: "ok", DurationMs: 120, Attributes: map[string]any{"model": "deepseek-v4-pro"}},
			{SpanID: "s5", ParentSpanID: "root", Name: "sse_emit", Kind: KindChain, Status: "ok", DurationMs: 1},
		},
	}

	digest := tr.BuildProcessDigest()
	if digest.TraceID != "trc_process" {
		t.Fatalf("TraceID = %q, want trc_process", digest.TraceID)
	}
	if digest.TurnType != "agent_reading" {
		t.Fatalf("TurnType = %q, want agent_reading", digest.TurnType)
	}
	if len(digest.Phases) != 4 {
		t.Fatalf("phases = %d, want 4", len(digest.Phases))
	}
	if digest.Phases[0].Label != "路由判断" {
		t.Fatalf("phase 0 label = %q, want 路由判断", digest.Phases[0].Label)
	}
	if digest.Phases[len(digest.Phases)-1].Label != "结果验收与生成" {
		t.Fatalf("last phase label = %q, want 结果验收与生成", digest.Phases[len(digest.Phases)-1].Label)
	}
	if digest.Phases[2].Meta["hits"] != 2 {
		t.Fatalf("prepare phase hits = %v, want 2", digest.Phases[2].Meta["hits"])
	}
	if digest.Phases[3].Meta["model"] != "deepseek-v4-pro" {
		t.Fatalf("answer phase model = %v, want deepseek-v4-pro", digest.Phases[3].Meta["model"])
	}
	if digest.Phases[2].Summary == "" {
		t.Fatal("prepare phase summary should not be empty")
	}
}

func TestBuildProcessDigest_RollsUpWorstStatusPerPhase(t *testing.T) {
	now := time.Now()
	tr := &TurnTrace{
		TraceID:   "trc_process_status",
		TurnType:  "agent_reading",
		StartedAt: now,
		Status:    "fallback",
		Spans: []TraceSpan{
			{SpanID: "root", Name: "chat.turn", Kind: KindAgent, Status: "fallback", StartedAt: now, DurationMs: 1000},
			{SpanID: "s1", ParentSpanID: "root", Name: "prefill", Kind: KindChain, Status: "ok", DurationMs: 15},
			{SpanID: "s2", ParentSpanID: "root", Name: "bazi_calc", Kind: KindTool, Status: "fallback", DurationMs: 25},
		},
	}

	digest := tr.BuildProcessDigest()
	if len(digest.Phases) != 1 {
		t.Fatalf("phases = %d, want 1", len(digest.Phases))
	}
	if digest.Phases[0].Status != "fallback" {
		t.Fatalf("phase status = %q, want fallback", digest.Phases[0].Status)
	}
	if digest.Phases[0].Ms != 40 {
		t.Fatalf("phase ms = %d, want 40", digest.Phases[0].Ms)
	}
}
