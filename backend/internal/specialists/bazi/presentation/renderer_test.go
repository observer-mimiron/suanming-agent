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

func TestPresentationUsesGejuConclusionAndEvidence(t *testing.T) {
	output := RenderFinalReply(FinalReplyInput{
		StaticSynthesis: StaticSynthesis{
			PatternName:       "伤官格",
			PatternRoute:      "伤官佩印",
			PatternEvaluation: "成格受限",
		},
	})
	for _, want := range []string{"## 格局视角", "**主格：伤官格", "成局路线：伤官佩印\n子平定性：成格受限"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q: %s", want, output)
		}
	}
	for _, unwanted := range []string{"候选主轴", "规则口径", "判读口径", "断语所限", "命格层次", "第6级", "保守定位"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("output leaked %q: %s", unwanted, output)
		}
	}
}

func TestPresentationShowsUsageSummaryBelowStrength(t *testing.T) {
	output := RenderFinalReply(FinalReplyInput{
		Facts: ChartFacts{UsageSummary: "喜神：木；用神：火；忌神：金、水"},
	})
	strength := strings.Index(output, "## 强弱视角")
	tiaohou := strings.Index(output, "## 调候视角")
	usage := strings.Index(output, "- **喜神**：木")
	if strength < 0 || usage < strength || tiaohou < usage {
		t.Fatalf("usage summary must appear below strength: %s", output)
	}
	for _, want := range []string{"- **喜神**：木", "- **用神**：火", "- **忌神**：金、水"} {
		if !strings.Contains(output, want) {
			t.Fatalf("usage output missing %q: %s", want, output)
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

func TestPresentationFiltersClassicalReferencesConflictingWithChart(t *testing.T) {
	output := RenderFinalReply(FinalReplyInput{
		Facts: ChartFacts{DayMaster: "丙", MonthCommand: "亥"},
		EvidenceBundle: EvidenceBundle{Citations: []Citation{
			{Classic: "穷通宝鉴", Quotes: []string{"冬月之金，形寒性冷。"}},
			{Classic: "穷通宝鉴", Quotes: []string{"论丙火：三春丙火，秉象至威。"}},
			{Classic: "子平真诠", Quotes: []string{"用神既定，则须观其成败救应。"}},
		}},
	})
	for _, unwanted := range []string{"冬月之金", "三春丙火"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("output kept conflicting seasonal quote %q: %s", unwanted, output)
		}
	}
	if !strings.Contains(output, "用神既定，则须观其成败救应。") {
		t.Fatalf("output removed generic reference: %s", output)
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

func TestValidateFinalWriterOutputRequiresCompactOverview(t *testing.T) {
	plan := baziAnalysisPlan{WriterTemplate: "full"}
	output := RenderFinalReply(FinalReplyInput{AnalysisPlan: AnalysisPlan{WriterTemplate: "full"}})
	for _, tc := range []struct {
		name    string
		output  string
		wantErr bool
	}{
		{name: "accepts compact overview", output: output},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFinalWriterOutput(plan, baziCharterState{}, tc.output)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateFinalWriterOutput() error = %v, wantErr %t", err, tc.wantErr)
			}
		})
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

	want := "- **庚子运（4-13岁）｜扶助用神**：庚为正官；此运有助于发挥本命用神。"
	if !strings.Contains(output, want) {
		t.Fatalf("full report missing compact lifetime entry %q: %s", want, output)
	}
}

func TestBaziPresentationDayunPeriodsOmitsTimestampRange(t *testing.T) {
	periods := baziPresentationDayunPeriods(map[string]any{
		"dayun_analyzed": map[string]any{"dayun_analyzed": []any{map[string]any{
			"ganZhi": "甲午", "startAge": 64, "endAge": 73,
			"startAt": "2058-01-21 12:08:00", "endAtExclusive": "2068-01-21 12:08:00",
		}}},
	})
	if len(periods) != 1 || periods[0].Label != "甲午运（64-73岁）" {
		t.Fatalf("period labels = %#v", periods)
	}
}

func TestPresentationKeepsAcceptedLifetimeLabels(t *testing.T) {
	output := RenderFinalReply(FinalReplyInput{
		AnalysisPlan:      AnalysisPlan{NeedLifetimeDayun: true},
		Facts:             ChartFacts{DayunPeriods: []DayunPeriod{{Ref: "dayun[0]", Label: "庚寅运（24-33岁）", GanZhi: "庚寅", TenGod: "偏财"}}},
		StaticSynthesis:   StaticSynthesis{},
		LifetimeSynthesis: LifetimeDayunSynthesis{Status: "accepted", PeriodClaims: []LifetimeDayunClaim{{PeriodRef: "dayun[0]", PeriodEffect: "support_use"}}},
	})
	for _, want := range []string{"扶助用神", "此运有助于发挥本命用神"} {
		if !strings.Contains(output, want) {
			t.Fatalf("provisional report missing lifetime label %q: %s", want, output)
		}
	}
}

func TestPresentationLimitationTextAvoidsTerminalPunctuationBeforeJoin(t *testing.T) {
	output := buildPresentationLimitationText(FinalReplyInput{StaticSynthesis: StaticSynthesis{
		CounterEvidence: "调候有效性尚待核验。",
	}})
	if strings.Contains(output, "。；") {
		t.Fatalf("limitation contains bad punctuation: %q", output)
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
	for _, forbidden := range []string{"**年性**", "**限制**"} {
		if strings.Contains(currentSection, forbidden) {
			t.Fatalf("facts-only dynamic output leaked %q: %s", forbidden, currentSection)
		}
	}
	if !strings.Contains(currentSection, "**流年干支**：丙午") {
		t.Fatalf("facts-only dynamic output missing calculated fact: %s", currentSection)
	}
}
