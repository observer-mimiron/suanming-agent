// This file belongs to the local observability layer.
// It owns cheap-gate reporting for this package.
// It summarizes evidence for operators; it is not an acceptance source by itself.
package observability

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// CheapGateHit records a single cheap gate reuse decision for later rollup and
// sampling. It is intentionally append-only and line-oriented.
type CheapGateHit struct {
	Timestamp           string `json:"timestamp"`
	SessionID           string `json:"session_id,omitempty"`
	TraceID             string `json:"trace_id,omitempty"`
	Message             string `json:"message,omitempty"`
	PrimaryDomain       string `json:"primary_domain,omitempty"`
	TaskIntent          string `json:"task_intent,omitempty"`
	DecisionSource      string `json:"decision_source,omitempty"`
	GateReason          string `json:"gate_reason,omitempty"`
	ExecutionMode       string `json:"execution_mode,omitempty"`
	ReuseCachedResult   bool   `json:"reuse_cached_result,omitempty"`
	ReuseSessionProfile bool   `json:"reuse_session_profile,omitempty"`
}

// CheapGateReporter appends cheap gate hit samples to a local JSONL report.
type CheapGateReporter struct {
	mu   sync.Mutex
	path string
}

// NewCheapGateReporter creates a local reporter rooted at the given file path.
func NewCheapGateReporter(path string) *CheapGateReporter {
	if path == "" {
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	return &CheapGateReporter{path: path}
}

// Record writes one cheap gate hit sample. Failures are non-fatal to runtime.
func (r *CheapGateReporter) Record(ctx context.Context, message string, snapshot contracts.ExecutionSnapshot) error {
	if r == nil || r.path == "" || snapshot.Gate.Reason != "cheap_followup_reuse" {
		return nil
	}

	hit := CheapGateHit{
		Timestamp:           time.Now().Format(time.RFC3339Nano),
		Message:             message,
		PrimaryDomain:       snapshot.PrimaryDomain,
		TaskIntent:          snapshot.TaskIntent,
		DecisionSource:      "cheap_followup_reuse",
		GateReason:          snapshot.Gate.Reason,
		ExecutionMode:       snapshot.Gate.ExecutionMode,
		ReuseCachedResult:   snapshot.Gate.ReuseCachedResult,
		ReuseSessionProfile: snapshot.Gate.ReuseSessionProfile,
	}
	if tr := tracing.TraceFromContext(ctx); tr != nil {
		hit.TraceID = tr.TraceID
		hit.SessionID = tr.SessionID
		if hit.SessionID == "" {
			if raw, ok := tr.Attributes["session_id"].(string); ok {
				hit.SessionID = raw
			}
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(hit)
}
