// This file belongs to the manager-owned runtime layer.
// It owns Repair Harness retry budget accounting only.
// It must not classify domain errors or run repair nodes.
package runtime

import "strings"

const (
	// DefaultMaxTurnRepairAttempts is the first-version whole-turn business repair cap.
	DefaultMaxTurnRepairAttempts = 2
	// DefaultMaxNodeRepairAttempts is the first-version per stage/field repair cap.
	DefaultMaxNodeRepairAttempts = 1
)

// NewRepairState returns a repair state with the fixed v1 budget defaults.
func NewRepairState() RepairState {
	return RepairState{
		MaxTurnRepairAttempts: DefaultMaxTurnRepairAttempts,
		MaxNodeRepairAttempts: DefaultMaxNodeRepairAttempts,
	}
}

// WithDefaultRepairBudget fills missing budget caps without mutating attempts.
func WithDefaultRepairBudget(state RepairState) RepairState {
	if state.MaxTurnRepairAttempts <= 0 {
		state.MaxTurnRepairAttempts = DefaultMaxTurnRepairAttempts
	}
	if state.MaxNodeRepairAttempts <= 0 {
		state.MaxNodeRepairAttempts = DefaultMaxNodeRepairAttempts
	}
	return state
}

// RepairAttemptsFor returns how often the same domain/stage/field was already repaired.
func RepairAttemptsFor(state RepairState, failure RepairFailure) int {
	key := repairBudgetKey(failure.Domain, failure.Stage, failure.Field)
	count := 0
	for _, attempt := range state.Attempts {
		if attempt.Action != RepairActionRepairNode {
			continue
		}
		if repairBudgetKey(attempt.Domain, attempt.Stage, attempt.Field) == key {
			count++
		}
	}
	return count
}

// RepairBudgetExhausted reports whether another business repair would exceed
// the whole-turn or same-stage/field caps.
func RepairBudgetExhausted(state RepairState, failure RepairFailure) bool {
	state = WithDefaultRepairBudget(state)
	if businessRepairAttemptCount(state) >= state.MaxTurnRepairAttempts {
		return true
	}
	return RepairAttemptsFor(state, failure) >= state.MaxNodeRepairAttempts
}

// RecordRepairAttempt appends one attempt after applying default budget caps.
func RecordRepairAttempt(state RepairState, attempt RepairAttempt) RepairState {
	state = WithDefaultRepairBudget(state)
	state.Attempts = append(state.Attempts, attempt)
	return state
}

// businessRepairAttemptCount counts only business repair-node attempts so
// transport retries can stay on their separate budget.
func businessRepairAttemptCount(state RepairState) int {
	count := 0
	for _, attempt := range state.Attempts {
		if attempt.Action == RepairActionRepairNode {
			count++
		}
	}
	return count
}

// repairBudgetKey normalizes the per-node budget identity.
func repairBudgetKey(domain, stage, field string) string {
	parts := []string{strings.TrimSpace(domain), strings.TrimSpace(stage), strings.TrimSpace(field)}
	return strings.Join(parts, "/")
}
