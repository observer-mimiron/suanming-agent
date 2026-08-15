// package presentation 包含 Manager 拥有的八字执行主链。
//
// 本文件负责把 Graph 后的已验收状态映射为 presentation 窄 DTO；
// 不负责渲染文本、重新校验合同或接触 Session、模型和传输层。
package presentation

import (
	"fmt"
	"strconv"
	"strings"

	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
)

func birthYearFromBaziResult(result map[string]any) int {
	value := strings.TrimSpace(bazidomain.StringValue(result["birthday"]))
	if len(value) < 4 {
		return 0
	}
	year, _ := strconv.Atoi(value[:4])
	return year
}

func targetYearFromLiunian(liunian map[string]any) int {
	switch value := liunian["liunian_year"].(type) {
	case int:
		return value
	case float64:
		return int(value)
	default:
		return 0
	}
}

type baziPresentationInput = FinalReplyInput
type baziPresentationCitation = Citation

// buildBaziFinalPresentationInput 只投影最终展示需要的已验收槽位。
func buildBaziFinalPresentationInput(plan baziAnalysisPlan, state baziCharterState, _ string) baziPresentationInput {
	presentationPlan := baziPresentationPlan(plan)
	// 原入口只用参数 plan 选择模板；全程大运章节仍由图状态中的已验收计划决定。
	presentationPlan.NeedLifetimeDayun = state.AnalysisPlan.NeedLifetimeDayun
	input := baziPresentationInput{
		AnalysisPlan: presentationPlan,
		Facts:        baziPresentationFacts(state),
		EvidenceBundle: EvidenceBundle{
			Citations: baziPresentationCitations(state.EvidenceBundle.Citations),
		},
		StaticSynthesis:   baziPresentationStatic(state.StaticSynthesis),
		LifetimeSynthesis: baziPresentationLifetime(state.LifetimeSynthesis),
		DynamicSynthesis:  baziPresentationDynamic(state.DynamicSynthesis),
	}
	input.FactsOnlyDynamicSynthesis = baziPresentationDynamic(bazidomain.BuildFactsOnlyDynamicSynthesis(state.Input, state.StaticSynthesis, state.DynamicSynthesis.RecoveryReason))
	return input
}

// baziPresentationPlan removes plan fields that cannot affect final presentation.
func baziPresentationPlan(plan baziAnalysisPlan) AnalysisPlan {
	return AnalysisPlan{
		NeedLifetimeDayun: plan.NeedLifetimeDayun,
		WriterTemplate:    plan.WriterTemplate,
		TopicMode:         plan.TopicMode,
	}
}

// baziPresentationFacts projects deterministic facts without exposing raw tool maps.
func baziPresentationFacts(state baziCharterState) ChartFacts {
	capsule := bazidomain.FactCapsuleForState(state)
	current := bazidomain.MapValue(state.Input.Liunian, "current_dayun")
	strength := strings.TrimSpace(state.StaticSynthesis.StrengthBalance)
	if strength == "" {
		strength = bazidomain.StrengthEvidenceSummary(state.Input.Yongshen)
	}
	return ChartFacts{
		PillarsSummary:     bazidomain.PillarFactSummary(state.Input.BaziResult["pillars"]),
		DayMaster:          strings.TrimSpace(bazidomain.StringValue(state.Input.BaziResult["dayGan"])),
		StrengthEvidence:   strength,
		PatternSummary:     bazidomain.StaticPatternFactSummary(state.Input),
		LiunianGanZhi:      strings.TrimSpace(bazidomain.StringValue(state.Input.Liunian["liunian_ganzhi"])),
		LiunianTenGod:      strings.TrimSpace(bazidomain.StringValue(state.Input.Liunian["liunian_shi_shen"])),
		CurrentDayunGanZhi: strings.TrimSpace(bazidomain.StringValue(current["ganZhi"])),
		LiunianRelations:   append([]string(nil), bazidomain.RelationTextList(state.Input.Liunian["liunian_chonghe"])...),
		DayunPeriods:       baziPresentationDayunPeriods(state.Input.Dayun),
		RuleProfileID:      strings.TrimSpace(state.Input.RuleProfile.ID),
		SubjectAgeBand: bazidomain.BuildSubjectContext(bazidomain.SubjectContextInput{
			BirthYear:  birthYearFromBaziResult(state.Input.BaziResult),
			TargetYear: targetYearFromLiunian(state.Input.Liunian),
		}).AgeBand,
		MonthCommand:           capsule.MonthCommand,
		OfficialVisible:        capsule.OfficialVisible,
		OfficialHidden:         capsule.OfficialHidden,
		FireEffectivenessKnown: capsule.FireEffectivenessKnown,
	}
}

// baziPresentationDayunPeriods projects only the stable facts used by labels and headings.
func baziPresentationDayunPeriods(dayun map[string]any) []DayunPeriod {
	periods := bazidomain.DayunPeriods(dayun)
	out := make([]DayunPeriod, 0, len(periods))
	for index, period := range periods {
		out = append(out, DayunPeriod{
			Ref:    fmt.Sprintf("dayun[%d]", index),
			Label:  compactDayunPeriodLabel(bazidomain.DayunPeriodDisplayLabel(period)),
			GanZhi: strings.TrimSpace(bazidomain.StringValue(period["ganZhi"])),
			TenGod: strings.TrimSpace(bazidomain.StringValue(period["tenGod"])),
		})
	}
	return out
}

// compactDayunPeriodLabel 在最终报告中隐藏精确交运时刻，保留用户阅读所需的起止年龄。
func compactDayunPeriodLabel(label string) string {
	label = strings.TrimSpace(label)
	index := strings.Index(label, "；")
	if index < 0 || !strings.Contains(label[:index], "（") || !strings.Contains(label[index:], "）") {
		return label
	}
	return strings.TrimSpace(label[:index]) + "）"
}

// baziPresentationStatic copies only renderer-owned static slots.
func baziPresentationStatic(static baziStaticSynthesis) StaticSynthesis {
	return StaticSynthesis{
		FactsOnly:         isFactsOnlyStaticSynthesis(static),
		RecoveryReason:    static.RecoveryReason,
		RuleProfile:       static.RuleProfile,
		MainAxis:          static.MainAxis,
		AxisConsistency:   static.AxisConsistency,
		PatternOutcome:    static.PatternOutcome,
		CounterEvidence:   static.CounterEvidence,
		TiaohouConstraint: static.TiaohouConstraint,
		TiaohouAnchor:     static.TiaohouAnchor,
		StrengthBalance:   static.StrengthBalance,
		Strength: StrengthJudgment{
			Conclusion: static.Strength.Conclusion,
			Reasoning:  static.Strength.Reasoning,
		},
		Usage: UsageLayers{
			Fuyi:    static.Usage.Fuyi,
			Tiaohou: static.Usage.Tiaohou,
		},
		TierStatus:        static.TierAssessment.Status,
		TierJudgment:      static.TierJudgment,
		TierBasis:         static.TierBasis,
		ReasoningSummary:  static.ReasoningSummary,
		TopicDirectAnswer: static.TopicDirectAnswer,
		TopicFocusAnswer:  static.TopicFocusAnswer,
		Advantages:        append([]string(nil), static.Advantages...),
		Risks:             append([]string(nil), static.Risks...),
	}
}

// baziPresentationDynamic copies dynamic slots and marks facts-only status explicitly.
func baziPresentationDynamic(dynamic baziDynamicSynthesis) DynamicSynthesis {
	return DynamicSynthesis{
		FactsOnly:                isFactsOnlyDynamicSynthesis(dynamic),
		RecoveryReason:           dynamic.RecoveryReason,
		CurrentTrend:             dynamic.CurrentTrend,
		CurrentPeriodRealization: dynamic.CurrentPeriodRealization,
		ConsistencyFlags:         append([]string(nil), dynamic.ConsistencyFlags...),
		DayunPath:                append([]string(nil), dynamic.DayunPath...),
		CurrentDayunIndex:        dynamic.CurrentDayunIndex,
		LiunianFocus:             dynamic.LiunianFocus,
		WindowLevel:              dynamic.WindowLevel,
		TriggerSignals:           append([]string(nil), dynamic.TriggerSignals...),
		Risks:                    append([]string(nil), dynamic.Risks...),
		ReasoningSummary:         dynamic.ReasoningSummary,
		ReasoningSteps:           append([]string(nil), dynamic.ReasoningSteps...),
		OutcomeDomains:           append([]string(nil), dynamic.OutcomeDomains...),
	}
}

// baziPresentationLifetime copies the independent full-life display contract.
func baziPresentationLifetime(lifetime baziLifetimeDayunSynthesis) LifetimeDayunSynthesis {
	claims := make([]LifetimeDayunClaim, 0, len(lifetime.PeriodClaims))
	for _, claim := range lifetime.PeriodClaims {
		claims = append(claims, LifetimeDayunClaim{PeriodRef: claim.PeriodRef, PeriodEffect: claim.PeriodEffect})
	}
	return LifetimeDayunSynthesis{
		Status:       lifetime.Status,
		Trajectory:   lifetime.Trajectory,
		PeriodClaims: claims,
		Summary:      lifetime.Summary,
	}
}

// baziPresentationCitations copies short citation DTOs without exposing evidence internals.
func baziPresentationCitations(citations []baziCitation) []baziPresentationCitation {
	out := make([]baziPresentationCitation, 0, len(citations))
	for _, citation := range citations {
		out = append(out, baziPresentationCitation{Classic: citation.Classic, Quotes: append([]string(nil), citation.Quotes...)})
	}
	return out
}
