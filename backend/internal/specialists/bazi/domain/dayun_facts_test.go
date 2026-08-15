package domain

import "testing"

func TestDayunFactsNormalizeLegacyPayloadAndRenderJudgment(t *testing.T) {
	periods := DayunPeriods(map[string]any{
		"dayun_analyzed": map[string]any{"dayun_analyzed": []any{map[string]any{
			"ganZhi": "甲子", "startAge": 3, "endAge": 12,
			"startAt": "2000-01-02 03:04:05", "endAtExclusive": "2010-01-02 03:04:05",
		}}},
	})
	if len(periods) != 1 {
		t.Fatalf("period count = %d, want 1", len(periods))
	}
	if got, want := DayunPeriodDisplayLabel(periods[0]), "甲子运（3-12岁；2000-01-02 03:04至2010-01-02 03:04前）"; got != want {
		t.Fatalf("label = %q, want %q", got, want)
	}
	lines := RenderDayunJudgmentLines([]DayunJudgment{{GanZhi: "甲子", Trend: "平稳", Interpretation: "只作结构观察。", Evidence: []string{"  工具事实  ", ""}}})
	if len(lines) != 1 || lines[0] != "### 甲子：平稳\n**解读**：只作结构观察。\n- **依据**：工具事实" {
		t.Fatalf("judgment lines = %#v", lines)
	}
}
