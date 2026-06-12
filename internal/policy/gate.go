package policy

import (
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// ApprovedRoute is the policy-gate-approved execution route.
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

// Apply validates a supervisor decision against phase-1 policy rules.
// Stub — real implementation in Task 7.
func Apply(decision schemas.SupervisorDecision, st *state.SessionState) ApprovedRoute {
	return ApprovedRoute{
		ConversationIntent:    decision.ConversationIntent,
		PrimaryDomain:         decision.PrimaryDomain,
		SecondaryDomains:      decision.SecondaryDomains,
		TaskIntent:            decision.TaskIntent,
		NeedsClarification:    decision.NeedsClarification,
		ClarificationQuestion: decision.ClarificationQuestion,
		ParallelAllowed:       decision.Parallelizable,
		Slots:                 decision.Slots,
		PolicyHints:           decision.PolicyHints,
	}
}
