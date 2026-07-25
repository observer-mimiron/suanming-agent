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

	if got.ClaimStrength != "倾向成立" {
		t.Fatalf("claim_strength = %q, want 倾向成立", got.ClaimStrength)
	}
	if got.SupportLevel != "得力" {
		t.Fatalf("support_level = %q, want 得力", got.SupportLevel)
	}
	if got.WordingCap != "保守" {
		t.Fatalf("wording_cap = %q, want 保守", got.WordingCap)
	}
	if got.AxisLevel != "结构可见" {
		t.Fatalf("axis_level = %q, want 结构可见", got.AxisLevel)
	}
	if got.EffectOnTiaohou != "冲突" {
		t.Fatalf("effect_on_tiaohou = %q, want 冲突", got.EffectOnTiaohou)
	}
	if got.EffectOnCoreDisease != "缓解" {
		t.Fatalf("effect_on_core_disease = %q, want 缓解", got.EffectOnCoreDisease)
	}
	if got.EffectOnJiShenDirection != "缓解" {
		t.Fatalf("effect_on_jishen_direction = %q, want 缓解", got.EffectOnJiShenDirection)
	}
	if got.AxisCeiling != "可作主轴" {
		t.Fatalf("axis_ceiling = %q, want 可作主轴", got.AxisCeiling)
	}
}

func TestRunStaticSynthesisWithFeedback_DegradesAfterRepeatedValidationFailure(t *testing.T) {
	executor := &Executor{}
	chartState := baziCharterState{}
	invalid := validStaticSynthesisForConsistencyTests()
	invalid.AxisLevel = "可以拔高"
	invalid.AxisCeiling = "结构信号"
	invalid.PatternOutcome = "这条路线已经贵格已成，可以拔高。"
	invalid.TierBasis = "已经具备拔高到上等的条件。"

	calls := 0
	out, err := executor.runStaticSynthesisWithFeedback(chartState, func(payload map[string]any) (baziStaticSynthesis, error) {
		calls++
		return invalid, nil
	})
	if err != nil {
		t.Fatalf("expected graceful degradation instead of error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("static synthesis calls = %d, want 2", calls)
	}
	if err := validateStaticSynthesisResult(chartState, out); err != nil {
		t.Fatalf("expected degraded static synthesis to validate, got %v", err)
	}
	if strings.Contains(out.PatternOutcome, "贵格已成") {
		t.Fatalf("degraded pattern_outcome should drop escalation wording, got %q", out.PatternOutcome)
	}
	if !containsAnyText([]string{out.PatternOutcome, out.CounterEvidence, out.TierBasis}, []string{"受限", "不宜拔高", "力度受限"}) {
		t.Fatalf("degraded static synthesis must preserve limitation wording, got %+v", out)
	}
}

func TestRecoverDynamicSynthesis_DegradesInvalidOutputWithoutError(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
	}
	candidate := baziDynamicSynthesis{
		CurrentTrend:     "从这里开始会一路顺行，明显起飞。",
		ClaimStrength:    "明确",
		SupportLevel:     "强",
		LimitationLevel:  "明显",
		WordingCap:       "明确",
		ConsistencyFlags: []string{"机会伴随强变动"},
		DayunPath: []string{
			"当前运势其实仍有明显限制。",
		},
		LiunianFocus:     "这是关键翻身年，可以一飞冲天。",
		WindowLevel:      "机会年",
		ReasoningSummary: "整体会彻底起势。",
		ReasoningSteps:   []string{"先看机会，再直接下拔高结论。"},
	}

	out := recoverDynamicSynthesis(state, candidate, fmt.Errorf("dynamic validation failed"))
	state.DynamicSynthesis = out

	if err := validateDynamicStage(state); err != nil {
		t.Fatalf("expected degraded dynamic synthesis to pass stage validation, got %v", err)
	}
	if err := validateCharterConsistency(state); err != nil {
		t.Fatalf("expected degraded dynamic synthesis to pass consistency validation, got %v", err)
	}
	if out.WindowLevel != "窗口年" {
		t.Fatalf("window_level = %q, want 窗口年", out.WindowLevel)
	}
	if strings.Contains(out.LiunianFocus, "一飞冲天") {
		t.Fatalf("degraded liunian_focus should drop escalation wording, got %q", out.LiunianFocus)
	}
}

func TestRecoverDynamicSynthesis_HealsCurrentDayunDirectionSplit(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
	}
	candidate := baziDynamicSynthesis{
		CurrentTrend:     "当前甲午大运承托静态主轴，但同时放大病点，整体属于吉中有阻。",
		ClaimStrength:    "倾向成立",
		SupportLevel:     "有气",
		LimitationLevel:  "明显",
		WordingCap:       "明确",
		ConsistencyFlags: []string{"吉中有阻"},
		DayunPath: []string{
			"甲午运（30-39岁）：天干甲木七杀为用神，地支午火印星为忌神。此运为偏压，但并非全凶，是承压中求发展的阶段。",
		},
		LiunianFocus:     "这一年有推进窗口，但仍会同步触发内耗与反复。",
		WindowLevel:      "窗口年",
		ReasoningSummary: "先看甲午运对主轴有承托，再看午火把原局病点一并放大，所以只能按吉中有阻落判。",
		ReasoningSteps:   []string{"先看当前甲午大运承托主轴，但不能把放大病点这一层忽略。"},
	}

	out := recoverDynamicSynthesis(state, candidate, fmt.Errorf("current dayun path contradicts current trend direction"))
	state.DynamicSynthesis = out

	if err := validateDynamicStage(state); err != nil {
		t.Fatalf("expected recovered dynamic synthesis to pass stage validation, got %v", err)
	}
	if err := validateCharterConsistency(state); err != nil {
		t.Fatalf("expected recovered dynamic synthesis to pass consistency validation, got %v", err)
	}
	if !containsAnyText([]string{out.DayunPath[0], out.LiunianFocus, out.ReasoningSummary}, []string{"吉中有阻", "限制", "并存"}) {
		t.Fatalf("expected recovered output to preserve mixed-direction wording, got %+v", out)
	}
}
