// This test file belongs to the manager-owned runtime layer.
// It verifies BaZi validation recovery and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"fmt"
	"strings"
	"testing"
)

func TestNormalizeStaticSynthesis_CanonicalizesSynonymEnums(t *testing.T) {
	static := validStaticSynthesisForConsistencyTests()
	static.ClaimStrength = "倾向"
	static.SupportLevel = "有力"
	static.WordingCap = "克制"
	static.AxisLevel = "结构存在"
	static.EffectOnTiaohou = "不利"
	static.EffectOnCoreDisease = "减轻"
	static.EffectOnJiShenDirection = "削弱"
	static.AxisCeiling = "主轴可立"

	got := normalizeStaticSynthesis(static)

	if got.ClaimStrength != "倾向成立" || got.SupportLevel != "得力" || got.WordingCap != "保守" {
		t.Fatalf("static enum normalization failed: %+v", got)
	}
	if got.AxisLevel != "结构可见" || got.EffectOnTiaohou != "冲突" || got.EffectOnCoreDisease != "缓解" || got.EffectOnJiShenDirection != "缓解" || got.AxisCeiling != "可作主轴" {
		t.Fatalf("static axis normalization failed: %+v", got)
	}
}

func TestValidateStaticStrengthAgainstEvidence_RejectsOppositeDirection(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{Yongshen: map[string]any{
			"strength":          "偏弱",
			"strength_evidence": map[string]any{"support_score": float64(6), "pressure_score": float64(10)},
		}},
		StaticSynthesis: baziStaticSynthesis{StrengthBalance: "日主偏强，喜克泄耗。"},
	}
	if err := validateStaticStrengthAgainstEvidence(state); err == nil {
		t.Fatal("expected an opposite strength direction to be rejected")
	}

	recovered := recoverStaticSynthesis(state, state.StaticSynthesis, fmt.Errorf("static strength reverses balance evidence: 偏弱"))
	if recovered.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("recovery source = %q, want facts-only degraded", recovered.Source)
	}
	if got, want := recovered.StrengthBalance, "日主偏弱；扶身证据 6、泄耗克证据 10。"; got != want {
		t.Fatalf("recovered facts = %q, want %q", got, want)
	}
}

func TestValidateStaticAxisAgainstChartFacts_DoesNotHardCodeMethodologyDisputes(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{
			Yongshen: map[string]any{"shi_shen_power": map[string]map[string]float64{
				"七杀": {"gan_count": 0, "zhi_count": 1},
			}},
		},
		StaticSynthesis: baziStaticSynthesis{
			MainAxis: "月劫格，以食神制杀为用。",
		},
	}
	if err := validateStaticAxisAgainstChartFacts(state); err != nil {
		t.Fatalf("methodology disputes must be handled by synthesis/eval, not runtime hard branches: %v", err)
	}
}

func TestRecoverStaticSynthesis_DropsCandidateTextAndReturnsFactsOnly(t *testing.T) {
	state := baziCharterState{Input: baziCharterInput{
		Yongshen: map[string]any{
			"geju_candidate": "月劫格",
			"strength":       "偏弱",
			"strength_evidence": map[string]any{
				"support_score":  float64(6),
				"pressure_score": float64(10),
			},
			"shi_shen_power": map[string]map[string]float64{"七杀": {"gan_count": 0, "zhi_count": 1}},
		},
	}}
	candidate := validStaticSynthesisForConsistencyTests()
	candidate.MainAxis = "月劫格，以食神制杀为用。"
	candidate.PatternOutcome = "丁火坐酉死地，食神制杀主轴成立。"
	candidate.TiaohouAnchor = "先丙后癸。"

	out := recoverStaticSynthesis(state, candidate, fmt.Errorf("static synthesis promotes food-god-controls-killing without visible seven-killing"))
	if out.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("recovery source = %q, want facts-only degraded", out.Source)
	}
	text := strings.Join([]string{out.MainAxis, out.PatternOutcome, out.TiaohouAnchor, out.ReasoningSummary}, "\n")
	if containsAnyText([]string{text}, []string{"食神制杀", "丁火坐酉死地", "先丙后癸"}) {
		t.Fatalf("facts-only recovery must drop candidate judgment text, got %+v", out)
	}
	if !strings.Contains(out.PatternBasis, "月令取格候选：月劫格") {
		t.Fatalf("facts-only recovery must retain tool facts, got %q", out.PatternBasis)
	}
}

func TestValidateDynamicFireBureauFacts_RejectsUndeclaredDayunClaim(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{Dayun: map[string]any{"dayun_analyzed": []map[string]any{{
			"ganZhi": "丙戌", "dayun_chonghe": []map[string]any{},
		}}}},
		DynamicSynthesis: baziDynamicSynthesis{DayunPath: []string{
			"丙戌运与巳、未会成火局，财星极旺。",
		}},
	}
	if err := validateDynamicFireBureauFacts(state); err == nil {
		t.Fatal("expected undeclared fire bureau to fail")
	}

	state.Input.Dayun["dayun_analyzed"] = []map[string]any{{
		"ganZhi": "壬午", "dayun_chonghe": []map[string]any{{"description": "大运参与巳午未会火局"}},
	}}
	state.DynamicSynthesis.DayunPath = []string{"壬午运参与巳午未会火局，关系已计算。"}
	if err := validateDynamicFireBureauFacts(state); err != nil {
		t.Fatalf("expected declared fire bureau to pass, got %v", err)
	}
}

func TestValidateDynamicRelationFacts_DetectsUndeclaredPeriodRelationForSoftAudit(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{Dayun: map[string]any{"dayun_analyzed": []map[string]any{{
			"ganZhi": "丙戌", "dayun_chonghe": []map[string]any{},
		}}}},
		DynamicSynthesis: baziDynamicSynthesis{DayunPath: []string{
			"丙戌运与日支未构成戌未相刑。",
		}},
	}
	if err := validateDynamicRelationFacts(state); err == nil {
		t.Fatal("expected undeclared period relation to fail")
	}

	state.Input.Dayun["dayun_analyzed"] = []map[string]any{{
		"ganZhi": "甲申", "dayun_chonghe": []map[string]any{{"description": "大运申亥害月柱亥"}},
	}}
	state.DynamicSynthesis.DayunPath = []string{"甲申运申亥相害，关系已计算。"}
	if err := validateDynamicRelationFacts(state); err != nil {
		t.Fatalf("expected declared relation to pass, got %v", err)
	}
}

func TestDynamicValidatorErrorsRemainClassifiable(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{Dayun: map[string]any{"dayun_analyzed": []map[string]any{{
			"ganZhi": "丙戌", "dayun_chonghe": []map[string]any{},
		}}}},
		DynamicSynthesis: baziDynamicSynthesis{DayunPath: []string{
			"丙戌运与日支未构成戌未相刑。",
		}},
	}

	validationErr := validateDynamicRelationFacts(state)
	if validationErr == nil {
		t.Fatal("expected undeclared relation to fail dynamic validation")
	}
	failure, ok := baziContractFailureFromError("dynamic_projection", validationErr)
	if !ok {
		t.Fatalf("dynamic validator error must remain classifiable: %v", validationErr)
	}
	if failure.Class != baziContractFailureFactConflict {
		t.Fatalf("failure class = %q, want %q", failure.Class, baziContractFailureFactConflict)
	}
	if failure.RecoveryPolicy != baziRecoveryPolicyHardError {
		t.Fatalf("recovery policy = %q, want %q", failure.RecoveryPolicy, baziRecoveryPolicyHardError)
	}
}

func TestSanitizeDynamicPresentationBoundaries_DoesNotRewriteUnsupportedOutcomeLanguage(t *testing.T) {
	input := baziDynamicSynthesis{
		DayunPath:      []string{"辛巳运称为冲开财库，并说可破财。", "壬午运写成关系触发，易有财务纠纷。"},
		Risks:          []string{"关系触发可能带来意外。", "关系触发仍需核对。"},
		LiunianFocus:   "这一年可能一飞冲天。",
		ReasoningSteps: []string{"该运存在明确破财风险。", "扶抑上大吉。"},
		CurrentTrend:   "当前仅作结构观察。",
	}
	out := sanitizeDynamicPresentationBoundaries(input)
	text := strings.Join(append([]string{out.LiunianFocus}, append(append(out.DayunPath, out.Risks...), out.ReasoningSteps...)...), "\n")
	if !containsAnyText([]string{text}, []string{"冲开财库", "破财", "财务纠纷", "一飞冲天", "大吉"}) {
		t.Fatalf("sanitizer must not rewrite unsupported outcomes; validator owns rejection, got %+v", out)
	}
}

func TestSanitizeDynamicPresentationBoundaries_NormalizesConsistencyFlags(t *testing.T) {
	out := sanitizeDynamicPresentationBoundaries(baziDynamicSynthesis{ConsistencyFlags: []string{"财务纠纷", "承接与扰动并存", "吉中带阻"}})
	if containsAnyText(out.ConsistencyFlags, []string{"财务纠纷", "承接与扰动并存", "吉中带阻"}) {
		t.Fatalf("unsupported consistency flags must be normalized, got %+v", out.ConsistencyFlags)
	}
	for _, want := range []string{dynamicFlagStructureOnly, dynamicFlagMixedConstraint} {
		if !containsString(out.ConsistencyFlags, want) {
			t.Fatalf("expected normalized flag %q, got %+v", want, out.ConsistencyFlags)
		}
	}
	if !containsString(out.FieldAudit, "dynamic_consistency_flags") {
		t.Fatalf("expected field audit to record enum normalization, got %+v", out.FieldAudit)
	}
}

func TestValidateDynamicConsistencyFlags_RejectsUnsupportedFlag(t *testing.T) {
	err := validateDynamicConsistencyFlags([]string{"承接与扰动并存"})
	if err == nil {
		t.Fatal("expected unsupported dynamic consistency flag to fail")
	}
	if !strings.Contains(err.Error(), dynamicFlagStructureOnly) {
		t.Fatalf("error should include allowed flags, got %v", err)
	}
}

func TestValidateDynamicStage_RejectsUnsupportedBoundaryTerms(t *testing.T) {
	state := baziCharterState{DynamicSynthesis: baziDynamicSynthesis{
		CurrentTrend:     "当前承接与扰动并存。",
		ClaimStrength:    "倾向成立",
		SupportLevel:     "有气",
		LimitationLevel:  "明显",
		WordingCap:       "中性",
		DayunPath:        []string{"壬午运：关系触发较强，据此直接断为官非。"},
		LiunianFocus:     "这一年有承接也有扰动。",
		WindowLevel:      "窗口年",
		ReasoningSummary: "只按结构观察。",
		ReasoningSteps:   []string{"先看大运和流年关系。"},
	}}
	if err := validateDynamicStage(state); err == nil {
		t.Fatal("expected unsupported dynamic boundary wording to fail validation")
	}
}

func TestValidateDynamicStage_RejectsInvestmentAdviceWording(t *testing.T) {
	state := baziCharterState{DynamicSynthesis: baziDynamicSynthesis{
		CurrentTrend:     "当前仅作结构观察。",
		ClaimStrength:    "倾向成立",
		SupportLevel:     "有气",
		LimitationLevel:  "明显",
		WordingCap:       "中性",
		DayunPath:        []string{"己巳运只能说明结构触发，不能推断投资建议。"},
		LiunianFocus:     "这一年仅作结构观察。",
		WindowLevel:      "窗口年",
		ReasoningSummary: "只按结构观察。",
		ReasoningSteps:   []string{"先看大运和流年关系。"},
	}}
	if err := validateDynamicStage(state); err == nil {
		t.Fatal("expected investment advice wording to fail validation")
	}
}

func TestValidateDynamicStage_AllowsOrdinaryBaziInterpretiveLanguage(t *testing.T) {
	state := baziCharterState{DynamicSynthesis: baziDynamicSynthesis{
		CurrentTrend:     "当前承接与扰动并存。",
		ClaimStrength:    "倾向成立",
		SupportLevel:     "有气",
		LimitationLevel:  "明显",
		WordingCap:       "中性",
		DayunPath:        []string{"丙戌运：戌为火库，燥土可调候暖局，才华初显但根基未稳。"},
		LiunianFocus:     "这一年有承接也有扰动。",
		WindowLevel:      "窗口年",
		ReasoningSummary: "只按结构观察。",
		ReasoningSteps:   []string{"先看大运和流年关系。"},
	}}
	if err := validateDynamicStage(state); err != nil {
		t.Fatalf("ordinary bazi interpretive language must not be blocked by lexical patching: %v", err)
	}
}

func TestRecoverStaticSynthesisAfterRetry_RecoversEvidenceOverclaimOnly(t *testing.T) {
	chartState := baziCharterState{
		EvidenceQuality: baziEvidenceQuality{
			RequiredTopics: []string{"geju", "bingyao"},
			CoveredTopics:  []string{"geju"},
			MissingTopics:  []string{"bingyao"},
		},
	}
	candidate := validStaticSynthesisForConsistencyTests()
	candidate.TierJudgment = "中上"
	candidate.TierBasis = "已经具备拔高到上等的条件。"
	cause := baziContractAuditError("static", baziContractAuditFinding{
		Code:    "evidence_topic_overclaim",
		Field:   "static.tier",
		Excerpt: "中上",
		Reason:  "static.tier 已声明 withheld_missing_evidence 但继续硬断层次",
	})

	out, err := recoverStaticSynthesisAfterRetry(chartState, candidate, cause)
	if err != nil {
		t.Fatalf("expected evidence overclaim to recover as facts-only, got %v", err)
	}
	if out.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("source = %q, want facts-only degraded", out.Source)
	}
	if !containsString(out.FieldAudit, "contract_failure_class:"+baziContractFailureEvidenceOverclaim) ||
		!containsString(out.FieldAudit, "recovery_policy:"+baziRecoveryPolicyStaticFactsOnly) {
		t.Fatalf("facts-only recovery must record classification, got %v", out.FieldAudit)
	}
	text := strings.Join([]string{out.MainAxis, out.PatternOutcome, out.TierBasis, out.ReasoningSummary}, "\n")
	if containsAnyText([]string{text}, []string{"中上", "可以拔高", "拔高到上等"}) {
		t.Fatalf("facts-only recovery must discard invalid candidate text, got %+v", out)
	}
}

func TestRecoverStaticSynthesisAfterRetry_RejectsMethodContractFailure(t *testing.T) {
	chartState := baziCharterState{}
	candidate := validStaticSynthesisForConsistencyTests()
	cause := baziContractAuditError("static", baziContractAuditFinding{
		Code:   "hidden_axis_uncompared",
		Field:  "static.pattern_adjudication",
		Reason: "hidden axis selected without complete comparison",
	})

	if _, err := recoverStaticSynthesisAfterRetry(chartState, candidate, cause); err == nil {
		t.Fatal("method-contract failures must not silently become facts-only")
	}
}

func TestBuildStaticSynthesisFeedback_RepeatsTierEvidenceBoundary(t *testing.T) {
	chartState := baziCharterState{
		EvidenceQuality: baziEvidenceQuality{
			RequiredTopics: []string{"geju", "bingyao"},
			CoveredTopics:  []string{"geju"},
			MissingTopics:  []string{"bingyao"},
		},
	}
	cause := baziContractAuditError("static", baziContractAuditFinding{
		Code:   "evidence_topic_overclaim",
		Field:  "static.tier",
		Reason: "tier overclaim",
	})

	feedback := buildStaticSynthesisFeedback(chartState, validStaticSynthesisForConsistencyTests(), cause)
	for _, want := range []string{
		"missing_topics=bingyao",
		"static.tier",
		"tier_judgment",
		"tier_basis",
		"assertions[].verdict",
		"命格层次中等（保守定位）",
		"暂不定级",
		"不得输出“中上”",
	} {
		if !strings.Contains(feedback, want) {
			t.Fatalf("static feedback missing %q in %s", want, feedback)
		}
	}
}

func TestRunStaticSynthesisWithFeedback_AcceptsDisplayOnlyPartialAfterRetry(t *testing.T) {
	executor := &Executor{}
	chartState := baziCharterState{}
	first := validStaticSynthesisForConsistencyTests()
	first.AxisLevel = "可以拔高"
	first.AxisCeiling = "结构信号"
	first.PatternOutcome = "这条路线已经贵格已成，可以拔高。"
	second := validStaticSynthesisForConsistencyTests()
	second.TierBasis = ""

	calls := 0
	out, err := executor.runStaticSynthesisWithFeedback(chartState, func(payload map[string]any) (baziStaticSynthesis, error) {
		calls++
		if calls == 1 {
			return first, nil
		}
		return second, nil
	})
	if err != nil {
		t.Fatalf("display-only missing field should be accepted as partial after retry: %v", err)
	}
	if calls != 2 {
		t.Fatalf("static synthesis calls = %d, want 2", calls)
	}
	if out.Source != baziSynthesisSourceModelPartial {
		t.Fatalf("source = %q, want model_partial", out.Source)
	}
	if !containsAnyText(out.FieldAudit, []string{"static_partial_omitted:"}) {
		t.Fatalf("expected static partial field audit, got %+v", out.FieldAudit)
	}
	if strings.TrimSpace(out.TierBasis) != "" {
		t.Fatalf("partial output should keep missing display field omitted, got %q", out.TierBasis)
	}
	if strings.TrimSpace(out.MainAxis) == "" || strings.TrimSpace(out.PatternOutcome) == "" {
		t.Fatalf("partial output must retain core judgments, got %+v", out)
	}
}

func TestAcceptPartialDynamicSynthesisAfterRetry_AcceptsDisplayOnlyMissingField(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{Dayun: map[string]any{
			"dayun_analyzed": []map[string]any{{"ganZhi": "甲午"}},
		}},
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
	}
	candidate := baziDynamicSynthesis{
		CurrentTrend:     "当前只作结构观察，承接中仍有扰动。",
		ClaimStrength:    "倾向成立",
		SupportLevel:     "有气",
		LimitationLevel:  "明显",
		WordingCap:       "中性",
		ConsistencyFlags: []string{dynamicFlagStructureOnly},
		DayunPath:        []string{"甲午运：只按已计算关系作结构观察。"},
		DayunJudgments: []baziDayunJudgment{{
			GanZhi:         "甲午",
			Trend:          "先看结构，不下吉凶",
			Interpretation: "只按已计算关系作结构观察。",
			OutcomeDomains: []string{"structure"},
		}},
		LiunianFocus:     "流年只观察结构触发，不展开具体应事。",
		WindowLevel:      "窗口年",
		ReasoningSummary: "按大运与流年结构观察。",
		OutcomeDomains:   []string{"structure"},
	}

	out, ok := acceptPartialDynamicSynthesisAfterRetry(state, candidate, fmt.Errorf("missing dynamic synthesis reasoning steps"))
	if !ok {
		t.Fatalf("display-only dynamic omission should be accepted as partial")
	}
	if out.Source != baziSynthesisSourceModelPartial {
		t.Fatalf("source = %q, want model_partial", out.Source)
	}
	if !containsAnyText(out.FieldAudit, []string{"dynamic_partial_omitted:"}) {
		t.Fatalf("expected dynamic partial field audit, got %+v", out.FieldAudit)
	}
	if len(out.ReasoningSteps) != 0 {
		t.Fatalf("partial output should keep missing reasoning steps omitted, got %+v", out.ReasoningSteps)
	}
}

func TestAcceptPartialDynamicSynthesisAfterRetry_RejectsFatalScopeFailure(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		Input: baziCharterInput{BaziResult: map[string]any{
			"birthSolar": "2025-11-11 00:15",
		}},
	}
	candidate := baziDynamicSynthesis{
		CurrentTrend:     "未来运势会进入事业升迁和财富兑现阶段。",
		ClaimStrength:    "倾向成立",
		SupportLevel:     "有气",
		LimitationLevel:  "明显",
		WordingCap:       "中性",
		ConsistencyFlags: []string{dynamicFlagStructureOnly},
		DayunPath:        []string{"甲午运：未来事业升迁明显。"},
		LiunianFocus:     "流年只观察结构触发。",
		WindowLevel:      "窗口年",
		ReasoningSummary: "按大运与流年结构观察。",
	}

	out, ok := acceptPartialDynamicSynthesisAfterRetry(state, candidate, fmt.Errorf("missing dynamic synthesis reasoning steps"))
	if ok {
		t.Fatalf("fatal age-scope failure must not be accepted as partial: %+v", out)
	}
}

func TestRecoverDynamicSynthesis_ReturnsFactsOnlyAndDropsCandidateText(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		Input: baziCharterInput{Dayun: map[string]any{"dayun_analyzed": []map[string]any{{
			"ganZhi": "甲午", "tenGod": "七杀", "dayun_chonghe": []map[string]any{{"description": "大运午午自刑时柱午"}},
		}}}},
	}
	candidate := baziDynamicSynthesis{
		CurrentTrend:     "从这里开始会一路顺行，明显起飞。",
		ClaimStrength:    "明确",
		SupportLevel:     "强",
		LimitationLevel:  "明显",
		WordingCap:       "明确",
		ConsistencyFlags: []string{"机会伴随强变动"},
		DayunPath:        []string{"当前运势可能引发官非。"},
		LiunianFocus:     "这是关键翻身年，可以一飞冲天。",
		WindowLevel:      "机会年",
		ReasoningSummary: "整体会彻底起势。",
		ReasoningSteps:   []string{"先看机会，再直接下拔高结论。"},
	}

	out := recoverDynamicSynthesis(state, candidate, fmt.Errorf("dynamic synthesis overstates unsupported legal or medical outcome"))
	state.DynamicSynthesis = out

	if out.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("dynamic recovery source = %q, want facts-only degraded", out.Source)
	}
	if err := validateDynamicStage(state); err != nil {
		t.Fatalf("expected facts-only dynamic synthesis to pass stage validation, got %v", err)
	}
	textParts := append([]string{out.CurrentTrend, out.LiunianFocus, out.ReasoningSummary}, out.DayunPath...)
	textParts = append(textParts, out.TriggerSignals...)
	textParts = append(textParts, out.ReasoningSteps...)
	text := strings.Join(textParts, "\n")
	if containsAnyText([]string{text}, []string{"一路顺", "明显起飞", "官非", "一飞冲天", "彻底起势"}) {
		t.Fatalf("facts-only dynamic recovery must drop candidate judgment text, got %+v", out)
	}
	if !strings.Contains(text, "甲午") || !strings.Contains(text, "午午自刑") {
		t.Fatalf("facts-only dynamic recovery must retain computed dayun facts, got %+v", out)
	}
}

func TestRecoverDynamicSynthesisAfterRetry_ReplacesUnauthorizedAdultOutcome(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		Input: baziCharterInput{
			BaziResult: map[string]any{"birthday": "1994-01-21 20:15"},
			Dayun: map[string]any{"dayun_analyzed": []map[string]any{{
				"ganZhi": "戊辰", "tenGod": "伤官", "dayun_chonghe": []map[string]any{{"description": "大运辰戌冲时柱戌"}},
			}}},
			Liunian: map[string]any{
				"liunian_year":     2026,
				"liunian_ganzhi":   "丙午",
				"liunian_shi_shen": "劫财",
				"liunian_chonghe":  []map[string]any{{"description": "流年丑午害月柱丑"}},
			},
		},
	}
	candidate := baziDynamicSynthesis{
		CurrentTrend:      "当前财务机会明显，可考虑投资与收入兑现。",
		ClaimStrength:     "倾向成立",
		SupportLevel:      "有气",
		LimitationLevel:   "明显",
		WordingCap:        "中性",
		ConsistencyFlags:  []string{dynamicFlagStructureOnly},
		DayunPath:         []string{"戊辰运：财务建议更积极，适合投资。"},
		CurrentDayunIndex: 0,
		LiunianFocus:      "丙午流年利收入。",
		WindowLevel:       "窗口年",
		OutcomeDomains:    []string{"structure", "user_requested_authorized_domain"},
		ReasoningSummary:  "候选越过了用户未授权的财务领域。",
		ReasoningSteps:    []string{"先看财务机会。"},
	}

	out, err := recoverDynamicSynthesisAfterRetry(state, candidate, baziContractAuditError("dynamic", baziContractAuditFinding{
		Code:           "outcome_domain_mismatch",
		Field:          "dynamic.dayun_judgments[0].interpretation",
		DetectedDomain: "finance",
		Reason:         "未授权财务领域",
	}))
	if err != nil {
		t.Fatalf("expected dynamic retry recovery to be valid, got %v", err)
	}
	if out.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("dynamic retry recovery source = %q, want facts-only degraded", out.Source)
	}
	state.DynamicSynthesis = out
	if err := validateDynamicSynthesisResult(state, out); err != nil {
		t.Fatalf("recovered dynamic synthesis must pass validation, got %v", err)
	}
	textParts := append([]string{out.CurrentTrend, out.LiunianFocus, out.ReasoningSummary}, out.DayunPath...)
	textParts = append(textParts, out.TriggerSignals...)
	textParts = append(textParts, out.ReasoningSteps...)
	text := strings.Join(textParts, "\n")
	if containsAnyText([]string{text}, []string{"财务机会", "投资", "收入兑现", "利收入"}) {
		t.Fatalf("retry recovery must discard unauthorized candidate text, got %+v", out)
	}
	if !strings.Contains(text, "戊辰") || !strings.Contains(text, "辰戌冲") || !strings.Contains(text, "丙午") {
		t.Fatalf("retry recovery must retain computed dynamic facts, got %+v", out)
	}
}

func TestRecoverDynamicSynthesisAfterRetry_RejectsBranchTenGodConflict(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		Input: baziCharterInput{Dayun: map[string]any{"dayun_analyzed": []map[string]any{{
			"ganZhi": "戊辰", "tenGod": "伤官",
		}}}},
	}
	candidate := baziDynamicSynthesis{
		CurrentTrend:     "仅作结构观察。",
		ClaimStrength:    "保守判断",
		SupportLevel:     "出现",
		LimitationLevel:  "明显",
		WordingCap:       "保守",
		ConsistencyFlags: []string{dynamicFlagStructureOnly},
		DayunPath:        []string{"戊辰运：仅作结构观察。"},
		LiunianFocus:     "仅作结构观察。",
		WindowLevel:      "扰动年",
		ReasoningSummary: "仅作结构观察。",
		ReasoningSteps:   []string{"先看戊辰运。"},
	}
	cause := baziContractAuditError("dynamic", baziContractAuditFinding{
		Code:   "branch_tengod_conflict",
		Field:  "dynamic.dayun_judgments[0].evidence",
		Reason: "地支本气十神与输入不一致",
	})

	if _, err := recoverDynamicSynthesisAfterRetry(state, candidate, cause); err == nil {
		t.Fatal("branch ten-god conflicts must not default to facts-only recovery")
	}
}
