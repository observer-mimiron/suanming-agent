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
	if !strings.Contains(out, "未启用运行时规则 profile") {
		t.Fatalf("renderer must expose that no runtime profile is selected: %s", out)
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
	for _, required := range []string{static.MainAxis, static.TierJudgment, "## 大运验证", "甲午运", "动态综合未通过，本轮仅展示已计算的大运事实"} {
		if !strings.Contains(out, required) {
			t.Fatalf("mixed output missing %q:\n%s", required, out)
		}
	}
	for _, forbidden := range []string{"dynamic synthesis uses undeclared branch relation", "fact_ref_missing", "## 命盘事实"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("mixed output leaked internal state %q:\n%s", forbidden, out)
		}
	}
	if err := validateFinalWriterOutput(baziAnalysisPlan{WriterTemplate: "full"}, state, out); err != nil {
		t.Fatalf("mixed output must satisfy final contract: %v", err)
	}
}
