// This file belongs to the deterministic tool layer.
// It owns tool contract governance for this package.
// It executes governed tools; user-facing synthesis stays in runtime.
package tools

// RiskLevel describes how dangerous a tool is if selected or repeated incorrectly.
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// SideEffectLevel describes whether a tool changes external or durable state.
type SideEffectLevel string

const (
	SideEffectNone        SideEffectLevel = "none"
	SideEffectRead        SideEffectLevel = "read"
	SideEffectWrite       SideEffectLevel = "write"
	SideEffectDestructive SideEffectLevel = "destructive"
)

// ToolErrorClass describes why a tool call failed.
type ToolErrorClass string

const (
	ToolErrorInvalidParams    ToolErrorClass = "invalid_params"
	ToolErrorTransient        ToolErrorClass = "transient"
	ToolErrorPermissionDenied ToolErrorClass = "permission_denied"
	ToolErrorBusinessRejected ToolErrorClass = "business_rejected"
	ToolErrorInternal         ToolErrorClass = "internal_error"
	ToolErrorNotFound         ToolErrorClass = "not_found"
	ToolErrorApprovalRequired ToolErrorClass = "approval_required"
)

// RetryPolicy declares which tool failures can be retried by ToolRunner.
type RetryPolicy struct {
	MaxAttempts       int
	BackoffMillis     int
	RetryErrorClasses []ToolErrorClass
}

// ParamSpec is a small runtime schema for parameters that should be checked before tool execution.
type ParamSpec struct {
	Name     string
	Type     string
	Required bool
}

// ToolContract is the runtime contract for a tool. It is the source of truth for
// visibility, risk, retry, trace, and future write-operation controls.
type ToolContract struct {
	Name                   string
	Version                string
	Description            string
	ReadOnly               bool
	Idempotent             bool
	RequiresApproval       bool
	RequiresIdempotencyKey bool
	SideEffect             SideEffectLevel
	RiskLevel              RiskLevel
	TimeoutMillis          int
	Retry                  RetryPolicy
	Params                 []ParamSpec
}

// DefaultContractFor returns a conservative contract for existing tools.
func DefaultContractFor(name string) ToolContract {
	contract := ToolContract{
		Name:          name,
		Version:       "v1",
		ReadOnly:      true,
		Idempotent:    true,
		SideEffect:    SideEffectRead,
		RiskLevel:     RiskLow,
		TimeoutMillis: 10_000,
		Retry: RetryPolicy{
			MaxAttempts:   1,
			BackoffMillis: 0,
		},
	}

	switch name {
	case "bazi_calc", "yongshen", "dayun_analyzer", "bazi_liunian", "qimen_dunjia", "ziwei_calc", "ziwei_liunian":
		contract.SideEffect = SideEffectNone
	case "knowledge_search":
		contract.Params = []ParamSpec{
			{Name: "query", Type: "string", Required: true},
			{Name: "top_k", Type: "number", Required: false},
		}
		contract.Retry = RetryPolicy{
			MaxAttempts:       2,
			BackoffMillis:     100,
			RetryErrorClasses: []ToolErrorClass{ToolErrorTransient, ToolErrorInternal},
		}
	case "knowledge_catalog":
		contract.Params = []ParamSpec{
			{Name: "prefix", Type: "string", Required: false},
		}
		contract.Retry = RetryPolicy{
			MaxAttempts:       2,
			BackoffMillis:     100,
			RetryErrorClasses: []ToolErrorClass{ToolErrorTransient, ToolErrorInternal},
		}
	}

	return contract
}
