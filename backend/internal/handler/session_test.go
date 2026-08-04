// This test file belongs to the HTTP and SSE adapter layer.
// It verifies session API behavior and protects the related contract from regressions.
// It adapts HTTP/SSE; route approval and domain execution stay below this layer.
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

type stubPeekStore struct {
	session *state.SessionState
	ok      bool
}

func (s stubPeekStore) Peek(_ string) (*state.SessionState, bool) {
	return s.session, s.ok
}

func TestBuildSessionSnapshot_PreservesHistoryAndRuntimeContracts(t *testing.T) {
	st := state.NewSession("sess-1")
	st.RecordTurn("user", "你好")
	st.RecordTurn("assistant", "你好，我来看看。")
	st.LastInput = contracts.LastInputState{
		PreferredDomain: "bazi",
		QuestionText:    "你好",
	}
	st.Execution = contracts.ExecutionSnapshot{
		PrimaryDomain:      "bazi",
		TaskIntent:         "direct_bazi",
		RequiredArtifacts:  []string{"bazi_chart"},
		ConversationIntent: "consult",
	}

	got := buildSessionSnapshot("", st)
	if got.SessionID != "sess-1" {
		t.Fatalf("SessionID = %q, want sess-1", got.SessionID)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(got.Messages))
	}
	if got.LastInput.PreferredDomain != "bazi" {
		t.Fatalf("LastInput.PreferredDomain = %q, want bazi", got.LastInput.PreferredDomain)
	}
	if got.Execution == nil {
		t.Fatal("Execution = nil, want runtime snapshot")
	}
	if got.Execution.PrimaryDomain != "bazi" {
		t.Fatalf("Execution.PrimaryDomain = %q, want bazi", got.Execution.PrimaryDomain)
	}
}

func TestHandleGetSession_ReturnsSnapshotForExistingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	st := state.NewSession("sess-existing")
	st.RecordTurn("user", "帮我看事业")
	st.Execution = contracts.ExecutionSnapshot{
		PrimaryDomain: "ziwei",
	}

	h := NewSessionHandler(stubPeekStore{session: st, ok: true}, "")
	r := gin.New()
	r.GET("/api/session/:sessionID", h.HandleGetSession)

	req := httptest.NewRequest(http.MethodGet, "/api/session/sess-existing", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var payload SessionSnapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.SessionID != "sess-existing" {
		t.Fatalf("SessionID = %q, want sess-existing", payload.SessionID)
	}
	if payload.Execution == nil || payload.Execution.PrimaryDomain != "ziwei" {
		t.Fatalf("Execution = %+v, want primary_domain=ziwei", payload.Execution)
	}
}

func TestHandleGetSession_RejectsInvalidSessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewSessionHandler(stubPeekStore{}, "")
	r := gin.New()
	r.GET("/api/session/:sessionID", h.HandleGetSession)

	req := httptest.NewRequest(http.MethodGet, "/api/session/../bad", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 400 or 404 based on router normalization", rec.Code)
	}
}

func TestBuildSessionSnapshot_RestoresLatestAssistantSegmentsFromDebugLog(t *testing.T) {
	debugDir := t.TempDir()
	logPath := filepath.Join(debugDir, "235632-sess-restore.jsonl")
	raw := strings.Join([]string{
		`{"session_id":"sess-restore","event_type":"thinking","payload":{"agent":"bazi_graph","text":"先核对格局。"}}`,
		`{"session_id":"sess-restore","event_type":"text","payload":{"content":"## 直接回答\n"}}`,
		`{"session_id":"sess-restore","event_type":"component","payload":{"type":"run-inspection","payload":{"trace_id":"trc_restore","status":"ok","total_ms":12,"summary":{"inspection_text":"本轮运行未发现确定性异常。"},"diagnostics":[],"spans":[]}}}`,
		`{"session_id":"sess-restore","event_type":"done","payload":{}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(logPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	st := state.NewSession("sess-restore")
	st.RecordTurn("user", "帮我看看")
	st.RecordTurn("assistant", "## 直接回答")

	got := buildSessionSnapshot(debugDir, st)
	if len(got.Segments) != 3 {
		t.Fatalf("segments len = %d, want 3", len(got.Segments))
	}
	if got.Segments[0].Type != "thinking" {
		t.Fatalf("segment 0 type = %q, want thinking", got.Segments[0].Type)
	}
	if got.Segments[2].ComponentType != "run-inspection" {
		t.Fatalf("segment 2 component_type = %q, want run-inspection", got.Segments[2].ComponentType)
	}
}
