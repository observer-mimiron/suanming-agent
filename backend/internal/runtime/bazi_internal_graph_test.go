// This test file belongs to the manager-owned runtime layer.
// It verifies BaZi internal graph behavior and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

func TestBaziInternalGraphBranches(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	staticOnly := &baziInternalGraphState{}
	if got, err := baziNeedDynamicEvidenceBranch(ctx, staticOnly); err != nil || got != baziInternalNodeCanonicalSynthesis {
		t.Fatalf("static-only evidence branch = %q, %v; want %q", got, err, baziInternalNodeCanonicalSynthesis)
	}

	needsDynamic := &baziInternalGraphState{
		ChartState: baziCharterState{AnalysisPlan: baziAnalysisPlan{NeedDynamic: true}},
	}
	if got, err := baziNeedDynamicEvidenceBranch(ctx, needsDynamic); err != nil || got != baziInternalNodeDynamicEvidence {
		t.Fatalf("dynamic evidence branch = %q, %v; want %q", got, err, baziInternalNodeDynamicEvidence)
	}
	if got, err := baziAfterStaticValidationBranch(ctx, needsDynamic); err != nil || got != baziInternalNodeDynamicValidation {
		t.Fatalf("after static branch = %q, %v; want %q", got, err, baziInternalNodeDynamicValidation)
	}

	staticFailed := &baziInternalGraphState{Failure: errors.New("projection failed")}
	if got, err := baziAfterStaticValidationBranch(ctx, staticFailed); err != nil || got != baziInternalNodeRepairDecision {
		t.Fatalf("after static failure branch = %q, %v; want %q", got, err, baziInternalNodeRepairDecision)
	}

	failed := &baziInternalGraphState{Failure: errors.New("projection failed")}
	if got, err := baziAfterDynamicValidationBranch(ctx, failed); err != nil || got != baziInternalNodeRecoveryDecision {
		t.Fatalf("after dynamic failure branch = %q, %v; want %q", got, err, baziInternalNodeRecoveryDecision)
	}

	canonicalClean := &baziInternalGraphState{}
	if got, err := baziCanonicalBranch(ctx, canonicalClean); err != nil || got != baziInternalNodeProjection {
		t.Fatalf("canonical clean branch = %q, %v; want %q", got, err, baziInternalNodeProjection)
	}

	repairable := &baziInternalGraphState{RepairAction: RepairActionRepairNode}
	if got, err := baziAfterRepairDecisionBranch(ctx, repairable); err != nil || got != baziInternalNodeCanonicalRepair {
		t.Fatalf("after repair decision branch = %q, %v; want %q", got, err, baziInternalNodeCanonicalRepair)
	}
	notRepairable := &baziInternalGraphState{RepairAction: RepairActionHardError}
	if got, err := baziAfterRepairDecisionBranch(ctx, notRepairable); err != nil || got != baziInternalNodeRecoveryDecision {
		t.Fatalf("after hard repair decision branch = %q, %v; want %q", got, err, baziInternalNodeRecoveryDecision)
	}
	repaired := &baziInternalGraphState{}
	if got, err := baziAfterCanonicalRepairBranch(ctx, repaired); err != nil || got != baziInternalNodeProjection {
		t.Fatalf("after canonical repair branch = %q, %v; want %q", got, err, baziInternalNodeProjection)
	}
	repairFailed := &baziInternalGraphState{Failure: errors.New("repair failed")}
	if got, err := baziAfterCanonicalRepairBranch(ctx, repairFailed); err != nil || got != baziInternalNodeRecoveryDecision {
		t.Fatalf("after failed canonical repair branch = %q, %v; want %q", got, err, baziInternalNodeRecoveryDecision)
	}
}

func TestBaziRepairDecisionStaticTiaohouProjectionRepairOnce(t *testing.T) {
	t.Parallel()

	state := baziCharterState{
		EvidenceQuality: baziEvidenceQuality{CoveredTopics: []string{"tiaohou"}},
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
	}
	state.StaticSynthesis.TiaohouAnchor = "本轮只确认季节环境与调候边界。"
	validationErr := validateStaticTiaohouEvidenceWording(state)
	if validationErr == nil {
		t.Fatal("tiaohou projection should fail before repair decision")
	}

	out, err := (&Executor{}).baziRepairDecisionNode(context.Background(), &baziInternalGraphState{
		FailureStage: "static_projection",
		Failure:      validationErr,
		RepairState:  NewRepairState(),
	})
	if err != nil {
		t.Fatalf("repair decision returned error: %v", err)
	}
	if out.RepairAction != RepairActionRepairNode {
		t.Fatalf("repair action = %q; want %q", out.RepairAction, RepairActionRepairNode)
	}
	if got := RepairAttemptsFor(out.RepairState, out.RepairFailure); got != 1 {
		t.Fatalf("repair attempts = %d; want 1", got)
	}
	if out.RepairFeedback["field"] != "static.tiaohou_anchor" {
		t.Fatalf("repair feedback field = %#v", out.RepairFeedback["field"])
	}
	if keys := RepairFeedbackKeys(out.RepairFeedback); !containsString(keys, "failed_stage") || !containsString(keys, "forbidden") {
		t.Fatalf("repair feedback keys = %#v", keys)
	}

	out, err = (&Executor{}).baziRepairDecisionNode(context.Background(), &baziInternalGraphState{
		FailureStage:  "static_projection",
		Failure:       validationErr,
		RepairState:   out.RepairState,
		RepairFailure: out.RepairFailure,
	})
	if err != nil {
		t.Fatalf("second repair decision returned error: %v", err)
	}
	if out.RepairAction != RepairActionFallback {
		t.Fatalf("second repair action = %q; want %q", out.RepairAction, RepairActionFallback)
	}
}

func TestBaziRecoveryDecisionStaticTiaohouBudgetExhaustedFactsOnly(t *testing.T) {
	t.Parallel()

	state := baziCharterState{
		EvidenceQuality: baziEvidenceQuality{CoveredTopics: []string{"tiaohou"}},
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		Input: baziCharterInput{
			BaziResult: map[string]any{"pillars": map[string]any{"year": "辛未"}},
		},
	}
	state.StaticSynthesis.TiaohouAnchor = "本轮只确认季节环境与调候边界。"
	validationErr := validateStaticTiaohouEvidenceWording(state)
	if validationErr == nil {
		t.Fatal("tiaohou projection should fail before repair decision")
	}

	first, err := (&Executor{}).baziRepairDecisionNode(context.Background(), &baziInternalGraphState{
		FailureStage: "static_projection",
		Failure:      validationErr,
		RepairState:  NewRepairState(),
	})
	if err != nil {
		t.Fatalf("first repair decision returned error: %v", err)
	}
	exhausted, err := (&Executor{}).baziRepairDecisionNode(context.Background(), &baziInternalGraphState{
		FailureStage: "static_projection",
		Failure:      validationErr,
		ChartState:   state,
		RepairState:  first.RepairState,
	})
	if err != nil {
		t.Fatalf("second repair decision returned error: %v", err)
	}
	if exhausted.RepairAction != RepairActionFallback {
		t.Fatalf("exhausted repair action = %q; want %q", exhausted.RepairAction, RepairActionFallback)
	}

	out, err := (&Executor{}).baziRecoveryDecisionNode(context.Background(), exhausted)
	if err != nil {
		t.Fatalf("budget exhausted tiaohou recovery returned error: %v", err)
	}
	if out.ChartState.StaticSynthesis.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("static source = %q; want %q", out.ChartState.StaticSynthesis.Source, baziSynthesisSourceFactsOnlyDegraded)
	}
}

func TestBaziCanonicalRepairFailureFallsBackToFactsOnlyForStaticTiaohou(t *testing.T) {
	t.Parallel()

	state := baziCharterState{
		EvidenceQuality: baziEvidenceQuality{CoveredTopics: []string{"tiaohou"}},
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		Input: baziCharterInput{
			BaziResult: map[string]any{"pillars": map[string]any{"year": "辛未"}},
		},
	}
	state.StaticSynthesis.TiaohouAnchor = "本轮只确认季节环境与调候边界。"
	validationErr := validateStaticTiaohouEvidenceWording(state)
	if validationErr == nil {
		t.Fatal("tiaohou projection should fail before repair decision")
	}
	repairFailure, ok := repairFailureFromBaziContract("static_projection", validationErr)
	if !ok {
		t.Fatal("tiaohou validation error should map to RepairFailure")
	}
	repairState := RecordRepairAttempt(NewRepairState(), RepairAttempt{
		Domain:  repairFailure.Domain,
		Stage:   repairFailure.Stage,
		Class:   repairFailure.Class,
		Field:   repairFailure.Field,
		Attempt: 1,
		Action:  RepairActionRepairNode,
	})

	afterRepair, err := (&Executor{}).baziCanonicalRepairNode(context.Background(), &baziInternalGraphState{
		Session:       nil,
		Question:      "看八字",
		ChartState:    state,
		FailureStage:  "static_projection",
		Failure:       validationErr,
		RepairState:   repairState,
		RepairFailure: repairFailure,
		RepairAction:  RepairActionRepairNode,
		RepairFeedback: buildBaziCanonicalRepairFeedback(
			repairFailure,
			1,
		),
	})
	if err != nil {
		t.Fatalf("canonical repair node returned error: %v", err)
	}
	if afterRepair.Failure == nil {
		t.Fatal("failed canonical repair should preserve original failure for recovery")
	}

	out, err := (&Executor{}).baziRecoveryDecisionNode(context.Background(), afterRepair)
	if err != nil {
		t.Fatalf("failed canonical repair recovery returned error: %v", err)
	}
	if out.ChartState.StaticSynthesis.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("static source = %q; want %q", out.ChartState.StaticSynthesis.Source, baziSynthesisSourceFactsOnlyDegraded)
	}
}

func TestBaziRecoveryDecisionStaticStrengthBalanceFactsOnlyAfterRepair(t *testing.T) {
	t.Parallel()

	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		Input: baziCharterInput{
			BaziResult: map[string]any{"pillars": map[string]any{"year": "辛未"}},
			Yongshen: map[string]any{
				"strength": "偏强",
				"strength_evidence": map[string]any{
					"support_score":  float64(12),
					"pressure_score": float64(7),
				},
			},
		},
	}
	state.StaticSynthesis.Strength.Conclusion = "日主偏弱，宜扶身。"
	state.StaticSynthesis.StrengthBalance = "日主偏弱，扶身为先。"
	validationErr := validateStaticStrengthAgainstEvidence(state)
	if validationErr == nil {
		t.Fatal("strength balance projection should fail before recovery")
	}
	if failure, ok := baziContractFailureFromError("static_projection", validationErr); !ok || failure.RecoveryPolicy != baziRecoveryPolicyStaticFactsOnly {
		t.Fatalf("strength balance failure = %+v, %v; want static facts-only", failure, ok)
	}
	tiaohouFailure := RepairFailure{
		Domain: "bazi",
		Stage:  "static_projection",
		Class:  RepairProjectionMismatch,
		Field:  "static.tiaohou_anchor",
	}
	repairState := RecordRepairAttempt(NewRepairState(), RepairAttempt{
		Domain:  tiaohouFailure.Domain,
		Stage:   tiaohouFailure.Stage,
		Class:   tiaohouFailure.Class,
		Field:   tiaohouFailure.Field,
		Attempt: 1,
		Action:  RepairActionRepairNode,
	})

	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()
	afterDecision, err := (&Executor{}).baziRepairDecisionNode(ctx, &baziInternalGraphState{
		FailureStage: "static_projection",
		Failure:      validationErr,
		ChartState:   state,
		RepairState:  repairState,
	})
	if err != nil {
		t.Fatalf("repair decision returned error: %v", err)
	}
	if afterDecision.RepairAction != RepairActionHardError {
		t.Fatalf("strength balance repair action = %q; want %q", afterDecision.RepairAction, RepairActionHardError)
	}
	if len(afterDecision.RepairState.Attempts) != 1 {
		t.Fatalf("strength balance should not add another model repair attempt: %#v", afterDecision.RepairState.Attempts)
	}

	out, err := (&Executor{}).baziRecoveryDecisionNode(ctx, afterDecision)
	if err != nil {
		t.Fatalf("strength balance recovery returned error: %v", err)
	}
	if out.ChartState.StaticSynthesis.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("static source = %q; want %q", out.ChartState.StaticSynthesis.Source, baziSynthesisSourceFactsOnlyDegraded)
	}
	if out.RecoveryState != baziRecoveryStateStaticFactsOnlyDegraded {
		t.Fatalf("recovery state = %q; want %q", out.RecoveryState, baziRecoveryStateStaticFactsOnlyDegraded)
	}
	if got := tracing.TraceFromContext(ctx).Attributes["repair.final_action"]; got != string(RepairActionFallback) {
		t.Fatalf("repair.final_action = %v; want %s", got, RepairActionFallback)
	}
}

func TestBaziRepairDecisionStaticFactConflictNoRepair(t *testing.T) {
	t.Parallel()

	out, err := (&Executor{}).baziRepairDecisionNode(context.Background(), &baziInternalGraphState{
		FailureStage: "static_projection",
		Failure: baziViolationError(
			baziViolationFactConflict,
			"static.pattern",
			"static.pattern",
			"static projection conflicts with chart facts",
			nil,
			nil,
		),
		RepairState: NewRepairState(),
	})
	if err != nil {
		t.Fatalf("repair decision returned error: %v", err)
	}
	if out.RepairAction != RepairActionHardError {
		t.Fatalf("repair action = %q; want %q", out.RepairAction, RepairActionHardError)
	}
	if len(out.RepairState.Attempts) != 0 {
		t.Fatalf("fact conflict should not consume repair attempts: %#v", out.RepairState.Attempts)
	}
}

func TestBaziRepairDecisionStaticMethodContractHardError(t *testing.T) {
	t.Parallel()

	out, err := (&Executor{}).baziRepairDecisionNode(context.Background(), &baziInternalGraphState{
		FailureStage: "static_projection",
		Failure: baziViolationError(
			baziViolationMethodContract,
			"static.pattern",
			"static.pattern",
			"static projection violates method contract",
			nil,
			nil,
		),
		RepairState: NewRepairState(),
	})
	if err != nil {
		t.Fatalf("repair decision returned error: %v", err)
	}
	if out.RepairAction != RepairActionHardError {
		t.Fatalf("repair action = %q; want %q", out.RepairAction, RepairActionHardError)
	}
	if len(out.RepairState.Attempts) != 0 {
		t.Fatalf("method contract should not consume repair attempts: %#v", out.RepairState.Attempts)
	}

	_, err = (&Executor{}).baziRecoveryDecisionNode(context.Background(), out)
	if err == nil {
		t.Fatal("method contract recovery returned nil error")
	}
	var runtimeFailure *RuntimeFailure
	if !errors.As(err, &runtimeFailure) {
		t.Fatalf("method contract error type = %T; want *RuntimeFailure", err)
	}
	if runtimeFailure.Code != "BAZI_STATIC_PROJECTION_FAILED" {
		t.Fatalf("runtime failure code = %q; want BAZI_STATIC_PROJECTION_FAILED", runtimeFailure.Code)
	}
}

func TestBaziCanonicalRepairSuccessClearsContractFailureTrace(t *testing.T) {
	t.Parallel()

	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()
	err := baziViolationError(
		baziViolationEvidenceTopicMissing,
		"static.tiaohou_anchor",
		"",
		"static tiaohou anchor lacks concrete verdict despite covered authority evidence",
		[]string{"tiaohou"},
		nil,
	)
	in := &baziInternalGraphState{
		FailureStage: "static_projection",
		Failure:      err,
		RepairState:  NewRepairState(),
	}
	baziRecordInternalFailure(ctx, in, "static_projection", err, "canonical_static_projection_facts_only")
	if got := tracing.TraceFromContext(ctx).Attributes["bazi.contract.failure_class"]; got == nil || got == "clean" {
		t.Fatalf("expected contract failure trace before repair success, got %v", got)
	}
	repairFailure, ok := repairFailureFromBaziContract("static_projection", err)
	if !ok {
		t.Fatal("expected repair failure mapping")
	}
	in.RepairFailure = repairFailure
	in.RepairState = RecordRepairAttempt(in.RepairState, RepairAttempt{
		Domain:  repairFailure.Domain,
		Stage:   repairFailure.Stage,
		Class:   repairFailure.Class,
		Field:   repairFailure.Field,
		Attempt: 1,
		Action:  RepairActionRepairNode,
	})

	baziAcceptCanonicalRepair(ctx, in, baziCanonicalSynthesis{})

	attrs := tracing.TraceFromContext(ctx).Attributes
	if got := attrs["bazi.contract.failure_class"]; got != "clean" {
		t.Fatalf("failure_class = %v; want clean", got)
	}
	if got := attrs["bazi.contract.recovery_policy"]; got != "" {
		t.Fatalf("recovery_policy = %v; want empty", got)
	}
	if got := attrs["bazi.contract.finding_code"]; got != "" {
		t.Fatalf("finding_code = %v; want empty", got)
	}
	if got := attrs["bazi.contract.finding_field"]; got != "" {
		t.Fatalf("finding_field = %v; want empty", got)
	}
	if got := attrs["repair.final_action"]; got != string(RepairActionAccept) {
		t.Fatalf("repair.final_action = %v; want accept", got)
	}
}

func TestBaziRepairTraceLearningHintCountMatchesFeedback(t *testing.T) {
	t.Parallel()

	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()
	validationErr := baziContractAuditError("static_projection", baziContractAuditFinding{
		Code:   "static_projection_mismatch",
		Field:  "static.tiaohou_anchor",
		Reason: "调候锚点缺少明确裁断",
	})

	out, err := (&Executor{}).baziRepairDecisionNode(ctx, &baziInternalGraphState{
		FailureStage: "static_projection",
		Failure:      validationErr,
		RepairState:  NewRepairState(),
	})
	if err != nil {
		t.Fatalf("repair decision returned error: %v", err)
	}
	want := RepairLearningHintCount(out.RepairFeedback)
	if want == 0 {
		t.Fatalf("repair feedback should include learning hints: %#v", out.RepairFeedback)
	}
	attrs := tracing.TraceFromContext(ctx).Attributes
	if got := attrs["repair.learning_hint_count"]; got != want {
		t.Fatalf("decision repair.learning_hint_count = %v; want %d", got, want)
	}

	baziAcceptCanonicalRepair(ctx, out, baziCanonicalSynthesis{})
	if got := attrs["repair.learning_hint_count"]; got != want {
		t.Fatalf("final repair.learning_hint_count = %v; want %d", got, want)
	}
}

func TestBaziRecoveryDecisionDynamicDomainOverreachFactsOnly(t *testing.T) {
	t.Parallel()

	in := &baziInternalGraphState{
		FailureStage: "dynamic_projection",
		Failure: baziViolationError(
			baziViolationUnsupportedConcreteOutcome,
			"dynamic.dayun[0].interpretation",
			"",
			"dynamic projection entered unauthorized outcome domain",
			nil,
			nil,
		),
		ChartState: baziCharterState{
			StaticSynthesis: baziStaticSynthesis{
				Source:   "model",
				MainAxis: "静态裁决已通过。",
			},
		},
	}

	out, err := (&Executor{}).baziRecoveryDecisionNode(context.Background(), in)
	if err != nil {
		t.Fatalf("dynamic domain overreach recovery returned error: %v", err)
	}
	if out.Failure != nil || out.FailureStage != "" {
		t.Fatalf("recovery did not clear failure state: %#v", out)
	}
	if out.ChartState.DynamicSynthesis.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("dynamic source = %q; want %q", out.ChartState.DynamicSynthesis.Source, baziSynthesisSourceFactsOnlyDegraded)
	}
	if !containsString(out.ChartState.DynamicSynthesis.FieldAudit, "contract_failure_class:"+baziContractFailureDomainUnauthorized) {
		t.Fatalf("dynamic field audit missing domain failure: %#v", out.ChartState.DynamicSynthesis.FieldAudit)
	}
}

func TestBaziRecoveryDecisionDynamicFactConflictHardError(t *testing.T) {
	t.Parallel()

	in := &baziInternalGraphState{
		FailureStage: "dynamic_projection",
		Failure: baziViolationError(
			baziViolationFactConflict,
			"dynamic.dayun[0].relation",
			"",
			"dynamic projection conflicts with deterministic relation facts",
			nil,
			nil,
		),
	}

	_, err := (&Executor{}).baziRecoveryDecisionNode(context.Background(), in)
	if err == nil {
		t.Fatal("dynamic fact conflict recovery returned nil error")
	}
	var runtimeFailure *RuntimeFailure
	if !errors.As(err, &runtimeFailure) {
		t.Fatalf("dynamic fact conflict error type = %T; want *RuntimeFailure", err)
	}
	if runtimeFailure.Code != "BAZI_DYNAMIC_PROJECTION_FAILED" {
		t.Fatalf("runtime failure code = %q; want BAZI_DYNAMIC_PROJECTION_FAILED", runtimeFailure.Code)
	}
}

func TestBaziDynamicValidatorFailureClassificationAndPolicy(t *testing.T) {
	t.Parallel()

	validationErr := validateDynamicConsistencyFlags([]string{"unsupported"})
	failure, ok := baziContractFailureFromError("dynamic_projection", validationErr)
	if !ok || failure.Class != baziContractFailureProjectionMismatch || failure.Field != "dynamic.consistency_flags" {
		t.Fatalf("dynamic validator failure = %+v, classified=%v; want projection mismatch on consistency flags", failure, ok)
	}
	repairFailure, ok := repairFailureFromBaziContract("dynamic_projection", validationErr)
	if !ok || repairFailure.Class != RepairProjectionMismatch || repairFailure.Field != "dynamic.consistency_flags" {
		t.Fatalf("global repair failure = %+v, classified=%v; want projection mismatch", repairFailure, ok)
	}

	degraded, err := (&Executor{}).baziRecoveryDecisionNode(context.Background(), &baziInternalGraphState{
		FailureStage: "dynamic_projection",
		Failure: baziViolationError(
			baziViolationUnsupportedConcreteOutcome,
			"dynamic.dayun[0].interpretation",
			"",
			"dynamic projection entered unauthorized outcome domain",
			nil,
			nil,
		),
		ChartState: baziCharterState{StaticSynthesis: baziStaticSynthesis{MainAxis: "静态主轴"}},
	})
	if err != nil || degraded.ChartState.DynamicSynthesis.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("dynamic facts-only fallback = %+v, err=%v", degraded, err)
	}

	for _, code := range []baziViolationCode{baziViolationFactConflict, baziViolationMethodContract} {
		failure, ok := repairFailureFromBaziContract("dynamic_projection", baziViolationError(code, "dynamic.contract", "", "must remain hard", nil, nil))
		if !ok {
			t.Fatalf("%s was not classified", code)
		}
		decision := DefaultRepairPolicy().Decide(failure, NewRepairState())
		if decision.Action != RepairActionHardError || decision.Repairable || decision.Retryable {
			t.Fatalf("%s policy = %+v; want non-repairable hard error", code, decision)
		}
	}
}

func TestBaziRecoveryDecisionStaticEvidenceOverclaimFactsOnly(t *testing.T) {
	t.Parallel()

	in := &baziInternalGraphState{
		FailureStage: "static_projection",
		RecoveryCode: "canonical_static_projection_facts_only",
		Failure: baziViolationError(
			baziViolationEvidenceTopicMissing,
			"static.tier",
			"static.tier",
			"tier verdict overclaims missing evidence topics",
			[]string{"bingyao"},
			nil,
		),
		ChartState: baziCharterState{
			Input: baziCharterInput{
				BaziResult: map[string]any{"pillars": map[string]any{"year": "甲戌"}},
			},
		},
	}

	out, err := (&Executor{}).baziRecoveryDecisionNode(context.Background(), in)
	if err != nil {
		t.Fatalf("static evidence overclaim recovery returned error: %v", err)
	}
	if out.ChartState.StaticSynthesis.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("static source = %q; want %q", out.ChartState.StaticSynthesis.Source, baziSynthesisSourceFactsOnlyDegraded)
	}
	if out.RecoveryState != baziRecoveryStateStaticFactsOnlyDegraded {
		t.Fatalf("recovery state = %q; want %q", out.RecoveryState, baziRecoveryStateStaticFactsOnlyDegraded)
	}
}

func TestBaziRecoveryDecisionStaticEvidenceBoundaryValidatorFactsOnly(t *testing.T) {
	t.Parallel()

	static := validStaticSynthesisForConsistencyTests()
	static.AxisLevel = "可以拔高"
	state := baziCharterState{
		EvidenceQuality: baziEvidenceQuality{MissingTopics: []string{"bingyao"}},
		StaticSynthesis: static,
		Input: baziCharterInput{
			BaziResult: map[string]any{"pillars": map[string]any{"year": "辛未"}},
		},
	}
	err := validateStaticEvidenceCoverageBoundary(state)
	if err == nil {
		t.Fatal("static evidence boundary should fail before recovery")
	}

	out, err := (&Executor{}).baziRecoveryDecisionNode(context.Background(), &baziInternalGraphState{
		FailureStage: "static_projection",
		RecoveryCode: "canonical_static_projection_facts_only",
		Failure:      err,
		ChartState:   state,
	})
	if err != nil {
		t.Fatalf("static evidence boundary recovery returned error: %v", err)
	}
	if out.ChartState.StaticSynthesis.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("static source = %q; want %q", out.ChartState.StaticSynthesis.Source, baziSynthesisSourceFactsOnlyDegraded)
	}
}

func TestBaziRecoveryDecisionStaticTiaohouProjectionFactsOnly(t *testing.T) {
	t.Parallel()

	state := baziCharterState{
		EvidenceQuality: baziEvidenceQuality{CoveredTopics: []string{"tiaohou"}},
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		Input: baziCharterInput{
			BaziResult: map[string]any{"pillars": map[string]any{"year": "辛未"}},
		},
	}
	state.StaticSynthesis.TiaohouAnchor = "秋月戊土，调候需丙火照暖、甲木疏劈、癸水滋润；原局火土偏旺，金水偏弱，调候上喜水润局，但水星不透，调候力度受限。"
	err := validateStaticTiaohouEvidenceWording(state)
	if err == nil {
		t.Fatal("tiaohou projection should fail before recovery")
	}

	out, err := (&Executor{}).baziRecoveryDecisionNode(context.Background(), &baziInternalGraphState{
		FailureStage: "static_projection",
		RecoveryCode: "canonical_static_projection_facts_only",
		Failure:      err,
		ChartState:   state,
	})
	if err != nil {
		t.Fatalf("static tiaohou projection recovery returned error: %v", err)
	}
	if out.ChartState.StaticSynthesis.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("static source = %q; want %q", out.ChartState.StaticSynthesis.Source, baziSynthesisSourceFactsOnlyDegraded)
	}
	if out.Failure != nil || out.FailureStage != "" {
		t.Fatalf("recovery did not clear failure: %#v", out)
	}
}

func TestBaziRecoveryDecisionStaticFactConflictHardError(t *testing.T) {
	t.Parallel()

	in := &baziInternalGraphState{
		FailureStage: "static_projection",
		Failure: baziViolationError(
			baziViolationFactConflict,
			"static.pattern",
			"static.pattern",
			"static projection conflicts with chart facts",
			nil,
			nil,
		),
	}

	_, err := (&Executor{}).baziRecoveryDecisionNode(context.Background(), in)
	if err == nil {
		t.Fatal("static fact conflict recovery returned nil error")
	}
	var runtimeFailure *RuntimeFailure
	if !errors.As(err, &runtimeFailure) {
		t.Fatalf("static fact conflict error type = %T; want *RuntimeFailure", err)
	}
	if runtimeFailure.Code != "BAZI_STATIC_PROJECTION_FAILED" {
		t.Fatalf("runtime failure code = %q; want BAZI_STATIC_PROJECTION_FAILED", runtimeFailure.Code)
	}
}
