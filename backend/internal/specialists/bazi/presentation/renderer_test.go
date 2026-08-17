package presentation

import (
	"strings"
	"testing"

	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
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

func TestValidateFinalWriterOutputRequiresOverviewTierAndBoundary(t *testing.T) {
	plan := baziAnalysisPlan{WriterTemplate: "full"}
	output := RenderFinalReply(FinalReplyInput{AnalysisPlan: AnalysisPlan{WriterTemplate: "full"}})
	for _, tc := range []struct {
		name    string
		output  string
		wantErr bool
	}{
		{name: "accepts current labels", output: output},
		{name: "rejects missing tier", output: strings.Replace(output, "**格局评价**", "**其他评价**", 1), wantErr: true},
		{name: "rejects missing boundary", output: strings.Replace(output, "**判断边界**", "**其他说明**", 1), wantErr: true},
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

func TestBaziPresentationStaticCarriesTierStatus(t *testing.T) {
	static := baziStaticSynthesis{TierAssessment: bazidomain.TierAssessment{Status: "provisional"}}
	if got := baziPresentationStatic(static).TierStatus; got != "provisional" {
		t.Fatalf("tier status = %q", got)
	}
}

func TestPresentationProvisionalTierSuppressesFortuneProse(t *testing.T) {
	output := RenderFinalReply(FinalReplyInput{
		AnalysisPlan: AnalysisPlan{NeedLifetimeDayun: true},
		Facts: ChartFacts{
			LiunianGanZhi: "丙午", LiunianTenGod: "比肩",
			DayunPeriods: []DayunPeriod{{Ref: "dayun[0]", Label: "庚寅运（24-33岁）", GanZhi: "庚寅", TenGod: "偏财"}},
		},
		StaticSynthesis: StaticSynthesis{
			TierStatus:     "provisional",
			TierJudgment:   "格局判断暂定",
			MainAxis:       "以七杀格为主轴，食神制杀与杀印相生并见。",
			PatternOutcome: "食神制杀条件不足，食神弱需印星转化。",
		},
		LifetimeSynthesis: LifetimeDayunSynthesis{
			Status: "accepted", Summary: "结构兑现较顺", Trajectory: "smooth_realization",
			PeriodClaims: []LifetimeDayunClaim{{PeriodRef: "dayun[0]", PeriodEffect: "support_use"}},
		},
		DynamicSynthesis: DynamicSynthesis{
			CurrentTrend: "当前大运结构兑现较顺。", LiunianFocus: "吉中带险。", WindowLevel: "扰动年",
		},
	})
	for _, forbidden := range []string{"结构兑现较顺", "吉中带险", "扰动年", "食神弱需印星转化"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("provisional report leaked %q: %s", forbidden, output)
		}
	}
	for _, want := range []string{
		"**庚寅运（24-33岁）｜扶助用神**：庚为偏财；此运有助于发挥本命用神。",
		"**候选主轴**：以七杀格为主轴，食神制杀与杀印相生并见。",
		"候选主轴仍须完成清浊、病药与救应等条件核验；当前独立证据未全，本轮不据此定局。",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("provisional report missing %q: %s", want, output)
		}
	}
	if !strings.Contains(output, "**流年干支**：丙午") {
		t.Fatalf("provisional report lost calculated fact: %s", output)
	}
}

func TestPresentationProvisionalTierKeepsAcceptedLifetimeLabels(t *testing.T) {
	output := RenderFinalReply(FinalReplyInput{
		AnalysisPlan:      AnalysisPlan{NeedLifetimeDayun: true},
		Facts:             ChartFacts{DayunPeriods: []DayunPeriod{{Ref: "dayun[0]", Label: "庚寅运（24-33岁）", GanZhi: "庚寅", TenGod: "偏财"}}},
		StaticSynthesis:   StaticSynthesis{TierStatus: "provisional", TierJudgment: "格局判断暂定"},
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
		TierBasis:       "清浊关系仍需继续核对。",
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
	for _, forbidden := range []string{"**年性**", "**依据**", "**限制**"} {
		if strings.Contains(currentSection, forbidden) {
			t.Fatalf("facts-only dynamic output leaked %q: %s", forbidden, currentSection)
		}
	}
	if !strings.Contains(currentSection, "**流年干支**：丙午") {
		t.Fatalf("facts-only dynamic output missing calculated fact: %s", currentSection)
	}
}
