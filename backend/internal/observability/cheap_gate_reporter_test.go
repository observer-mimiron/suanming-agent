package observability

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

func TestCheapGateReporter_Record_AppendsJSONLHit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cheap-gate", "hits.jsonl")
	reporter := NewCheapGateReporter(path)

	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()
	tracing.SetTraceAttribute(ctx, "session_id", "sess-report")

	err := reporter.Record(ctx, "那事业这两年呢", contracts.ExecutionSnapshot{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
		Gate: contracts.GateContract{
			Reason:              "cheap_followup_reuse",
			ExecutionMode:       "reuse_followup",
			ReuseCachedResult:   true,
			ReuseSessionProfile: true,
		},
	})
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "\"decision_source\":\"cheap_followup_reuse\"") {
		t.Fatalf("report = %q, want decision_source", text)
	}
	if !strings.Contains(text, "\"session_id\":\"sess-report\"") {
		t.Fatalf("report = %q, want session_id", text)
	}
}
