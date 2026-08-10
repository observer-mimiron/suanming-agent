// This test file belongs to the manager-owned runtime layer.
// It verifies BaZi assertion-contract validation and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"errors"
	"fmt"
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

func TestValidateBaziAssertions_RejectsUnknownFactRef(t *testing.T) {
	state := assertionTestState()
	state.StaticSynthesis.Assertions = []baziAssertion{{
		ID:        "static.main_axis",
		Kind:      baziAssertionMainAxis,
		Subject:   "chart",
		Verdict:   "仅作结构观察。",
		FactRefs:  []baziFactRef{"chart.not_declared"},
		ClaimRefs: []baziClaimRef{"ziping_month_order_candidate"},
	}}
	assertBaziViolationCode(t, validateBaziAssertions(state, state.StaticSynthesis.Assertions), baziViolationUndeclaredFactClaim)
}

func TestValidateBaziAssertions_RejectsUnknownClaimRef(t *testing.T) {
	state := assertionTestState()
	state.StaticSynthesis.Assertions = []baziAssertion{{
		ID:        "static.main_axis",
		Kind:      baziAssertionMainAxis,
		Subject:   "chart",
		Verdict:   "仅作结构观察。",
		FactRefs:  []baziFactRef{"chart.month_branch"},
		ClaimRefs: []baziClaimRef{"unknown_claim"},
	}}
	assertBaziViolationCode(t, validateBaziAssertions(state, state.StaticSynthesis.Assertions), baziViolationUndeclaredFactClaim)
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

func TestValidateBaziAssertions_RejectsUnregisteredFactAliases(t *testing.T) {
	state := assertionTestState()
	state.Input.RuleProfile.Claims = []baziProfileClaim{{ID: "ziping_balance_evidence"}, {ID: "ziping_dynamic_four_dimension"}}
	state.Input.Dayun = map[string]any{"dayun_analyzed": []map[string]any{{
		"ganZhi":        "甲午",
		"tenGod":        "七杀",
		"dayun_chonghe": []map[string]any{{"type": "自刑"}},
	}}}
	for _, ref := range []baziFactRef{
		"yongshen.strength_evidence.support_score",
		"dynamic_facts.dayun.dayun_analyzed[0].ganZhi",
		"dayun[0].relations[0]",
	} {
		assertions := []baziAssertion{{
			ID:        "static.strength",
			Kind:      baziAssertionStrength,
			Subject:   "day_master",
			Verdict:   "仅作结构观察。",
			FactRefs:  []baziFactRef{ref},
			ClaimRefs: []baziClaimRef{"ziping_balance_evidence"},
		}}
		if err := validateBaziAssertions(state, assertions); err == nil {
			t.Fatalf("unregistered fact alias %q must be rejected", ref)
		} else {
			assertBaziViolationCode(t, err, baziViolationUndeclaredFactClaim)
		}
	}
}

func TestValidateDynamicAssertions_AllowsOnlyCurrentDayunCoverage(t *testing.T) {
	state := assertionTestState()
	state.Input.RuleProfile.Claims = []baziProfileClaim{{ID: "ziping_dynamic_four_dimension"}}
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
	if err := validateDynamicAssertions(state); err != nil {
		t.Fatalf("dynamic assertion may cover only the current period: %v", err)
	}
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

func TestRegression1991Profile_Binds2026LiunianToJiawuDayun(t *testing.T) {
	state := assertionTestState()
	state.Input.BaziResult = map[string]any{"birthday": "1991-10-05 12:40:00"}
	state.Input.Dayun = map[string]any{"dayun_analyzed": []map[string]any{
		{"ganZhi": "癸巳"}, {"ganZhi": "甲午"},
	}}
	state.Input.Liunian = map[string]any{"liunian_year": float64(2026), "current_dayun": map[string]any{"ganZhi": "甲午"}}
	assertions := []baziAssertion{
		{ID: "dynamic.dayun.0", Kind: baziAssertionDayunPeriod, Subject: "dayun[0]", Verdict: "癸巳运只作结构观察", FactRefs: []baziFactRef{"dayun[0].gan_zhi"}},
		{ID: "dynamic.dayun.1", Kind: baziAssertionDayunPeriod, Subject: "dayun[1]", Verdict: "甲午运只作结构观察", FactRefs: []baziFactRef{"dayun[1].gan_zhi"}},
		{ID: "dynamic.liunian", Kind: baziAssertionLiunian, Verdict: "2026年只作结构观察", FactRefs: []baziFactRef{"liunian.gan_zhi", "dayun[0].gan_zhi"}},
	}
	assertBaziViolationCode(t, validateLiunianAssertionsAgainstCurrentDayun(state, assertions), baziViolationFactRefMissing)
	assertions[2].FactRefs[1] = "dayun[1].gan_zhi"
	if err := validateLiunianAssertionsAgainstCurrentDayun(state, assertions); err != nil {
		t.Fatalf("2026 liunian must bind 甲午运: %v", err)
	}
}

func TestRegression1991DynamicJudgment_RequiresJiawuCurrentPeriod(t *testing.T) {
	state := assertionTestState()
	state.Input.BaziResult = map[string]any{"birthday": "1991-10-05 12:40:00"}
	state.Input.Dayun = map[string]any{"dayun_analyzed": []map[string]any{{"ganZhi": "癸巳"}, {"ganZhi": "甲午", "dayun_chonghe": []map[string]any{{"description": "流年关系"}}}}}
	state.Input.Liunian = map[string]any{"liunian_year": float64(2026), "current_dayun": map[string]any{"ganZhi": "甲午"}}
	judgment := baziStructuredDynamicSynthesis{CurrentPeriodRef: "dayun[0]", CurrentPeriodRealization: "maintain", PeriodClaims: []baziStructuredPeriodClaim{{PeriodRef: "dayun[0]"}}}
	assertBaziViolationCode(t, validateBaziDynamicJudgmentPolicy(state, judgment), baziViolationFactConflict)
	judgment.CurrentPeriodRef, judgment.PeriodClaims[0].PeriodRef = "dayun[1]", "dayun[1]"
	if err := validateBaziDynamicJudgmentPolicy(state, judgment); err != nil {
		t.Fatalf("2026 dynamic judgment must bind 甲午运: %v", err)
	}
}

func TestEnsureDynamicAssertionsResolvesSparseJudgmentByGanZhi(t *testing.T) {
	state := assertionTestState()
	state.Input.Dayun = map[string]any{"dayun_analyzed": []map[string]any{
		{"ganZhi": "丙申"}, {"ganZhi": "乙未"}, {"ganZhi": "甲午"},
	}}
	dynamic := ensureDynamicAssertions(state, baziDynamicSynthesis{
		DayunPath: []string{"丙申运（事实目录）", "乙未运（事实目录）", "甲午运（事实目录）"},
		DayunJudgments: []baziDayunJudgment{{
			GanZhi: "甲午", Trend: "甲午运承接主轴", Interpretation: "仅作结构观察",
		}},
	})
	if len(dynamic.Assertions) != 1 {
		t.Fatalf("dynamic assertions = %#v, want only the model judgment", dynamic.Assertions)
	}
	assertion := dynamic.Assertions[0]
	if assertion.Subject != "dayun[2]" || !containsString(factRefsToStrings(assertion.FactRefs), "dayun[2].gan_zhi") {
		t.Fatalf("sparse judgment must bind 甲午's catalog index: %#v", assertion)
	}
	if err := validateDayunAssertionsAgainstFacts(state, dynamic.Assertions); err != nil {
		t.Fatalf("甲午 judgment must validate against 甲午 period: %v", err)
	}
}

func TestStaticJudgmentPolicy_WithholdsTierWithoutIndependentGrounds(t *testing.T) {
	state := assertionTestState()
	state.EvidenceQuality.CoveredTopics = []string{"geju"}
	state.Input.Yongshen["official_visibility"] = map[string]any{"visible": []map[string]any{{"stem": "辛"}}}
	judgment := baziStructuredStaticSynthesis{AxisStatus: "candidate", TierAssessment: tierAssessmentForTest("rated", 5), NatalRiskStatus: "none"}
	assertBaziViolationCode(t, validateBaziStaticJudgmentPolicy(state, judgment), baziViolationEvidenceTopicMissing)
	judgment.TierAssessment = tierAssessmentForTest("provisional", 5)
	if err := validateBaziStaticJudgmentPolicy(state, judgment); err != nil {
		t.Fatalf("incomplete tier evidence must retain a provisional grade: %v", err)
	}
}

func TestBaziTierAssessmentAcceptsEveryLevelWithinTypedBand(t *testing.T) {
	state := assertionTestState()
	state.EvidenceQuality.CoveredTopics = append([]string{}, baziTierEvidenceTopics...)
	facts := buildBaziFactCapsule(state)
	for level := 1; level <= 9; level++ {
		t.Run(fmt.Sprintf("level_%d", level), func(t *testing.T) {
			if err := validateBaziTierAssessment(facts, "established", ratedTierAssessmentForLevel(level)); err != nil {
				t.Fatalf("level %d should fit its typed evidence band: %v", level, err)
			}
		})
	}
}

func TestCurrentPeriodRealizationDoesNotRewriteStaticTier(t *testing.T) {
	state := baziCharterState{
		AnalysisPlan: baziAnalysisPlan{NeedDynamic: true},
		Input: baziCharterInput{
			Dayun: map[string]any{"dayun_analyzed": []map[string]any{{"ganZhi": "甲午"}}},
			Liunian: map[string]any{
				"current_dayun": map[string]any{"ganZhi": "甲午"},
			},
		},
	}
	static := baziStaticSynthesis{TierAssessment: tierAssessmentForTest("rated", 5), TierJudgment: "命格基础层次：第5级（中格，有路但利弊并见）"}
	dynamic := projectCanonicalDynamicSynthesis(state, baziCanonicalSynthesis{
		DayunOverview:            baziCanonicalUnit{Verdict: "甲午运承接主轴", Confidence: "倾向成立"},
		Liunian:                  baziCanonicalUnit{Verdict: "流年按当前大运观察", Confidence: "倾向成立"},
		CurrentPeriodRealization: "suppress",
	}, static)
	if static.TierAssessment.Status != "rated" || static.TierAssessment.Level != 5 {
		t.Fatalf("dynamic projection rewrote natal tier: %+v", static.TierAssessment)
	}
	if dynamic.CurrentPeriodRealization != "suppress" {
		t.Fatalf("dynamic realization = %q, want suppress", dynamic.CurrentPeriodRealization)
	}
}

func TestStaticJudgmentPolicy_WithholdsNatalRiskWithoutVisibleOfficial(t *testing.T) {
	state := assertionTestState()
	state.Input.Yongshen["official_visibility"] = map[string]any{"visible": []map[string]any{}, "hidden": []map[string]any{{"stem": "癸"}}}
	judgment := baziStructuredStaticSynthesis{AxisStatus: "candidate", TierAssessment: tierAssessmentForTest("provisional", 5), NatalRiskStatus: "none"}
	assertBaziViolationCode(t, validateBaziStaticJudgmentPolicy(state, judgment), baziViolationFactConflict)
	judgment.NatalRiskStatus = "withheld"
	if err := validateBaziStaticJudgmentPolicy(state, judgment); err != nil {
		t.Fatalf("hidden official must force withheld static risk: %v", err)
	}
}

func TestNormalizeBaziStaticJudgmentProjectsNatalRiskFromFacts(t *testing.T) {
	state := assertionTestState()
	state.Input.Yongshen["official_visibility"] = map[string]any{"visible": []map[string]any{}, "hidden": []map[string]any{{"stem": "癸"}}}
	judgment := normalizeBaziStaticJudgment(state, baziStructuredStaticSynthesis{NatalRiskStatus: "none"})
	if judgment.NatalRiskStatus != "withheld" {
		t.Fatalf("natal risk status = %q; want withheld", judgment.NatalRiskStatus)
	}
}

func TestStaticJudgmentPolicyRejectsFreeTextStatus(t *testing.T) {
	state := assertionTestState()
	state.EvidenceQuality.CoveredTopics = append([]string{}, baziTierEvidenceTopics...)
	state.Input.Yongshen["official_visibility"] = map[string]any{"visible": []map[string]any{}, "hidden": []map[string]any{{"stem": "乙"}}}
	judgment := baziStructuredStaticSynthesis{
		AxisStatus:      "candidate",
		TierAssessment:  tierAssessmentForTest("provisional", 5),
		NatalRiskStatus: "withheld",
		Claims: []baziStructuredStaticClaim{
			{Verdict: "主轴候选仍需比较", Status: "candidate"},
			{Verdict: "日主偏强", Status: "candidate"},
			{Verdict: "调候条件受限", Status: "limited"},
			{Verdict: "格局取用尚待比较", Status: "candidate"},
		},
	}
	if err := validateBaziStaticJudgmentPolicy(state, judgment); err != nil {
		t.Fatalf("baseline typed static judgment should pass: %v", err)
	}

	judgment.Claims[3].Status = "层次中等偏上"
	assertBaziViolationCode(t, validateBaziStaticJudgmentPolicy(state, judgment), baziViolationMethodContract)
	judgment.Claims[3].Status = "candidate"

	judgment.Claims[2].Status = "fire_effective=false"
	assertBaziViolationCode(t, validateBaziStaticJudgmentPolicy(state, judgment), baziViolationMethodContract)
	judgment.Claims[2].Status = "limited"

	judgment.NatalRiskStatus = "none"
	assertBaziViolationCode(t, validateBaziStaticJudgmentPolicy(state, judgment), baziViolationFactConflict)
}

func TestStaticJudgmentPolicyRejectsMainAxisPatternConflict(t *testing.T) {
	state := assertionTestState()
	state.Input.Yongshen["geju_candidate"] = "伤官格(官未透)"
	judgment := baziStructuredStaticSynthesis{
		AxisStatus:      "established",
		NatalRiskStatus: "withheld",
		Claims: []baziStructuredStaticClaim{{
			Verdict: "建禄月劫之格，取伤官泄秀为轴",
			Status:  "established",
		}},
		TierAssessment: tierAssessmentForTest("provisional", 5),
	}
	assertBaziViolationCode(t, validateBaziStaticJudgmentPolicy(state, judgment), baziViolationFactConflict)
}

func TestValidateStaticMainAxisPatternAllowsUsageRouteWithinToolPattern(t *testing.T) {
	state := assertionTestState()
	state.Input.Yongshen["geju_candidate"] = "伤官格(官未透)"
	judgment := baziStructuredStaticSynthesis{Claims: []baziStructuredStaticClaim{{
		Verdict: "伤官格取伤官佩印为用",
	}}}
	if err := validateStaticMainAxisPattern(state, judgment); err != nil {
		t.Fatalf("usage route within the deterministic pattern must pass: %v", err)
	}
}

func TestValidateBaziAssertionsRejectsCatalogDerivedInternalFieldName(t *testing.T) {
	state := assertionTestState()
	err := validateBaziAssertions(state, []baziAssertion{{
		ID:       "static.tiaohou",
		Kind:     baziAssertionTiaohou,
		Subject:  "chart",
		Verdict:  "fire_effective=false",
		FactRefs: []baziFactRef{"fact_capsule.fire_effective"},
	}})
	assertBaziViolationCode(t, err, baziViolationMethodContract)
}

func TestValidateBaziAssertions_RejectsUnsupportedConcreteOutcome(t *testing.T) {
	state := assertionTestState()
	state.Input.RuleProfile.Claims = []baziProfileClaim{{ID: "ziping_dynamic_four_dimension"}}
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

func TestEnsureStaticAssertionsKeepsLegacyTierEvidenceSeparateFromTypedAssessment(t *testing.T) {
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
	if assertion.EvidenceStatus != baziEvidenceSupported {
		t.Fatalf("legacy tier assertion should not override typed evidence state, got %q", assertion.EvidenceStatus)
	}
	if containsString(assertion.EvidenceTopics, "bingyao") {
		t.Fatalf("typed tier dimensions, not legacy assertions, own independent tier topics: %v", assertion.EvidenceTopics)
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

func TestValidateStaticTierWithheldBoundaryAllowsDeferredTier(t *testing.T) {
	static := validStaticSynthesisForConsistencyTests()
	static.TierJudgment = "命格层次暂不定级（仅作结构观察）"
	static.TierBasis = "病药救应链条尚未完全闭合，因此不作高低定级。"
	state := baziCharterState{
		EvidenceQuality: baziEvidenceQuality{
			RequiredTopics: []string{"geju", "bingyao"},
			CoveredTopics:  []string{"geju"},
			MissingTopics:  []string{"bingyao"},
		},
		StaticSynthesis: static,
	}

	if err := validateStaticAssertions(state); err != nil {
		t.Fatalf("deferred tier should pass, got %v", err)
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
		BaziResult: map[string]any{
			"dayGan": "戊",
			"pillars": []map[string]any{
				{"name": "年柱", "stem": "辛", "branch": "未", "hideGan": []string{"己", "丁", "乙"}},
				{"name": "月柱", "stem": "丁", "branch": "酉", "hideGan": []string{"辛"}},
				{"name": "日柱", "stem": "戊", "branch": "申", "hideGan": []string{"庚", "壬", "戊"}},
				{"name": "时柱", "stem": "戊", "branch": "午", "hideGan": []string{"丁", "己"}},
			},
		},
		Yongshen: yongshen,
	}}
}

func tierAssessmentForTest(status string, level int) baziTierAssessment {
	dimension := func(state string, topics ...string) baziTierDimension {
		return baziTierDimension{
			State:          state,
			FactRefs:       []baziFactRef{"chart.month_branch"},
			EvidenceTopics: append([]string{}, topics...),
		}
	}
	return baziTierAssessment{
		Status:     status,
		Level:      level,
		Confidence: "保守判断",
		Dimensions: baziTierDimensions{
			MainAxis:   dimension("mixed", "geju"),
			YouQing:    dimension("mixed", "geju"),
			YouLi:      dimension("mixed", "geju"),
			QingZhuo:   dimension("mixed", "qingzhuo"),
			Disease:    dimension("moderate", "bingyao"),
			Remedy:     dimension("mixed", "bingyao"),
			Rescue:     dimension("mixed", "jiuying"),
			Tiaohou:    dimension("mixed", "tiaohou"),
			HeZhiZhang: dimension("mixed", "hezhizhang"),
		},
	}
}

func ratedTierAssessmentForLevel(level int) baziTierAssessment {
	assessment := tierAssessmentForTest("rated", level)
	setGeneralState := func(state string) {
		for _, dimension := range []*baziTierDimension{
			&assessment.Dimensions.MainAxis,
			&assessment.Dimensions.YouQing,
			&assessment.Dimensions.YouLi,
			&assessment.Dimensions.QingZhuo,
			&assessment.Dimensions.Remedy,
			&assessment.Dimensions.Rescue,
			&assessment.Dimensions.Tiaohou,
			&assessment.Dimensions.HeZhiZhang,
		} {
			dimension.State = state
		}
	}
	switch level {
	case 1, 2:
		setGeneralState("missing")
		assessment.Dimensions.Disease.State = "critical"
	case 3:
		setGeneralState("limited")
		assessment.Dimensions.Disease.State = "critical"
	case 4:
		setGeneralState("missing")
		assessment.Dimensions.MainAxis.State = "limited"
		assessment.Dimensions.Disease.State = "heavy"
	case 5:
		setGeneralState("mixed")
		assessment.Dimensions.MainAxis.State = "limited"
		assessment.Dimensions.Disease.State = "moderate"
	case 6:
		setGeneralState("mixed")
		assessment.Dimensions.Disease.State = "moderate"
	case 7, 8:
		setGeneralState("usable")
		assessment.Dimensions.Disease.State = "light"
	case 9:
		setGeneralState("strong")
		assessment.Dimensions.Disease.State = "light"
	}
	return assessment
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
