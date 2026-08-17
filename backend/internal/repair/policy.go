package repair

// Policy contains the fixed v1 repair limits.
type Policy struct {
	MaxTurnRepairAttempts int
	MaxNodeRepairAttempts int
}

// Decision is the policy result for one failure and budget state.
type Decision struct {
	Action      Action
	Retryable   bool
	Repairable  bool
	Exhausted   bool
	MaxAttempts int
}

// DefaultPolicy returns the fixed Phase 0 policy.
func DefaultPolicy() Policy {
	return Policy{
		MaxTurnRepairAttempts: DefaultMaxTurnRepairAttempts,
		MaxNodeRepairAttempts: DefaultMaxNodeRepairAttempts,
	}
}

// Decide selects the next bounded action for a classified repair failure.
func (p Policy) Decide(failure Failure, state State) Decision {
	p = p.withDefaults()
	base := p.actionForFailure(failure)
	decision := Decision{
		Action:      base,
		Retryable:   base == ActionRepairNode,
		Repairable:  base == ActionRepairNode,
		MaxAttempts: p.maxAttemptsForFailure(failure),
	}
	if base != ActionRepairNode {
		return decision
	}
	state.MaxTurnRepairAttempts = p.MaxTurnRepairAttempts
	state.MaxNodeRepairAttempts = p.MaxNodeRepairAttempts
	if BudgetExhausted(state, failure) {
		decision.Exhausted = true
		decision.Retryable = false
		decision.Repairable = false
		decision.Action = fallbackOrHardError(failure)
	}
	return decision
}

func (p Policy) withDefaults() Policy {
	if p.MaxTurnRepairAttempts <= 0 {
		p.MaxTurnRepairAttempts = DefaultMaxTurnRepairAttempts
	}
	if p.MaxNodeRepairAttempts <= 0 {
		p.MaxNodeRepairAttempts = DefaultMaxNodeRepairAttempts
	}
	return p
}

func (p Policy) actionForFailure(failure Failure) Action {
	switch failure.Class {
	case ParseError, SchemaError, ProjectionMismatch, EvidenceOverclaim, DomainUnauthorized, FactConflict, MethodContract:
		return ActionRepairNode
	case DeterministicConflict, GuardrailBlocked:
		return ActionHardError
	default:
		return fallbackOrHardError(failure)
	}
}

func (p Policy) maxAttemptsForFailure(failure Failure) int {
	return p.MaxNodeRepairAttempts
}

func fallbackOrHardError(failure Failure) Action {
	if failure.Fallback != "" {
		return ActionFallback
	}
	return ActionHardError
}
