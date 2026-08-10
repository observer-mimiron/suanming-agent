// This test file belongs to the manager-owned runtime layer.
// It verifies BaZi profile synthesis behavior and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"strings"
	"testing"
)

func TestProfileStaticSynthesis_FactsOnlyDoesNotProjectClaims(t *testing.T) {
	yongshen := map[string]any{
		"geju_candidate":   "伤官格(官未透)",
		"geju_basis":       "月令酉本气辛透出天干，先以本气十神伤官定主格为伤官格",
		"geju_combination": "伤官佩印候选（伤官在年干、正印在月干）",
		"strength":         "中和附近",
		"strength_evidence": map[string]any{
			"support_score":  float64(7),
			"pressure_score": float64(8),
		},
	}
	static := buildProfileStaticSynthesis(baziCharterInput{
		BaziResult: map[string]any{"dayGan": "戊", "pillars": []map[string]any{{"name": "年柱", "stem": "辛", "branch": "未"}, {"name": "月柱", "stem": "丁", "branch": "酉"}}},
		Yongshen:   yongshen,
	})

	if static.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("profile fallback source = %q, want facts-only degraded", static.Source)
	}
	if !strings.Contains(static.MainAxis, "不输出主轴裁断") || !strings.Contains(static.TierJudgment, "不输出层次裁断") {
		t.Fatalf("facts-only static synthesis must withhold judgment, got %+v", static)
	}
	if !strings.Contains(static.StrengthBalance, "扶身合计 7") || !strings.Contains(static.StrengthBalance, "食伤泄身、财耗与官杀克合计 8") || !strings.Contains(static.PatternBasis, "月令取格候选") {
		t.Fatalf("facts-only static synthesis must retain tool facts, got %+v", static)
	}
	if containsAnyText([]string{static.MainAxis, static.PatternOutcome, static.TierBasis, static.ReasoningSummary}, []string{
		"优先按伤官佩印", "先丙后癸", "中等（保守定位）", "食神制杀",
	}) {
		t.Fatalf("facts-only static synthesis leaked synthesized reading: %+v", static)
	}
	if err := validateStaticSynthesisResult(baziCharterState{}, static); err != nil {
		t.Fatalf("facts-only static synthesis must validate as degraded output: %v", err)
	}
}

func TestPeriodHeadline_StripsMarkdownHeadingPrefix(t *testing.T) {
	got := periodHeadline("### 甲午运（30-39岁）：阻力偏重；十神：七杀\n**综合解读**：过程宜保守观察。")
	if got != "甲午运（30-39岁）：阻力偏重；十神：七杀" {
		t.Fatalf("period headline = %q", got)
	}
}

func TestProfileDynamicSynthesis_FactsOnlyRendersDeclaredFacts(t *testing.T) {
	input := baziCharterInput{
		Dayun: map[string]any{"dayun_analyzed": []map[string]any{{
			"ganZhi":         "甲午",
			"tenGod":         "七杀",
			"startAge":       30,
			"endAge":         39,
			"startAt":        "2020-10-05 12:52:00",
			"endAtExclusive": "2030-10-05 12:52:00",
			"dayun_chonghe":  []map[string]any{{"description": "大运午午自刑时柱午"}},
		}}},
		Liunian: map[string]any{
			"current_dayun":     map[string]any{"ganZhi": "甲午"},
			"liunian_ganzhi":    "丙午",
			"liunian_shi_shen":  "偏印",
			"liunian_chonghe":   []map[string]any{{"description": "流年午午自刑时柱午"}},
			"liunian_target_at": "2026-07-28 12:00:00",
		},
	}
	dynamic := buildProfileDynamicSynthesis(input, baziStaticSynthesis{})

	if dynamic.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("dynamic source = %q, want facts-only degraded", dynamic.Source)
	}
	all := strings.Join(append(append(append([]string{dynamic.CurrentTrend, dynamic.LiunianFocus}, dynamic.DayunPath...), dynamic.TriggerSignals...), dynamic.ReasoningSteps...), "\n")
	for _, want := range []string{"甲午", "七杀", "30-39岁", "2020-10-05 12:52", "午午自刑", "丙午"} {
		if !strings.Contains(all, want) {
			t.Fatalf("facts-only dynamic output missing %q:\n%s", want, all)
		}
	}
	for _, forbidden := range []string{"结构承接", "承托与压力", "**解读**", "大吉", "官非", "破财", "倾向有利", "动态综合未通过", "runtime", "模型动态综合不可用"} {
		if strings.Contains(all, forbidden) {
			t.Fatalf("facts-only dynamic output leaked synthesized wording %q:\n%s", forbidden, all)
		}
	}
	if dynamic.CurrentDayunIndex != 0 {
		t.Fatalf("current dayun index = %d, want 0", dynamic.CurrentDayunIndex)
	}
	if err := validateDynamicStage(baziCharterState{Input: input, DynamicSynthesis: dynamic}); err != nil {
		t.Fatalf("facts-only dynamic synthesis must validate as degraded output: %v", err)
	}
}

func TestProfileDynamicSynthesis_CurrentDayunIndexFollowsChronologicalPath(t *testing.T) {
	input := baziCharterInput{
		Dayun: map[string]any{"dayun_analyzed": []map[string]any{
			{"ganZhi": "丙申", "startAge": 10, "endAge": 19},
			{"ganZhi": "乙未", "startAge": 20, "endAge": 29},
			{"ganZhi": "甲午", "startAge": 30, "endAge": 39},
		}},
		Liunian: map[string]any{"current_dayun": map[string]any{"ganZhi": "甲午"}, "liunian_ganzhi": "丙午"},
	}
	dynamic := buildProfileDynamicSynthesis(input, baziStaticSynthesis{})
	if dynamic.CurrentDayunIndex != 2 {
		t.Fatalf("current dayun index = %d, want 2", dynamic.CurrentDayunIndex)
	}
}

func TestProfileDynamicSynthesis_RecoversCurrentDayunFromAnnotatedBoundaries(t *testing.T) {
	input := baziCharterInput{
		Dayun: map[string]any{"dayun_analyzed": []map[string]any{
			{"ganZhi": "乙未", "startAt": "2010-10-05 12:00:00", "endAtExclusive": "2020-10-05 12:00:00"},
			{"ganZhi": "甲午", "startAt": "2020-10-05 12:00:00", "endAtExclusive": "2030-10-05 12:00:00"},
		}},
		Liunian: map[string]any{
			"liunian_target_at": "2026-07-28 12:00:00",
			"liunian_ganzhi":    "丙午",
			"current_dayun":     map[string]any{},
		},
	}
	dynamic := buildProfileDynamicSynthesis(input, baziStaticSynthesis{})
	if dynamic.CurrentDayunIndex != 1 || !strings.Contains(dynamic.ReasoningSteps[1], "甲午运") {
		t.Fatalf("boundary recovery must locate 甲午, got index=%d steps=%v", dynamic.CurrentDayunIndex, dynamic.ReasoningSteps)
	}
}

func TestProfileDynamicSynthesis_MissingCurrentDayunIsExplicitWithoutInventingOne(t *testing.T) {
	input := baziCharterInput{
		Dayun:   map[string]any{"dayun_analyzed": []map[string]any{{"ganZhi": "甲午"}}},
		Liunian: map[string]any{"liunian_ganzhi": "丙午", "current_dayun": map[string]any{}},
	}
	dynamic := buildProfileDynamicSynthesis(input, baziStaticSynthesis{})
	if dynamic.CurrentDayunIndex != -1 || !strings.Contains(dynamic.ReasoningSteps[1], "事实未能定位") {
		t.Fatalf("missing current dayun must remain explicit, got index=%d steps=%v", dynamic.CurrentDayunIndex, dynamic.ReasoningSteps)
	}
}

func TestProfileDynamicSynthesis_PreStartDayunIsNotReportedAsMissingFact(t *testing.T) {
	input := baziCharterInput{
		Dayun: map[string]any{"dayun_analyzed": []map[string]any{{
			"ganZhi": "丙戌", "startAt": "2027-01-01 12:00:00", "endAtExclusive": "2037-01-01 12:00:00",
		}}},
		Liunian: map[string]any{
			"liunian_target_at": "2026-07-28 12:00:00", "current_dayun": map[string]any{},
		},
	}
	dynamic := buildProfileDynamicSynthesis(input, baziStaticSynthesis{})
	if !strings.Contains(dynamic.ReasoningSteps[1], "尚未交入第一步大运") || strings.Contains(dynamic.ReasoningSteps[1], "事实未能定位") {
		t.Fatalf("pre-start chart must be explained as pre-start, got steps=%v", dynamic.ReasoningSteps)
	}
}
