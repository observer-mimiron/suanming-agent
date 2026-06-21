package tracing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestTurnTrace_BuildDigest(t *testing.T) {
	now := time.Now()
	tr := &TurnTrace{
		TraceID:   "trc_test",
		SessionID: "sess_1",
		TurnType:  "full_reading",
		StartedAt: now,
		Status:    "ok",
		Spans: []TraceSpan{
			{SpanID: "root", Name: "chat.turn", Kind: KindAgent, Status: "ok", StartedAt: now, DurationMs: 1000},
			{SpanID: "s1", ParentSpanID: "root", Name: "classify_and_extract", Kind: KindChain, Status: "ok", DurationMs: 100},
			{SpanID: "s2", ParentSpanID: "root", Name: "bazi_calc", Kind: KindTool, Status: "ok", DurationMs: 17},
			{SpanID: "s3", ParentSpanID: "root", Name: "knowledge_search", Kind: KindRetriever, Status: "degraded", DurationMs: 50, Attributes: map[string]any{"hits": 0}},
			{SpanID: "s4", ParentSpanID: "root", Name: "qimen_dunjia", Kind: KindTool, Status: "fallback", DurationMs: 200, Error: "timeout"},
			{SpanID: "s5", ParentSpanID: "root", Name: "llm_generate", Kind: KindLLM, Status: "ok", DurationMs: 500, Attributes: map[string]any{"model": "deepseek-v4-pro", "output_tokens": nil}},
		},
	}

	digest := tr.BuildDigest()

	if digest.TraceID != "trc_test" {
		t.Errorf("TraceID = %s, want trc_test", digest.TraceID)
	}
	if digest.TurnType != "full_reading" {
		t.Errorf("TurnType = %s, want full_reading", digest.TurnType)
	}
	if digest.Status != "ok" {
		t.Errorf("Status = %s, want ok", digest.Status)
	}

	// Should skip the AGENT span
	if len(digest.Steps) != 5 {
		t.Fatalf("steps = %d, want 5 (AGENT span skipped)", len(digest.Steps))
	}

	// LLM span should have model in meta
	llmStep := digest.Steps[4]
	if llmStep.Label != "命理解读" {
		t.Errorf("LLM step label = %s, want 命理解读", llmStep.Label)
	}
	if llmStep.Meta["model"] != "deepseek-v4-pro" {
		t.Errorf("LLM model = %v", llmStep.Meta["model"])
	}
	if llmStep.Meta["output_tokens"] != nil {
		t.Errorf("LLM output_tokens should be nil, got %v", llmStep.Meta["output_tokens"])
	}

	// Knowledge search: Status field is "degraded" directly, not via attribute hack
	ksStep := digest.Steps[2]
	if ksStep.Status != "degraded" {
		t.Errorf("knowledge_search status = %s, want degraded (from Status field, not attribute hack)", ksStep.Status)
	}

	// Qimen: Status field is "fallback" directly
	qmStep := digest.Steps[3]
	if qmStep.Status != "fallback" {
		t.Errorf("qimen status = %s, want fallback (from Status field)", qmStep.Status)
	}

	// Step labels
	if digest.Steps[0].Label != "意图识别" {
		t.Errorf("step 0 label = %s", digest.Steps[0].Label)
	}
	if digest.Steps[1].Label != "八字排盘" {
		t.Errorf("step 1 label = %s", digest.Steps[1].Label)
	}
}

func TestTurnTrace_BuildDigest_ErrorTurn(t *testing.T) {
	tr := &TurnTrace{
		TraceID:   "trc_err",
		SessionID: "sess_1",
		TurnType:  "full_reading",
		StartedAt: time.Now(),
		Status:    "error",
		Spans: []TraceSpan{
			{SpanID: "root", Name: "chat.turn", Kind: KindAgent, Status: "error"},
			{SpanID: "s1", ParentSpanID: "root", Name: "bazi_calc", Kind: KindTool, Status: "error", Error: "排盘失败"},
		},
	}

	digest := tr.BuildDigest()
	if digest.Status != "error" {
		t.Errorf("digest.Status = %s, want error", digest.Status)
	}
	if len(digest.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(digest.Steps))
	}
	if digest.Steps[0].Status != "error" {
		t.Errorf("bazi_calc step status = %s, want error", digest.Steps[0].Status)
	}
}

func TestTurnTrace_BuildDigest_UsesEndedAtWhenPresent(t *testing.T) {
	start := time.Now().Add(-10 * time.Second)
	end := start.Add(1500 * time.Millisecond)
	tr := &TurnTrace{
		TraceID:   "trc_ended",
		SessionID: "sess_1",
		TurnType:  "full_reading",
		StartedAt: start,
		EndedAt:   end,
		Status:    "ok",
		Spans: []TraceSpan{
			{SpanID: "root", Name: "chat.turn", Kind: KindAgent, Status: "ok", StartedAt: start, EndedAt: end, DurationMs: 1500},
		},
	}

	digest := tr.BuildDigest()
	if digest.TotalMs != 1500 {
		t.Fatalf("digest.TotalMs = %d, want 1500 from EndedAt-StartedAt", digest.TotalMs)
	}
}

func TestTurnTrace_BuildDigest_MapsSupervisorLabels(t *testing.T) {
	now := time.Now()
	tr := &TurnTrace{
		TraceID:   "trc_supervisor",
		SessionID: "sess_1",
		TurnType:  "followup_reading",
		StartedAt: now,
		Status:    "ok",
		Spans: []TraceSpan{
			{SpanID: "root", Name: "chat.turn", Kind: KindAgent, Status: "ok", StartedAt: now, DurationMs: 1000},
			{SpanID: "s1", ParentSpanID: "root", Name: "supervisor_decision", Kind: KindChain, Status: "ok", DurationMs: 10},
			{SpanID: "s2", ParentSpanID: "root", Name: "policy_gate", Kind: KindChain, Status: "ok", DurationMs: 8},
			{SpanID: "s3", ParentSpanID: "root", Name: "domain_dispatch", Kind: KindChain, Status: "ok", DurationMs: 6},
		},
	}

	digest := tr.BuildDigest()
	if len(digest.Steps) != 3 {
		t.Fatalf("steps = %d, want 3", len(digest.Steps))
	}
	if digest.Steps[0].Label != "路由决策" {
		t.Fatalf("step 0 label = %q, want %q", digest.Steps[0].Label, "路由决策")
	}
	if digest.Steps[1].Label != "策略校验" {
		t.Fatalf("step 1 label = %q, want %q", digest.Steps[1].Label, "策略校验")
	}
	if digest.Steps[2].Label != "领域调度" {
		t.Fatalf("step 2 label = %q, want %q", digest.Steps[2].Label, "领域调度")
	}
}

func TestTurnTrace_BuildDigest_SkipsSSEEmitSteps(t *testing.T) {
	now := time.Now()
	tr := &TurnTrace{
		TraceID:   "trc_sse_skip",
		SessionID: "sess_1",
		TurnType:  "agent_reading",
		StartedAt: now,
		Status:    "ok",
		Spans: []TraceSpan{
			{SpanID: "root", Name: "chat.turn", Kind: KindAgent, Status: "ok", StartedAt: now, DurationMs: 1000},
			{SpanID: "s1", ParentSpanID: "root", Name: "knowledge_search", Kind: KindRetriever, Status: "ok", DurationMs: 20},
			{SpanID: "s2", ParentSpanID: "root", Name: "sse_emit", Kind: KindChain, Status: "ok", DurationMs: 1, Attributes: map[string]any{"event_type": "text"}},
		},
	}

	digest := tr.BuildDigest()
	if len(digest.Steps) != 1 {
		t.Fatalf("steps = %d, want 1 after skipping sse_emit", len(digest.Steps))
	}
	if digest.Steps[0].Label != "知识检索" {
		t.Fatalf("step 0 label = %q, want %q", digest.Steps[0].Label, "知识检索")
	}
}

func TestFileCollector_Save(t *testing.T) {
	dir := t.TempDir()
	fc := NewFileCollector(dir)

	now := time.Now()
	tr := &TurnTrace{
		TraceID:   "trc_test_save",
		SessionID: "sess_1",
		TurnType:  "full_reading",
		StartedAt: now,
		EndedAt:   now.Add(time.Second),
		Status:    "ok",
		Spans: []TraceSpan{
			{SpanID: "root", Name: "chat.turn", Kind: KindAgent, Status: "ok", StartedAt: now, EndedAt: now.Add(time.Second), DurationMs: 1000},
		},
	}

	if err := fc.Save(tr); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	dateDir := now.Format("2006-01-02")
	path := filepath.Join(dir, dateDir, "trc_test_save.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var loaded TurnTrace
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if loaded.TraceID != "trc_test_save" {
		t.Errorf("TraceID = %s", loaded.TraceID)
	}
	if loaded.SessionID != "sess_1" {
		t.Errorf("SessionID = %s", loaded.SessionID)
	}
}

func TestFileCollector_SaveNil(t *testing.T) {
	fc := NewFileCollector(t.TempDir())
	if err := fc.Save(nil); err == nil {
		t.Error("expected error for nil trace")
	}
}

func TestRealTracer_CreatesTrace(t *testing.T) {
	rt := NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")

	tr := TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	if !strings.HasPrefix(tr.TraceID, "trc_") {
		t.Errorf("TraceID = %s, want trc_ prefix", tr.TraceID)
	}

	sp := trace.StartSpan("classify")
	sp.SetKind(KindChain)
	sp.SetAttribute("action", "new_profile")
	sp.End()

	trace.End()

	if len(tr.Spans) != 2 {
		t.Fatalf("spans = %d, want 2", len(tr.Spans))
	}
	if tr.Status != "ok" {
		t.Errorf("Status = %s, want ok", tr.Status)
	}
}

func TestRealTracer_ErrorTurnStatus(t *testing.T) {
	rt := NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")

	trace.SetStatus("error")
	trace.End()

	tr := TraceFromContext(ctx)
	if tr.Status != "error" {
		t.Errorf("TurnTrace.Status = %s, want error after trace.SetStatus(\"error\")", tr.Status)
	}

	// Digest should also show error
	digest := tr.BuildDigest()
	if digest.Status != "error" {
		t.Errorf("digest.Status = %s, want error", digest.Status)
	}
}

func TestRealTracer_SetStatusOnSpan(t *testing.T) {
	rt := NewRealTracer(nil)
	ctx, _ := rt.StartTrace(context.Background(), "chat.turn")

	// Use SpanFromContext to create a child span and set degraded status
	sp := SpanFromContext(ctx, "knowledge_search", KindRetriever)
	sp.SetStatus("degraded")
	sp.SetAttribute("hits", 0)
	sp.End()

	sp2 := SpanFromContext(ctx, "qimen_dunjia", KindTool)
	sp2.SetStatus("fallback")
	sp2.RecordError(os.ErrNotExist)
	sp2.End()

	// Close the incomplete trace manually
	tr := TraceFromContext(ctx)
	tr.EndedAt = time.Now()
	// Verify raw trace has correct status on spans
	var ksSpan, qmSpan *TraceSpan
	for i := range tr.Spans {
		switch tr.Spans[i].Name {
		case "knowledge_search":
			ksSpan = &tr.Spans[i]
		case "qimen_dunjia":
			qmSpan = &tr.Spans[i]
		}
	}
	if ksSpan == nil || ksSpan.Status != "degraded" {
		t.Errorf("knowledge_search span status = %s, want degraded (from SetStatus, not attribute)", ksSpan.Status)
	}
	if qmSpan == nil || qmSpan.Status != "fallback" {
		t.Errorf("qimen span status = %s, want fallback", qmSpan.Status)
	}

	// Digest should match raw trace status
	digest := tr.BuildDigest()
	for _, s := range digest.Steps {
		switch s.Label {
		case "知识检索":
			if s.Status != "degraded" {
				t.Errorf("digest knowledge_search = %s, want degraded", s.Status)
			}
		case "奇门遁甲":
			if s.Status != "fallback" {
				t.Errorf("digest qimen = %s, want fallback", s.Status)
			}
		}
	}
}

func TestRealTracer_NoCollectorStillProvidesTrace(t *testing.T) {
	// Simulates default config (DEBUG_TRACE=0): real tracer with nil collector
	rt := NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")

	tr := TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("RealTracer(nil) must still create trace for UI digest — trace is nil")
	}

	// SpanFromContext must work even without collector
	sp := SpanFromContext(ctx, "classify", KindChain)
	sp.SetStatus("ok")
	sp.End()

	trace.End()

	if len(tr.Spans) < 2 {
		t.Errorf("spans = %d, want at least 2 (root + child)", len(tr.Spans))
	}

	// Digest must be buildable
	digest := tr.BuildDigest()
	if digest.TraceID == "" {
		t.Error("digest has empty trace_id")
	}
}

func TestMiddleware_SkipsChatRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tracer := NewRealTracer(nil)
	router := gin.New()
	router.Use(Middleware(tracer))
	router.POST("/api/chat", func(c *gin.Context) {
		if TraceFromContext(c.Request.Context()) != nil {
			t.Fatal("expected no middleware trace for /api/chat")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/chat", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
