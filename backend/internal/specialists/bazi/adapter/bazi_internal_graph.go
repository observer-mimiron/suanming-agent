// Package adapter 包含八字内部 Graph 的运行时节点适配。
//
// 本文件把 authority-first 八字链路拆成显式节点和分支；只编排规划、
// 取证、裁断、投影、校验、repair 和恢复，不新增命盘专项裁断，
// 也不改写 renderer 语义或 specialist 最终答复边界。
package adapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/repair"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	baziapplication "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/application"
	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

const (
	baziInternalNodeBootstrap    = "bootstrap"
	baziInternalNodeAnalysisPlan = "analysis_plan"
	baziInternalNodeRender       = "render"
)

const (
	baziRecoveryStateClean                    = bazidomain.RecoveryStateClean
	baziRecoveryStateRetryableModelFailure    = bazidomain.RecoveryStateRetryableModelFailure
	baziRecoveryStateEvidenceOverclaim        = bazidomain.RecoveryStateEvidenceOverclaim
	baziRecoveryStateDomainUnauthorized       = bazidomain.RecoveryStateDomainUnauthorized
	baziRecoveryStateFactConflict             = bazidomain.RecoveryStateFactConflict
	baziRecoveryStateMethodContractViolation  = bazidomain.RecoveryStateMethodContractViolation
	baziRecoveryStateStaticFactsOnlyDegraded  = bazidomain.RecoveryStateStaticFactsOnlyDegraded
	baziRecoveryStateDynamicFactsOnlyDegraded = bazidomain.RecoveryStateDynamicFactsOnlyDegraded
	baziRecoveryStateHardError                = bazidomain.RecoveryStateHardError
)

// baziInternalGraphState 是八字内部 graph 的单轮状态。
// 它只保存可描述的输入、候选、合同、预算和终止信息；会话、Executor
// 与 SSE sink 通过本轮 context 注入，避免把运行时指针带入 Graph state；
// repair 仅保存短快照，完整反馈只在当前节点调用期间存在。
type baziInternalGraphState struct {
	Question string

	ChartState     baziCharterState
	Canonical      baziCanonicalSynthesis
	ChartInput     baziCharterInput
	FactCapsule    BaziFactCapsule
	RuntimeCatalog baziRuntimeCatalog

	Phase       string
	LoopStep    int
	MaxRunSteps int

	StaticCandidate   baziStaticSynthesis
	LifetimeCandidate baziLifetimeDayunSynthesis
	DynamicCandidate  baziDynamicSynthesis
	AcceptedStatic    baziStaticSynthesis
	AcceptedDynamic   baziDynamicSynthesis
	StaticAttempted   bool
	StaticAccepted    bool
	LifetimeAttempted bool
	LifetimeAccepted  bool
	DynamicAttempted  bool
	DynamicAccepted   bool
	EvidenceValidated bool

	Failure           graphFailure
	FailureStage      string
	RecoveryCode      string
	RecoveryState     string
	FailureClass      string
	RecoveryPolicy    string
	RepairState       repair.State
	RepairFailure     repair.FailureSnapshot
	RepairAction      repair.Action
	RepairedStage     string
	BranchPath        []string
	Output            string
	EvidenceAttempts  int
	TransportAttempts int
	RepairAttempts    int
	TerminationReason string
}

// runBaziInternalGraph executes the bounded BaZi graph and returns rendered
// text. The legacy string boundary remains for outer callers; state-machine
// failures are converted after the graph reaches terminal state.
func (e *Executor) runBaziInternalGraph(ctx context.Context, sink EventSink, view *specialists.SessionView, question string) (string, error) {
	result, err := e.baziGraphRuntimeResult(ctx, sink, view, question)
	if err != nil {
		return "", err
	}
	return baziGraphTerminalText(result)
}

// baziBootstrapNode validates the prefilled chart artifact and creates the
// runtime-owned charter state consumed by every later node.
func (e *Executor) baziBootstrapNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeBootstrap); err != nil {
		return nil, err
	}
	runtime, err := baziGraphRuntimeFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if len(runtime.Session.BaziResult) == 0 {
		err := fmt.Errorf("pure bazi charter graph requires bazi result")
		annotateBaziGraphError(ctx, "bootstrap", err)
		// 缺失命盘是可分类的资产合同失败，保留稳定错误码，避免被泛化为
		// RUNTIME_EXECUTION_FAILED 后让上层无法区分“没准备好资产”和基础设施故障。
		in.Failure = graphFailure{
			FailureClass: failureClassArtifactMissing,
			FailureStage: "bootstrap",
			FailureCode:  "BAZI_CHART_MISSING",
			Domain:       "bazi",
			Retryable:    false,
			Message:      "八字命盘事实未就绪，无法继续内部裁断。",
		}
		in.FailureStage = "bootstrap"
		in.FailureClass = failureClassArtifactMissing
		in.RecoveryPolicy = baziRecoveryPolicyHardError
		in.RecoveryState = baziRecoveryStateHardError
		return in, nil
	}
	in.ChartState = baziCharterState{
		Input: baziCharterInput{
			UserQuestion: in.Question,
			BaziResult:   runtime.Session.BaziResult,
			Yongshen:     mapValue(runtime.Session.BaziResult, "yongshen"),
			Dayun:        mapValue(runtime.Session.BaziResult, "dayun_analyzed"),
			Liunian:      mapValue(runtime.Session.BaziResult, "liunian"),
		},
	}
	in.ChartInput = in.ChartState.Input
	in.FactCapsule = buildBaziFactCapsule(in.ChartState)
	in.RuntimeCatalog = buildBaziRuntimeCatalog(in.ChartState)
	return in, nil
}

// baziAnalysisPlanNode runs the model planner but preserves the existing
// deterministic fallback when planning is unavailable.
func (e *Executor) baziAnalysisPlanNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeAnalysisPlan); err != nil {
		return nil, err
	}
	runtime, err := baziGraphRuntimeFromContext(ctx)
	if err != nil {
		return nil, err
	}
	analysisPlan, err := e.runBaziAnalysisPlanner(ctx, runtime.Session, in.Question, in.ChartState.Input)
	if err != nil {
		annotateBaziGraphError(ctx, "analysis_planner", err)
		analysisPlan = defaultBaziAnalysisPlan(in.Question)
	}
	analysisPlan = normalizeBaziAnalysisPlan(analysisPlan)
	in.ChartState.AnalysisPlan = analysisPlan
	emitBaziStageThinking(ctx, runtime.Sink, "bazi_graph", analysisPlan.StageSummary)
	return in, nil
}

// baziRenderNode emits visible summaries, trace attributes and the final text
// from structured synthesis. It must not introduce new BaZi judgments.
func (e *Executor) baziRenderNode(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, baziInternalNodeRender); err != nil {
		return nil, err
	}
	runtime, err := baziGraphRuntimeFromContext(ctx)
	if err != nil {
		return nil, err
	}
	in.ChartState.FieldAudit = append(in.ChartState.FieldAudit, in.ChartState.StaticSynthesis.FieldAudit...)
	in.ChartState.FieldAudit = append(in.ChartState.FieldAudit, in.ChartState.DynamicSynthesis.FieldAudit...)
	emitBaziStageThinking(ctx, runtime.Sink, "bazi_graph", baziapplication.BuildStaticStageSummary(in.ChartState))
	if !isFactsOnlyStaticSynthesis(in.ChartState.StaticSynthesis) {
		emitBaziReasoningSteps(ctx, runtime.Sink, "静态推演", in.ChartState.StaticSynthesis.ReasoningSteps)
	}
	if in.ChartState.AnalysisPlan.NeedDynamic {
		emitBaziStageThinking(ctx, runtime.Sink, "bazi_graph", baziapplication.BuildDynamicStageSummary(in.ChartState))
		if !isFactsOnlyDynamicSynthesis(in.ChartState.DynamicSynthesis) {
			emitBaziReasoningSteps(ctx, runtime.Sink, "动态推演", in.ChartState.DynamicSynthesis.ReasoningSteps)
		}
	}
	annotateBaziSynthesisSources(ctx, in.ChartState)
	annotateBaziSoftAudit(ctx, in.ChartState)
	baziAnnotateInternalGraphPath(ctx, in)
	output, err := e.runFinalWriter(ctx, runtime.Session, in.ChartState, in.Question)
	if err != nil {
		baziRecordInternalFailure(ctx, in, failureStageFinalWriter, err, "final_writer_failed")
		in.TerminationReason = "hard_error"
		return in, nil
	}
	in.Output = output
	return in, nil
}

// baziAnnotateRepairFinalAction 更新最后 repair 结果，不投影 feedback value。
func baziAnnotateRepairFinalAction(ctx context.Context, in *baziInternalGraphState, action repair.Action) {
	if in == nil || in.RepairFailure.Domain == "" {
		return
	}
	failure := in.RepairFailure.Failure()
	attempt := repair.AttemptsFor(in.RepairState, failure)
	decision := repair.DefaultPolicy().Decide(failure, in.RepairState)
	exhausted := decision.Exhausted
	if action == repair.ActionAccept {
		exhausted = false
	}
	feedbackKeys, hintCount := baziLastRepairAttemptMetadata(in.RepairState)
	attrs := repair.TraceAttrs(repair.TraceEvent{
		Failure:           failure,
		Attempt:           attempt,
		MaxAttempts:       decision.MaxAttempts,
		Action:            in.RepairAction,
		FeedbackKeys:      feedbackKeys,
		LearningHintCount: hintCount,
		Exhausted:         exhausted,
		FinalAction:       action,
	})
	if initial := in.RepairState.InitialFailure; initial.Domain != "" {
		attrs["repair.initial_class"] = string(initial.Class)
		attrs["repair.initial_stage"] = initial.Stage
		attrs["repair.initial_field"] = initial.Field
	}
	if last := in.RepairState.LastFailure; last.Domain != "" {
		attrs["repair.last_class"] = string(last.Class)
		attrs["repair.last_stage"] = last.Stage
		attrs["repair.last_field"] = last.Field
	}
	attrs["repair.final_class"] = string(failure.Class)
	if action == repair.ActionAccept {
		attrs["repair.candidate_status"] = "accepted_after_repair"
	}
	attrs["recovery.final_state"] = in.RecoveryState
	attrs["recovery.final_policy"] = in.RecoveryPolicy
	tracing.SetTraceAttributes(ctx, attrs)
}

// baziLastRepairAttemptMetadata returns only metadata retained from the most recent repair prompt.
func baziLastRepairAttemptMetadata(state repair.State) ([]string, int) {
	for index := len(state.Attempts) - 1; index >= 0; index-- {
		attempt := state.Attempts[index]
		if attempt.Action == repair.ActionRepairNode {
			return append([]string(nil), attempt.FeedbackKeys...), attempt.LearningHintCount
		}
	}
	return nil, 0
}

// baziAcceptRepair 标记领域 repair 成功并清理旧失败状态。
func baziAcceptRepair(ctx context.Context, in *baziInternalGraphState, canonical baziCanonicalSynthesis) {
	if in == nil {
		return
	}
	in.Canonical = canonical
	in.Failure = graphFailure{}
	in.FailureStage = ""
	in.RecoveryCode = ""
	in.FailureClass = ""
	in.RecoveryPolicy = ""
	in.RecoveryState = baziRecoveryStateClean
	in.RepairAction = repair.ActionAccept
	baziFinalizeContractTrace(ctx)
	baziAnnotateRepairFinalAction(ctx, in, repair.ActionAccept)
}

// baziFinalizeContractTrace clears transient graph errors and preserves repair history.
func baziFinalizeContractTrace(ctx context.Context) {
	tracing.SetTraceAttributes(ctx, map[string]any{
		"bazi.graph.error":                   "",
		"bazi.graph.error_stage":             "",
		"bazi.static.error":                  "",
		"bazi.static.error_stage":            "",
		"bazi.dynamic.error":                 "",
		"bazi.dynamic.error_stage":           "",
		"bazi.inner_agent.error":             "",
		"bazi.inner_agent.stage":             "",
		"bazi.internal_graph.recovery_state": baziRecoveryStateClean,
	})
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
// later generic error. It also projects a safe terminal repair result when a
// prior business repair is followed by a failure that is no longer repairable.
func baziRecordInternalFailure(ctx context.Context, in *baziInternalGraphState, stage string, err error, recoveryCode string) {
	if in == nil {
		return
	}
	in.Failure = graphFailureFromError("bazi", stage, err)
	in.FailureStage = stage
	in.RecoveryCode = recoveryCode
	in.RecoveryState = baziRecoveryStateRetryableModelFailure
	in.FailureClass = ""
	in.RecoveryPolicy = baziRecoveryPolicyHardError

	attrs := map[string]any{
		"bazi.internal_graph.recovery_state": in.RecoveryState,
		"bazi.contract.failure_class":        "",
		"bazi.contract.recovery_policy":      "",
		"bazi.contract.finding_code":         "",
		"bazi.contract.finding_field":        "",
	}
	var currentRepairFailure repair.Failure
	var hasCurrentRepairFailure bool
	if recoveryCode != "" {
		attrs["bazi.contract.recovery_code"] = recoveryCode
	}
	if violation, ok := baziViolationFromError(err); ok {
		if len(violation.MissingRefs) > 0 {
			attrs["bazi.contract.invalid_refs"] = strings.Join(violation.MissingRefs, ",")
		}
		if len(violation.AllowedRefs) > 0 {
			attrs["bazi.contract.allowed_refs"] = strings.Join(violation.AllowedRefs, ",")
		}
	}
	if failure, ok := baziContractFailureFromError(stage, err); ok {
		in.FailureClass = failure.Class
		in.RecoveryPolicy = failure.RecoveryPolicy
		in.RecoveryState = baziRecoveryStateForFailure(failure)
		in.Failure = graphFailure{
			FailureClass: failure.Class,
			FailureStage: stage,
			FailureCode:  firstNonEmpty(failure.FindingCode, failure.Class),
			Domain:       "bazi",
			Retryable:    failure.RecoveryPolicy == baziRecoveryPolicyRetryOnly,
			Degraded:     failure.RecoveryPolicy != baziRecoveryPolicyHardError,
			Message:      firstNonEmpty(failure.Reason, err.Error()),
			MissingRefs:  append([]string(nil), failure.MissingRefs...),
			AllowedRefs:  append([]string(nil), failure.AllowedRefs...),
		}
		repairFailure, repairOK := baziapplication.RepairFailureFromError(stage, err)
		if !repairOK {
			repairFailure = repair.Failure{
				Domain: "bazi", Stage: stage, Class: repair.DeterministicConflict,
				Origin: repair.OriginSystem, Code: failure.FindingCode, Message: failure.Reason,
			}
		}
		in.RepairFailure = repairFailure.Snapshot()
		currentRepairFailure = repairFailure
		hasCurrentRepairFailure = true
		attrs["bazi.internal_graph.recovery_state"] = in.RecoveryState
		attrs["bazi.contract.failure_class"] = failure.Class
		attrs["bazi.contract.recovery_policy"] = failure.RecoveryPolicy
		if failure.FindingCode != "" {
			attrs["bazi.contract.finding_code"] = failure.FindingCode
		}
		if failure.Field != "" {
			attrs["bazi.contract.finding_field"] = failure.Field
		}
	} else if repairFailure, ok := baziapplication.RepairFailureFromError(stage, err); ok {
		in.FailureClass = string(repairFailure.Class)
		in.RecoveryPolicy = baziRecoveryPolicyRetryOnly
		in.Failure = graphFailure{
			FailureClass: string(repairFailure.Class),
			FailureStage: stage,
			FailureCode:  firstNonEmpty(repairFailure.Code, string(repairFailure.Class)),
			Domain:       "bazi",
			Retryable:    repairFailure.Retryable,
			Message:      repairFailure.Message,
			MissingRefs:  append([]string(nil), repairFailure.MissingRefs...),
			AllowedRefs:  append([]string(nil), repairFailure.AllowedRefs...),
		}
		in.RepairFailure = repairFailure.Snapshot()
		currentRepairFailure = repairFailure
		hasCurrentRepairFailure = true
		attrs["bazi.contract.failure_class"] = in.FailureClass
		attrs["bazi.contract.recovery_policy"] = in.RecoveryPolicy
		attrs["bazi.contract.finding_field"] = repairFailure.Field
	}
	// 仅在本轮已有业务 repair 且当前 policy 不再允许继续 repair 时补写终态；
	// 复用安全投影，避免把 repair 后的候选正文或 feedback value 写入 trace。
	if hasCurrentRepairFailure {
		in.RepairState = repair.RecordFailure(in.RepairState, currentRepairFailure)
		decision := repair.DefaultPolicy().Decide(currentRepairFailure, in.RepairState)
		if initial := in.RepairState.InitialFailure; initial.Domain != "" {
			attrs["repair.initial_class"] = string(initial.Class)
			attrs["repair.initial_stage"] = initial.Stage
			attrs["repair.initial_field"] = initial.Field
		}
		if last := in.RepairState.LastFailure; last.Domain != "" {
			attrs["repair.last_class"] = string(last.Class)
			attrs["repair.last_stage"] = last.Stage
			attrs["repair.last_field"] = last.Field
			attrs["repair.failure_origin"] = string(last.Origin)
		}
		businessRepairAttempts := 0
		for _, attempt := range in.RepairState.Attempts {
			if attempt.Action == repair.ActionRepairNode {
				businessRepairAttempts++
			}
		}
		if businessRepairAttempts > 0 && !decision.Repairable {
			tracing.SetTraceAttributes(ctx, repair.TraceAttrs(repair.TraceEvent{
				Failure:     currentRepairFailure,
				Attempt:     businessRepairAttempts,
				MaxAttempts: decision.MaxAttempts,
				Action:      decision.Action,
				Exhausted:   decision.Exhausted,
				FinalAction: decision.Action,
			}))
		}
	}
	tracing.SetTraceAttributes(ctx, attrs)
}

// baziRecoveryStateForFailure converts the closed contract taxonomy into the
// graph state-machine labels used by branches, tests and trace search.
func baziRecoveryStateForFailure(failure baziContractFailure) string {
	return bazidomain.RecoveryStateForFailure(failure)
}

// baziStaticJudgmentV2Node makes the single static model judgment; there is no canonical precursor.
func (e *Executor) baziStaticJudgmentV2Node(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, "static_judgment"); err != nil {
		return nil, err
	}
	runtime, err := baziGraphRuntimeFromContext(ctx)
	if err != nil {
		return nil, err
	}
	in.StaticAttempted = true
	canonical, err := e.runStaticSynthesis(ctx, runtime.Session, in.ChartState, baziCanonicalSynthesis{}, in.Question)
	if err != nil {
		baziRecordInternalFailure(ctx, in, "static_synthesis", err, "static_judgment_failed")
		return in, nil
	}
	canonical.Source, canonical.ContractAudit = "model", baziContractAudit{Compliant: true}
	in.Canonical = canonical
	// Projection is a one-way runtime adapter; semantic acceptance remains in contract_gate.
	in.ChartState.StaticSynthesis = projectCanonicalStaticSynthesis(in.ChartState, canonical)
	in.StaticCandidate = in.ChartState.StaticSynthesis
	return in, nil
}

// baziDynamicJudgmentV2Node evaluates only the runtime-bound current period and target year.
// 尚未交入第一步大运时没有合法的 current_period_ref，直接保留确定性事实，避免模型伪造 dayun[n]。
func (e *Executor) baziDynamicJudgmentV2Node(ctx context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
	if err := baziMarkInternalNode(ctx, in, "dynamic_judgment"); err != nil {
		return nil, err
	}
	// 先记录节点已处理，首步未交运也必须消耗本次 dynamic 动作；否则
	// decide_next 会把 facts-only 分支再次派回 dynamic，直到撞上步数上限。
	in.DynamicAttempted = true
	if in.Failure.hasFailure() || !in.ChartState.AnalysisPlan.NeedDynamic {
		return in, nil
	}
	if buildBaziFactCapsule(in.ChartState).CurrentPeriodRef == "" {
		in.Canonical.Source = baziSynthesisSourceFactsOnlyDegraded
		in.Canonical.RecoveryReason = "当前尚未交入第一步大运，动态层仅展示可复算事实。"
		return in, nil
	}
	runtime, err := baziGraphRuntimeFromContext(ctx)
	if err != nil {
		return nil, err
	}
	canonical, err := e.runDynamicSynthesis(ctx, runtime.Session, in.ChartState, in.Canonical, in.Question)
	if err != nil {
		baziRecordInternalFailure(ctx, in, "dynamic_synthesis", err, "dynamic_judgment_failed")
		return in, nil
	}
	in.Canonical = canonical
	in.DynamicCandidate = projectCanonicalDynamicSynthesis(in.ChartState, canonical, in.ChartState.StaticSynthesis)
	return in, nil
}

// baziValidateV2Projection applies the final static and dynamic contracts without rendering text.
func baziValidateV2Projection(in *baziInternalGraphState) error {
	if err := validateStaticSynthesisResult(in.ChartState, in.ChartState.StaticSynthesis); err != nil {
		return err
	}
	if !in.ChartState.AnalysisPlan.NeedDynamic {
		return nil
	}
	in.ChartState.DynamicSynthesis = projectCanonicalDynamicSynthesis(in.ChartState, in.Canonical, in.ChartState.StaticSynthesis)
	return validateDynamicSynthesisAfterGraphNormalization(in.ChartState)
}
