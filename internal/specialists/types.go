package specialists

import (
	"context"

	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// Event is a single event emitted during specialist execution.
type Event struct {
	Type string
	Data any
}

// EventSink receives events from a specialist.
type EventSink func(ctx context.Context, evt Event) error

// ApprovedRoute is the policy-gate-approved execution route.
// In Task 7 this will be aligned with policy.ApprovedRoute.
type ApprovedRoute struct {
	ConversationIntent    string
	PrimaryDomain         string
	SecondaryDomains      []string
	TaskIntent            string
	NeedsClarification    bool
	ClarificationQuestion string
	ParallelAllowed       bool
	Slots                 schemas.DecisionSlots
	PolicyHints           schemas.PolicyHints
}

// DomainHandler is the contract that every domain specialist must implement.
type DomainHandler interface {
	Name() string
	Run(ctx context.Context, st *state.SessionState, route ApprovedRoute, sink EventSink) (schemas.DomainResult, error)
}
