// This file belongs to the deterministic tool layer.
// It owns runner behavior for this package.
// It executes governed tools; user-facing synthesis stays in runtime.
package tools

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// ToolRunStatus is the normalized execution result status for a tool call.
type ToolRunStatus string

const (
	ToolRunStatusOK       ToolRunStatus = "ok"
	ToolRunStatusFallback ToolRunStatus = "fallback"
	ToolRunStatusBlocked  ToolRunStatus = "blocked"
	ToolRunStatusError    ToolRunStatus = "error"
)

// ToolRunRequest is a normalized request to execute a registered tool.
type ToolRunRequest struct {
	ToolName       string
	Params         map[string]any
	DecisionSource string
	IdempotencyKey string
	Approved       bool
}

// ToolRunResult is a structured envelope for every tool execution.
type ToolRunResult struct {
	ToolName       string
	Version        string
	Status         ToolRunStatus
	Data           any
	Error          error
	ErrorClass     ToolErrorClass
	Retryable      bool
	Fallback       bool
	Attempts       int
	DecisionSource string
	DurationMs     int64
}

// ToolRunner executes tools through contracts, policy, retry, and tracing.
type ToolRunner struct {
	reg *Registry
}

// NewToolRunner creates a runner backed by the shared tool registry.
func NewToolRunner(reg *Registry) *ToolRunner {
	return &ToolRunner{reg: reg}
}

// Run executes one tool and always returns a structured result envelope.
func (r *ToolRunner) Run(ctx context.Context, req ToolRunRequest) ToolRunResult {
	start := time.Now()
	result := ToolRunResult{
		ToolName:       req.ToolName,
		Status:         ToolRunStatusError,
		DecisionSource: req.DecisionSource,
	}

	if r == nil || r.reg == nil {
		result.ErrorClass = ToolErrorNotFound
		result.Error = fmt.Errorf("tool registry is nil")
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	tool, ok := r.reg.Get(req.ToolName)
	if !ok {
		result.ErrorClass = ToolErrorNotFound
		result.Error = fmt.Errorf("tool %s not found", req.ToolName)
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	contract, ok := r.reg.Contract(req.ToolName)
	if !ok {
		contract = DefaultContractFor(req.ToolName)
	}
	result.Version = contract.Version

	span := tracing.SpanFromContext(ctx, req.ToolName, tracing.KindTool)
	span.SetAttribute("tool.name", req.ToolName)
	span.SetAttribute("tool.version", contract.Version)
	span.SetAttribute("tool.risk_level", string(contract.RiskLevel))
	span.SetAttribute("tool.side_effect", string(contract.SideEffect))
	span.SetAttribute("tool.read_only", contract.ReadOnly)
	span.SetAttribute("tool.idempotent", contract.Idempotent)
	span.SetAttribute("tool.decision_source", req.DecisionSource)
	span.SetAttribute("tool.param_keys", summarizeParamKeys(req.Params))
	defer func() {
		span.SetAttribute("tool.attempts", result.Attempts)
		span.SetAttribute("tool.status", string(result.Status))
		span.SetAttribute("tool.duration_ms", result.DurationMs)
		if result.ErrorClass != "" {
			span.SetAttribute("tool.error_class", string(result.ErrorClass))
		}
		span.End()
	}()

	if err := validateParams(contract, req.Params); err != nil {
		result.Error = err
		result.ErrorClass = ToolErrorInvalidParams
		result.DurationMs = time.Since(start).Milliseconds()
		span.RecordError(err)
		span.SetStatus("error")
		return result
	}

	if contract.RequiresApproval && !req.Approved {
		err := fmt.Errorf("tool %s requires approval", req.ToolName)
		result.Status = ToolRunStatusBlocked
		result.Error = err
		result.ErrorClass = ToolErrorApprovalRequired
		result.DurationMs = time.Since(start).Milliseconds()
		span.RecordError(err)
		span.SetStatus("error")
		return result
	}

	if contract.RequiresIdempotencyKey && strings.TrimSpace(req.IdempotencyKey) == "" {
		err := fmt.Errorf("tool %s requires idempotency key", req.ToolName)
		result.Status = ToolRunStatusBlocked
		result.Error = err
		result.ErrorClass = ToolErrorInvalidParams
		result.DurationMs = time.Since(start).Milliseconds()
		span.RecordError(err)
		span.SetStatus("error")
		return result
	}

	attempts := normalizedAttempts(contract.Retry.MaxAttempts)
	for attempt := 1; attempt <= attempts; attempt++ {
		result.Attempts = attempt

		execCtx := ctx
		cancel := func() {}
		if contract.TimeoutMillis > 0 {
			execCtx, cancel = context.WithTimeout(ctx, time.Duration(contract.TimeoutMillis)*time.Millisecond)
		}
		data, err := tool.Execute(execCtx, req.Params)
		cancel()

		if err == nil && data != nil {
			result.Status = ToolRunStatusOK
			result.Data = data
			result.DurationMs = time.Since(start).Milliseconds()
			if isFallbackResult(data) {
				result.Status = ToolRunStatusFallback
				result.Fallback = true
				span.SetStatus("fallback")
			}
			return result
		}

		if err == nil {
			err = fmt.Errorf("tool %s returned nil result", req.ToolName)
		}
		result.Error = err
		result.ErrorClass = ClassifyToolError(err)
		result.Retryable = canRetry(result.ErrorClass, contract.Retry)
		if !result.Retryable || attempt == attempts {
			break
		}
		if contract.Retry.BackoffMillis > 0 {
			time.Sleep(time.Duration(contract.Retry.BackoffMillis) * time.Millisecond)
		}
	}

	result.DurationMs = time.Since(start).Milliseconds()
	span.RecordError(result.Error)
	span.SetStatus("error")
	return result
}

// ClassifyToolError maps raw errors into governance-level classes.
func ClassifyToolError(err error) ToolErrorClass {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return ToolErrorTransient
	}
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "required"), strings.Contains(text, "invalid"), strings.Contains(text, "out of range"):
		return ToolErrorInvalidParams
	case strings.Contains(text, "permission"), strings.Contains(text, "unauthorized"), strings.Contains(text, "forbidden"):
		return ToolErrorPermissionDenied
	case strings.Contains(text, "business rejected"), strings.Contains(text, "rejected"):
		return ToolErrorBusinessRejected
	case strings.Contains(text, "timeout"), strings.Contains(text, "temporary"), strings.Contains(text, "rate limit"):
		return ToolErrorTransient
	default:
		return ToolErrorInternal
	}
}

func validateParams(contract ToolContract, params map[string]any) error {
	if len(contract.Params) > 0 {
		allowed := make(map[string]struct{}, len(contract.Params))
		for _, spec := range contract.Params {
			allowed[spec.Name] = struct{}{}
		}
		for name := range params {
			if _, ok := allowed[name]; !ok {
				return fmt.Errorf("unknown parameter %s", name)
			}
		}
	}
	for _, spec := range contract.Params {
		value, ok := params[spec.Name]
		if spec.Required && (!ok || value == nil || value == "") {
			return fmt.Errorf("%s is required", spec.Name)
		}
		if !ok || value == nil {
			continue
		}
		switch spec.Type {
		case "string":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("%s must be string", spec.Name)
			}
		case "number":
			switch value.(type) {
			case int, int64, float64, float32:
			default:
				return fmt.Errorf("%s must be number", spec.Name)
			}
		case "object":
			if _, ok := value.(map[string]any); !ok {
				return fmt.Errorf("%s must be object", spec.Name)
			}
		}
	}
	return nil
}

func normalizedAttempts(maxAttempts int) int {
	if maxAttempts <= 0 {
		return 1
	}
	return maxAttempts
}

func canRetry(class ToolErrorClass, policy RetryPolicy) bool {
	for _, allowed := range policy.RetryErrorClasses {
		if allowed == class {
			return true
		}
	}
	return false
}

func isFallbackResult(data any) bool {
	payload, ok := data.(map[string]any)
	if !ok {
		return false
	}
	fallback, _ := payload["fallback"].(bool)
	return fallback
}

func summarizeParamKeys(params map[string]any) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}
