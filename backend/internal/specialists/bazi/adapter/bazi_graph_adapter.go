// package adapter 包含 Manager 拥有的执行主链。
//
// 本文件把 runtime 的模型、检索、追踪和事件能力适配给八字 Graph；
// 不负责选择 Graph 动作、角色选择或循环预算。
package adapter

import (
	"context"
	"fmt"

	"github.com/observer-mimiron/suanming-agent/internal/repair"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	bazigraph "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/graph"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// baziGraphNode is one existing domain operation adapted into the graph-owned loop.
type baziGraphNode func(context.Context, *baziInternalGraphState) (*baziInternalGraphState, error)

// baziGraphDeps binds an Executor's per-turn services to the narrow BaZi graph API.
func (e *Executor) baziGraphDeps() bazigraph.Deps {
	invoke := func(node baziGraphNode) func(context.Context, *bazigraph.State) error {
		return func(ctx context.Context, graphState *bazigraph.State) error {
			return e.runBaziGraphNode(ctx, graphState, node)
		}
	}
	return bazigraph.Deps{
		Bootstrap:        invoke(e.baziBootstrapNode),
		AnalysisPlan:     invoke(e.baziAnalysisPlanNode),
		Evidence:         invoke(e.baziEvidenceActionNode),
		ValidateEvidence: invoke(e.baziValidateEvidenceNode),
		Static:           invoke(e.baziStaticJudgmentV2Node),
		Lifetime:         invoke(e.baziLifetimeDayunJudgmentNode),
		Dynamic:          invoke(e.baziDynamicJudgmentV2Node),
		ContractCheck:    invoke(e.baziContractCheckNode),
		Repair:           invoke(e.baziRepairNode),
		RecoverFacts:     invoke(e.baziRecoverFactsNode),
		Render:           invoke(e.baziRenderNode),
		HardError:        invoke(e.baziHardErrorNode),
		TraceAttributes:  tracing.SetTraceAttributes,
	}
}

// runBaziGraphNode applies graph control facts before a domain operation and
// projects the operation's state-safe results back after it returns.
func (e *Executor) runBaziGraphNode(ctx context.Context, graphState *bazigraph.State, node baziGraphNode) error {
	if graphState == nil {
		return fmt.Errorf("bazi graph state is nil")
	}
	in, ok := graphState.Payload.(*baziInternalGraphState)
	if !ok || in == nil {
		return fmt.Errorf("bazi graph payload is invalid")
	}
	applyBaziGraphControl(in, graphState)
	out, err := node(ctx, in)
	if err != nil {
		return err
	}
	if out == nil {
		return fmt.Errorf("bazi graph node returned nil state")
	}
	graphState.Payload = out
	projectBaziGraphControl(graphState, out)
	return nil
}

// applyBaziGraphControl keeps the legacy domain state aligned with the graph's
// authoritative loop fields while preserving the domain payload itself.
func applyBaziGraphControl(in *baziInternalGraphState, graphState *bazigraph.State) {
	if in == nil || graphState == nil {
		return
	}
	in.Phase = graphState.Phase
	in.LoopStep = graphState.LoopStep
	in.MaxRunSteps = graphState.MaxRunSteps
	in.StaticAttempted = graphState.StaticAttempted
	in.StaticAccepted = graphState.StaticAccepted
	in.LifetimeAttempted = graphState.LifetimeAttempted
	in.LifetimeAccepted = graphState.LifetimeAccepted
	in.DynamicAttempted = graphState.DynamicAttempted
	in.DynamicAccepted = graphState.DynamicAccepted
	in.EvidenceValidated = graphState.EvidenceValidated
	in.EvidenceAttempts = graphState.EvidenceAttempts
	in.TransportAttempts = graphState.TransportAttempts
	in.RepairAttempts = graphState.RepairAttempts
	in.TerminationReason = graphState.TerminationReason
	in.RecoveryState = graphState.RecoveryState
	in.RecoveryPolicy = graphState.RecoveryPolicy
	in.RepairState = graphState.RepairState
	in.RepairAction = graphState.RepairAction
	in.Failure = graphFailureFromBaziGraph(graphState.Failure)
	in.FailureStage = graphState.Failure.Stage
	in.RepairFailure = graphState.RepairFailure
}

// projectBaziGraphControl copies only graph control facts out of the runtime
// payload; candidates and deterministic facts stay inside the domain payload.
func projectBaziGraphControl(graphState *bazigraph.State, in *baziInternalGraphState) {
	if graphState == nil || in == nil {
		return
	}
	// Graph 持有当前 phase。领域节点可以在分类失败时更新本地 phase，但不能
	// 反写 Graph，否则会形成第二个动作选择权，旧 payload 可能劫持下一条边。
	graphState.ChartReady = len(in.ChartState.Input.BaziResult) > 0
	graphState.AnalysisPlanned = in.ChartState.AnalysisPlan.Mode != ""
	graphState.NeedDynamic = in.ChartState.AnalysisPlan.NeedDynamic
	graphState.NeedLifetimeDayun = in.ChartState.AnalysisPlan.NeedLifetimeDayun
	graphState.CurrentPeriodReady = in.FactCapsule.CurrentPeriodRef != ""
	graphState.EvidenceValidated = in.EvidenceValidated
	graphState.EvidenceNeedsAction = baziEvidenceNeedsAction(in)
	graphState.StaticAttempted = in.StaticAttempted
	graphState.StaticAccepted = in.StaticAccepted
	graphState.LifetimeAttempted = in.LifetimeAttempted
	graphState.LifetimeAccepted = in.LifetimeAccepted
	graphState.DynamicAttempted = in.DynamicAttempted
	graphState.DynamicAccepted = in.DynamicAccepted
	graphState.Failure = baziGraphFailureFromRuntime(in.Failure)
	graphState.RecoveryPolicy = in.RecoveryPolicy
	graphState.RepairState = in.RepairState
	graphState.RepairFailure = in.RepairFailure
	graphState.RepairAction = in.RepairAction
	graphState.EvidenceAttempts = in.EvidenceAttempts
	graphState.TransportAttempts = in.TransportAttempts
	graphState.RepairAttempts = in.RepairAttempts
	graphState.RecoveryState = in.RecoveryState
	graphState.TerminationReason = in.TerminationReason
	graphState.Output = in.Output
}

// baziGraphFailureFromRuntime converts the runtime's common failure contract at
// the one allowed adapter boundary.
func baziGraphFailureFromRuntime(failure graphFailure) bazigraph.Failure {
	return bazigraph.Failure{
		Class: failure.FailureClass, Stage: failure.FailureStage, Code: failure.FailureCode,
		Domain: failure.Domain, Retryable: failure.Retryable, Degraded: failure.Degraded,
		Message: failure.Message, MissingRefs: append([]string(nil), failure.MissingRefs...), AllowedRefs: append([]string(nil), failure.AllowedRefs...),
	}
}

// graphFailureFromBaziGraph restores the common RuntimeFailure projection only
// after the domain graph has reached an explicit terminal state.
func graphFailureFromBaziGraph(failure bazigraph.Failure) graphFailure {
	return graphFailure{
		FailureClass: failure.Class, FailureStage: failure.Stage, FailureCode: failure.Code,
		Domain: failure.Domain, Retryable: failure.Retryable, Degraded: failure.Degraded,
		Message: failure.Message, MissingRefs: append([]string(nil), failure.MissingRefs...), AllowedRefs: append([]string(nil), failure.AllowedRefs...),
	}
}

// runBaziDomainGraph is the runtime entrypoint for the specialist-owned graph.
func (e *Executor) runBaziDomainGraph(ctx context.Context, sink EventSink, view *specialists.SessionView, question string) (bazigraph.Result, error) {
	if view == nil {
		return bazigraph.Result{}, fmt.Errorf("bazi graph requires session view")
	}
	ctx = withBaziGraphRuntime(ctx, &baziGraphRuntime{Executor: e, Session: view, Sink: sink})
	payload := &baziInternalGraphState{
		Question: question, Phase: baziPhaseAnalysisPlan, MaxRunSteps: bazigraph.MaxRunSteps,
		RecoveryState: baziRecoveryStateClean, RepairState: repair.NewState(),
	}
	result, err := bazigraph.Run(ctx, e.baziGraphDeps(), &bazigraph.State{
		Phase: baziPhaseAnalysisPlan, MaxRunSteps: bazigraph.MaxRunSteps, RecoveryState: baziRecoveryStateClean,
		RepairState: repair.NewState(), Payload: payload,
	})
	if err != nil {
		return bazigraph.Result{}, err
	}
	terminal, ok := result.Payload.(*baziInternalGraphState)
	if !ok || terminal == nil {
		return bazigraph.Result{}, fmt.Errorf("bazi graph terminal payload is invalid")
	}
	// Graph owns the terminal result; runtime only adds the already-validated audit
	// projection that is not part of action selection.
	result.ContractAudit = terminal.ChartState.StaticSynthesis.ContractAudit
	return result, nil
}
