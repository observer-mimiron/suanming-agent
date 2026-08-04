// This test file belongs to the trace projection layer.
// It verifies turn trace behavior and protects the related contract from regressions.
// It projects runtime evidence; it must not change execution decisions.
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

	// RunInspection must be buildable even without a collector.
	inspection := tr.BuildRunInspection()
	if inspection.TraceID == "" {
		t.Error("inspection has empty trace_id")
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
