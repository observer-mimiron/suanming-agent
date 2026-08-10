// This test file belongs to the manager-owned runtime layer.
// It verifies BaZi canonical synthesis contract and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"errors"
	"strings"
	"testing"
)

func TestProjectCanonicalSynthesis_WithheldTierUsesBoundedRuntimeTemplate(t *testing.T) {
	state := baziCharterState{
		AnalysisPlan: baziAnalysisPlan{NeedDynamic: true},
		Input: baziCharterInput{
			BaziResult: map[string]any{"dayGan": "丁", "pillars": []map[string]any{{"name": "月柱", "stem": "乙", "branch": "丑"}}},
			Yongshen:   map[string]any{"strength": "中和附近", "geju_candidate": "食神格", "geju_basis": "月令候选"},
			Dayun:      map[string]any{"dayun_analyzed": []map[string]any{{"ganZhi": "甲子", "tenGod": "正印"}}},
			Liunian:    map[string]any{"liunian_year": float64(2026), "liunian_ganzhi": "丙午"},
		},
		EvidenceQuality: baziEvidenceQuality{
			RequiredTopics: []string{"geju", "tiaohou", "bingyao"},
			CoveredTopics:  []string{"geju"},
			MissingTopics:  []string{"tiaohou", "bingyao"},
		},
	}
	canonical := baziCanonicalSynthesis{
		Source:   "model",
		MainAxis: baziCanonicalUnit{Verdict: "以食神制杀为主轴候选，但力度受限", EvidenceTopics: []string{"geju"}, Confidence: "倾向成立"},
		Strength: baziCanonicalUnit{Verdict: "中和附近", Boundary: "扶抑不直接等于格局取用", EvidenceTopics: []string{"geju"}},
		Tiaohou:  baziCanonicalUnit{Verdict: "调候证据不足，只能确认季节环境", Boundary: "不作具体先后裁断", EvidenceTopics: []string{"tiaohou"}},
		Pattern:  baziCanonicalUnit{Verdict: "食神路线可观察", Boundary: "不能拔高为贵格已成", EvidenceTopics: []string{"geju"}},
		Tier:     baziCanonicalUnit{Verdict: "中上", Boundary: "模型误写硬等级", EvidenceTopics: []string{"geju"}},
		Liunian:  baziCanonicalUnit{Verdict: "仅作结构观察", FactRefs: []string{"liunian.gan_zhi"}},
		DayunOverview: baziCanonicalUnit{
			Verdict:  "大运只作结构观察",
			FactRefs: []string{"dayun[0].gan_zhi"},
		},
	}

	static, dynamic := projectCanonicalSynthesis(state, canonical)
	if static.TierJudgment != "命格层次暂不定级（仅作结构观察）" {
		t.Fatalf("tier judgment = %q", static.TierJudgment)
	}
	if !strings.Contains(static.TierBasis, "调候") || !strings.Contains(static.TierBasis, "病药救应") {
		t.Fatalf("tier basis should name missing topics, got %q", static.TierBasis)
	}
	if strings.Contains(static.TierJudgment, "中等") || strings.Contains(static.TierJudgment, "中上") {
		t.Fatalf("runtime tier judgment must not invent a rank: %q", static.TierJudgment)
	}
	if static.Assertions[len(static.Assertions)-1].EvidenceStatus != baziEvidenceWithheld {
		t.Fatalf("tier assertion status = %q", static.Assertions[len(static.Assertions)-1].EvidenceStatus)
	}
	if err := validateStaticSynthesisResult(state, static); err != nil {
		t.Fatalf("runtime-withheld tier projection should validate: %v", err)
	}
	if len(dynamic.DayunPath) == 0 {
		t.Fatalf("dynamic projection must retain deterministic dayun facts")
	}
}

func TestProjectCanonicalDayunJudgmentsKeepsOnlySelectedClaims(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{
			Dayun: map[string]any{
				"dayun_analyzed": []map[string]any{
					{"ganZhi": "癸巳"}, {"ganZhi": "甲午"}, {"ganZhi": "乙未"},
				},
			},
		},
	}
	canonical := baziCanonicalSynthesis{DayunPeriods: []baziCanonicalDayunUnit{
		{Index: intPointer(1), GanZhi: "甲午", Verdict: "承托与扰动并见", Boundary: "只解释当前承接。"},
	}}

	judgments := projectCanonicalDayunJudgments(state, canonical)
	if len(judgments) != 1 || judgments[0].GanZhi != "甲午" {
		t.Fatalf("unselected periods must remain facts-only, got %#v", judgments)
	}
	facts := []string{"### 癸巳运", "### 甲午运", "### 乙未运"}
	merged := mergeCanonicalDayunPath(facts, judgments)
	if !strings.Contains(merged[1], "当前承接") || strings.Contains(merged[0], "当前承接") || strings.Contains(merged[2], "当前承接") {
		t.Fatalf("selected period claim was not attached to the matching fact line: %#v", merged)
	}
}

func intPointer(value int) *int {
	return &value
}

func TestProjectCanonicalSynthesis_PreservesTiaohouVerdict(t *testing.T) {
	state := baziCharterState{
		AnalysisPlan: baziAnalysisPlan{NeedDynamic: false},
		Input: baziCharterInput{
			BaziResult: map[string]any{"dayGan": "甲"},
			Yongshen:   map[string]any{"strength": "中和附近", "geju_candidate": "月令候选"},
			Dayun: map[string]any{"dayun_analyzed": []map[string]any{{
				"ganZhi":        "甲午",
				"dayun_chonghe": []map[string]any{{"description": "大运关系事实"}},
			}}},
		},
	}
	canonical := baziCanonicalSynthesis{
		MainAxis:      baziCanonicalUnit{Kind: "main_axis", Verdict: "主轴可观察"},
		Strength:      baziCanonicalUnit{Kind: "strength", Verdict: "中和附近"},
		Tiaohou:       baziCanonicalUnit{Kind: "tiaohou", Verdict: "调候火有但弱"},
		Pattern:       baziCanonicalUnit{Kind: "pattern_usage", Verdict: "格局可观察"},
		Tier:          baziCanonicalUnit{Kind: "tier", Verdict: "仅作结构观察"},
		DayunOverview: baziCanonicalUnit{Kind: "dayun_period", Verdict: "大运只作结构观察"},
		Liunian:       baziCanonicalUnit{Kind: "liunian", Verdict: "流年只作结构观察"},
	}
	static, _ := projectCanonicalSynthesis(state, canonical)
	if static.TiaohouAnchor != "调候火有但弱" {
		t.Fatalf("tiaohou anchor = %q, want model verdict preserved", static.TiaohouAnchor)
	}
}

func TestApplyStaticClaimsUsesSmallStructuredDTO(t *testing.T) {
	state := assertionTestState()
	output := baziStructuredStaticSynthesis{Claims: []baziStructuredStaticClaim{
		{Verdict: "以月令与透干关系立主轴候选", Status: "candidate"},
		{Verdict: "日主受扶较多，偏强", Status: "candidate"},
		{Verdict: "调候所需火力尚待核验", Status: "limited"},
		{Verdict: "格局取用仍需比较承接与反证", Status: "candidate"},
	}, AxisStatus: "candidate", TierAssessment: tierAssessmentForTest("provisional", 5), NatalRiskStatus: "withheld"}

	canonical, err := applyStaticClaims(state, baziCanonicalSynthesis{}, output)
	if err != nil {
		t.Fatalf("apply static claims: %v", err)
	}
	if canonical.MainAxis.Verdict != output.Claims[0].Verdict || canonical.Tiaohou.Verdict != output.Claims[2].Verdict {
		t.Fatalf("static verdicts were not preserved: %#v", canonical)
	}
	if canonical.MainAxis.Kind != string(baziAssertionMainAxis) || canonical.Strength.Kind != string(baziAssertionStrength) {
		t.Fatalf("runtime must restore fixed claim kinds: %#v", canonical)
	}
	if strings.Contains(canonical.MainAxis.Boundary, "保守边界") || !strings.Contains(canonical.Strength.Boundary, "扶身合计") {
		t.Fatalf("static boundaries must come from fact capsule: %#v", canonical)
	}
	if strings.Contains(canonical.StaticReasoningSummary, "主轴可观察") || len(canonical.ReasoningSteps) != 3 {
		t.Fatalf("runtime static explanation must not reuse model summary: %#v", canonical)
	}
}

func TestApplyStaticClaims_WithheldTierCannotLeakModelRank(t *testing.T) {
	output := baziStructuredStaticSynthesis{Claims: []baziStructuredStaticClaim{
		{Verdict: "主轴暂缓判断", Status: "withheld"},
		{Verdict: "强弱暂缓判断", Status: "withheld"},
		{Verdict: "调候暂缓判断", Status: "withheld"},
		{Verdict: "格局取用暂缓判断", Status: "withheld"},
	}, AxisStatus: "withheld", TierAssessment: tierAssessmentForTest("withheld", 0), NatalRiskStatus: "withheld"}
	canonical, err := applyStaticClaims(assertionTestState(), baziCanonicalSynthesis{}, output)
	if err != nil {
		t.Fatalf("apply static claims: %v", err)
	}
	if canonical.Tier.Verdict != "命格基础层次暂缓判定" {
		t.Fatalf("withheld tier leaked model rank: %q", canonical.Tier.Verdict)
	}
}

func TestCanonicalTriggerSignalsProjectsReadableCurrentRelations(t *testing.T) {
	state := baziCharterState{Input: baziCharterInput{
		Dayun:   map[string]any{"dayun_analyzed": []map[string]any{{"ganZhi": "甲午", "dayun_chonghe": []map[string]any{{"description": "大运午午自刑时柱午"}}}}},
		Liunian: map[string]any{"current_dayun": map[string]any{"ganZhi": "甲午"}, "liunian_chonghe": []map[string]any{{"description": "流年午午自刑大运午"}}},
	}}
	got := strings.Join(canonicalTriggerSignals(state, baziCanonicalSynthesis{DayunOverview: baziCanonicalUnit{FactRefs: []string{"dayun[0].gan_zhi"}}}), "；")
	if strings.Contains(got, "dayun[") || !strings.Contains(got, "大运午午自刑时柱午") {
		t.Fatalf("trigger signals must be readable facts, got %q", got)
	}
}

func TestApplyDynamicClaimsProjectsUnselectedPeriodsAsFactsOnly(t *testing.T) {
	state := baziCharterState{Input: baziCharterInput{Dayun: map[string]any{"dayun_analyzed": []map[string]any{
		{"ganZhi": "甲子"},
		{"ganZhi": "乙丑"},
	}}}}
	output := baziStructuredDynamicSynthesis{
		CurrentPeriodRef: "dayun[1]",
		PeriodClaims: []baziStructuredPeriodClaim{{
			PeriodRef: "dayun[1]", Verdict: "重点运承接主轴", Boundary: "模型写入的层次边界不得进入投影",
			Confidence: "倾向成立",
		}},
		LiunianClaim: baziStructuredClaim{Verdict: "结构观察", Boundary: "模型边界不得进入投影"},
	}
	canonical, err := applyDynamicClaims(state, baziCanonicalSynthesis{}, output)
	if err != nil {
		t.Fatalf("apply dynamic claims: %v", err)
	}
	if len(canonical.DayunPeriods) != 2 || canonical.DayunPeriods[0].GanZhi != "甲子" || canonical.DayunPeriods[1].GanZhi != "乙丑" {
		t.Fatalf("runtime did not preserve all period facts: %#v", canonical.DayunPeriods)
	}
	if canonical.DayunPeriods[0].Verdict != "" || canonical.DayunPeriods[1].Verdict != "重点运承接主轴" {
		t.Fatalf("period claim selection was not projected by period_ref: %#v", canonical.DayunPeriods)
	}
	if strings.Contains(canonical.DayunPeriods[1].Boundary, "模型写入") || !strings.Contains(canonical.DayunPeriods[1].Boundary, "当前已绑定大运") {
		t.Fatalf("period boundary must be runtime-owned: %q", canonical.DayunPeriods[1].Boundary)
	}
	if strings.Contains(canonical.Liunian.Boundary, "模型边界") || !strings.Contains(canonical.Liunian.Boundary, "当前大运") {
		t.Fatalf("liunian boundary must be runtime-owned: %q", canonical.Liunian.Boundary)
	}
}

func TestCanonicalFailureFactsOnly_UsesSpecificAuditCode(t *testing.T) {
	static, dynamic := canonicalFailureFactsOnly(baziCharterState{}, errors.New("projection failed"), "canonical_static_projection_facts_only", "投影失败，已降级展示事实。")
	if !containsString(static.FieldAudit, "canonical_static_projection_facts_only") {
		t.Fatalf("static audit = %#v", static.FieldAudit)
	}
	if !containsString(dynamic.FieldAudit, "canonical_static_projection_facts_only") {
		t.Fatalf("dynamic audit = %#v", dynamic.FieldAudit)
	}
}

func TestRenderFactsOnlyDegradedTemplateIncludesStrengthFacts(t *testing.T) {
	out := renderFactsOnlyDegradedTemplate(baziCharterState{StaticSynthesis: baziStaticSynthesis{StrengthBalance: "日主偏强；扶身证据 13、泄耗克证据 9。"}})
	for _, want := range []string{"## 强弱事实", "强弱证据", "日主偏强"} {
		if !strings.Contains(out, want) {
			t.Fatalf("facts-only output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderFullTemplateSanitizesInternalReferencesAndControlsAxisEcho(t *testing.T) {
	axis := "伤官格为主轴"
	output := renderFullTemplate(baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis:        axis,
			PatternBasis:    axis,
			PatternOutcome:  axis,
			CounterEvidence: axis,
			Advantages:      []string{axis},
			Risks:           []string{axis},
			Usage:           baziUsageLayers{Pattern: axis},
			TierJudgment:    "命格基础层次：第5级（中格，有路但利弊并见）",
			TierBasis:       "层次依据为已验证的结构事实。",
		},
		DynamicSynthesis: baziDynamicSynthesis{
			Source:                   "model",
			CurrentTrend:             axis,
			CurrentPeriodRealization: "maintain",
			LiunianFocus:             axis,
			TriggerSignals:           []string{"dayun[2].relations", "pressure_score"},
		},
	})
	for _, forbidden := range []string{"dayun[2]", "liunian.shi_shen", "support_score", "pressure_score"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("user output leaked internal reference %q:\n%s", forbidden, output)
		}
	}
	if !strings.Contains(output, "已计算的结构事实") {
		t.Fatalf("internal references should project to readable facts:\n%s", output)
	}
	if count := strings.Count(output, axis); count != 1 {
		t.Fatalf("main axis echo count = %d, want 1:\n%s", count, output)
	}
}

func TestRenderFullTemplateCombinesNatalBaselineAndCurrentMomentum(t *testing.T) {
	out := renderFullTemplate(baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			TierJudgment: "命格基础层次：第5级（中格）",
			TierBasis:    "本命主轴有路，但限制并见。",
		},
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:             "甲午运进入较强发力窗口，但仍伴随扰动。",
			CurrentPeriodRealization: "assist",
		},
	})
	section := sectionContent(out, "## 综合判定", "")
	for _, want := range []string{"命格基础层次：第5级（中格）", "当前岁运走势：甲午运进入较强发力窗口", "**判定依据**", "**岁运兑现**"} {
		if !strings.Contains(section, want) {
			t.Fatalf("combined assessment missing %q:\n%s", want, section)
		}
	}
}

func TestBuildBaziCanonicalRepairFeedbackSkipsUnmatchedLearningHints(t *testing.T) {
	feedback := buildBaziCanonicalRepairFeedback(RepairFailure{
		Domain:  "bazi",
		Stage:   "static_projection",
		Class:   RepairProjectionMismatch,
		Field:   "static.pattern",
		Message: "格局投影缺少裁断",
	}, 1)
	hints, ok := feedback["learning_hints"].([]map[string]string)
	if !ok {
		t.Fatalf("learning_hints type = %T, want []map[string]string", feedback["learning_hints"])
	}
	if len(hints) != 0 {
		t.Fatalf("unmatched repair should not receive hints: %#v", hints)
	}
}

func TestRepairLearningHintsForCapsPerField(t *testing.T) {
	hints := RepairLearningHintsFor(RepairFailure{
		Domain: "bazi",
		Stage:  "static_projection",
		Class:  RepairProjectionMismatch,
		Field:  "static.tiaohou_anchor",
	})
	if len(hints) == 0 || len(hints) > maxRepairLearningHintsPerField {
		t.Fatalf("hint count = %d, want 1..%d", len(hints), maxRepairLearningHintsPerField)
	}
}

func repairHintsContain(hints []map[string]string, key, needle string) bool {
	for _, hint := range hints {
		if strings.Contains(hint[key], needle) {
			return true
		}
	}
	return false
}

func TestBaziFieldAuditResult_IgnoresRuntimeTierWithholdingNote(t *testing.T) {
	if got := baziFieldAuditResult([]string{"canonical_tier_withheld_by_runtime"}); got != "clean" {
		t.Fatalf("audit result = %q, want clean", got)
	}
	if got := baziFieldAuditResult([]string{"canonical_tier_withheld_by_runtime", "canonical_dynamic_projection_facts_only"}); got != "clean" {
		t.Fatalf("facts-only audit result = %q, want clean", got)
	}
}

func TestBaziFieldAuditResult_DeduplicatesRecoveryNotes(t *testing.T) {
	got := baziFieldAuditResult([]string{
		"canonical_static_projection_facts_only",
		"canonical_static_projection_facts_only",
		"contract_failure_class:domain_unauthorized",
		"contract_failure_class:domain_unauthorized",
	})
	if got != "repaired: canonical_static_projection_facts_only, contract_failure_class:domain_unauthorized" {
		t.Fatalf("audit result = %q", got)
	}
}

func TestCanonicalDynamicFailureFactsOnly_RecordsDynamicRecoveryOnly(t *testing.T) {
	static := baziStaticSynthesis{Source: "model", MainAxis: "静态主轴保留"}
	err := baziViolationError(baziViolationUnsupportedConcreteOutcome, "dynamic", "", "dynamic synthesis overstates unsupported concrete outcome", nil, nil)

	dynamic := canonicalDynamicFailureFactsOnly(baziCharterState{}, static, err)
	if dynamic.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("dynamic source = %q", dynamic.Source)
	}
	if !containsString(dynamic.FieldAudit, "canonical_dynamic_projection_facts_only") ||
		!containsString(dynamic.FieldAudit, "contract_failure_class:"+baziContractFailureDomainUnauthorized) ||
		!containsString(dynamic.FieldAudit, "recovery_policy:"+baziRecoveryPolicyDynamicFactsOnly) {
		t.Fatalf("dynamic audit = %#v", dynamic.FieldAudit)
	}
	if static.Source != "model" || static.MainAxis != "静态主轴保留" {
		t.Fatalf("static should remain unchanged: %+v", static)
	}
}

func TestProjectCanonicalSynthesis_MinorStaticProjectionDropsUnauthorizedOutcomes(t *testing.T) {
	state := baziCharterState{
		AnalysisPlan: baziAnalysisPlan{NeedDynamic: false},
		Input: baziCharterInput{
			BaziResult: map[string]any{"birthday": "2025-11-11 00:15:00", "dayGan": "甲"},
			Yongshen: map[string]any{
				"strength":       "偏弱",
				"geju_candidate": "伤官格",
				"strength_evidence": map[string]any{
					"support_score":  float64(4),
					"pressure_score": float64(8),
				},
			},
			Liunian: map[string]any{"liunian_year": float64(2026)},
		},
		EvidenceQuality: baziEvidenceQuality{RequiredTopics: []string{"geju"}, CoveredTopics: []string{"geju"}},
	}
	canonical := baziCanonicalSynthesis{
		Source:   "model",
		MainAxis: baziCanonicalUnit{Verdict: "结构主轴可观察，但不要延伸到健康风险", Confidence: "保守判断"},
		Strength: baziCanonicalUnit{
			Verdict:  "偏弱，需注意健康",
			Boundary: "强弱只说明受力，不能推出事业或财富",
		},
		Tiaohou: baziCanonicalUnit{Verdict: "调候只作季节环境观察"},
		Pattern: baziCanonicalUnit{
			Verdict:  "伤官格候选，但不推出事业突破",
			Boundary: "不展开婚姻、财富或健康应事",
		},
		Tier: baziCanonicalUnit{Verdict: "仅作结构观察"},
		Advantages: []string{
			"结构主轴可观察",
			"未来事业突破倾向明显",
		},
		Risks: []string{
			"健康风险需留意",
			"只按成长环境观察",
		},
		ReasoningSteps: []string{
			"先看强弱与调候。",
			"再推事业、财富和婚姻。",
		},
	}

	static, _ := projectCanonicalSynthesis(state, canonical)
	visible := staticSynthesisUserVisibleText(static)
	if domain, term := firstUnauthorizedMinorOutcomeSignal(visible); domain != "" {
		t.Fatalf("minor static projection kept unauthorized %s outcome %q in %q", domain, term, visible)
	}
	if !strings.Contains(visible, "强弱") || !strings.Contains(visible, "调候") {
		t.Fatalf("projection should keep structural sections, got %q", visible)
	}
	state.StaticSynthesis = static
	if err := validateStaticSynthesisResult(state, static); err != nil {
		t.Fatalf("sanitized minor static projection should validate: %v", err)
	}
}

func TestProjectCanonicalSynthesis_DayunJudgmentsKeepDeterministicPeriods(t *testing.T) {
	state := baziCharterState{
		AnalysisPlan: baziAnalysisPlan{NeedDynamic: true},
		Input: baziCharterInput{
			BaziResult: map[string]any{"birthday": "1994-01-21 20:30:00", "dayGan": "丁"},
			Yongshen:   map[string]any{"strength": "中和附近", "geju_candidate": "食神格"},
			Dayun: map[string]any{"dayun_analyzed": []map[string]any{
				{"ganZhi": "甲子", "tenGod": "正印", "startAge": float64(1), "endAge": float64(10)},
				{"ganZhi": "乙丑", "tenGod": "偏印", "startAge": float64(11), "endAge": float64(20)},
			}},
			Liunian: map[string]any{
				"liunian_year":   float64(2026),
				"liunian_ganzhi": "丙午",
				"current_dayun":  map[string]any{"ganZhi": "乙丑"},
			},
		},
		EvidenceQuality: baziEvidenceQuality{RequiredTopics: []string{"geju"}, CoveredTopics: []string{"geju"}},
	}
	canonical := baziCanonicalSynthesis{
		Source:        "model",
		MainAxis:      baziCanonicalUnit{Verdict: "主轴候选成立但受限", EvidenceTopics: []string{"geju"}},
		Strength:      baziCanonicalUnit{Verdict: "中和附近"},
		Tiaohou:       baziCanonicalUnit{Verdict: "只作季节环境观察"},
		Pattern:       baziCanonicalUnit{Verdict: "格局路线可观察", EvidenceTopics: []string{"geju"}},
		Tier:          baziCanonicalUnit{Verdict: "仅作结构观察", EvidenceTopics: []string{"geju"}},
		DayunOverview: baziCanonicalUnit{Verdict: "当前乙丑运只作结构承接观察"},
		DayunPeriods: []baziCanonicalDayunUnit{{
			Index:    intPtr(1),
			Verdict:  "乙丑运承接主轴但不纯顺",
			Boundary: "只说明结构承接，不展开具体事业财务事件",
		}},
		Liunian: baziCanonicalUnit{Verdict: "丙午年触发明显，但仅作结构观察"},
	}

	_, dynamic := projectCanonicalSynthesis(state, canonical)
	if len(dynamic.DayunJudgments) != 1 {
		t.Fatalf("dayun judgments got %d, want only the selected model period", len(dynamic.DayunJudgments))
	}
	if dynamic.DayunJudgments[0].GanZhi != "乙丑" || !strings.Contains(dynamic.DayunJudgments[0].Interpretation, "结构承接") {
		t.Fatalf("current period judgment not projected: %+v", dynamic.DayunJudgments[0])
	}
	if strings.Contains(strings.Join(dynamic.DayunPath, "\n"), "未被模型列为重点窗口") {
		t.Fatalf("non-key period must remain a plain deterministic fact line: %v", dynamic.DayunPath)
	}
}

func TestProjectCanonicalDayunJudgments_ProjectsReadableEvidence(t *testing.T) {
	state := baziCharterState{Input: baziCharterInput{Dayun: map[string]any{"dayun_analyzed": []map[string]any{{
		"ganZhi": "甲午", "ageRange": "29-38岁",
	}}}}}
	judgments := projectCanonicalDayunJudgments(state, baziCanonicalSynthesis{DayunPeriods: []baziCanonicalDayunUnit{{
		Index: intPtr(0), Verdict: "只作结构观察", FactRefs: []string{"dayun[0].gan_zhi"},
	}}})
	if got := strings.Join(judgments[0].Evidence, "；"); strings.Contains(got, "dayun[") || !strings.Contains(got, "甲午") {
		t.Fatalf("dayun evidence must be user-readable facts, got %q", got)
	}
}

func TestRegression1991Profile_RejectsFirstDayunAsCurrentPeriodClaim(t *testing.T) {
	state := baziCharterState{Input: baziCharterInput{
		Dayun:   map[string]any{"dayun_analyzed": []map[string]any{{"ganZhi": "癸巳"}, {"ganZhi": "甲午"}}},
		Liunian: map[string]any{"liunian_year": float64(2026), "current_dayun": map[string]any{"ganZhi": "甲午"}},
	}}
	err := validateDynamicPeriodClaims(state, []baziStructuredPeriodClaim{{PeriodRef: "dayun[0]"}})
	assertBaziViolationCode(t, err, baziViolationMethodContract)
}

func TestProjectCanonicalSynthesis_DayunPeriodMatchesGanZhiWhenIndexMissing(t *testing.T) {
	state := baziCharterState{
		AnalysisPlan: baziAnalysisPlan{NeedDynamic: true},
		Input: baziCharterInput{
			BaziResult: map[string]any{"dayGan": "丁"},
			Yongshen:   map[string]any{"strength": "偏弱", "geju_candidate": "食神格"},
			Dayun: map[string]any{"dayun_analyzed": []map[string]any{
				{"ganZhi": "甲子", "tenGod": "正印"},
				{"ganZhi": "乙丑", "tenGod": "偏印"},
			}},
			Liunian: map[string]any{"liunian_ganzhi": "丙午"},
		},
		EvidenceQuality: baziEvidenceQuality{RequiredTopics: []string{"geju"}, CoveredTopics: []string{"geju"}},
	}
	canonical := baziCanonicalSynthesis{
		Source:        "model",
		MainAxis:      baziCanonicalUnit{Verdict: "主轴候选成立但受限"},
		Strength:      baziCanonicalUnit{Verdict: "偏弱"},
		Tiaohou:       baziCanonicalUnit{Verdict: "调候只作边界观察"},
		Pattern:       baziCanonicalUnit{Verdict: "食神路线可观察"},
		Tier:          baziCanonicalUnit{Verdict: "仅作结构观察"},
		DayunOverview: baziCanonicalUnit{Verdict: "大运只作结构观察"},
		DayunPeriods: []baziCanonicalDayunUnit{{
			GanZhi:   "乙丑",
			Verdict:  "乙丑运承接主轴但不纯顺",
			Boundary: "只说明结构承接，不展开具体事件",
		}},
		Liunian: baziCanonicalUnit{Verdict: "丙午年仅作结构观察"},
	}

	_, dynamic := projectCanonicalSynthesis(state, canonical)
	if len(dynamic.DayunJudgments) != 1 || dynamic.DayunJudgments[0].GanZhi != "乙丑" || !strings.Contains(dynamic.DayunJudgments[0].Interpretation, "结构承接") {
		t.Fatalf("gan_zhi match should bind verdict only to 乙丑: %+v", dynamic.DayunJudgments)
	}
}

func intPtr(value int) *int {
	return &value
}
