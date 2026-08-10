// This test file belongs to the manager-owned runtime layer.
// It protects strict BaZi validation and generic reference catalog behavior.
package runtime

import (
	"strings"
	"testing"
)

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
}

func TestValidateStaticAxisAgainstChartFacts_DoesNotHardCodeMethodologyDisputes(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{
			Yongshen: map[string]any{"shi_shen_power": map[string]map[string]float64{
				"七杀": {"gan_count": 0, "zhi_count": 1},
			}},
		},
		StaticSynthesis: baziStaticSynthesis{MainAxis: "月劫格，以食神制杀为用。"},
	}
	if err := validateStaticAxisAgainstChartFacts(state); err != nil {
		t.Fatalf("methodology disputes must be handled by synthesis/eval, not runtime hard branches: %v", err)
	}
}

func TestBaziReferenceCatalog_RejectsUndeclaredRelationWithoutSpecialValidator(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{Dayun: map[string]any{"dayun_analyzed": []map[string]any{{
			"ganZhi": "丙戌", "dayun_chonghe": []map[string]any{},
		}}}},
	}
	assertion := baziAssertion{
		ID: "dynamic.dayun.0", Kind: baziAssertionDayunPeriod,
		Verdict:      "丙戌运与巳、未会成火局。",
		RelationRefs: []baziRelationRef{"relation.fire_bureau"},
	}
	assertBaziViolationCode(t, validateBaziReferenceCatalog(state, []baziAssertion{assertion}), baziViolationUndeclaredFactClaim)

	state.Input.Dayun["dayun_analyzed"] = []map[string]any{{
		"ganZhi": "壬午", "dayun_chonghe": []map[string]any{{"description": "大运参与巳午未会火局"}},
	}}
	assertion.RelationRefs = []baziRelationRef{"relation.dayun.0.0"}
	if err := validateBaziReferenceCatalog(state, []baziAssertion{assertion}); err != nil {
		t.Fatalf("declared relation ref must pass, got %v", err)
	}
}

func TestValidateDynamicConsistencyFlags_RejectsAliasesInsteadOfRewriting(t *testing.T) {
	for _, flag := range []string{"承接与扰动并存", "吉中带阻"} {
		t.Run(flag, func(t *testing.T) {
			err := validateDynamicConsistencyFlags([]string{flag})
			if flag == dynamicFlagMixedConstraint {
				if err != nil {
					t.Fatalf("canonical enum should pass: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), dynamicFlagStructureOnly) {
				t.Fatalf("alias must be rejected with allowed enum list, got %v", err)
			}
		})
	}
}

func TestValidateDynamicStage_RejectsUnsupportedBoundaryTerms(t *testing.T) {
	state := baziCharterState{DynamicSynthesis: baziDynamicSynthesis{
		CurrentTrend: "当前仅作结构观察。", ClaimStrength: "倾向成立", SupportLevel: "有气",
		LimitationLevel: "明显", WordingCap: "中性",
		DayunPath:    []string{"壬午运：关系触发较强，据此直接断为官非。"},
		LiunianFocus: "这一年有承接也有扰动。", WindowLevel: "窗口年",
		ReasoningSummary: "只按结构观察。", ReasoningSteps: []string{"先看大运和流年关系。"},
	}}
	if err := validateDynamicStage(state); err == nil {
		t.Fatal("expected unsupported dynamic boundary wording to fail validation")
	}
}

func TestValidateDynamicStage_RejectsInvestmentAdviceWording(t *testing.T) {
	state := baziCharterState{DynamicSynthesis: baziDynamicSynthesis{
		CurrentTrend: "当前仅作结构观察。", ClaimStrength: "倾向成立", SupportLevel: "有气",
		LimitationLevel: "明显", WordingCap: "中性",
		DayunPath:    []string{"己巳运只能说明结构触发，不能推断投资建议。"},
		LiunianFocus: "这一年仅作结构观察。", WindowLevel: "窗口年",
		ReasoningSummary: "只按结构观察。", ReasoningSteps: []string{"先看大运和流年关系。"},
	}}
	if err := validateDynamicStage(state); err == nil {
		t.Fatal("expected investment advice wording to fail validation")
	}
}

func TestValidateDynamicStage_AllowsOrdinaryBaziInterpretiveLanguage(t *testing.T) {
	state := baziCharterState{DynamicSynthesis: baziDynamicSynthesis{
		CurrentTrend: "当前承接与扰动并存。", ClaimStrength: "倾向成立", SupportLevel: "有气",
		LimitationLevel: "明显", WordingCap: "中性",
		DayunPath:    []string{"丙戌运：戌为火库，燥土可调候暖局，才华初显但根基未稳。"},
		LiunianFocus: "这一年有承接也有扰动。", WindowLevel: "窗口年",
		ReasoningSummary: "只按结构观察。", ReasoningSteps: []string{"先看大运和流年关系。"},
	}}
	if err := validateDynamicStage(state); err != nil {
		t.Fatalf("ordinary bazi interpretive language must remain valid: %v", err)
	}
}
