package tracing

import (
	"testing"
	"time"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
)

func TestBuildExecutionTree_GroupsIntoPhases(t *testing.T) {
	now := time.Now()
	tr := &TurnTrace{
		TraceID:   "trc_test",
		TurnType:  "agent_reading",
		StartedAt: now,
		Status:    "ok",
		Spans: []TraceSpan{
			{SpanID: "s1", Name: "supervisor_model", Kind: KindLLM, Status: "ok", DurationMs: 3000, Attributes: map[string]any{"model": "deepseek-v4-flash", "output_tokens": 474}},
			{SpanID: "s2", Name: "output", Kind: KindTool, Status: "ok", DurationMs: 1},
			{SpanID: "s3", Name: "preflight", Kind: KindChain, Status: "ok", DurationMs: 5},
			{SpanID: "s4", Name: "bazi_calc", Kind: KindTool, Status: "ok", DurationMs: 1200},
			{SpanID: "s5", Name: "yongshen", Kind: KindTool, Status: "ok", DurationMs: 800},
			{SpanID: "s6", Name: "knowledge_catalog", Kind: KindTool, Status: "ok", DurationMs: 500},
			{SpanID: "s7", Name: "knowledge_search", Kind: KindRetriever, Status: "ok", DurationMs: 300, Attributes: map[string]any{"hits": 10}},
			{SpanID: "s8", Name: "knowledge_search", Kind: KindRetriever, Status: "ok", DurationMs: 200, Attributes: map[string]any{"hits": 5}},
			{SpanID: "s9", Name: "bazi_specialist", Kind: KindTool, Status: "error", DurationMs: 21700},
			{SpanID: "s10", Name: "sse_emit", Kind: KindChain, Status: "ok", DurationMs: 1, Attributes: map[string]any{"event_type": "text"}},
			{SpanID: "s11", Name: "sse_emit", Kind: KindChain, Status: "ok", DurationMs: 1, Attributes: map[string]any{"event_type": "text"}},
			{SpanID: "s12", Name: "sse_emit", Kind: KindChain, Status: "ok", DurationMs: 1, Attributes: map[string]any{"event_type": "thinking"}},
		},
	}

	tree := tr.BuildExecutionTree()

	if tree.Root.Label != "chat.turn" {
		t.Fatalf("root label = %q, want chat.turn", tree.Root.Label)
	}

	if len(tree.Root.Children) != 6 {
		t.Fatalf("children count = %d, want 6 (5 semantic phases + sse output)", len(tree.Root.Children))
	}

	// Phase 0: routing decision
	ph0 := tree.Root.Children[0]
	if ph0.Label != "路由决策" {
		t.Fatalf("ph0 label = %q, want 路由决策", ph0.Label)
	}
	if len(ph0.Children) != 2 {
		t.Fatalf("ph0 children = %d, want 2 (supervisor_model + output)", len(ph0.Children))
	}

	// Phase 1: preflight
	ph1 := tree.Root.Children[1]
	if ph1.Label != "执行前校验" {
		t.Fatalf("ph1 label = %q, want 执行前校验", ph1.Label)
	}

	// Phase 2: chart calc (index 2, order 2 prefill has no matching spans)
	ph2 := tree.Root.Children[2]
	if ph2.Label != "命盘计算" {
		t.Fatalf("ph2 label = %q, want 命盘计算", ph2.Label)
	}
	if len(ph2.Children) != 2 {
		t.Fatalf("ph2 children = %d, want 2 (bazi_calc + yongshen)", len(ph2.Children))
	}

	// Phase 3: knowledge retrieval
	ph3 := tree.Root.Children[3]
	if ph3.Label != "知识检索" {
		t.Fatalf("ph3 label = %q, want 知识检索", ph3.Label)
	}
	if len(ph3.Children) != 3 {
		t.Fatalf("ph3 children = %d, want 3 (knowledge_catalog + 2x knowledge_search)", len(ph3.Children))
	}

	// Phase 4: specialist analysis — contains error
	ph4 := tree.Root.Children[4]
	if ph4.Status != "error" {
		t.Fatalf("ph4 status = %q, want error", ph4.Status)
	}

	// Phase 5: SSE output — compacted sse_emit_batch
	phSSE := tree.Root.Children[len(tree.Root.Children)-1]
	if phSSE.Label != "SSE 输出" {
		t.Fatalf("last child label = %q, want SSE 输出", phSSE.Label)
	}
	if len(phSSE.Children) != 1 {
		t.Fatalf("sse phase children = %d, want 1 (compacted sse_emit_batch)", len(phSSE.Children))
	}
}

func TestBuildExecutionTree_EmptyTrace(t *testing.T) {
	now := time.Now()
	tr := &TurnTrace{
		TraceID:   "trc_empty",
		StartedAt: now,
		Status:    "ok",
		Spans:     []TraceSpan{},
	}
	tree := tr.BuildExecutionTree()
	if len(tree.Root.Children) != 0 {
		t.Fatalf("empty trace children = %d, want 0", len(tree.Root.Children))
	}
}

func TestBuildExecutionTree_ExposesRuntimeSnapshotMeta(t *testing.T) {
	tr := &TurnTrace{
		TraceID:   "trc_tree_runtime",
		TurnType:  "agent_reading",
		StartedAt: time.Now(),
		Status:    "ok",
	}
	SetLocalExecutionSnapshot(tr, contracts.ExecutionSnapshot{
		PrimaryDomain:     "ziwei",
		RequiredArtifacts: []string{"ziwei_chart"},
	})

	tree := tr.BuildExecutionTree()
	if tree.Runtime == nil {
		t.Fatal("Runtime = nil, want runtime meta")
	}
	if tree.Runtime["primary_domain"] != "ziwei" {
		t.Fatalf("runtime.primary_domain = %v, want ziwei", tree.Runtime["primary_domain"])
	}
}
