package runtime

import (
	"context"
	"errors"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

const (
	failureClassArtifactMissing             = "artifact_missing"
	failureClassSpecialistContractViolation = "specialist_contract_violation"
	failureClassModelContractViolation      = "model_contract_violation"
	failureClassInvariantFailure            = "invariant_failure"

	failureStagePrefill     = "prefill"
	failureStageFinalGuard  = "final_guard"
	failureStageAgent       = "agent"
	failureStageFinalWriter = "final_writer"
)

// RuntimeFailure is the lightweight structured failure shape shared by runtime
// validation, trace projection, and regression tests.
type RuntimeFailure struct {
	Class       string
	Stage       string
	Domain      string
	Code        string
	Retryable   bool
	Degraded    bool
	UserVisible bool
	Message     string
	Cause       error
}

// Error returns the user-facing runtime failure message.
func (e *RuntimeFailure) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Unwrap returns the underlying cause so errors.As / errors.Is keep working.
func (e *RuntimeFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func annotateRuntimeFailureTrace(ctx context.Context, err error) {
	var rf *RuntimeFailure
	if !errors.As(err, &rf) || rf == nil {
		return
	}
	tracing.SetTraceAttributes(ctx, map[string]any{
		"failure.class":        rf.Class,
		"failure.stage":        rf.Stage,
		"failure.domain":       rf.Domain,
		"failure.code":         rf.Code,
		"failure.retryable":    rf.Retryable,
		"failure.degraded":     rf.Degraded,
		"failure.user_visible": rf.UserVisible,
	})
}

func classifyRuntimeFailure(domain, stage string, err error) error {
	if err == nil {
		return nil
	}
	var rf *RuntimeFailure
	if errors.As(err, &rf) {
		return rf
	}
	return &RuntimeFailure{
		Class:       failureClassInvariantFailure,
		Stage:       firstFailureText(stage, failureStageAgent),
		Domain:      domain,
		Code:        "RUNTIME_EXECUTION_FAILED",
		Retryable:   true,
		Degraded:    false,
		UserVisible: true,
		Message:     "本轮执行过程中遇到系统错误，已停止输出。请稍后重试。",
		Cause:       err,
	}
}

// RuntimeFailureEventData converts an internal runtime error into the stable
// SSE error payload. Internal error details stay in trace/logs; the frontend
// receives a predictable code, stage, retryability, trace id and Chinese text.
func RuntimeFailureEventData(ctx context.Context, err error, fallbackStage string) map[string]any {
	rf := runtimeFailureForEvent(fallbackStage, err)
	code := firstFailureText(rf.Code, strings.ToUpper(strings.ReplaceAll(rf.Class, "-", "_")))
	message := firstFailureText(publicRuntimeFailureMessage(rf), "本轮执行过程中遇到系统错误，已停止输出。请稍后重试。")
	return map[string]any{
		"code":      code,
		"stage":     firstFailureText(rf.Stage, fallbackStage),
		"retryable": rf.Retryable,
		"degraded":  rf.Degraded,
		"trace_id":  tracing.TraceIDFromContext(ctx),
		"message":   message,
	}
}

func runtimeFailureForEvent(fallbackStage string, err error) *RuntimeFailure {
	var rf *RuntimeFailure
	if errors.As(err, &rf) && rf != nil {
		return rf
	}
	return &RuntimeFailure{
		Class:       failureClassInvariantFailure,
		Stage:       fallbackStage,
		Code:        "RUNTIME_EXECUTION_FAILED",
		Retryable:   true,
		UserVisible: true,
		Message:     "本轮执行过程中遇到系统错误，已停止输出。请稍后重试。",
		Cause:       err,
	}
}

func publicRuntimeFailureMessage(rf *RuntimeFailure) string {
	if rf == nil {
		return ""
	}
	switch rf.Class {
	case failureClassArtifactMissing:
		return "本轮没有拿到必需的命盘或问事盘结果，无法继续解释。请稍后重试。"
	case failureClassModelContractViolation:
		return "本轮输出未通过最终合同校验，已停止展示不稳定内容。请稍后重试。"
	}
	if rf.UserVisible {
		return strings.TrimSpace(rf.Message)
	}
	return ""
}

func firstFailureText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
