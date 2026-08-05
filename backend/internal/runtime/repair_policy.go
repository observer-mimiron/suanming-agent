// This file belongs to the manager-owned runtime layer.
// It owns global Repair Harness policy decisions after failures are classified.
// It must not perform model calls, domain validation or final answer composition.
package runtime

import "net/http"

const (
	// DefaultTransportMaxAttempts is the v1 cap for model-call retry phases.
	DefaultTransportMaxAttempts = 2
	// DefaultJSONParseMaxAttempts is the v1 cap for malformed JSON repair phases.
	DefaultJSONParseMaxAttempts = 1
)

// RepairPolicy contains the fixed v1 repair limits.
type RepairPolicy struct {
	MaxTurnRepairAttempts int
	MaxNodeRepairAttempts int
	TransportMaxAttempts  int
	JSONParseMaxAttempts  int
}

// RepairDecision is the policy result for one failure and budget state.
type RepairDecision struct {
	Action      RepairAction
	Retryable   bool
	Repairable  bool
	Exhausted   bool
	MaxAttempts int
}

// DefaultRepairPolicy returns the fixed Phase 0 policy.
func DefaultRepairPolicy() RepairPolicy {
	return RepairPolicy{
		MaxTurnRepairAttempts: DefaultMaxTurnRepairAttempts,
		MaxNodeRepairAttempts: DefaultMaxNodeRepairAttempts,
		TransportMaxAttempts:  DefaultTransportMaxAttempts,
		JSONParseMaxAttempts:  DefaultJSONParseMaxAttempts,
	}
}

// Decide selects the next bounded action for a classified repair failure.
func (p RepairPolicy) Decide(failure RepairFailure, state RepairState) RepairDecision {
	p = p.withDefaults()
	base := p.actionForFailure(failure)
	decision := RepairDecision{
		Action:      base,
		Retryable:   base == RepairActionRetry || base == RepairActionRepairNode,
		Repairable:  base == RepairActionRepairNode,
		MaxAttempts: p.maxAttemptsForFailure(failure),
	}
	if base != RepairActionRepairNode {
		return decision
	}
	state.MaxTurnRepairAttempts = p.MaxTurnRepairAttempts
	state.MaxNodeRepairAttempts = p.MaxNodeRepairAttempts
	if RepairBudgetExhausted(state, failure) {
		decision.Exhausted = true
		decision.Retryable = false
		decision.Repairable = false
		decision.Action = fallbackOrHardError(failure)
	}
	return decision
}

// RepairClassForHTTPStatus maps model transport status codes into retry classes.
func RepairClassForHTTPStatus(status int) RepairClass {
	switch status {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusPaymentRequired:
		return RepairTransportFatal
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return RepairTransportTransient
	}
	if status >= 500 && status <= 599 {
		return RepairTransportTransient
	}
	return RepairUnknown
}

// RepairHTTPStatusRetryable reports whether a status may use transport retry.
func RepairHTTPStatusRetryable(status int) bool {
	return RepairClassForHTTPStatus(status) == RepairTransportTransient
}

// withDefaults applies v1 fixed limits when a caller provides a zero policy.
func (p RepairPolicy) withDefaults() RepairPolicy {
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

// actionForFailure encodes the non-negotiable classes that models may not repair.
func (p RepairPolicy) actionForFailure(failure RepairFailure) RepairAction {
	switch failure.Class {
	case RepairTransportTransient:
		return RepairActionRetry
	case RepairParseError, RepairSchemaError, RepairProjectionMismatch, RepairEvidenceOverclaim, RepairGuardrailBlocked:
		return RepairActionRepairNode
	case RepairTransportFatal, RepairFactConflict, RepairMethodContract:
		return RepairActionHardError
	case RepairDomainUnauthorized:
		return fallbackOrHardError(failure)
	default:
		return fallbackOrHardError(failure)
	}
}

// maxAttemptsForFailure returns the relevant cap for trace and later executors.
func (p RepairPolicy) maxAttemptsForFailure(failure RepairFailure) int {
	switch failure.Class {
	case RepairTransportTransient:
		return p.TransportMaxAttempts
	case RepairParseError, RepairSchemaError:
		return p.JSONParseMaxAttempts
	default:
		return p.MaxNodeRepairAttempts
	}
}

// fallbackOrHardError chooses deterministic degradation when a classified
// failure has an approved fallback path.
func fallbackOrHardError(failure RepairFailure) RepairAction {
	if failure.Fallback != "" {
		return RepairActionFallback
	}
	return RepairActionHardError
}
