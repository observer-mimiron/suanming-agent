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
	if static.TierJudgment != "命格层次中等（保守定位）" {
		t.Fatalf("tier judgment = %q", static.TierJudgment)
	}
	if !strings.Contains(static.TierBasis, "调候") || !strings.Contains(static.TierBasis, "病药救应") {
		t.Fatalf("tier basis should name missing topics, got %q", static.TierBasis)
	}
	if strings.Contains(static.TierJudgment, "中上") {
		t.Fatalf("runtime tier judgment must not preserve model hard rank: %q", static.TierJudgment)
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

func TestCanonicalFailureFactsOnly_UsesSpecificAuditCode(t *testing.T) {
	static, dynamic := canonicalFailureFactsOnly(baziCharterState{}, errors.New("projection failed"), "canonical_static_projection_facts_only", "投影失败，已降级展示事实。")
	if !containsString(static.FieldAudit, "canonical_static_projection_facts_only") {
		t.Fatalf("static audit = %#v", static.FieldAudit)
	}
	if !containsString(dynamic.FieldAudit, "canonical_static_projection_facts_only") {
		t.Fatalf("dynamic audit = %#v", dynamic.FieldAudit)
	}
}

func TestBaziFieldAuditResult_IgnoresRuntimeTierWithholdingNote(t *testing.T) {
	if got := baziFieldAuditResult([]string{"canonical_tier_withheld_by_runtime"}); got != "clean" {
		t.Fatalf("audit result = %q, want clean", got)
	}
	if got := baziFieldAuditResult([]string{"canonical_tier_withheld_by_runtime", "canonical_dynamic_projection_facts_only"}); got != "repaired: canonical_dynamic_projection_facts_only" {
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
	if err := validateStaticSynthesisResult(withStaticSynthesis(state, static), static); err != nil {
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
	if len(dynamic.DayunJudgments) != 2 {
		t.Fatalf("dayun judgments got %d, want full deterministic period count", len(dynamic.DayunJudgments))
	}
	if dynamic.DayunJudgments[1].GanZhi != "乙丑" || !strings.Contains(dynamic.DayunJudgments[1].Interpretation, "结构承接") {
		t.Fatalf("current period judgment not projected: %+v", dynamic.DayunJudgments[1])
	}
	if dynamic.DayunJudgments[0].GanZhi != "甲子" || !strings.Contains(dynamic.DayunJudgments[0].Interpretation, "工具事实") {
		t.Fatalf("non-key period should retain facts-only interpretation: %+v", dynamic.DayunJudgments[0])
	}
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
	if dynamic.DayunJudgments[0].GanZhi != "甲子" || !strings.Contains(dynamic.DayunJudgments[0].Interpretation, "工具事实") {
		t.Fatalf("missing index must not bind model verdict to first period: %+v", dynamic.DayunJudgments[0])
	}
	if dynamic.DayunJudgments[1].GanZhi != "乙丑" || !strings.Contains(dynamic.DayunJudgments[1].Interpretation, "结构承接") {
		t.Fatalf("gan_zhi match should bind verdict to 乙丑: %+v", dynamic.DayunJudgments[1])
	}
}

func intPtr(value int) *int {
	return &value
}
