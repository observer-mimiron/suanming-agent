package runtime

import (
	"context"
	"errors"

	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

const (
	failureClassArtifactMissing             = "artifact_missing"
	failureClassSpecialistContractViolation = "specialist_contract_violation"

	failureStagePrefill    = "prefill"
	failureStageFinalGuard = "final_guard"
)

// RuntimeFailure is the lightweight structured failure shape shared by runtime
// validation, trace projection, and regression tests.
type RuntimeFailure struct {
	Class       string
	Stage       string
	Domain      string
	Degraded    bool
	UserVisible bool
	Message     string
	Cause       error
}

// Error returns the user-facing runtime failure message.
func (e *RuntimeFailure) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Unwrap returns the underlying cause so errors.As / errors.Is keep working.
func (e *RuntimeFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func annotateRuntimeFailureTrace(ctx context.Context, err error) {
	var rf *RuntimeFailure
	if !errors.As(err, &rf) || rf == nil {
		return
	}
	tracing.SetTraceAttributes(ctx, map[string]any{
		"failure.class":        rf.Class,
		"failure.stage":        rf.Stage,
		"failure.domain":       rf.Domain,
		"failure.degraded":     rf.Degraded,
		"failure.user_visible": rf.UserVisible,
	})
}
