package repair

import "strings"

const (
	// DefaultMaxTurnRepairAttempts is the first-version whole-turn business repair cap.
	DefaultMaxTurnRepairAttempts = 2
	// DefaultMaxNodeRepairAttempts is the first-version per stage/field repair cap.
	DefaultMaxNodeRepairAttempts = 1
)

// NewState returns a repair state with the fixed v1 budget defaults.
func NewState() State {
	return State{
		MaxTurnRepairAttempts: DefaultMaxTurnRepairAttempts,
		MaxNodeRepairAttempts: DefaultMaxNodeRepairAttempts,
	}
}

// WithDefaultBudget fills missing budget caps without mutating attempts.
func WithDefaultBudget(state State) State {
	if state.MaxTurnRepairAttempts <= 0 {
		state.MaxTurnRepairAttempts = DefaultMaxTurnRepairAttempts
	}
	if state.MaxNodeRepairAttempts <= 0 {
		state.MaxNodeRepairAttempts = DefaultMaxNodeRepairAttempts
	}
	return state
}

// AttemptsFor returns how often the same domain/stage/field was already repaired.
func AttemptsFor(state State, failure Failure) int {
	key := budgetKey(failure.Domain, failure.Stage, failure.Field)
	count := 0
	for _, attempt := range state.Attempts {
		if attempt.Action != ActionRepairNode {
			continue
		}
		if budgetKey(attempt.Domain, attempt.Stage, attempt.Field) == key {
			count++
		}
	}
	return count
}

// BudgetExhausted reports whether another business repair would exceed the
// whole-turn or same-stage/field caps.
func BudgetExhausted(state State, failure Failure) bool {
	state = WithDefaultBudget(state)
	if businessAttemptCount(state) >= state.MaxTurnRepairAttempts {
		return true
	}
	return AttemptsFor(state, failure) >= state.MaxNodeRepairAttempts
}

// RecordAttempt appends one attempt after applying default budget caps.
func RecordAttempt(state State, attempt Attempt) State {
	state = WithDefaultBudget(state)
	state.Attempts = append(state.Attempts, attempt)
	return state
}

func businessAttemptCount(state State) int {
	count := 0
	for _, attempt := range state.Attempts {
		if attempt.Action == ActionRepairNode {
			count++
		}
	}
	return count
}

func budgetKey(domain, stage, field string) string {
	return strings.Join([]string{strings.TrimSpace(domain), strings.TrimSpace(stage), strings.TrimSpace(field)}, "/")
}
