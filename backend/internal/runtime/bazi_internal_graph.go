// Package runtime contains the manager-owned BaZi execution graph.
//
// This file turns the authority-first BaZi chain into explicit nodes and
// branches. The nodes reuse the existing planner, evidence, synthesis,
// projection and validation functions; they must not add chart-specific
// judgment rules or rewrite renderer semantics.
package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

const (
	baziInternalNodeBootstrap          = "bootstrap"
	baziInternalNodeAnalysisPlan       = "analysis_plan"
	baziInternalNodeEvidence           = "evidence"
	baziInternalNodeEvidenceReflection = "evidence_reflection"
	baziInternalNodeEvidenceValidation = "evidence_validation"
	baziInternalNodeDynamicEvidence    = "dynamic_evidence"
	baziInternalNodeCanonicalSynthesis = "canonical_synthesis"
	baziInternalNodeProjection         = "projection"
	baziInternalNodeStaticValidation   = "static_validation"
	baziInternalNodeDynamicValidation  = "dynamic_validation"
	baziInternalNodeRecoveryDecision   = "recovery_decision"
	baziInternalNodeRender             = "render"
	baziInternalNodeDone               = "done"
)

const (
	baziRecoveryStateClean                    = "clean"
	baziRecoveryStateRetryableModelFailure    = "retryable_model_failure"
	baziRecoveryStateEvidenceOverclaim        = "evidence_overclaim"
	baziRecoveryStateDomainUnauthorized       = "domain_unauthorized"
	baziRecoveryStateFactConflict             = "fact_conflict"
	baziRecoveryStateMethodContractViolation  = "method_contract_violation"
	baziRecoveryStateStaticFactsOnlyDegraded  = "static_facts_only_degraded"
	baziRecoveryStateDynamicFactsOnlyDegraded = "dynamic_facts_only_degraded"
	baziRecoveryStateHardError                = "hard_error"
)

// baziInternalGraphState is the request-local state passed through the inner
// BaZi graph. It keeps model-owned canonical judgments separate from runtime
// projection, validation and recovery decisions.
type baziInternalGraphState struct {
	Session  *state.SessionState
	Sink     EventSink
	Question string

	ChartState baziCharterState
	Canonical  baziCanonicalSynthesis

	Failure        error
	FailureStage   string
	RecoveryCode   string
	RecoveryState  string
	FailureClass   string
	RecoveryPolicy string
	BranchPath     []string
	Output         string
}

// runBaziInternalGraph executes the observable BaZi DAG and returns rendered
// text. The render node still only consumes structured runtime state; it does
// not re-adjudicate BaZi semantics.
func (e *Executor) runBaziInternalGraph(ctx context.Context, sink EventSink, st *state.SessionState, question string) (string, error) {
	runnable, err := e.buildBaziInternalGraph(ctx)
	if err != nil {
		return "", err
	}
	out, err := runnable.Invoke(ctx, &baziInternalGraphState{
		Session:       st,
		Sink:          sink,
		Question:      question,
		RecoveryState: baziRecoveryStateClean,
	})
	if err != nil {
		return "", err
	}
	if out == nil {
		return "", fmt.Errorf("bazi internal graph returned nil state")
	}
	return out.Output, nil
}

// buildBaziInternalGraph wires the BaZi authority-first chain as a graph so
// recovery decisions are explicit branches rather than hidden fallthrough in a
// single long function.
func (e *Executor) buildBaziInternalGraph(ctx context.Context) (compose.Runnable[*baziInternalGraphState, *baziInternalGraphState], error) {
	g := compose.NewGraph[*baziInternalGraphState, *baziInternalGraphState]()

	nodes := []struct {
		key string
		fn  func(context.Context, *baziInternalGraphState) (*baziInternalGraphState, error)
	}{
		{baziInternalNodeBootstrap, e.baziBootstrapNode},
		{baziInternalNodeAnalysisPlan, e.baziAnalysisPlanNode},
		{baziInternalNodeEvidence, e.baziEvidenceNode},
		{baziInternalNodeEvidenceReflection, e.baziEvidenceReflectionNode},
		{baziInternalNodeEvidenceValidation, e.baziEvidenceValidationNode},
		{baziInternalNodeDynamicEvidence, e.baziDynamicEvidenceNode},
		{baziInternalNodeCanonicalSynthesis, e.baziCanonicalSynthesisNode},
		{baziInternalNodeProjection, e.baziProjectionNode},
		{baziInternalNodeStaticValidation, e.baziStaticValidationNode},
		{baziInternalNodeDynamicValidation, e.baziDynamicValidationNode},
		{baziInternalNodeRecoveryDecision, e.baziRecoveryDecisionNode},
		{baziInternalNodeRender, e.baziRenderNode},
		{baziInternalNodeDone, e.baziDoneNode},
	}
	for _, node := range nodes {
		if err := g.AddLambdaNode(node.key,
			compose.InvokableLambda(node.fn),
			compose.WithNodeName("bazi."+node.key)); err != nil {
			return nil, fmt.Errorf("add bazi %s node: %w", node.key, err)
		}
	}

	if err := g.AddEdge(compose.START, baziInternalNodeBootstrap); err != nil {
		return nil, fmt.Errorf("edge START->bazi.bootstrap: %w", err)
	}
	for _, edge := range [][2]string{
		{baziInternalNodeBootstrap, baziInternalNodeAnalysisPlan},
		{baziInternalNodeAnalysisPlan, baziInternalNodeEvidence},
		{baziInternalNodeEvidence, baziInternalNodeEvidenceReflection},
		{baziInternalNodeEvidenceReflection, baziInternalNodeEvidenceValidation},
	} {
		if err := g.AddEdge(edge[0], edge[1]); err != nil {
			return nil, fmt.Errorf("edge bazi.%s->bazi.%s: %w", edge[0], edge[1], err)
		}
	}

	if err := g.AddBranch(baziInternalNodeEvidenceValidation, compose.NewGraphBranch(
		baziNeedDynamicEvidenceBranch,
		map[string]bool{baziInternalNodeDynamicEvidence: true, baziInternalNodeCanonicalSynthesis: true},
	)); err != nil {
		return nil, fmt.Errorf("add bazi evidence branch: %w", err)
	}
	if err := g.AddEdge(baziInternalNodeDynamicEvidence, baziInternalNodeCanonicalSynthesis); err != nil {
		return nil, fmt.Errorf("edge bazi.dynamic_evidence->bazi.canonical_synthesis: %w", err)
	}
	if err := g.AddBranch(baziInternalNodeCanonicalSynthesis, compose.NewGraphBranch(
		baziCanonicalBranch,
		map[string]bool{baziInternalNodeRecoveryDecision: true, baziInternalNodeProjection: true},
	)); err != nil {
		return nil, fmt.Errorf("add bazi canonical branch: %w", err)
	}
	if err := g.AddEdge(baziInternalNodeProjection, baziInternalNodeStaticValidation); err != nil {
		return nil, fmt.Errorf("edge bazi.projection->bazi.static_validation: %w", err)
	}
	if err := g.AddBranch(baziInternalNodeStaticValidation, compose.NewGraphBranch(
		baziAfterStaticValidationBranch,
		map[string]bool{
			baziInternalNodeRecoveryDecision:  true,
			baziInternalNodeDynamicValidation: true,
			baziInternalNodeRender:            true,
		},
	)); err != nil {
		return nil, fmt.Errorf("add bazi static validation branch: %w", err)
	}
	if err := g.AddBranch(baziInternalNodeDynamicValidation, compose.NewGraphBranch(
		baziAfterDynamicValidationBranch,
		map[string]bool{baziInternalNodeRecoveryDecision: true, baziInternalNodeRender: true},
	)); err != nil {
		return nil, fmt.Errorf("add bazi dynamic validation branch: %w", err)
	}
	if err := g.AddEdge(baziInternalNodeRecoveryDecision, baziInternalNodeRender); err != nil {
		return nil, fmt.Errorf("edge bazi.recovery_decision->bazi.render: %w", err)
	}
	if err := g.AddEdge(baziInternalNodeRender, baziInternalNodeDone); err != nil {
		return nil, fmt.Errorf("edge bazi.render->bazi.done: %w", err)
	}
	if err := g.AddEdge(baziInternalNodeDone, compose.END); err != nil {
		return nil, fmt.Errorf("edge bazi.done->END: %w", err)
	}

	return g.Compile(ctx, compose.WithGraphName("bazi_authority_first_internal"))
}

// baziBootstrapNode validates the prefilled chart artifact and creates the
// runtime-owned charter state consumed by every later node.
func (e *Executor) baziBootstrapNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeBootstrap); err != nil {
		return nil, err
	}
	if in.Session == nil || !in.Session.HasBaziResult() {
		err := fmt.Errorf("pure bazi charter graph requires bazi result")
		annotateBaziGraphError(ctx, "bootstrap", err)
		return nil, err
	}
	in.ChartState = baziCharterState{
		Input: baziCharterInput{
			UserQuestion: in.Question,
			BaziResult:   in.Session.BaziResult,
			Yongshen:     mapValue(in.Session.BaziResult, "yongshen"),
			Dayun:        mapValue(in.Session.BaziResult, "dayun_analyzed"),
			Liunian:      mapValue(in.Session.BaziResult, "liunian"),
		},
	}
	return in, nil
}

// baziAnalysisPlanNode runs the model planner but preserves the existing
// deterministic fallback when planning is unavailable.
func (e *Executor) baziAnalysisPlanNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeAnalysisPlan); err != nil {
		return nil, err
	}
	analysisPlan, err := e.runBaziAnalysisPlanner(ctx, in.Session, in.Question, in.ChartState.Input)
	if err != nil {
		annotateBaziGraphError(ctx, "analysis_planner", err)
		analysisPlan = defaultBaziAnalysisPlan(in.Question)
	}
	analysisPlan = normalizeBaziAnalysisPlan(analysisPlan)
	in.ChartState.AnalysisPlan = analysisPlan
	emitBaziStageThinking(ctx, in.Sink, "bazi_graph", analysisPlan.StageSummary)
	return in, nil
}

// baziEvidenceNode retrieves and scores the authority evidence selected by the
// planner. It does not perform synthesis.
func (e *Executor) baziEvidenceNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeEvidence); err != nil {
		return nil, err
	}
	plan, bundle, quality, err := e.runBaziEvidenceStage(ctx, in.Session, in.Question, in.ChartState.Input, in.ChartState.AnalysisPlan)
	if err != nil {
		annotateBaziGraphError(ctx, "evidence_stage", err)
		return nil, err
	}
	in.ChartState.EvidencePlan = plan
	in.ChartState.EvidenceBundle = bundle
	in.ChartState.EvidenceQuality = quality
	return in, nil
}

// baziEvidenceReflectionNode retries only missing or conflicting authority
// topics before synthesis sees the evidence bundle.
func (e *Executor) baziEvidenceReflectionNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeEvidenceReflection); err != nil {
		return nil, err
	}
	chartState, err := e.maybeReflectOnBaziEvidence(ctx, in.Session, in.ChartState)
	if err != nil {
		annotateBaziGraphError(ctx, "evidence_reflection", err)
		return nil, err
	}
	in.ChartState = chartState
	return in, nil
}

// baziEvidenceValidationNode enforces evidence preconditions before model
// judgment, preventing later nodes from treating missing A-level topics as
// supported evidence.
func (e *Executor) baziEvidenceValidationNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeEvidenceValidation); err != nil {
		return nil, err
	}
	if err := validateEvidenceBundlePreconditions(in.ChartState); err != nil {
		annotateBaziGraphError(ctx, "evidence_validation", err)
		return nil, err
	}
	emitBaziStageThinking(ctx, in.Sink, "bazi_graph", buildEvidenceStageSummary(in.ChartState))
	return in, nil
}

// baziDynamicEvidenceNode supplements luck-period evidence only when the
// planner requested dynamic analysis.
func (e *Executor) baziDynamicEvidenceNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeDynamicEvidence); err != nil {
		return nil, err
	}
	chartState, err := e.supplementDynamicEvidenceIfNeeded(ctx, in.Session, in.Question, in.ChartState)
	if err != nil {
		annotateBaziGraphError(ctx, "dynamic_evidence", err)
		return nil, err
	}
	in.ChartState = chartState
	return in, nil
}

// baziCanonicalSynthesisNode is the only model-owned whole-chart judgment
// node. Runtime projection and evidence status stay downstream.
func (e *Executor) baziCanonicalSynthesisNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeCanonicalSynthesis); err != nil {
		return nil, err
	}
	canonical, err := e.runCanonicalSynthesis(ctx, in.Session, in.ChartState, in.Question)
	if err != nil {
		annotateBaziGraphError(ctx, "canonical_synthesis", err)
		baziRecordInternalFailure(ctx, in, "canonical_synthesis", err, "canonical_parse_failure_facts_only")
		return in, nil
	}
	canonical.ContractAudit = baziContractAudit{Compliant: true}
	annotateCanonicalSynthesis(ctx, canonical)
	in.Canonical = canonical
	return in, nil
}

// baziProjectionNode derives legacy static/dynamic renderer fields from the
// canonical judgment. Models do not get a second chance to restate legacy
// semantics here.
func (e *Executor) baziProjectionNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeProjection); err != nil {
		return nil, err
	}
	staticSynthesis, dynamicSynthesis := projectCanonicalSynthesis(in.ChartState, in.Canonical)
	in.ChartState.StaticSynthesis = staticSynthesis
	in.ChartState.DynamicSynthesis = dynamicSynthesis
	return in, nil
}

// baziStaticValidationNode validates runtime projection into the legacy static
// renderer shape. Only evidence overclaim can recover to static facts-only;
// fact and method conflicts remain hard errors.
func (e *Executor) baziStaticValidationNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeStaticValidation); err != nil {
		return nil, err
	}
	if err := validateStaticSynthesisResult(in.ChartState, in.ChartState.StaticSynthesis); err != nil {
		annotateBaziGraphError(ctx, "static_projection", err)
		baziRecordInternalFailure(ctx, in, "static_projection", err, "canonical_static_projection_facts_only")
	}
	return in, nil
}

// baziDynamicValidationNode validates dynamic projection and records recoverable
// domain overreach for the recovery node. Method or fact conflicts remain hard
// errors through baziRecoveryDecisionNode.
func (e *Executor) baziDynamicValidationNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeDynamicValidation); err != nil {
		return nil, err
	}
	in.ChartState.DynamicSynthesis = sanitizeDynamicPresentationBoundaries(normalizeDynamicSynthesis(in.ChartState.DynamicSynthesis))
	if err := validateDynamicSynthesisAfterGraphNormalization(in.ChartState); err != nil {
		annotateBaziGraphError(ctx, "dynamic_projection", err)
		baziRecordInternalFailure(ctx, in, "dynamic_projection", err, "canonical_dynamic_projection_facts_only")
	}
	return in, nil
}

// baziRecoveryDecisionNode is the single state-machine transition for synthesis
// recovery. It keeps candidate model text out of the renderer whenever a
// recovery path is selected.
func (e *Executor) baziRecoveryDecisionNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeRecoveryDecision); err != nil {
		return nil, err
	}
	if in.Failure == nil {
		return in, nil
	}

	switch in.FailureStage {
	case "canonical_synthesis":
		staticSynthesis, dynamicSynthesis := canonicalFailureFactsOnly(in.ChartState, in.Failure, in.RecoveryCode, "最小裁断综合未能生成可解析 JSON，已降级展示可复算事实。")
		in.ChartState.StaticSynthesis = staticSynthesis
		in.ChartState.DynamicSynthesis = dynamicSynthesis
		in.RecoveryState = baziRecoveryStateStaticFactsOnlyDegraded
	case "static_projection":
		failure, ok := baziContractFailureFromError("static_projection", in.Failure)
		if !ok || failure.RecoveryPolicy != baziRecoveryPolicyStaticFactsOnly {
			in.RecoveryState = baziRecoveryStateHardError
			return nil, baziSynthesisRuntimeFailure("static_projection", "BAZI_STATIC_PROJECTION_FAILED", in.Failure)
		}
		staticSynthesis, dynamicSynthesis := canonicalFailureFactsOnly(in.ChartState, in.Failure, in.RecoveryCode, "最小裁断静态投影未通过合同校验，已降级展示可复算事实。")
		if err := validateStaticSynthesisResult(withStaticSynthesis(in.ChartState, staticSynthesis), staticSynthesis); err != nil {
			in.RecoveryState = baziRecoveryStateHardError
			return nil, baziSynthesisRuntimeFailure("static_projection", "BAZI_STATIC_PROJECTION_FAILED", err)
		}
		in.ChartState.StaticSynthesis = staticSynthesis
		in.ChartState.DynamicSynthesis = dynamicSynthesis
		in.RecoveryState = baziRecoveryStateStaticFactsOnlyDegraded
	case "dynamic_projection":
		failure, ok := baziContractFailureFromError("dynamic_projection", in.Failure)
		if ok && failure.RecoveryPolicy == baziRecoveryPolicyDynamicFactsOnly {
			in.ChartState.DynamicSynthesis = canonicalDynamicFailureFactsOnly(in.ChartState, in.ChartState.StaticSynthesis, in.Failure)
			in.RecoveryState = baziRecoveryStateDynamicFactsOnlyDegraded
		} else {
			in.RecoveryState = baziRecoveryStateHardError
			return nil, baziSynthesisRuntimeFailure("dynamic_projection", "BAZI_DYNAMIC_PROJECTION_FAILED", in.Failure)
		}
	default:
		in.RecoveryState = baziRecoveryStateHardError
		return nil, in.Failure
	}

	in.Failure = nil
	in.FailureStage = ""
	in.RecoveryCode = ""
	return in, nil
}

// baziRenderNode emits visible summaries, trace attributes and the final text
// from structured synthesis. It must not introduce new BaZi judgments.
func (e *Executor) baziRenderNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeRender); err != nil {
		return nil, err
	}
	in.ChartState.FieldAudit = append(in.ChartState.FieldAudit, in.ChartState.StaticSynthesis.FieldAudit...)
	in.ChartState.FieldAudit = append(in.ChartState.FieldAudit, in.ChartState.DynamicSynthesis.FieldAudit...)
	emitBaziStageThinking(ctx, in.Sink, "bazi_graph", buildStaticStageSummary(in.ChartState))
	if !isFactsOnlyStaticSynthesis(in.ChartState.StaticSynthesis) && !isPartialSynthesisSource(in.ChartState.StaticSynthesis.Source) {
		emitBaziReasoningSteps(ctx, in.Sink, "静态推演", in.ChartState.StaticSynthesis.ReasoningSteps)
	}
	if in.ChartState.AnalysisPlan.NeedDynamic {
		emitBaziStageThinking(ctx, in.Sink, "bazi_graph", buildDynamicStageSummary(in.ChartState))
		if !isFactsOnlyDynamicSynthesis(in.ChartState.DynamicSynthesis) && !isPartialSynthesisSource(in.ChartState.DynamicSynthesis.Source) {
			emitBaziReasoningSteps(ctx, in.Sink, "动态推演", in.ChartState.DynamicSynthesis.ReasoningSteps)
		}
	}
	annotateBaziSynthesisSources(ctx, in.ChartState)
	annotateBaziSoftAudit(ctx, in.ChartState)
	baziAnnotateInternalGraphPath(ctx, in)
	output, err := e.runFinalWriter(ctx, in.Session, in.ChartState, in.Question)
	if err != nil {
		return nil, err
	}
	in.Output = output
	return in, nil
}

// baziDoneNode marks the successful terminal state after render has produced
// user-visible text.
func (e *Executor) baziDoneNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeDone); err != nil {
		return nil, err
	}
	return in, nil
}

// baziNeedDynamicEvidenceBranch skips dynamic evidence retrieval when the
// planner selected a static-only answer path.
func baziNeedDynamicEvidenceBranch(ctx context.Context, in *baziInternalGraphState) (string, error) {
	if in.ChartState.AnalysisPlan.NeedDynamic {
		return baziMarkInternalBranch(ctx, in, "need_dynamic", baziInternalNodeDynamicEvidence), nil
	}
	return baziMarkInternalBranch(ctx, in, "static_only", baziInternalNodeCanonicalSynthesis), nil
}

// baziCanonicalBranch routes malformed or invalid canonical synthesis into
// deterministic facts-only recovery before any renderer sees model text.
func baziCanonicalBranch(ctx context.Context, in *baziInternalGraphState) (string, error) {
	if in.Failure != nil {
		return baziMarkInternalBranch(ctx, in, "canonical_recovery", baziInternalNodeRecoveryDecision), nil
	}
	return baziMarkInternalBranch(ctx, in, "canonical_clean", baziInternalNodeProjection), nil
}

// baziAfterStaticValidationBranch decides whether the clean static projection
// needs dynamic validation or can proceed directly to final rendering.
func baziAfterStaticValidationBranch(ctx context.Context, in *baziInternalGraphState) (string, error) {
	if in.Failure != nil {
		return baziMarkInternalBranch(ctx, in, "static_recovery", baziInternalNodeRecoveryDecision), nil
	}
	if in.ChartState.AnalysisPlan.NeedDynamic {
		return baziMarkInternalBranch(ctx, in, "dynamic_validation", baziInternalNodeDynamicValidation), nil
	}
	return baziMarkInternalBranch(ctx, in, "static_final", baziInternalNodeRender), nil
}

// baziAfterDynamicValidationBranch sends recoverable dynamic failures to the
// recovery state and clean outputs to finalization.
func baziAfterDynamicValidationBranch(ctx context.Context, in *baziInternalGraphState) (string, error) {
	if in.Failure != nil {
		return baziMarkInternalBranch(ctx, in, "dynamic_recovery", baziInternalNodeRecoveryDecision), nil
	}
	return baziMarkInternalBranch(ctx, in, "dynamic_clean", baziInternalNodeRender), nil
}

// baziMarkInternalNode records the latest node and path for trace search. The
// final path is emitted as one compact attribute in baziRenderNode.
func baziMarkInternalNode(ctx context.Context, in *baziInternalGraphState, node string) error {
	if in == nil {
		return fmt.Errorf("bazi internal graph state is nil at %s", node)
	}
	in.BranchPath = append(in.BranchPath, "node:"+node)
	tracing.SetTraceAttributes(ctx, map[string]any{
		"bazi.internal_graph.node": node,
	})
	return nil
}

// baziMarkInternalBranch records why graph control moved to the next node.
func baziMarkInternalBranch(ctx context.Context, in *baziInternalGraphState, branch, target string) string {
	if in != nil {
		in.BranchPath = append(in.BranchPath, "branch:"+branch+"->"+target)
	}
	tracing.SetTraceAttributes(ctx, map[string]any{
		"bazi.internal_graph.branch": branch,
		"bazi.internal_graph.target": target,
	})
	return target
}

// baziAnnotateInternalGraphPath exposes the compact graph trajectory for
// debugging stability failures without storing full candidate synthesis text.
func baziAnnotateInternalGraphPath(ctx context.Context, in *baziInternalGraphState) {
	if in == nil {
		return
	}
	attrs := map[string]any{}
	if len(in.BranchPath) > 0 {
		attrs["bazi.internal_graph.path"] = strings.Join(in.BranchPath, " | ")
	}
	if strings.TrimSpace(in.RecoveryState) != "" {
		attrs["bazi.internal_graph.recovery_state"] = in.RecoveryState
	}
	if strings.TrimSpace(in.FailureClass) != "" {
		attrs["bazi.contract.failure_class"] = in.FailureClass
	}
	if strings.TrimSpace(in.RecoveryPolicy) != "" {
		attrs["bazi.contract.recovery_policy"] = in.RecoveryPolicy
	}
	if len(attrs) > 0 {
		tracing.SetTraceAttributes(ctx, attrs)
	}
}

// baziRecordInternalFailure preserves one failure as state-machine metadata so
// recovery decisions and trace attributes do not have to infer policy from a
// later generic error.
func baziRecordInternalFailure(ctx context.Context, in *baziInternalGraphState, stage string, err error, recoveryCode string) {
	in.Failure = err
	in.FailureStage = stage
	in.RecoveryCode = recoveryCode
	in.RecoveryState = baziRecoveryStateRetryableModelFailure
	in.RecoveryPolicy = baziRecoveryPolicyHardError

	attrs := map[string]any{
		"bazi.internal_graph.recovery_state": in.RecoveryState,
	}
	if recoveryCode != "" {
		attrs["bazi.contract.recovery_code"] = recoveryCode
	}
	if failure, ok := baziContractFailureFromError(stage, err); ok {
		in.FailureClass = failure.Class
		in.RecoveryPolicy = failure.RecoveryPolicy
		in.RecoveryState = baziRecoveryStateForFailure(failure)
		attrs["bazi.internal_graph.recovery_state"] = in.RecoveryState
		attrs["bazi.contract.failure_class"] = failure.Class
		attrs["bazi.contract.recovery_policy"] = failure.RecoveryPolicy
		if failure.FindingCode != "" {
			attrs["bazi.contract.finding_code"] = failure.FindingCode
		}
		if failure.Field != "" {
			attrs["bazi.contract.finding_field"] = failure.Field
		}
	} else if stage == "canonical_synthesis" {
		in.RecoveryPolicy = baziRecoveryPolicyFullFactsOnly
		attrs["bazi.contract.failure_class"] = baziRecoveryStateRetryableModelFailure
		attrs["bazi.contract.recovery_policy"] = in.RecoveryPolicy
	}
	tracing.SetTraceAttributes(ctx, attrs)
}

// baziRecoveryStateForFailure converts the closed contract taxonomy into the
// graph state-machine labels used by branches, tests and trace search.
func baziRecoveryStateForFailure(failure baziContractFailure) string {
	switch failure.Class {
	case baziContractFailureEvidenceOverclaim:
		return baziRecoveryStateEvidenceOverclaim
	case baziContractFailureDomainUnauthorized:
		return baziRecoveryStateDomainUnauthorized
	case baziContractFailureFactConflict:
		return baziRecoveryStateFactConflict
	case baziContractFailureMethodContract:
		return baziRecoveryStateMethodContractViolation
	default:
		return baziRecoveryStateRetryableModelFailure
	}
}

// withStaticSynthesis returns a copy of the graph state for validating a
// recovered static projection without mutating other fields first.
func withStaticSynthesis(state baziCharterState, static baziStaticSynthesis) baziCharterState {
	state.StaticSynthesis = static
	return state
}
