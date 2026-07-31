package runtime

import (
	"errors"
	"os"
	"strings"
	"testing"
)

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
