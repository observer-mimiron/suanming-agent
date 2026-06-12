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

// Run executes the qimen specialist flow. In phase 1, the result is always
// supplemental — it augments the bazi mainline, never replaces it.
func (s *Specialist) Run(ctx context.Context, st *state.SessionState, route specialists.ApprovedRoute, sink specialists.EventSink) (schemas.DomainResult, error) {
	// Stub — real implementation in Task 12.
	return schemas.DomainResult{Domain: "qimen", Summary: "stub"}, nil
}
