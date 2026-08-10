// Package repair contains the shared, state-safe repair contract.
//
// This package owns failure classes, actions and per-turn repair state. It does
// not classify domain errors, call models or compose user-facing answers.
package repair

// Class names the closed taxonomy for repair-harness failures.
type Class string

const (
	// TransportTransient covers retryable transport failures such as 429, 5xx and timeouts.
	TransportTransient Class = "transport_transient"
	// TransportFatal covers non-retryable transport failures such as 400, 401 and 402.
	TransportFatal Class = "transport_fatal"
	// ParseError covers malformed model output that cannot be decoded.
	ParseError Class = "parse_error"
	// SchemaError covers decoded output that violates a structured schema.
	SchemaError Class = "schema_error"
	// ProjectionMismatch covers fields that fail deterministic projection contracts.
	ProjectionMismatch Class = "projection_mismatch"
	// EvidenceOverclaim covers conclusions stronger than available evidence allows.
	EvidenceOverclaim Class = "evidence_overclaim"
	// DomainUnauthorized covers output outside the authorized domain scope.
	DomainUnauthorized Class = "domain_unauthorized"
	// FactConflict covers contradictions with deterministic facts.
	FactConflict Class = "fact_conflict"
	// MethodContract covers domain-method violations that should not be model-patched.
	MethodContract Class = "method_contract"
	// GuardrailBlocked covers final output blocked by runtime guardrails.
	GuardrailBlocked Class = "guardrail_blocked"
	// Unknown covers failures that cannot be safely classified yet.
	Unknown Class = "unknown"
)

// Action names the next runtime action chosen for one classified failure.
type Action string

const (
	// ActionAccept means validation succeeded and no repair is needed.
	ActionAccept Action = "accept"
	// ActionRetry means the same model/tool call may retry without business repair.
	ActionRetry Action = "retry"
	// ActionRepairNode means a bounded business repair node may run.
	ActionRepairNode Action = "repair_node"
	// ActionFallback means runtime should degrade through a deterministic fallback.
	ActionFallback Action = "fallback"
	// ActionHardError means runtime should stop instead of asking the model to guess.
	ActionHardError Action = "hard_error"
)

// Failure is the machine-readable shape passed between validators, repair
// policy, trace projection and domain adapters.
type Failure struct {
	Domain string
	Stage  string
	Class  Class
	Field  string

	Code        string
	Message     string
	Excerpt     string
	MissingRefs []string
	AllowedRefs []string

	Retryable  bool
	Repairable bool
	Fallback   string
	Cause      error
}

// Attempt records one bounded repair decision without retaining prompts, full
// trace data or user-authored private text.
type Attempt struct {
	Domain   string
	Stage    string
	Class    Class
	Field    string
	Attempt  int
	Action   Action
	Feedback map[string]any
}

// State tracks the current turn's business repair budget.
type State struct {
	Attempts              []Attempt
	MaxTurnRepairAttempts int
	MaxNodeRepairAttempts int
}
