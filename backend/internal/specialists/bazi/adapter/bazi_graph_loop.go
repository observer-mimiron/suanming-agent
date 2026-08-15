// Package adapter 负责八字 Graph 的有限动作循环适配。
//
// 本文件负责八字确定性 Graph 的状态机、有限动作循环和终态恢复；
// 不负责领域角色选择、完整会话所有权或最终答复合成。
package adapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/repair"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	baziapplication "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/application"
	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
	bazigraph "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/graph"
)

const (
	baziPhaseAnalysisPlan = "analysis_plan"
	baziPhaseEvidence     = "evidence"
	baziPhaseStatic       = "static"
	baziPhaseLifetime     = "lifetime"
	baziPhaseDynamic      = "dynamic"
	baziPhaseRepair       = "repair"
)

type baziGraphRuntime struct {
	Executor *Executor
	Session  *specialists.SessionView
	Sink     EventSink
}

type baziGraphRuntimeCtxKey struct{}

// baziRepairFailureState 保留 runtime 私有调用点名称；状态由八字 Graph 所有。
type baziRepairFailureState = bazidomain.RepairFailureState
type Result = bazigraph.Result

func withBaziGraphRuntime(ctx context.Context, runtime *baziGraphRuntime) context.Context {
	return context.WithValue(ctx, baziGraphRuntimeCtxKey{}, runtime)
}

func baziGraphRuntimeFromContext(ctx context.Context) (*baziGraphRuntime, error) {
	runtime, _ := ctx.Value(baziGraphRuntimeCtxKey{}).(*baziGraphRuntime)
	if runtime == nil || runtime.Executor == nil || runtime.Session == nil {
		return nil, fmt.Errorf("bazi graph runtime is incomplete")
	}
	return runtime, nil
}

func repairFailureStateFromRuntime(failure repair.Failure) baziRepairFailureState {
	return baziRepairFailureState{
		Domain:      failure.Domain,
		Stage:       failure.Stage,
		Class:       failure.Class,
		Field:       failure.Field,
		Code:        failure.Code,
		Message:     failure.Message,
		Excerpt:     failure.Excerpt,
		MissingRefs: append([]string(nil), failure.MissingRefs...),
		AllowedRefs: append([]string(nil), failure.AllowedRefs...),
		Fallback:    failure.Fallback,
		Retryable:   failure.Retryable,
		Repairable:  failure.Repairable,
	}
}

func baziEvidenceNeedsAction(in *baziInternalGraphState) bool {
	if in == nil {
		return false
	}
	return in.EvidenceAttempts == 0
}

// baziEvidenceActionNode performs the bounded initial and supplemental evidence actions for this run.
func (e *Executor) baziEvidenceActionNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, "evidence_action"); err != nil {
		return nil, err
	}
	runtime, err := baziGraphRuntimeFromContext(ctx)
	if err != nil {
		return nil, err
	}
	in.EvidenceAttempts++
	in.Failure = graphFailure{}
	plan, bundle, quality := e.runBaziEvidenceStage(ctx, runtime.Session, in.Question, in.ChartState.Input, in.ChartState.AnalysisPlan)
	in.ChartState.EvidencePlan = plan
	in.ChartState.EvidenceBundle = bundle
	in.ChartState.EvidenceQuality = quality
	emitBaziStageThinking(ctx, runtime.Sink, "bazi_graph", baziapplication.BuildEvidenceStageSummary(in.ChartState))
	return in, nil
}

// baziValidateEvidenceNode only checks the deterministic plan shape. Empty evidence
// is valid because retrieval is optional enhancement, not a judgment precondition.
func (e *Executor) baziValidateEvidenceNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, "validate_evidence"); err != nil {
		return nil, err
	}
	in.EvidenceValidated = true
	if err := validateEvidenceBundlePreconditions(in.ChartState); err != nil {
		if recordErr := recordGraphFailure(ctx, &in.Failure, "bazi", "evidence_validation", err); recordErr != nil {
			return nil, recordErr
		}
	}
	return in, nil
}

// baziContractCheckNode validates the current candidate only. It never calls a
// model; decide_next chooses repair or facts-only after this node reports.
func (e *Executor) baziContractCheckNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, "contract_check"); err != nil {
		return nil, err
	}
	if in.Failure.hasFailure() && in.Phase != baziPhaseRepair {
		return in, nil
	}
	var err error
	stage := "static_projection"
	if in.Phase == baziPhaseDynamic || (in.Phase == baziPhaseRepair && strings.HasPrefix(in.Failure.FailureStage, "dynamic")) {
		stage = "dynamic_projection"
		err = baziValidateV2Projection(in)
	} else if in.Phase == baziPhaseLifetime {
		stage = "lifetime_projection"
		err = validateLifetimeDayunSynthesis(in.ChartState)
	} else {
		err = validateStaticSynthesisResult(in.ChartState, in.ChartState.StaticSynthesis)
	}
	if err != nil {
		baziRecordInternalFailure(ctx, in, stage, err, "contract_check_failed")
		return in, nil
	}
	in.Failure = graphFailure{}
	in.FailureStage = ""
	in.RecoveryCode = ""
	in.FailureClass = ""
	in.RecoveryPolicy = ""
	if stage == "dynamic_projection" {
		in.ChartState.DynamicSynthesis.ContractAudit = baziContractAudit{Compliant: true}
		in.DynamicAccepted = true
		in.AcceptedDynamic = in.ChartState.DynamicSynthesis
	} else if stage == "lifetime_projection" {
		in.LifetimeAccepted = true
	} else {
		in.ChartState.StaticSynthesis.ContractAudit = baziContractAudit{Compliant: true}
		in.StaticAccepted = true
		in.AcceptedStatic = in.ChartState.StaticSynthesis
	}
	in.RecoveryState = baziRecoveryStateClean
	baziClearContractFailureTraceAttrs(ctx)
	return in, nil
}

// baziLifetimeDayunJudgmentNode owns the all-period reading. Its only output is
// LifetimeSynthesis, so neither natal nor current-period fields can be overwritten.
func (e *Executor) baziLifetimeDayunJudgmentNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, "lifetime_dayun_judgment"); err != nil {
		return nil, err
	}
	in.LifetimeAttempted = true
	runtime, err := baziGraphRuntimeFromContext(ctx)
	if err != nil {
		return nil, err
	}
	out, err := e.runLifetimeDayunSynthesis(ctx, runtime.Session, in.ChartState, in.Question)
	if err != nil {
		baziRecordInternalFailure(ctx, in, "lifetime_synthesis", err, "lifetime_judgment_failed")
		return in, nil
	}
	in.ChartState.LifetimeSynthesis = out
	in.LifetimeCandidate = out
	return in, nil
}

// baziRepairNode is the only node allowed to invoke a business repair model.
func (e *Executor) baziRepairNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, "repair"); err != nil {
		return nil, err
	}
	runtime, err := baziGraphRuntimeFromContext(ctx)
	if err != nil {
		return nil, err
	}
	failure := in.RepairFailure.Runtime()
	decision := repair.DefaultPolicy().Decide(failure, in.RepairState)
	in.RepairAction = decision.Action
	if decision.Action != repair.ActionRepairNode || decision.Exhausted {
		return in, nil
	}
	attempt := repair.AttemptsFor(in.RepairState, failure) + 1
	in.RepairState = repair.RecordAttempt(in.RepairState, repair.Attempt{
		Domain:  failure.Domain,
		Stage:   failure.Stage,
		Class:   failure.Class,
		Field:   failure.Field,
		Attempt: attempt,
		Action:  repair.ActionRepairNode,
	})
	in.RepairAttempts++
	in.RepairFeedback = buildBaziCanonicalRepairFeedback(failure, attempt)
	feedback := in.RepairFeedback
	var repaired baziCanonicalSynthesis
	if strings.HasPrefix(in.Failure.FailureStage, "dynamic") {
		repaired, err = e.runDynamicSynthesisRepair(ctx, runtime.Session, in.ChartState, in.Canonical, in.Question, feedback)
	} else {
		repaired, err = e.runStaticSynthesisRepair(ctx, runtime.Session, in.ChartState, in.Canonical, in.Question, feedback)
	}
	if err != nil {
		baziRecordInternalFailure(ctx, in, in.Failure.FailureStage, err, "repair_failed")
		return in, nil
	}
	in.Canonical = repaired
	in.Canonical.Source = "model"
	if strings.HasPrefix(in.Failure.FailureStage, "dynamic") {
		in.ChartState.DynamicSynthesis = projectCanonicalDynamicSynthesis(in.ChartState, repaired, in.ChartState.StaticSynthesis)
	} else {
		in.ChartState.StaticSynthesis = projectCanonicalStaticSynthesis(in.ChartState, repaired)
	}
	in.Phase = baziPhaseRepair
	return in, nil
}

// baziRecoverFactsNode applies only an already-approved deterministic fallback.
func (e *Executor) baziRecoverFactsNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, "recover_facts"); err != nil {
		return nil, err
	}
	err := baziFailureErrorFromState(in)
	if !in.Failure.hasFailure() && in.Phase == baziPhaseDynamic && !in.DynamicAccepted {
		in.ChartState.DynamicSynthesis = canonicalDynamicFailureFactsOnly(
			in.ChartState,
			in.ChartState.StaticSynthesis,
			fmt.Errorf("动态层达到安全收口条件，已保留可复算事实"),
		)
		in.DynamicAccepted = true
		in.AcceptedDynamic = in.ChartState.DynamicSynthesis
		in.RecoveryState = baziRecoveryStateDynamicFactsOnlyDegraded
		in.TerminationReason = firstNonEmpty(in.TerminationReason, "graph_step_limit_degraded")
		return in, nil
	}
	if strings.HasPrefix(in.FailureStage, "dynamic") && in.RecoveryPolicy == baziRecoveryPolicyDynamicFactsOnly {
		in.ChartState.DynamicSynthesis = canonicalDynamicFailureFactsOnly(in.ChartState, in.ChartState.StaticSynthesis, err)
		in.DynamicAccepted = true
		in.AcceptedDynamic = in.ChartState.DynamicSynthesis
		in.RecoveryState = baziRecoveryStateDynamicFactsOnlyDegraded
	} else if in.RecoveryPolicy == baziRecoveryPolicyStaticFactsOnly || in.RecoveryPolicy == baziRecoveryPolicyFullFactsOnly {
		static, dynamic := canonicalFailureFactsOnly(in.ChartState, err, "contract_gate_facts_only", "结构化裁断未通过合同校验，已降级展示可复算事实。")
		in.ChartState.StaticSynthesis, in.ChartState.DynamicSynthesis = static, dynamic
		in.StaticAccepted = true
		in.DynamicAccepted = true
		in.AcceptedStatic, in.AcceptedDynamic = static, dynamic
		in.RecoveryState = baziRecoveryStateStaticFactsOnlyDegraded
	} else {
		in.TerminationReason = "hard_error"
		return in, nil
	}
	baziAnnotateRepairFinalAction(ctx, in, repair.ActionFallback)
	in.Failure = graphFailure{}
	in.FailureStage = ""
	in.RecoveryCode = ""
	return in, nil
}

// baziHardErrorNode terminates without exposing rejected model text.
func (e *Executor) baziHardErrorNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, "hard_error"); err != nil {
		return nil, err
	}
	in.TerminationReason = firstNonEmpty(in.TerminationReason, "hard_error")
	in.RecoveryState = baziRecoveryStateHardError
	return in, nil
}

func baziFailureErrorFromState(in *baziInternalGraphState) error {
	if in == nil {
		return nil
	}
	if in.RepairFailure.Message != "" {
		return fmt.Errorf("%s: %s", firstNonEmpty(in.RepairFailure.Code, string(in.RepairFailure.Class)), in.RepairFailure.Message)
	}
	return graphFailureError(in.Failure)
}

// baziGraphRuntimeResult is the graph boundary used by Executor and outer
// dispatch; it returns state-owned failures without returning them from nodes.
func (e *Executor) baziGraphRuntimeResult(ctx context.Context, sink EventSink, view *specialists.SessionView, question string) (Result, error) {
	return e.runBaziDomainGraph(ctx, sink, view, question)
}

// baziGraphTerminalText converts a typed graph failure at the legacy boundary
// into the existing RuntimeFailure contract without changing SSE shape.
func baziGraphTerminalText(result Result) (string, error) {
	if !result.Failure.HasFailure() {
		return result.Text, nil
	}
	failure := graphFailureFromBaziGraph(result.Failure)
	return "", baziSynthesisRuntimeFailure(failure.FailureStage, firstNonEmpty(failure.FailureCode, "BAZI_GRAPH_FAILED"), graphFailureError(failure))
}
