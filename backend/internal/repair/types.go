// Package repair contains the shared, state-safe repair contract.
//
// This package owns failure classes, actions and per-turn repair state. It does
// not classify domain errors, call models or compose user-facing answers.
package repair

// Class names the closed taxonomy for repair-harness failures.
type Class string

const (
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
	// MethodContract covers domain-method violations in a model candidate.
	MethodContract Class = "method_contract"
	// DeterministicConflict covers contradictory tools, durable assets or deterministic rules.
	DeterministicConflict Class = "deterministic_conflict"
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
	Origin Origin

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

// Origin identifies whether a failure came from a model candidate, tool result or system invariant.
type Origin string

const (
	OriginModelCandidate Origin = "model_candidate"
	OriginTool           Origin = "tool"
	OriginSystem         Origin = "system"
)

// FailureSnapshot is the state-safe portion of a failure retained across graph transitions.
type FailureSnapshot struct {
	Domain   string
	Stage    string
	Class    Class
	Field    string
	Code     string
	Origin   Origin
	Fallback string
}

// Snapshot returns the fields safe to persist in graph state and trace metadata.
func (failure Failure) Snapshot() FailureSnapshot {
	return FailureSnapshot{
		Domain: failure.Domain, Stage: failure.Stage, Class: failure.Class, Field: failure.Field,
		Code: failure.Code, Origin: failure.Origin, Fallback: failure.Fallback,
	}
}

// Failure rebuilds a repair decision input from state-safe metadata.
// Message, excerpts, references and causes intentionally remain node-local.
func (snapshot FailureSnapshot) Failure() Failure {
	return Failure{
		Domain: snapshot.Domain, Stage: snapshot.Stage, Class: snapshot.Class, Field: snapshot.Field,
		Code: snapshot.Code, Origin: snapshot.Origin, Fallback: snapshot.Fallback,
	}
}

// Attempt records one bounded repair decision without retaining prompts, full
// trace data or user-authored private text.
type Attempt struct {
	Domain            string
	Stage             string
	Class             Class
	Field             string
	Attempt           int
	Action            Action
	FeedbackKeys      []string
	LearningHintCount int
	PolicyVersion     string
	PromptVersion     string
	ValidatorVersion  string
}

// State tracks the current turn's business repair budget.
type State struct {
	Attempts              []Attempt
	MaxTurnRepairAttempts int
	MaxNodeRepairAttempts int
	InitialFailure        FailureSnapshot
	LastFailure           FailureSnapshot
}
