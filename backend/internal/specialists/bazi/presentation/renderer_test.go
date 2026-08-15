package presentation

import (
	"strings"
	"testing"
)

func TestPresentationTiaohouConclusionPrefersAnchor(t *testing.T) {
	input := FinalReplyInput{StaticSynthesis: StaticSynthesis{
		TiaohouAnchor:     "甲木生亥月，寒湿重，调候火有但弱。",
		TiaohouConstraint: "调候为环境约束，不直接决定格局成败。",
	}}
	if got := buildPresentationTiaohouConclusion(input); got != input.StaticSynthesis.TiaohouAnchor {
		t.Fatalf("tiaohou conclusion = %q", got)
	}
}

func TestPresentationLimitationDeduplicatesExactFallback(t *testing.T) {
	input := FinalReplyInput{StaticSynthesis: StaticSynthesis{
		CounterEvidence: "关系触发会增加过程反复，具体应事不作展开。",
		TierBasis:       "关系触发会增加过程反复，具体应事不作展开。",
	}}
	if got := buildPresentationLimitationText(input); strings.Count(got, input.StaticSynthesis.CounterEvidence) != 1 {
		t.Fatalf("expected exact duplicate limitation once, got %q", got)
	}
}

func TestPresentationPatternEvidenceDoesNotRepeatAxis(t *testing.T) {
	input := FinalReplyInput{
		Facts: ChartFacts{MonthCommand: "酉"},
		StaticSynthesis: StaticSynthesis{
			MainAxis:       "伤官佩印为主轴",
			PatternOutcome: "格局取用仍需结合清浊与破格风险。",
		},
	}
	if got := buildPresentationPatternEvidence(input); strings.Contains(got, input.StaticSynthesis.MainAxis) {
		t.Fatalf("pattern evidence must not repeat overview main axis: %q", got)
	}
}

func TestPresentationUsesGejuEvaluationWithoutInternalTierScale(t *testing.T) {
	output := RenderFinalReply(FinalReplyInput{
		StaticSynthesis: StaticSynthesis{
			TierJudgment: "格局评价已定",
			TierBasis:    "格局评价依据已验收的结构、证据与限制维度综合确定。",
		},
	})
	for _, want := range []string{"### 格局评价", "**格局评价**", "格局评价已定"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q: %s", want, output)
		}
	}
	for _, unwanted := range []string{"命格层次", "第6级", "保守定位"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("output leaked %q: %s", unwanted, output)
		}
	}
}

func TestPresentationShowsReadableClassicalReferences(t *testing.T) {
	output := RenderFinalReply(FinalReplyInput{
		EvidenceBundle: EvidenceBundle{Citations: []Citation{{
			Classic: "子平真诠",
			Quotes:  []string{"伤官用财，宜见财星。"},
		}}},
	})
	for _, want := range []string{"### 古籍参照", "《子平真诠》", "伤官用财，宜见财星。"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q: %s", want, output)
		}
	}
}

func TestConciseDisplayTextKeepsCompleteClauseOverCap(t *testing.T) {
	input := "流年子午冲时柱子水，增加变动与不稳定因素。"
	if got := conciseDisplayText(input, 8); got != input {
		t.Fatalf("conciseDisplayText() = %q, want complete clause %q", got, input)
	}
}

func TestPresentationFullReportPlacesOverviewFirstWithoutRepeat(t *testing.T) {
	output := RenderFinalReply(FinalReplyInput{})
	if !strings.HasPrefix(output, "## 总览结论") {
		t.Fatalf("full report must start with overview: %s", output)
	}
	if count := strings.Count(output, "## 总览结论"); count != 1 {
		t.Fatalf("overview count = %d, want 1: %s", count, output)
	}
}

func TestPresentationFullReportUsesCompactLifetimeEntries(t *testing.T) {
	output := RenderFinalReply(FinalReplyInput{
		AnalysisPlan: AnalysisPlan{NeedLifetimeDayun: true},
		Facts: ChartFacts{DayunPeriods: []DayunPeriod{{
			Ref: "dayun[0]", Label: "庚子运（4-13岁）", GanZhi: "庚子", TenGod: "正官",
		}}},
		LifetimeSynthesis: LifetimeDayunSynthesis{
			Status: "accepted",
			PeriodClaims: []LifetimeDayunClaim{{
				PeriodRef: "dayun[0]", PeriodEffect: "support_use",
			}},
		},
	})

	want := "- **庚子运（4-13岁）｜扶助用神**：庚为正官；此运对本命用神形成助力，具体力度仍以已计算关系为准。"
	if !strings.Contains(output, want) {
		t.Fatalf("full report missing compact lifetime entry %q: %s", want, output)
	}
}

func TestPresentationFactsOnlyDynamicSuppressesTrendFields(t *testing.T) {
	output := RenderFinalReply(FinalReplyInput{
		Facts: ChartFacts{LiunianGanZhi: "丙午", LiunianTenGod: "伤官"},
		DynamicSynthesis: DynamicSynthesis{
			FactsOnly:      true,
			WindowLevel:    "扰动年",
			TriggerSignals: []string{"流年子午冲月柱子"},
		},
	})
	currentSection := sectionContent(output, "## 当前应期", "")
	for _, forbidden := range []string{"**年性**", "**依据**", "**限制**"} {
		if strings.Contains(currentSection, forbidden) {
			t.Fatalf("facts-only dynamic output leaked %q: %s", forbidden, currentSection)
		}
	}
	if !strings.Contains(currentSection, "**流年干支**：丙午") {
		t.Fatalf("facts-only dynamic output missing calculated fact: %s", currentSection)
	}
}
