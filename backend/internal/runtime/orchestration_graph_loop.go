// Package runtime contains the manager-owned execution flow.
//
// This file owns the outer graph's bounded state machine. It decides which
// existing Manager-owned action runs next and never emits the final answer.
package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// decideNextNode is the only outer node allowed to choose the next action.
// Keeping the policy here makes retries, degraded support and termination
// visible in one state transition instead of hiding them in worker functions.
func decideNextNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	if oc.GS == nil || oc.Init == nil || oc.RT == nil {
		return "", fmt.Errorf("orchestration graph context is incomplete")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if oc.GS.MaxRunSteps <= 0 {
		oc.GS.MaxRunSteps = orchestrationMaxRunSteps
	}
	oc.GS.LoopStep++

	if oc.GS.LoopStep >= oc.GS.MaxRunSteps {
		if strings.TrimSpace(oc.GS.RawFinalText) != "" || primaryOutcomeReady(oc.GS.DomainOutcomes) {
			oc.GS.NextAction = orchestrationActionFinish
			oc.GS.TerminationReason = "graph_step_limit_degraded"
		} else {
			oc.GS.NextAction = orchestrationActionHardError
			oc.GS.TerminationReason = "graph_step_limit"
			oc.GS.Failure = graphFailure{
				FailureClass: failureClassInvariantFailure,
				FailureStage: "decide_next",
				FailureCode:  "ORCHESTRATION_MAX_STEPS",
				Domain:       oc.GS.Route.PrimaryDomain,
				Retryable:    false,
				Message:      "外层执行图达到本轮步数上限，未形成可安全展示的结果。",
			}
		}
		traceOrchestrationDecision(ctx, oc.GS)
		return in, nil
	}

	action := chooseOrchestrationAction(oc.GS)
	oc.GS.NextAction = action
	if action == orchestrationActionHardError && oc.GS.TerminationReason == "" {
		oc.GS.TerminationReason = "hard_error"
	}
	traceOrchestrationDecision(ctx, oc.GS)
	return in, nil
}

// chooseOrchestrationAction applies the fixed outer transition order.
func chooseOrchestrationAction(state *orchestrationGraphState) orchestrationNextAction {
	if state == nil {
		return orchestrationActionHardError
	}
	if supportOutcomeFailed(state.DomainOutcomes) {
		state.Degraded = true
	}
	if state.PreflightResult.ShortCircuit && !state.PrefillCompleted && len(state.DomainOutcomes) == 0 {
		return orchestrationActionShortCircuit
	}
	if state.Failure.hasFailure() {
		if state.Failure.FailureStage == failureStagePrefill && state.Failure.Retryable && state.PrefillAttempts < 2 {
			return orchestrationActionPrefill
		}
		if primaryOutcomeFailed(state.DomainOutcomes) && state.Failure.Retryable && state.DispatchAttempts < 2 {
			return orchestrationActionDispatch
		}
		if !primaryOutcomeFailed(state.DomainOutcomes) && supportOutcomeFailed(state.DomainOutcomes) {
			state.Degraded = true
			state.Failure = graphFailure{}
			return orchestrationActionFinish
		}
		return orchestrationActionHardError
	}
	if !state.PrefillCompleted {
		return orchestrationActionPrefill
	}
	if len(state.PendingDomainSteps) > 0 {
		return orchestrationActionDispatch
	}
	if strings.TrimSpace(state.RawFinalText) != "" || primaryOutcomeReady(state.DomainOutcomes) {
		return orchestrationActionFinish
	}
	state.Failure = graphFailure{
		FailureClass: failureClassInvariantFailure,
		FailureStage: "decide_next",
		FailureCode:  "ORCHESTRATION_NO_RESULT",
		Domain:       state.Route.PrimaryDomain,
		Retryable:    false,
		Message:      "外层执行图没有形成可安全展示的领域结果。",
	}
	return orchestrationActionHardError
}

// orchestrationBranch maps the state-machine action to a graph node.
func orchestrationBranch(ctx context.Context, _ string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	if oc.GS == nil {
		return "", fmt.Errorf("orchestration graph state is nil")
	}
	switch oc.GS.NextAction {
	case orchestrationActionShortCircuit:
		return "short_circuit", nil
	case orchestrationActionPrefill:
		return "prefill", nil
	case orchestrationActionDispatch:
		return "dispatch_batch", nil
	case orchestrationActionFinish:
		return "terminal", nil
	case orchestrationActionHardError:
		return "terminal_error", nil
	default:
		return "", fmt.Errorf("unknown orchestration action %q", oc.GS.NextAction)
	}
}

// shortCircuitNode buffers a preflight answer. Final guard and SSE emission
// remain in Executor.Execute so the answer is emitted exactly once.
func shortCircuitNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	oc.GS.RawFinalText = oc.GS.PreflightResult.Text
	oc.GS.TurnType = oc.GS.PreflightResult.TurnType
	oc.GS.TerminationReason = "short_circuit"
	return in, nil
}

// terminalNode copies the graph-owned output into the post-Invoke result side
// channel. It deliberately does not call guardFinalAnswerWithPlan or emit SSE.
func terminalNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(oc.GS.RawFinalText) == "" {
		oc.GS.RawFinalText = oc.GS.AggregatedResult.NormalizedSummary()
	}
	if oc.GS.TerminationReason == "" {
		oc.GS.TerminationReason = "completed"
	}
	result := getOrchestrationResult(ctx)
	if result != nil {
		result.GraphState = oc.GS
		result.RawFinalText = oc.GS.RawFinalText
		result.TurnType = firstNonEmpty(oc.GS.TurnType, "agent_reading")
		result.PrimaryDomain = oc.GS.Route.PrimaryDomain
		result.ReplyDomain = firstNonEmpty(oc.GS.AggregatedResult.Domain, oc.GS.Route.PrimaryDomain)
		result.Specialist = oc.GS.AggregatedResult
		result.Failure = oc.GS.Failure
		result.TerminationReason = oc.GS.TerminationReason
	}
	tracing.SetTraceAttributes(ctx, map[string]any{
		"orchestration.termination_reason": oc.GS.TerminationReason,
		"orchestration.raw_final_text_len": len([]rune(oc.GS.RawFinalText)),
	})
	return oc.GS.RawFinalText, nil
}

// terminalErrorNode closes classified failures without returning a Go error.
// Executor converts this state into RuntimeFailure for the stable SSE contract.
func terminalErrorNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	oc.GS.TerminationReason = firstNonEmpty(oc.GS.TerminationReason, "hard_error")
	if !oc.GS.Failure.hasFailure() {
		oc.GS.Failure = graphFailure{
			FailureClass: failureClassInvariantFailure,
			FailureStage: "terminal_error",
			FailureCode:  "ORCHESTRATION_HARD_ERROR",
			Domain:       oc.GS.Route.PrimaryDomain,
			Message:      "本轮执行未形成可安全展示的结果。",
		}
	}
	result := getOrchestrationResult(ctx)
	if result != nil {
		result.GraphState = oc.GS
		result.Failure = oc.GS.Failure
		result.TerminationReason = oc.GS.TerminationReason
		result.TurnType = "agent_error"
		result.PrimaryDomain = oc.GS.Route.PrimaryDomain
	}
	tracing.SetTraceAttributes(ctx, map[string]any{
		"orchestration.termination_reason": oc.GS.TerminationReason,
		"orchestration.failure_code":       oc.GS.Failure.FailureCode,
	})
	return "", nil
}

// dispatchBatchNode runs only pending domains and retains each outcome in graph
// state. A failed primary remains pending; successful domains are never rerun.
func dispatchBatchNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	oc.GS.DispatchAttempts++
	oc.GS.Failure = graphFailure{}
	steps := executionStepsFromContracts(oc.GS.PendingDomainSteps)
	if len(steps) == 0 {
		return in, nil
	}
	outcomes, runErr := oc.RT.Executor.dispatchExecutionSteps(ctx, oc.RT.Sink, oc.Init.St, oc.GS.Plan, oc.Init.UserMsg, steps)
	if runErr != nil {
		if err := recordGraphFailure(ctx, &oc.GS.Failure, oc.GS.Route.PrimaryDomain, failureStageAgent, runErr); err != nil {
			return "", err
		}
		return in, nil
	}
	for _, outcome := range outcomes {
		upsertOrchestrationOutcome(oc.GS, outcome)
	}
	oc.GS.PendingDomainSteps = pendingDomainStepsAfterOutcomes(oc.GS.PendingDomainSteps, outcomes)
	if primary := primaryFailureOutcome(outcomes); primary != nil {
		oc.GS.Failure = graphFailureFromError(primary.Domain, failureStageAgent, primary.Err)
		if !oc.GS.Failure.hasFailure() {
			oc.GS.Failure = graphFailure{
				FailureClass: failureClassSpecialistContractViolation,
				FailureStage: failureStageAgent,
				FailureCode:  "PRIMARY_SPECIALIST_FAILED",
				Domain:       primary.Domain,
				Retryable:    true,
				Message:      "主领域执行失败，正在按有限预算处理。",
			}
		}
	}
	if supportOutcomeFailed(oc.GS.DomainOutcomes) {
		oc.GS.Degraded = true
	}
	return in, nil
}

// aggregateNode performs the single fan-in and Manager composition step.
func aggregateNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	outcomes := executionOutcomesFromState(oc.GS.DomainOutcomes)
	primary := primaryDomainForSteps(executionStepsFromContracts(oc.GS.Plan.DomainSteps))
	result := aggregateExecutionOutcomes(outcomes, primary)
	result.DomainContextPatch = attachDynamicFacts(result.DomainContextPatch, oc.GS.DynamicFacts)
	if result.DomainContextPatch == nil {
		result.DomainContextPatch = make(map[string]any)
	}
	// TimeScope 是本轮唯一的动态展示授权；没有明确时间范围时，静态追问
	// 不应被 Prefill 缺口说明打断。
	result.DomainContextPatch[dynamicFactsNoticeRequiredKey] = strings.TrimSpace(oc.GS.Plan.Route.Slots.TimeScope) != ""
	result.Summary = oc.RT.Executor.manager.ComposeFinalReply(oc.Init.UserMsg, result)
	oc.GS.AggregatedResult = result
	oc.GS.RawFinalText = result.NormalizedSummary()
	if result.Domain == "" {
		result.Domain = firstNonEmpty(primary, oc.GS.Route.PrimaryDomain)
	}
	oc.GS.AggregatedResult = result
	return in, nil
}

// traceOrchestrationDecision records the state-machine transition fields used
// by trace search and bounded-loop regressions.
func traceOrchestrationDecision(ctx context.Context, state *orchestrationGraphState) {
	if state == nil {
		return
	}
	tracing.SetTraceAttributes(ctx, map[string]any{
		"orchestration.loop_step":     state.LoopStep,
		"orchestration.next_action":   string(state.NextAction),
		"orchestration.max_run_steps": state.MaxRunSteps,
	})
}

func executionStepsFromContracts(steps []contracts.DomainStep) []executionStep {
	if len(steps) == 0 {
		return nil
	}
	out := make([]executionStep, 0, len(steps))
	for _, step := range steps {
		out = append(out, executionStep{Domain: strings.TrimSpace(step.Domain), Role: strings.TrimSpace(step.Role)})
	}
	return out
}

func upsertOrchestrationOutcome(state *orchestrationGraphState, outcome executionStepOutcome) {
	if state == nil {
		return
	}
	converted := orchestrationDomainOutcome{
		Domain: outcome.Domain,
		Role:   outcome.Role,
		Status: outcome.Status,
		Result: outcome.Result,
	}
	if outcome.Err != nil {
		converted.Failure = graphFailureFromError(outcome.Domain, failureStageAgent, outcome.Err)
	}
	for index := range state.DomainOutcomes {
		if state.DomainOutcomes[index].Domain == converted.Domain {
			state.DomainOutcomes[index] = converted
			return
		}
	}
	state.DomainOutcomes = append(state.DomainOutcomes, converted)
}

func pendingDomainStepsAfterOutcomes(pending []contracts.DomainStep, outcomes []executionStepOutcome) []contracts.DomainStep {
	failedPrimary := make(map[string]bool)
	completed := make(map[string]bool)
	for _, outcome := range outcomes {
		if outcome.Role == executionStepRolePrimary && outcome.Status == executionStepStatusFailed {
			failedPrimary[outcome.Domain] = true
			continue
		}
		completed[outcome.Domain] = true
	}
	remaining := make([]contracts.DomainStep, 0, len(pending))
	for _, step := range pending {
		if failedPrimary[step.Domain] || !completed[step.Domain] {
			remaining = append(remaining, step)
		}
	}
	return remaining
}

func executionOutcomesFromState(outcomes []orchestrationDomainOutcome) []executionStepOutcome {
	converted := make([]executionStepOutcome, 0, len(outcomes))
	for _, outcome := range outcomes {
		converted = append(converted, executionStepOutcome{
			Domain: outcome.Domain,
			Role:   outcome.Role,
			Status: outcome.Status,
			Result: outcome.Result,
			Err:    graphFailureError(outcome.Failure),
		})
	}
	return converted
}

func primaryOutcomeFailed(outcomes []orchestrationDomainOutcome) bool {
	for _, outcome := range outcomes {
		if outcome.Role == executionStepRolePrimary && outcome.Status == executionStepStatusFailed {
			return true
		}
	}
	return false
}

// primaryOutcomeReady reports whether the graph has a safe primary result to
// preserve at the step limit. A support-only result cannot become the answer.
func primaryOutcomeReady(outcomes []orchestrationDomainOutcome) bool {
	for _, outcome := range outcomes {
		if outcome.Role == executionStepRolePrimary && outcome.Status == executionStepStatusReady && strings.TrimSpace(outcome.Result.NormalizedSummary()) != "" {
			return true
		}
	}
	return false
}

func supportOutcomeFailed(outcomes []orchestrationDomainOutcome) bool {
	for _, outcome := range outcomes {
		if outcome.Role == executionStepRoleSupport && outcome.Status != executionStepStatusReady {
			return true
		}
	}
	return false
}

func primaryFailureOutcome(outcomes []executionStepOutcome) *executionStepOutcome {
	for index := range outcomes {
		if outcomes[index].Role == executionStepRolePrimary && outcomes[index].Status == executionStepStatusFailed {
			return &outcomes[index]
		}
	}
	return nil
}
