// This file belongs to the manager-owned runtime layer.
// It owns the global Repair Harness contract shared by domain validators.
// It must not run domain repair loops or let specialists own final answers.
package runtime

// RepairClass names the closed taxonomy for repair-harness failures.
type RepairClass string

const (
	// RepairTransportTransient covers retryable transport failures such as 429, 5xx and timeouts.
	RepairTransportTransient RepairClass = "transport_transient"
	// RepairTransportFatal covers non-retryable transport failures such as 400, 401 and 402.
	RepairTransportFatal RepairClass = "transport_fatal"
	// RepairParseError covers malformed model output that cannot be decoded.
	RepairParseError RepairClass = "parse_error"
	// RepairSchemaError covers decoded output that violates a structured schema.
	RepairSchemaError RepairClass = "schema_error"
	// RepairProjectionMismatch covers fields that fail deterministic projection contracts.
	RepairProjectionMismatch RepairClass = "projection_mismatch"
	// RepairEvidenceOverclaim covers conclusions stronger than available evidence allows.
	RepairEvidenceOverclaim RepairClass = "evidence_overclaim"
	// RepairDomainUnauthorized covers output outside the authorized domain scope.
	RepairDomainUnauthorized RepairClass = "domain_unauthorized"
	// RepairFactConflict covers contradictions with deterministic facts.
	RepairFactConflict RepairClass = "fact_conflict"
	// RepairMethodContract covers domain-method violations that should not be model-patched.
	RepairMethodContract RepairClass = "method_contract"
	// RepairGuardrailBlocked covers final output blocked by runtime guardrails.
	RepairGuardrailBlocked RepairClass = "guardrail_blocked"
	// RepairUnknown covers failures that cannot be safely classified yet.
	RepairUnknown RepairClass = "unknown"
)

// RepairAction names the next runtime action chosen for one classified failure.
type RepairAction string

const (
	// RepairActionAccept means validation succeeded and no repair is needed.
	RepairActionAccept RepairAction = "accept"
	// RepairActionRetry means the same model/tool call may retry without business repair.
	RepairActionRetry RepairAction = "retry"
	// RepairActionRepairNode means a bounded business repair node may run.
	RepairActionRepairNode RepairAction = "repair_node"
	// RepairActionFallback means runtime should degrade through a deterministic fallback.
	RepairActionFallback RepairAction = "fallback"
	// RepairActionHardError means runtime should stop instead of asking the model to guess.
	RepairActionHardError RepairAction = "hard_error"
)

// RepairFailure is the machine-readable failure shape passed between validators,
// repair policy, trace projection and later domain adapters.
type RepairFailure struct {
	Domain string
	Stage  string
	Class  RepairClass
	Field  string

	Code    string
	Message string
	Excerpt string

	Retryable  bool
	Repairable bool
	Fallback   string
	Cause      error
}

// RepairAttempt records one bounded repair decision without retaining prompts,
// full trace data or user-authored private text.
type RepairAttempt struct {
	Domain   string
	Stage    string
	Class    RepairClass
	Field    string
	Attempt  int
	Action   RepairAction
	Feedback map[string]any
}

// RepairState tracks the current turn's business repair budget.
type RepairState struct {
	Attempts              []RepairAttempt
	MaxTurnRepairAttempts int
	MaxNodeRepairAttempts int
}
