// This test file belongs to the HTTP and SSE adapter layer.
// It verifies persisted debug trace responses and protects the handler contract from regressions.
// It adapts HTTP/SSE only; route approval and domain execution stay below this layer.
package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDebugTraceHandler_ReturnsPersistedTrace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	traceDir := t.TempDir()
	dateDir := filepath.Join(traceDir, "2026-08-03")
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	raw := "{\"trace_id\":\"trc_debug_1\",\"user_message\":\"完整用户输入\",\"spans\":[]}"
	if err := os.WriteFile(filepath.Join(dateDir, "trc_debug_1.json"), []byte(raw), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	h := NewDebugTraceHandler(traceDir)
	r := gin.New()
	r.GET("/api/debug/traces/:traceID", h.HandleGetTrace)

	req := httptest.NewRequest(http.MethodGet, "/api/debug/traces/trc_debug_1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "\"user_message\":\"完整用户输入\"") {
		t.Fatalf("raw trace response missing full payload: %s", rec.Body.String())
	}
}

func TestDebugTraceHandler_RejectsUnsafeTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewDebugTraceHandler(t.TempDir())
	r := gin.New()
	r.GET("/api/debug/traces/:traceID", h.HandleGetTrace)

	req := httptest.NewRequest(http.MethodGet, "/api/debug/traces/..%2Fbad", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 400 or 404 based on router normalization", rec.Code)
	}
}

func TestDebugTraceHandler_ReturnsNotFoundForMissingTrace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewDebugTraceHandler(t.TempDir())
	r := gin.New()
	r.GET("/api/debug/traces/:traceID", h.HandleGetTrace)

	req := httptest.NewRequest(http.MethodGet, "/api/debug/traces/trc_missing", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
