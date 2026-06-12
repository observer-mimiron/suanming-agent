package qimen

import (
	"context"

	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// Specialist is the qimen domain specialist. In phase 1 it acts as a thin
// boundary layer that provides supplemental timing insights.
type Specialist struct{}

// New creates a qimen specialist.
func New() *Specialist {
	return &Specialist{}
}

// Name returns the specialist identifier.
func (s *Specialist) Name() string {
	return "qimen"
}

// isTimingRelevant returns true if the route indicates a timing-oriented question.
func isTimingRelevant(route specialists.ApprovedRoute) bool {
	if route.PolicyHints.NeedsQimen {
		return true
	}
	switch route.TaskIntent {
	case "timing_followup", "cross_domain_consult":
		return true
	}
	return false
}

// Run executes the qimen specialist flow. In phase 1, the result is always
// supplemental (Final=false) — it augments the bazi mainline, never replaces it.
// Non-timing routes are skipped (returned with empty domain).
func (s *Specialist) Run(ctx context.Context, st *state.SessionState, route specialists.ApprovedRoute, sink specialists.EventSink) (schemas.DomainResult, error) {
	if sink != nil {
		sink(ctx, specialists.Event{Type: "specialist_qimen", Data: route.TaskIntent})
	}

	// Skip non-timing routes — qimen only activates for timing questions.
	if !isTimingRelevant(route) {
		return schemas.DomainResult{
			Domain: "",
			Final:  false,
		}, nil
	}

	// Phase 1: qimen result is always supplemental.
	// The orchestrator handles actual qimen tool invocation and SSE.
	return schemas.DomainResult{
		Domain: "qimen",
		Summary: "qimen timing supplement",
		StructuredData: map[string]any{
			"stage":      "supplemental",
			"time_scope": route.Slots.TimeScope,
		},
		Final: false,
	}, nil
}
