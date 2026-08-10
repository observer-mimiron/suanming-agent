package runtime

import (
	"strings"
	"testing"
)

func TestValidateFinalWriterOutputAcceptsLifetimeHeadings(t *testing.T) {
	plan := baziAnalysisPlan{WriterTemplate: "full", NeedLifetimeDayun: true}
	output := "## 强弱视角\n**结论：日主中和。**\n\n## 调候视角\n**结论：调候待核。**\n\n## 格局视角\n**结论：主轴可用。**\n- **规则口径**：已声明\n- **依据**：月令与透干\n\n### 命格层次\n- **判读口径**：按结构评价。\n**结论：第5级。**\n- **判定依据**：旧层次审计依据\n> 断语所限：仍有边界。\n\n## 全程运路\n**结论：起伏中有兑现。**\n\n## 当前应期\n### 当前大运\n**结论：当前承接。**\n\n### 流年应期\n**结论：本年触发。**\n\n## 总览结论\n### 本命总断\n**结论：本命结构可用。**\n- **命格层次**：第5级\n- **主要限制**：仍有边界。\n\n### 全程走势\n**结论：起伏中有兑现。**\n\n### 当前阶段\n**结论：当前承接。**"
	state := baziCharterState{AnalysisPlan: plan, StaticSynthesis: baziStaticSynthesis{TierJudgment: "第5级", TierBasis: "旧层次审计依据"}}
	if err := validateFinalWriterOutput(plan, state, output); err != nil {
		t.Fatalf("lifetime headings rejected: %v", err)
	}
	rendered := renderFullTemplate(state)
	if !strings.Contains(rendered, "### 命格层次\n- **判读口径**") || !strings.Contains(rendered, "**结论：第5级**") || !strings.Contains(rendered, "**判定依据**：旧层次审计依据") {
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
			map[string]any{"ganZhi": "戊子", "tenGod": "比肩", "startAge": 10, "endAge": 19},
			map[string]any{"ganZhi": "甲午", "tenGod": "七杀", "startAge": 30, "endAge": 39},
		}}},
		LifetimeSynthesis: baziLifetimeDayunSynthesis{Status: "accepted", PeriodClaims: []baziLifetimeDayunClaim{
			{PeriodRef: "dayun[0]", PeriodEffect: "carry_balance", Verdict: "己土劫财，承托与耗损并存。"},
			{PeriodRef: "dayun[1]", PeriodEffect: "support_use", Verdict: "以扶助用神的结构作用为主。"},
		}},
	}
	var b strings.Builder
	writeLifetimeDayunGroups(&b, state)
	output := b.String()
	for _, want := range []string{"**戊子运（10-19岁）**", "**定位**：平衡承接；戊为比肩", "**甲午运（30-39岁）**"} {
		if !strings.Contains(output, want) {
			t.Fatalf("lifetime output missing %q: %s", want, output)
		}
	}
	for _, forbidden := range []string{"早期运程", "中期运程", "后期运程", "工具事实"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("lifetime output must not expose legacy grouping text %q: %s", forbidden, output)
		}
	}
	if strings.Contains(output, "己土劫财") {
		t.Fatalf("lifetime model prose must not override deterministic stem-ten-god facts: %s", output)
	}
	if strings.Contains(output, "dayun[") {
		t.Fatalf("lifetime group output leaked internal ref: %s", output)
	}
}

func TestRenderFullTemplateDisplaysRetrievedClassicalReferences(t *testing.T) {
	out := renderFullTemplate(baziCharterState{
		StaticSynthesis: baziStaticSynthesis{MainAxis: "主轴可用。"},
		EvidenceBundle: baziEvidenceBundle{Citations: []baziCitation{
			{Classic: "穷通宝鉴", Quotes: []string{"戊土酉月，取火以温燥。"}},
			{Classic: "子平真诠", Quotes: []string{"…；此为变化之由也。", "九、论用神成败救应 > ⭐⭐⭐⭐⭐ | 清·沈孝瞻"}},
			{Classic: "无引文书", Quotes: nil},
		}},
	})
	for _, want := range []string{"### 古籍参照", "**《穷通宝鉴》**：戊土酉月，取火以温燥。"} {
		if !strings.Contains(out, want) {
			t.Fatalf("classical reference missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "无引文书") {
		t.Fatalf("citation without a quote must stay hidden:\n%s", out)
	}
	for _, forbidden := range []string{"…；此为变化之由也。", "⭐⭐", "清·沈孝瞻", "九、论用神成败救应 >"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("citation display leaked invalid retrieval text %q:\n%s", forbidden, out)
		}
	}
}

func TestFinalOverviewNormalizesTierPrefixAndDirectionPunctuation(t *testing.T) {
	out := renderFullTemplate(baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis:     "主轴可用。",
			TierJudgment: "命格基础层次：第5级。",
			Usage: baziUsageLayers{
				Fuyi:    "扶抑方向明确。",
				Tiaohou: "调候方向待验。",
			},
		},
	})
	if strings.Contains(out, "命格层次**：命格基础层次：") {
		t.Fatalf("final overview duplicated tier prefix:\n%s", out)
	}
	if strings.Contains(out, "。；") {
		t.Fatalf("final overview has malformed direction punctuation:\n%s", out)
	}
}
