package runtime

import (
	"strings"
	"testing"
)

func TestValidateFinalWriterOutputAcceptsLifetimeHeadings(t *testing.T) {
	plan := baziAnalysisPlan{WriterTemplate: "full", NeedLifetimeDayun: true}
	output := "## 总览结论\n**结论：本命结构可用。**\n> ▲ 限制：仍有边界。\n\n## 强弱视角\n**结论：日主中和。**\n\n## 调候视角\n**结论：调候待核。**\n\n## 格局视角\n**结论：主轴可用。**\n- **规则口径**：已声明\n\n## 全程运路\n**结论：起伏中有兑现。**\n\n## 当前大运\n**结论：当前承接。**\n\n## 流年应期\n**结论：本年触发。**\n\n## 评判标准\n**结论：按本命与运路观察。**\n\n## 综合判定\n**结论：本命、全程与当前分别成立。**\n- **判定依据**：旧层次审计依据"
	state := baziCharterState{AnalysisPlan: plan, StaticSynthesis: baziStaticSynthesis{TierJudgment: "第5级", TierBasis: "旧层次审计依据"}}
	if err := validateFinalWriterOutput(plan, state, output); err != nil {
		t.Fatalf("lifetime headings rejected: %v", err)
	}
	rendered := renderFullTemplate(state)
	if !strings.Contains(rendered, "### 本命命格层次\n**结论：第5级**") || !strings.Contains(rendered, "**判定依据**：旧层次审计依据") {
		t.Fatalf("lifetime template dropped nine-level natal assessment: %s", rendered)
	}
}

func TestRenderFullTemplateKeepsCurrentDayunSeparateFromLifetimeRoute(t *testing.T) {
	state := baziCharterState{
		AnalysisPlan: baziAnalysisPlan{WriterTemplate: "full", NeedLifetimeDayun: true},
		Input: baziCharterInput{
			Dayun: map[string]any{"dayun_analyzed": []any{
				map[string]any{"ganZhi": "丙申", "startAge": 10, "endAge": 19},
				map[string]any{"ganZhi": "甲午", "startAge": 20, "endAge": 29},
			}},
			Liunian: map[string]any{"current_dayun": map[string]any{"ganZhi": "甲午"}},
		},
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend: "甲午运维持原局承接。",
			DayunPath:    []string{"丙申旧运说明", "甲午当前运说明"},
		},
	}
	rendered := renderFullTemplate(state)
	if strings.Contains(rendered, "丙申旧运说明") {
		t.Fatalf("current-dayun section replayed a lifetime period: %s", rendered)
	}
	if !strings.Contains(rendered, "**当前大运事实**：甲午运") {
		t.Fatalf("current-dayun section did not render the runtime-bound period: %s", rendered)
	}
}

func TestValidateLifetimeDayunOutputRequiresEveryCalculatedPeriod(t *testing.T) {
	chart := baziCharterState{Input: baziCharterInput{Dayun: map[string]any{"dayun_analyzed": []any{
		map[string]any{"ganZhi": "甲子"}, map[string]any{"ganZhi": "乙丑"},
	}}}}
	out := baziLifetimeDayunSynthesis{PeriodClaims: []baziLifetimeDayunClaim{{PeriodRef: "dayun[0]", FactRefs: []baziFactRef{"dayun[0].gan_zhi"}}}}
	if err := validateLifetimeDayunOutput(chart, out); err == nil {
		t.Fatal("expected incomplete all-life period coverage to fail")
	}
}

func TestValidateLifetimeDayunOutputRejectsDuplicatePeriod(t *testing.T) {
	chart := baziCharterState{Input: baziCharterInput{Dayun: map[string]any{"dayun_analyzed": []any{
		map[string]any{"ganZhi": "甲子"}, map[string]any{"ganZhi": "乙丑"},
	}}}}
	out := baziLifetimeDayunSynthesis{PeriodClaims: []baziLifetimeDayunClaim{
		{PeriodRef: "dayun[0]", FactRefs: []baziFactRef{"dayun[0].gan_zhi"}},
		{PeriodRef: "dayun[0]", FactRefs: []baziFactRef{"dayun[0].gan_zhi"}},
	}}
	if err := validateLifetimeDayunOutput(chart, out); err == nil {
		t.Fatal("expected duplicate all-life period reference to fail")
	}
}

func TestValidateLifetimeDayunOutputRequiresOwnPeriodFact(t *testing.T) {
	chart := baziCharterState{Input: baziCharterInput{Dayun: map[string]any{"dayun_analyzed": []any{
		map[string]any{"ganZhi": "甲子"}, map[string]any{"ganZhi": "乙丑"},
	}}}}
	out := baziLifetimeDayunSynthesis{PeriodClaims: []baziLifetimeDayunClaim{
		{PeriodRef: "dayun[0]", FactRefs: []baziFactRef{"dayun[0].gan_zhi"}},
		{PeriodRef: "dayun[1]", FactRefs: []baziFactRef{"dayun[0].gan_zhi"}},
	}}
	if err := validateLifetimeDayunOutput(chart, out); err == nil {
		t.Fatal("expected a lifetime claim without its own period fact to fail")
	}
}

func TestWriteLifetimeDayunGroupsSeparatesRoleFromVerdict(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{Dayun: map[string]any{"dayun_analyzed": []any{
			map[string]any{"ganZhi": "丙申", "startAge": 10, "endAge": 19},
			map[string]any{"ganZhi": "甲午", "startAge": 30, "endAge": 39},
		}}},
		LifetimeSynthesis: baziLifetimeDayunSynthesis{Status: "accepted", PeriodClaims: []baziLifetimeDayunClaim{
			{PeriodRef: "dayun[0]", PeriodEffect: "carry_balance", Verdict: "以泄身与生扶的平衡承接为主。"},
			{PeriodRef: "dayun[1]", PeriodEffect: "support_use", Verdict: "以扶助用神的结构作用为主。"},
		}},
	}
	var b strings.Builder
	writeLifetimeDayunGroups(&b, state)
	output := b.String()
	for _, want := range []string{"### 早期运程（29岁前）", "#### 丙申运（10-19岁）", "**结构定位：平衡承接**", "#### 甲午运（30-39岁）"} {
		if !strings.Contains(output, want) {
			t.Fatalf("lifetime group output missing %q: %s", want, output)
		}
	}
	if strings.Contains(output, "dayun[") {
		t.Fatalf("lifetime group output leaked internal ref: %s", output)
	}
}
