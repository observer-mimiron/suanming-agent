// Package runtime contains the manager-owned execution flow.
//
// This file defines the serializable control contracts shared by the bounded
// orchestration graphs. It carries decisions and failure facts, not Go errors,
// model clients, executors or event sinks.
package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/specialists"
)

const orchestrationMaxRunSteps = 16

// orchestrationNextAction is the closed set of outer graph transitions.
type orchestrationNextAction string

const (
	orchestrationActionShortCircuit orchestrationNextAction = "short_circuit"
	orchestrationActionPrefill      orchestrationNextAction = "prefill"
	orchestrationActionDispatch     orchestrationNextAction = "dispatch_batch"
	orchestrationActionFinish       orchestrationNextAction = "finish"
	orchestrationActionHardError    orchestrationNextAction = "hard_error"
)

// orchestrationDomainOutcome is the state-safe domain result retained across
// dispatch retries. Failed support work is visible here but does not erase a
// successful primary result.
type orchestrationDomainOutcome struct {
	Domain  string             `json:"domain"`
	Role    string             `json:"role"`
	Status  string             `json:"status"`
	Result  specialists.Result `json:"result"`
	Failure graphFailure       `json:"failure"`
}

// graphFailure is the state-safe projection of an error that a graph can
// recover from or expose through its terminal result.
type graphFailure struct {
	FailureClass string   `json:"failure_class,omitempty"`
	FailureStage string   `json:"failure_stage,omitempty"`
	FailureCode  string   `json:"failure_code,omitempty"`
	Domain       string   `json:"domain,omitempty"`
	Retryable    bool     `json:"retryable,omitempty"`
	Degraded     bool     `json:"degraded,omitempty"`
	Message      string   `json:"message,omitempty"`
	MissingRefs  []string `json:"missing_refs,omitempty"`
	AllowedRefs  []string `json:"allowed_refs,omitempty"`
}

// hasFailure reports whether a graph has a recoverable or terminal failure.
func (f graphFailure) hasFailure() bool {
	return strings.TrimSpace(f.FailureClass) != "" ||
		strings.TrimSpace(f.FailureCode) != "" ||
		strings.TrimSpace(f.Message) != ""
}

// graphFailureFromError converts a runtime error into a state-safe failure.
func graphFailureFromError(domain, stage string, err error) graphFailure {
	if err == nil {
		return graphFailure{}
	}
	var runtimeFailure *RuntimeFailure
	if errors.As(err, &runtimeFailure) && runtimeFailure != nil {
		return graphFailure{
			FailureClass: runtimeFailure.Class,
			FailureStage: firstFailureText(runtimeFailure.Stage, stage),
			FailureCode:  runtimeFailure.Code,
			Domain:       firstFailureText(runtimeFailure.Domain, domain),
			Retryable:    runtimeFailure.Retryable,
			Degraded:     runtimeFailure.Degraded,
			Message:      runtimeFailure.Message,
		}
	}
	var specialistFailure *specialists.Failure
	if errors.As(err, &specialistFailure) && specialistFailure != nil {
		return graphFailure{
			FailureClass: specialistFailure.Class,
			FailureStage: firstFailureText(specialistFailure.Stage, stage),
			FailureCode:  specialistFailure.Code,
			Domain:       firstFailureText(specialistFailure.Domain, domain),
			Retryable:    specialistFailure.Retryable,
			Degraded:     specialistFailure.Degraded,
			Message:      specialistFailure.Message,
		}
	}
	return graphFailure{
		FailureClass: failureClassInvariantFailure,
		FailureStage: firstFailureText(stage, failureStageAgent),
		FailureCode:  "RUNTIME_EXECUTION_FAILED",
		Domain:       domain,
		Retryable:    true,
		Message:      err.Error(),
	}
}

// graphFailureError reconstructs a local error only at a node boundary. The
// graph state itself never stores this error interface.
func graphFailureError(f graphFailure) error {
	if !f.hasFailure() {
		return nil
	}
	return fmt.Errorf("%s: %s", firstFailureText(f.FailureCode, f.FailureClass), f.Message)
}

// recordGraphFailure writes one classified failure and rejects cancellation
// from being silently converted into a business recovery path.
func recordGraphFailure(ctx context.Context, failure *graphFailure, domain, stage string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if failure != nil {
		*failure = graphFailureFromError(domain, stage, err)
	}
	return nil
}
