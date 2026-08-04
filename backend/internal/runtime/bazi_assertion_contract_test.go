// This test file belongs to the manager-owned runtime layer.
// It verifies BaZi assertion-contract validation and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// TestValidateMainAxisAssertionConsistency_AllowsEquivalentParaphrase leaves
// natural-language semantic comparison to the independent contract audit.
func TestValidateMainAxisAssertionConsistency_AllowsEquivalentParaphrase(t *testing.T) {
	static := baziStaticSynthesis{
		MainAxis: "第一条主轴",
		Assertions: []baziAssertion{{
			ID:      "static.main_axis",
			Kind:    baziAssertionMainAxis,
			Verdict: "另一条主轴",
		}},
	}

	if err := validateMainAxisAssertionConsistency(static); err != nil {
		t.Fatalf("structured main-axis projections should allow a semantic audit: %v", err)
	}
}

func TestValidateBaziAssertions_AuditsUnknownFactRefWithoutRejectingReading(t *testing.T) {
	state := assertionTestState()
	state.StaticSynthesis.Assertions = []baziAssertion{{
		ID:        "static.main_axis",
		Kind:      baziAssertionMainAxis,
		Subject:   "chart",
		Verdict:   "仅作结构观察。",
		FactRefs:  []baziFactRef{"chart.not_declared"},
		ClaimRefs: []baziClaimRef{"ziping_month_order_candidate"},
	}}
	if err := validateBaziAssertions(state, state.StaticSynthesis.Assertions); err != nil {
		t.Fatalf("unknown fact-ref aliases must not reject an otherwise valid assertion: %v", err)
	}
	warnings := strings.Join(collectBaziSoftAuditWarnings(state), "\n")
	if !strings.Contains(warnings, "chart.not_declared") {
		t.Fatalf("soft audit must retain the unknown ref for trace review, got %q", warnings)
	}
}

func TestValidateBaziAssertions_AuditsUnknownClaimRefWithoutRejectingReading(t *testing.T) {
	state := assertionTestState()
	state.StaticSynthesis.Assertions = []baziAssertion{{
		ID:        "static.main_axis",
		Kind:      baziAssertionMainAxis,
		Subject:   "chart",
		Verdict:   "仅作结构观察。",
		FactRefs:  []baziFactRef{"chart.month_branch"},
		ClaimRefs: []baziClaimRef{"unknown_claim"},
	}}
	if err := validateBaziAssertions(state, state.StaticSynthesis.Assertions); err != nil {
		t.Fatalf("unknown claim refs must not reject an otherwise valid assertion: %v", err)
	}
	warnings := strings.Join(collectBaziSoftAuditWarnings(state), "\n")
	if !strings.Contains(warnings, "unknown_claim") {
		t.Fatalf("soft audit must retain the unknown claim ref, got %q", warnings)
	}
}

func TestValidateBaziAssertions_AuditsClaimKindMismatchWithoutRejectingReading(t *testing.T) {
	state := assertionTestState()
	state.Input.RuleProfile.Claims = []baziProfileClaim{{ID: "test_pattern_candidate", Category: "pattern_candidate"}}
	state.StaticSynthesis.Assertions = []baziAssertion{{
		ID:        "static.strength",
		Kind:      baziAssertionStrength,
		Subject:   "day_master",
		Verdict:   "日主中和附近。",
		FactRefs:  []baziFactRef{"yongshen.strength"},
		ClaimRefs: []baziClaimRef{"test_pattern_candidate"},
	}}
	if err := validateBaziAssertions(state, state.StaticSynthesis.Assertions); err != nil {
		t.Fatalf("known claim kind mismatches must not discard a reading: %v", err)
	}
	warnings := strings.Join(collectBaziSoftAuditWarnings(state), "\n")
	if !strings.Contains(warnings, "static.strength") || !strings.Contains(warnings, "test_pattern_candidate") {
		t.Fatalf("soft audit must retain the claim-kind mismatch, got %q", warnings)
	}
}

func TestValidateBaziAssertions_AllowsStructuredFactSubrefs(t *testing.T) {
	state := assertionTestState()
	state.Input.Dayun = map[string]any{"dayun_analyzed": []map[string]any{{
		"ganZhi":        "甲午",
		"tenGod":        "七杀",
		"dayun_chonghe": []map[string]any{{"type": "自刑"}},
	}}}
	assertions := []baziAssertion{
		{
			ID:        "static.strength",
			Kind:      baziAssertionStrength,
			Subject:   "day_master",
			Verdict:   "日主中和附近。",
			FactRefs:  []baziFactRef{"yongshen.strength_evidence.support_score"},
			ClaimRefs: []baziClaimRef{"ziping_balance_evidence"},
		},
		{
			ID:        "dynamic.dayun.0",
			Kind:      baziAssertionDayunPeriod,
			Subject:   "dayun[0]",
			Verdict:   "甲午运：关系触发仅作结构观察。",
			FactRefs:  []baziFactRef{"dynamic_facts.dayun.dayun_analyzed[0].ganZhi", "dayun[0].relations[0]"},
			ClaimRefs: []baziClaimRef{"ziping_dynamic_four_dimension"},
		},
	}
	if err := validateBaziAssertions(state, assertions); err != nil {
		t.Fatalf("expected structured fact subrefs to pass, got %v", err)
	}
}

func TestValidateDynamicAssertions_RejectsMissingDayunCoverage(t *testing.T) {
	state := assertionTestState()
	state.Input.Dayun = map[string]any{"dayun_analyzed": []map[string]any{
		{"ganZhi": "甲午"},
		{"ganZhi": "乙未"},
	}}
	state.DynamicSynthesis = baziDynamicSynthesis{Assertions: []baziAssertion{{
		ID:        "dynamic.dayun.0",
		Kind:      baziAssertionDayunPeriod,
		Subject:   "dayun[0]",
		Verdict:   "甲午运：仅作结构观察。",
		FactRefs:  []baziFactRef{"dayun[0].gan_zhi"},
		ClaimRefs: []baziClaimRef{"ziping_dynamic_four_dimension"},
	}}}
	assertBaziViolationCode(t, validateDynamicAssertions(state), baziViolationDayunCoverageMissing)
}

func TestValidateDynamicAssertions_RejectsDayunGanZhiConflict(t *testing.T) {
	state := assertionTestState()
	state.Input.Dayun = map[string]any{"dayun_analyzed": []map[string]any{{"ganZhi": "甲午"}}}
	state.DynamicSynthesis = baziDynamicSynthesis{Assertions: []baziAssertion{{
		ID:        "dynamic.dayun.0",
		Kind:      baziAssertionDayunPeriod,
		Subject:   "dayun[0]",
		Verdict:   "乙未运：仅作结构观察。",
		FactRefs:  []baziFactRef{"dayun[0].gan_zhi"},
		ClaimRefs: []baziClaimRef{"ziping_dynamic_four_dimension"},
	}}}
	assertBaziViolationCode(t, validateDynamicAssertions(state), baziViolationFactConflict)
}

func TestValidateBaziAssertions_RejectsUnsupportedConcreteOutcome(t *testing.T) {
	state := assertionTestState()
	state.Input.Dayun = map[string]any{"dayun_analyzed": []map[string]any{{"ganZhi": "甲午"}}}
	err := validateBaziAssertions(state, []baziAssertion{{
		ID:        "dynamic.dayun.0",
		Kind:      baziAssertionDayunPeriod,
		Subject:   "dayun[0]",
		Verdict:   "本运可能有官非。",
		FactRefs:  []baziFactRef{"dayun[0].gan_zhi"},
		ClaimRefs: []baziClaimRef{"ziping_dynamic_four_dimension"},
	}})
	assertBaziViolationCode(t, err, baziViolationUnsupportedConcreteOutcome)
}

func TestValidatePatternAdjudicationDoesNotRejectMonthCandidateOnlyForOpacity(t *testing.T) {
	state := assertionTestState()
	state.EvidenceQuality = baziEvidenceQuality{CoveredTopics: []string{"geju"}}
	state.StaticSynthesis.PatternAdjudication = baziPatternAdjudication{
		MonthCommandCandidateID:  "month",
		SelectedAxisCandidateIDs: []string{"visible"},
		Candidates: []baziPatternCandidate{
			{ID: "month", Name: "月劫格", Origin: "month_command", Role: "rejected", RejectionReasons: []string{"independent_break"}, ComparisonDimensions: requiredPatternComparisonDimensions},
			{ID: "visible", Name: "伤官路线", Origin: "visible_stem", Role: "selected_axis", ComparisonDimensions: requiredPatternComparisonDimensions},
		},
	}
	if err := validatePatternAdjudication(state, state.StaticSynthesis.PatternAdjudication); err != nil {
		t.Fatalf("independent break with complete comparison should be admissible: %v", err)
	}
}

func TestValidatePatternAdjudicationRequiresComparisonForHiddenAxis(t *testing.T) {
	state := assertionTestState()
	state.EvidenceQuality = baziEvidenceQuality{CoveredTopics: []string{"geju"}}
	state.StaticSynthesis.PatternAdjudication = baziPatternAdjudication{
		MonthCommandCandidateID:  "month",
		SelectedAxisCandidateIDs: []string{"hidden"},
		Candidates: []baziPatternCandidate{
			{ID: "month", Name: "月劫格", Origin: "month_command", Role: "pattern_foundation"},
			{ID: "hidden", Name: "藏支组合", Origin: "hidden_combination", Role: "selected_axis"},
		},
	}
	assertBaziViolationCode(t, validatePatternAdjudication(state, state.StaticSynthesis.PatternAdjudication), baziViolationMethodContract)
}

func TestEnsureStaticAssertionsCanonicalizesEvidenceStatusFromQuality(t *testing.T) {
	state := assertionTestState()
	state.EvidenceQuality = baziEvidenceQuality{
		RequiredTopics: []string{"geju", "bingyao"},
		CoveredTopics:  []string{"geju"},
		MissingTopics:  []string{"bingyao"},
	}
	static := ensureStaticAssertions(state, baziStaticSynthesis{Assertions: []baziAssertion{{
		ID:             "static.tier",
		Kind:           baziAssertionTier,
		Subject:        "chart",
		Verdict:        "层次只能保守看。",
		EvidenceTopics: []string{"geju"},
		EvidenceStatus: baziEvidenceSupported,
	}}})

	assertion := static.Assertions[0]
	if assertion.EvidenceStatus != baziEvidenceWithheld {
		t.Fatalf("evidence status should be runtime-derived, got %q", assertion.EvidenceStatus)
	}
	if !containsString(assertion.EvidenceTopics, "bingyao") {
		t.Fatalf("required missing topic should still be bound for audit, got %v", assertion.EvidenceTopics)
	}
	if err := validateStaticAssertionEvidenceTopics(state, static.Assertions); err != nil {
		t.Fatalf("canonicalized evidence metadata should validate: %v", err)
	}
}

func TestRequirePatternComparisonDimensionsAcceptsObjectShape(t *testing.T) {
	dimensions := make(map[string]any, len(requiredPatternComparisonDimensions))
	for _, dimension := range requiredPatternComparisonDimensions {
		dimensions[dimension] = "已比较"
	}
	if err := requirePatternComparisonDimensions(baziPatternCandidate{ID: "candidate", ComparisonDimensions: dimensions}); err != nil {
		t.Fatalf("keyed comparison object should satisfy the same contract: %v", err)
	}
}

func TestValidateBaziContractAuditRejectsNonCompliantReview(t *testing.T) {
	assertBaziViolationCode(t, validateBaziContractAudit("dynamic", baziContractAudit{
		Compliant: false,
		Findings:  []baziContractAuditFinding{{Code: "age_scope", Field: "dayun_judgments[0]", Reason: "unauthorized domain"}},
	}), baziViolationSemanticContract)
}

func TestValidateBaziContractAuditPreservesFindingClassification(t *testing.T) {
	err := validateBaziContractAudit("dynamic", baziContractAudit{
		Compliant: false,
		Findings: []baziContractAuditFinding{{
			Code:           "outcome_domain_mismatch",
			Field:          "dynamic.dayun_judgments[0].interpretation",
			DetectedDomain: "finance",
			Reason:         "未授权财务领域",
		}},
	})
	assertBaziViolationCode(t, err, baziViolationSemanticContract)
	failure, ok := baziContractFailureFromError("dynamic_synthesis", err)
	if !ok {
		t.Fatalf("expected classified contract failure, got %v", err)
	}
	if failure.FindingCode != "outcome_domain_mismatch" || failure.Field != "dynamic.dayun_judgments[0].interpretation" {
		t.Fatalf("finding metadata not preserved: %+v", failure)
	}
	if failure.Class != baziContractFailureDomainUnauthorized || failure.RecoveryPolicy != baziRecoveryPolicyDynamicFactsOnly {
		t.Fatalf("unexpected failure classification: %+v", failure)
	}
}

func TestValidateStaticTierWithheldBoundaryRejectsHardTier(t *testing.T) {
	static := validStaticSynthesisForConsistencyTests()
	static.TierJudgment = "中上"
	static.TierBasis = "主轴可立，层次中上。"
	state := baziCharterState{
		EvidenceQuality: baziEvidenceQuality{
			RequiredTopics: []string{"geju", "bingyao"},
			CoveredTopics:  []string{"geju"},
			MissingTopics:  []string{"bingyao"},
		},
		StaticSynthesis: static,
	}

	err := validateStaticAssertions(state)
	assertBaziViolationCode(t, err, baziViolationEvidenceTopicMissing)
	failure, ok := baziContractFailureFromError("static_synthesis", err)
	if !ok || failure.Class != baziContractFailureEvidenceOverclaim {
		t.Fatalf("expected evidence overclaim classification, got %+v / %v", failure, err)
	}
}

func TestValidateStaticTierWithheldBoundaryAllowsBoundedConservativeTier(t *testing.T) {
	static := validStaticSynthesisForConsistencyTests()
	static.TierJudgment = "命格层次中等（保守定位）"
	static.TierBasis = "按保守定级标准：病药救应链条尚未完全闭合，层次封顶为中等，不上推中上或上等。"
	state := baziCharterState{
		EvidenceQuality: baziEvidenceQuality{
			RequiredTopics: []string{"geju", "bingyao"},
			CoveredTopics:  []string{"geju"},
			MissingTopics:  []string{"bingyao"},
		},
		StaticSynthesis: static,
	}

	if err := validateStaticAssertions(state); err != nil {
		t.Fatalf("bounded conservative tier should pass, got %v", err)
	}
}

func TestBaziContractAuditSummaryMarksMissingAuditAsNotRun(t *testing.T) {
	if got := baziContractAuditSummary(baziContractAudit{}); got != "not_run" {
		t.Fatalf("empty audit summary = %q, want not_run", got)
	}
}

func TestValidationRecoverySourceHasNoCaseSpecificPhrasePatches(t *testing.T) {
	raw, err := os.ReadFile("bazi_validation_recovery.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	for _, forbidden := range []string{
		"食神制杀",
		"丁火坐酉",
		"印星在酉月",
		"先丙后癸",
		"sanitizeUnmatchedQiongtongText",
		"dynamicHardBoundaryTerms",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("runtime recovery must not contain case-specific patch term %q", forbidden)
		}
	}
}

func assertionTestState() baziCharterState {
	yongshen := map[string]any{
		"geju_candidate": "月劫格",
		"geju_basis":     "月令本气为劫财，列月劫格候选。",
		"strength":       "中和附近",
		"strength_evidence": map[string]any{
			"support_score":  float64(8),
			"pressure_score": float64(8),
		},
	}
	return baziCharterState{Input: baziCharterInput{
		Yongshen: yongshen,
	}}
}

func assertBaziViolationCode(t *testing.T, err error, want baziViolationCode) {
	t.Helper()
	var validationErr baziValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected baziValidationError %s, got %v", want, err)
	}
	if validationErr.Violation.Code != want {
		t.Fatalf("violation code = %s, want %s", validationErr.Violation.Code, want)
	}
}
