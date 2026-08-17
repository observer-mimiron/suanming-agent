// Package application 包含八字用例层的确定性投影与合同编排。
//
// 本文件只复用 domain 的公开数据合同；不持有 runtime、模型、会话或事件依赖。
package application

import (
	"strings"

	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
)

type baziCharterState = bazidomain.CharterState
type baziCharterInput = bazidomain.CharterInput
type baziCanonicalSynthesis = bazidomain.CanonicalSynthesis
type baziStaticSynthesis = bazidomain.StaticSynthesis
type baziDynamicSynthesis = bazidomain.DynamicSynthesis
type baziValidationViolation = bazidomain.ValidationViolation
type baziStructuredStaticSynthesis = bazidomain.StructuredStaticSynthesis
type baziStructuredDynamicSynthesis = bazidomain.StructuredDynamicSynthesis
type baziCanonicalUnit = bazidomain.CanonicalUnit
type baziTierAssessment = bazidomain.TierAssessment
type baziAssertion = bazidomain.Assertion
type baziAssertionKind = bazidomain.AssertionKind
type baziFactRef = bazidomain.FactRef
type baziClaimRef = bazidomain.ClaimRef
type baziRelationRef = bazidomain.RelationRef
type baziStrengthJudgment = bazidomain.StrengthJudgment
type baziUsageLayers = bazidomain.UsageLayers
type baziPatternAdjudication = bazidomain.PatternAdjudication
type baziDayunJudgment = bazidomain.DayunJudgment
type baziAnalysisPlan = bazidomain.AnalysisPlan
type baziEvidenceBundle = bazidomain.EvidenceBundle
type baziEvidenceQuality = bazidomain.EvidenceQuality
type baziCitation = bazidomain.Citation
type baziContractAudit = bazidomain.ContractAudit
type baziPatternCandidate = bazidomain.PatternCandidate
type baziCanonicalDayunUnit = bazidomain.CanonicalDayunUnit

const (
	baziEvidenceSupported     = bazidomain.EvidenceSupported
	baziEvidenceWithheld      = bazidomain.EvidenceWithheld
	baziAssertionMainAxis     = bazidomain.AssertionMainAxis
	baziAssertionStrength     = bazidomain.AssertionStrength
	baziAssertionTiaohou      = bazidomain.AssertionTiaohou
	baziAssertionPatternUsage = bazidomain.AssertionPatternUsage
	baziAssertionTier         = bazidomain.AssertionTier
	baziAssertionDayunPeriod  = bazidomain.AssertionDayunPeriod
	baziAssertionLiunian      = bazidomain.AssertionLiunian
)

// containsString reports exact membership in a local projection list.
func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func ensureDynamicAssertions(state baziCharterState, synthesis baziDynamicSynthesis) baziDynamicSynthesis {
	return bazidomain.EnsureDynamicAssertions(state, synthesis)
}

func stringValue(raw any) string { return bazidomain.StringValue(raw) }

func relationTextList(raw any) []string { return bazidomain.RelationTextList(raw) }

func periodHeadline(line string) string { return bazidomain.PeriodHeadline(line) }

func currentDayunIndexForInput(input baziCharterInput) int {
	return bazidomain.CurrentDayunIndexForInput(input)
}

func buildFactsOnlyStaticSynthesis(input baziCharterInput, reason string) baziStaticSynthesis {
	return bazidomain.BuildFactsOnlyStaticSynthesis(input, reason)
}

func containsAnyText(value any, needles []string) bool {
	var texts []string
	switch typed := value.(type) {
	case string:
		texts = []string{typed}
	case []string:
		texts = typed
	default:
		return false
	}
	for _, text := range texts {
		for _, needle := range needles {
			if needle != "" && strings.Contains(text, needle) {
				return true
			}
		}
	}
	return false
}

func isFactsOnlyStaticSynthesis(s baziStaticSynthesis) bool {
	return s.Source == "facts_only_degraded"
}

func isFactsOnlyDynamicSynthesis(s baziDynamicSynthesis) bool {
	return s.Source == "facts_only_degraded"
}

func hasDynamicSystemFacts(input baziCharterInput) bool {
	return len(input.Dayun) > 0 || len(input.Liunian) > 0
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func buildBaziFactCapsulePromptView(state baziCharterState, includeDynamic bool) map[string]any {
	periods := bazidomain.DayunPeriods(state.Input.Dayun)
	current := map[string]any{}
	if index := currentDayunIndexForInput(state.Input); index >= 0 && index < len(periods) {
		current = periods[index]
	}
	return bazidomain.BuildPromptView(bazidomain.FactInput{
		BaziResult: state.Input.BaziResult, Yongshen: state.Input.Yongshen,
		MonthCommand:  bazidomain.MonthBranchForEvidenceQuery(state.Input),
		CurrentPeriod: current,
	}, includeDynamic)
}
