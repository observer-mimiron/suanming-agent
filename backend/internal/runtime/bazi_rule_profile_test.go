// This test file belongs to the manager-owned runtime layer.
// It verifies legacy BaZi rule profile coverage and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"strings"
	"testing"
)

func TestBuildCoreChartView_AllowsNoSelectedRuleProfile(t *testing.T) {
	view := buildCoreChartView(baziCharterInput{})
	profile, ok := view["selected_rule_profile"].(baziRuleProfile)
	if !ok {
		t.Fatalf("expected selected_rule_profile slot to remain present, got %#v", view["selected_rule_profile"])
	}
	if profile.ID != "" || len(profile.Claims) != 0 || len(profile.Verdicts) != 0 {
		t.Fatalf("runtime must not inject a default rule profile, got %#v", profile)
	}
}

func TestBuildCoreChartView_DropsTiaohouImplementationPlaceholder(t *testing.T) {
	view := buildCoreChartView(baziCharterInput{Yongshen: map[string]any{
		"tiao_hou":              "待 qiongtong_tiaohou_v1 规则表实现",
		"seasonal_tiaohou_hint": "冬月丁火，寒湿需火暖局",
	}})
	if _, ok := view["tiao_hou"]; ok {
		t.Fatalf("core chart must not expose tiaohou implementation placeholder: %#v", view["tiao_hou"])
	}
	if view["seasonal_tiaohou_hint"] == "" {
		t.Fatalf("core chart should keep deterministic seasonal tiaohou hint")
	}
}

func TestRenderer_DoesNotDeriveAxisFromKeywords(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis:         "伤官配印仅作结构候选，仍待模型综合裁断。",
			ReasoningSummary: "上游裁断认为条件尚未齐备。",
			CounterEvidence:  "印星承接条件仍需核验。",
			PatternOutcome:   "方向受限。",
			TierJudgment:     "待裁定",
			TierBasis:        "尚无完整规则表。",
		},
	}
	out := renderBaziFinalReply(baziAnalysisPlan{}, state, "")
	if strings.Contains(out, "以伤官配印为主轴") {
		t.Fatalf("renderer must not promote a keyword to an axis: %s", out)
	}
	if !strings.Contains(out, "未选择专门规则口径") {
		t.Fatalf("renderer must expose that no specialized rule profile is selected: %s", out)
	}
	if strings.Contains(out, "runtime") || strings.Contains(out, "运行时规则 profile") {
		t.Fatalf("renderer must not leak implementation profile wording: %s", out)
	}
}

func TestRenderer_OnlyFormatsUpstreamVerdicts(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis: "上游明确给出的主轴句。",
		},
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:     "当前属于吉中有阻。",
			ConsistencyFlags: []string{"吉中有阻", "机会伴随强变动"},
		},
	}

	full := renderBaziFinalReply(baziAnalysisPlan{WriterTemplate: "full"}, state, "")
	topic := renderBaziFinalReply(baziAnalysisPlan{WriterTemplate: "topic", TopicMode: "explain_term"}, state, "解释一下")
	for _, output := range []string{full, topic} {
		for _, forbidden := range []string{
			"把压力化成可用之力",
			"先按月令本气与透干关系",
			"整盘升格",
			"先做自己能掌控的决定",
		} {
			if strings.Contains(output, forbidden) {
				t.Fatalf("renderer must not derive %q from upstream keywords:\n%s", forbidden, output)
			}
		}
	}
	if !strings.Contains(topic, state.StaticSynthesis.MainAxis) {
		t.Fatalf("renderer must display the upstream verdict verbatim:\n%s", topic)
	}
	if strings.Contains(full, "上游未提供") || strings.Contains(topic, "上游未提供") {
		t.Fatalf("renderer must not leak internal missing-field placeholders:\nfull=%s\ntopic=%s", full, topic)
	}
}

func TestRenderer_RendersEachDayunAsAnIndependentAnalysisBlock(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis: "上游给出静态主轴。",
		},
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend: "当前甲午运（30-39岁）：有转机，也有牵制。",
			DayunPath: []string{
				"### 丙申运（10-19岁）：有助力但不纯顺\n**解读**：扶抑两侧同时出现作用，不能按单边顺逆理解。\n- 扶抑面：丙火为承托；申金为牵制",
				"### 甲午运（30-39岁）：承托与扰动并见\n**解读**：承托条件存在，同时关系触发提示过程会有变化与反复。\n- 扶抑面：甲木为压力；午火为承托",
			},
		},
	}

	out := renderBaziFinalReply(baziAnalysisPlan{WriterTemplate: "full"}, state, "")
	for _, required := range []string{"### 丙申运（10-19岁）：有助力但不纯顺", "### 甲午运（30-39岁）：承托与扰动并见", "**解读**"} {
		if !strings.Contains(out, required) {
			t.Fatalf("dayun analysis missing %q:\n%s", required, out)
		}
	}
	if strings.Contains(out, "- ### 丙申") {
		t.Fatalf("dayun analysis must not be flattened into a nested list:\n%s", out)
	}
}

func TestRenderer_DayunHeadingsIncludeCalculatedDateBoundaries(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{
			Dayun: map[string]any{"dayun_analyzed": []map[string]any{
				{
					"ganZhi": "丙戌", "startAge": 3, "endAge": 12,
					"startAt": "2027-01-11 00:15:00", "endAtExclusive": "2037-01-11 00:15:00",
				},
				{
					"ganZhi": "乙酉", "startAge": 13, "endAge": 22,
					"startAt": "2037-01-11 00:15:00", "endAtExclusive": "2047-01-11 00:15:00",
				},
			}},
		},
		StaticSynthesis: baziStaticSynthesis{
			MainAxis:     "月令偏印格为分析入口；组合只列候选，暂不定成格。",
			TierJudgment: "中等",
		},
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend: "当前尚处童限，未交正式大运。",
			DayunPath: []string{
				"### 丙戌：偏顺\n**解读**：结构观察。",
				"### 乙酉：利阻并见\n**解读**：结构观察。",
			},
		},
	}

	out := renderBaziFinalReply(baziAnalysisPlan{WriterTemplate: "full"}, state, "")
	for _, required := range []string{
		"### 丙戌运（3-12岁；2027-01-11 00:15至2037-01-11 00:15前）：偏顺",
		"### 乙酉运（13-22岁；2037-01-11 00:15至2047-01-11 00:15前）：利阻并见",
	} {
		if !strings.Contains(out, required) {
			t.Fatalf("dayun heading missing calculated boundary %q:\n%s", required, out)
		}
	}
}

func TestRenderer_DynamicDegradationKeepsValidStaticReading(t *testing.T) {
	static := validStaticSynthesisForConsistencyTests()
	static.Source = "model"
	state := baziCharterState{
		Input: baziCharterInput{
			Dayun: map[string]any{"dayun_analyzed": []map[string]any{{
				"ganZhi": "甲午", "tenGod": "七杀", "startAge": 30, "endAge": 39,
				"dayun_chonghe": []map[string]any{{"description": "大运午午自刑时柱午"}},
			}}},
			Liunian: map[string]any{
				"liunian_ganzhi": "丙午", "liunian_shi_shen": "偏印",
				"liunian_chonghe": []map[string]any{{"description": "流年午午自刑时柱午"}},
			},
		},
		StaticSynthesis: static,
	}
	state.DynamicSynthesis = buildFactsOnlyDynamicSynthesis(state.Input, static, "dynamic synthesis uses undeclared branch relation")

	out := renderBaziFinalReply(baziAnalysisPlan{WriterTemplate: "full"}, state, "")
	for _, required := range []string{static.MainAxis, static.TierJudgment, "## 大运验证", "甲午运", "受授权边界限制，本轮只展示可复算大运事实"} {
		if !strings.Contains(out, required) {
			t.Fatalf("mixed output missing %q:\n%s", required, out)
		}
	}
	for _, forbidden := range []string{"dynamic synthesis uses undeclared branch relation", "fact_ref_missing", "## 命盘事实", "动态综合未通过", "runtime"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("mixed output leaked internal state %q:\n%s", forbidden, out)
		}
	}
	if err := validateFinalWriterOutput(baziAnalysisPlan{WriterTemplate: "full"}, state, out); err != nil {
		t.Fatalf("mixed output must satisfy final contract: %v", err)
	}
}

func TestRenderer_MinorDynamicDegradationShowsConciseGrowthFacts(t *testing.T) {
	static := validStaticSynthesisForConsistencyTests()
	static.Source = "model"
	state := baziCharterState{
		Input: baziCharterInput{
			BaziResult: map[string]any{"birthday": "2025-11-10 23:53:00"},
			Dayun: map[string]any{"dayun_analyzed": []map[string]any{
				{"ganZhi": "丙戌", "tenGod": "食神", "startAge": 3, "endAge": 12, "startAt": "2027-01-11 00:15:00", "endAtExclusive": "2037-01-11 00:15:00"},
				{"ganZhi": "乙酉", "tenGod": "劫财", "startAge": 13, "endAge": 22, "startAt": "2037-01-11 00:15:00", "endAtExclusive": "2047-01-11 00:15:00"},
				{"ganZhi": "甲申", "tenGod": "比肩", "startAge": 23, "endAge": 32, "startAt": "2047-01-11 00:15:00", "endAtExclusive": "2057-01-11 00:15:00"},
				{"ganZhi": "辛巳", "tenGod": "正官", "startAge": 53, "endAge": 62, "startAt": "2077-01-11 00:15:00", "endAtExclusive": "2087-01-11 00:15:00"},
			}},
			Liunian: map[string]any{
				"liunian_year":      2026,
				"liunian_target_at": "2026-08-03 12:00:00",
				"liunian_ganzhi":    "丙午",
				"liunian_shi_shen":  "食神",
				"current_dayun":     map[string]any{},
			},
		},
		StaticSynthesis: static,
	}
	state.DynamicSynthesis = buildFactsOnlyDynamicSynthesis(state.Input, static, "unsupported_concrete_outcome")

	out := renderBaziFinalReply(baziAnalysisPlan{WriterTemplate: "full"}, state, "")
	for _, required := range []string{"成长节奏", "当前阶段", "大运事实节选", "丙戌运", "乙酉运", "甲申运", "流年只展示干支"} {
		if !strings.Contains(out, required) {
			t.Fatalf("minor facts-only output missing %q:\n%s", required, out)
		}
	}
	for _, forbidden := range []string{"动态综合未通过", "runtime", "辛巳运", "事业突破", "财富增长", "健康风险"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("minor facts-only output leaked %q:\n%s", forbidden, out)
		}
	}
	if err := validateFinalWriterOutput(baziAnalysisPlan{WriterTemplate: "full"}, state, out); err != nil {
		t.Fatalf("minor facts-only output must satisfy final contract: %v", err)
	}
}

func TestRenderer_BoundedTierAppearsInOverviewAndJudgment(t *testing.T) {
	static := validStaticSynthesisForConsistencyTests()
	static.Source = "model"
	static.MainAxis = "偏印格候选成立，但成败与清浊待规则裁断，暂以偏印为结构主轴。"
	static.CounterEvidence = "调候证据不足，无法判断寒暖燥湿影响。"
	static.TiaohouAnchor = "冬令甲木调候需求未定，证据不足，暂不裁断。"
	static.TiaohouConstraint = "不能据此判断寒暖燥湿对格局的影响。"
	static.Usage.Pattern = "偏印格候选，清浊成败待规则裁断，暂以结构观察为主。"
	static.TierJudgment = "命格层次中等（保守定位）"
	static.TierBasis = "按保守定级标准：病药救应链条尚未完全闭合，层次封顶为中等，不上推中上或上等。"
	static.ReasoningSummary = static.MainAxis + "；" + static.TierBasis
	state := baziCharterState{
		StaticSynthesis: static,
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend: "大运吉凶待规则裁断，当前仅报告十神与地支关系。",
			DayunPath:    []string{"### 甲午运：仅作结构观察\n**解读**：本步大运未被模型列为重点窗口。"},
		},
	}

	out := renderBaziFinalReply(baziAnalysisPlan{WriterTemplate: "full"}, state, "")
	if strings.Contains(out, "暂不定级") || strings.Contains(out, "证据不足") {
		t.Fatalf("bounded tier output must not expose no-tier wording:\n%s", out)
	}
	if count := strings.Count(out, "命格层次中等（保守定位）"); count != 2 {
		t.Fatalf("bounded tier verdict should appear in overview and 综合判定, got %d:\n%s", count, out)
	}
	for _, forbidden := range []string{"调候规则未实现", "待规则裁断", "仅作结构观察"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("display should normalize repeated boundary phrase %q:\n%s", forbidden, out)
		}
	}
	if !strings.Contains(sectionContent(out, "## 总览结论", "## 强弱视角"), "命格层次中等（保守定位）") {
		t.Fatalf("overview must show the bounded tier verdict:\n%s", out)
	}
	if !strings.Contains(sectionContent(out, "## 综合判定", "## 命格总结"), "命格层次中等（保守定位）") {
		t.Fatalf("display must keep the canonical bounded tier at 综合判定:\n%s", out)
	}
}
