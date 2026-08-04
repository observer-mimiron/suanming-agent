// This test file belongs to the manager-owned runtime layer.
// It verifies BaZi internal graph behavior and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"errors"
	"testing"
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

	failed := &baziInternalGraphState{Failure: errors.New("projection failed")}
	if got, err := baziAfterDynamicValidationBranch(ctx, failed); err != nil || got != baziInternalNodeRecoveryDecision {
		t.Fatalf("after dynamic failure branch = %q, %v; want %q", got, err, baziInternalNodeRecoveryDecision)
	}

	canonicalClean := &baziInternalGraphState{}
	if got, err := baziCanonicalBranch(ctx, canonicalClean); err != nil || got != baziInternalNodeProjection {
		t.Fatalf("canonical clean branch = %q, %v; want %q", got, err, baziInternalNodeProjection)
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
