package bazi

import (
	"context"

	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// Specialist is the bazi domain specialist. In phase 1 it wraps the existing
// orchestrator chart logic as a thin boundary layer.
type Specialist struct {
	// Dependencies will be wired in Task 10.
}

// New creates a bazi specialist. Accepts nil for phase-1 stub.
func New(_ any) *Specialist {
	return &Specialist{}
}

// Name returns the specialist identifier.
func (s *Specialist) Name() string {
	return "bazi"
}

// Run executes the bazi specialist flow.
func (s *Specialist) Run(ctx context.Context, st *state.SessionState, route specialists.ApprovedRoute, sink specialists.EventSink) (schemas.DomainResult, error) {
	// Stub — real implementation in Task 10.
	return schemas.DomainResult{Domain: "bazi", Summary: "stub"}, nil
}
