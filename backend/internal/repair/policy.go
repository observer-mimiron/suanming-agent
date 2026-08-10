package repair

import "net/http"

const (
	// DefaultTransportMaxAttempts is the v1 cap for model-call retry phases.
	DefaultTransportMaxAttempts = 2
	// DefaultJSONParseMaxAttempts is the v1 cap for malformed JSON repair phases.
	DefaultJSONParseMaxAttempts = 1
)

// Policy contains the fixed v1 repair limits.
type Policy struct {
	MaxTurnRepairAttempts int
	MaxNodeRepairAttempts int
	TransportMaxAttempts  int
	JSONParseMaxAttempts  int
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
		TransportMaxAttempts:  DefaultTransportMaxAttempts,
		JSONParseMaxAttempts:  DefaultJSONParseMaxAttempts,
	}
}

// Decide selects the next bounded action for a classified repair failure.
func (p Policy) Decide(failure Failure, state State) Decision {
	p = p.withDefaults()
	base := p.actionForFailure(failure)
	decision := Decision{
		Action:      base,
		Retryable:   base == ActionRetry || base == ActionRepairNode,
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

// ClassForHTTPStatus maps model transport status codes into retry classes.
func ClassForHTTPStatus(status int) Class {
	switch status {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusPaymentRequired:
		return TransportFatal
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return TransportTransient
	}
	if status >= 500 && status <= 599 {
		return TransportTransient
	}
	return Unknown
}

// HTTPStatusRetryable reports whether a status may use transport retry.
func HTTPStatusRetryable(status int) bool {
	return ClassForHTTPStatus(status) == TransportTransient
}

func (p Policy) withDefaults() Policy {
	if p.MaxTurnRepairAttempts <= 0 {
		p.MaxTurnRepairAttempts = DefaultMaxTurnRepairAttempts
	}
	if p.MaxNodeRepairAttempts <= 0 {
		p.MaxNodeRepairAttempts = DefaultMaxNodeRepairAttempts
	}
	if p.TransportMaxAttempts <= 0 {
		p.TransportMaxAttempts = DefaultTransportMaxAttempts
	}
	if p.JSONParseMaxAttempts <= 0 {
		p.JSONParseMaxAttempts = DefaultJSONParseMaxAttempts
	}
	return p
}

func (p Policy) actionForFailure(failure Failure) Action {
	switch failure.Class {
	case TransportTransient:
		return ActionRetry
	case ParseError, SchemaError, ProjectionMismatch, EvidenceOverclaim, GuardrailBlocked:
		return ActionRepairNode
	case TransportFatal, FactConflict, MethodContract:
		return ActionHardError
	case DomainUnauthorized:
		return fallbackOrHardError(failure)
	default:
		return fallbackOrHardError(failure)
	}
}

func (p Policy) maxAttemptsForFailure(failure Failure) int {
	switch failure.Class {
	case TransportTransient:
		return p.TransportMaxAttempts
	case ParseError, SchemaError:
		return p.JSONParseMaxAttempts
	default:
		return p.MaxNodeRepairAttempts
	}
}

func fallbackOrHardError(failure Failure) Action {
	if failure.Fallback != "" {
		return ActionFallback
	}
	return ActionHardError
}
