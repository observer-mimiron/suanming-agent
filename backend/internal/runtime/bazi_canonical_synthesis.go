// This file belongs to the manager-owned runtime layer.
// It owns BaZi canonical synthesis contract for this package.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

const (
	baziCanonicalKindMainAxis      = "main_axis"
	baziCanonicalKindStrength      = "strength"
	baziCanonicalKindTiaohou       = "tiaohou"
	baziCanonicalKindPattern       = "pattern"
	baziCanonicalKindTier          = "tier"
	baziCanonicalKindDayunOverview = "dayun_overview"
	baziCanonicalKindDayunPeriod   = "dayun_period"
	baziCanonicalKindLiunian       = "liunian"
)

// runCanonicalSynthesis asks the model for only the minimum expert judgments.
// Runtime code owns evidence status, deterministic facts and renderer fields.
func (e *Executor) runCanonicalSynthesis(ctx context.Context, st *state.SessionState, chartState baziCharterState, question string) (baziCanonicalSynthesis, error) {
	payload := buildCanonicalSynthesisPayload(chartState, question)
	out, err := runBaziInnerAgentJSON[baziCanonicalSynthesis](ctx, e.builder, baziCanonicalSynthesisConfig(), st, buildBaziCharterPrompt("最小裁断综合", question, payload))
	if err != nil {
		return baziCanonicalSynthesis{}, err
	}
	out = normalizeCanonicalSynthesis(out)
	out.Source = firstNonEmptyTrim(out.Source, "model")
	if err := validateCanonicalSynthesis(chartState, out); err != nil {
		return out, err
	}
	return out, nil
}

// buildCanonicalSynthesisPayload selects only the facts and evidence needed by
// the expert judgment node. Legacy renderer schema is intentionally omitted.
func buildCanonicalSynthesisPayload(state baziCharterState, question string) map[string]any {
	return map[string]any{
		"input": map[string]any{
			"core_chart":      buildCoreChartView(state.Input),
			"dynamic_facts":   buildDynamicFactsView(state.Input),
			"subject_context": buildBaziSubjectContext(state.Input),
		},
		"analysis_plan":    state.AnalysisPlan,
		"evidence_plan":    state.EvidencePlan,
		"evidence_bundle":  buildEvidenceBundleView(state.EvidenceBundle, true),
		"evidence_quality": state.EvidenceQuality,
		"question":         question,
	}
}

// normalizeCanonicalSynthesis fills kind labels and removes empty slice entries
// so downstream projection has a single canonical shape to consume.
func normalizeCanonicalSynthesis(in baziCanonicalSynthesis) baziCanonicalSynthesis {
	in.MainAxis = normalizeCanonicalUnit(in.MainAxis, baziCanonicalKindMainAxis)
	in.Strength = normalizeCanonicalUnit(in.Strength, baziCanonicalKindStrength)
	in.Tiaohou = normalizeCanonicalUnit(in.Tiaohou, baziCanonicalKindTiaohou)
	in.Pattern = normalizeCanonicalUnit(in.Pattern, baziCanonicalKindPattern)
	in.Tier = normalizeCanonicalUnit(in.Tier, baziCanonicalKindTier)
	in.DayunOverview = normalizeCanonicalUnit(in.DayunOverview, baziCanonicalKindDayunOverview)
	in.Liunian = normalizeCanonicalUnit(in.Liunian, baziCanonicalKindLiunian)
	for i := range in.DayunPeriods {
		if strings.TrimSpace(in.DayunPeriods[i].GanZhi) == "" {
			in.DayunPeriods[i].GanZhi = ""
		}
		in.DayunPeriods[i].Verdict = strings.TrimSpace(in.DayunPeriods[i].Verdict)
		in.DayunPeriods[i].Boundary = strings.TrimSpace(in.DayunPeriods[i].Boundary)
		in.DayunPeriods[i].EvidenceTopics = filterNonEmpty(in.DayunPeriods[i].EvidenceTopics)
		in.DayunPeriods[i].FactRefs = filterNonEmpty(in.DayunPeriods[i].FactRefs)
		in.DayunPeriods[i].ClaimRefs = filterNonEmpty(in.DayunPeriods[i].ClaimRefs)
		in.DayunPeriods[i].Confidence = firstNonEmptyTrim(in.DayunPeriods[i].Confidence, "保守判断")
	}
	in.Limitations = filterNonEmpty(in.Limitations)
	in.Advantages = filterNonEmpty(in.Advantages)
	in.Risks = filterNonEmpty(in.Risks)
	in.ReasoningSteps = filterNonEmpty(in.ReasoningSteps)
	in.AdviceBoundary = strings.TrimSpace(in.AdviceBoundary)
	return in
}

func normalizeCanonicalUnit(unit baziCanonicalUnit, kind string) baziCanonicalUnit {
	unit.Kind = firstNonEmptyTrim(unit.Kind, kind)
	unit.Verdict = strings.TrimSpace(unit.Verdict)
	unit.Boundary = strings.TrimSpace(unit.Boundary)
	unit.FactRefs = filterNonEmpty(unit.FactRefs)
	unit.ClaimRefs = filterNonEmpty(unit.ClaimRefs)
	unit.EvidenceTopics = filterNonEmpty(unit.EvidenceTopics)
	unit.Confidence = firstNonEmptyTrim(unit.Confidence, "保守判断")
	return unit
}

// validateCanonicalSynthesis checks only core availability and hard scope. It
// does not recreate the old field-by-field legacy contract.
func validateCanonicalSynthesis(state baziCharterState, in baziCanonicalSynthesis) error {
	for _, unit := range []struct {
		field string
		unit  baziCanonicalUnit
	}{
		{field: "canonical.main_axis", unit: in.MainAxis},
		{field: "canonical.strength", unit: in.Strength},
		{field: "canonical.pattern", unit: in.Pattern},
		{field: "canonical.tier", unit: in.Tier},
	} {
		if strings.TrimSpace(unit.unit.Verdict) == "" {
			return baziViolationError(baziViolationScopeEscalation, unit.field, "", "canonical synthesis missing core verdict", nil, nil)
		}
	}
	if state.AnalysisPlan.NeedDynamic {
		if strings.TrimSpace(in.DayunOverview.Verdict) == "" {
			return baziViolationError(baziViolationScopeEscalation, "canonical.dayun_overview", "", "canonical synthesis missing dayun overview", nil, nil)
		}
		if strings.TrimSpace(in.Liunian.Verdict) == "" {
			return baziViolationError(baziViolationScopeEscalation, "canonical.liunian", "", "canonical synthesis missing liunian verdict", nil, nil)
		}
	}
	text := strings.Join([]string{
		in.MainAxis.Verdict, in.MainAxis.Boundary,
		in.Strength.Verdict, in.Strength.Boundary,
		in.Tiaohou.Verdict, in.Tiaohou.Boundary,
		in.Pattern.Verdict, in.Pattern.Boundary,
		in.Tier.Verdict, in.Tier.Boundary,
		in.DayunOverview.Verdict, in.DayunOverview.Boundary,
		in.Liunian.Verdict, in.Liunian.Boundary,
		strings.Join(in.Limitations, "；"),
		strings.Join(in.Risks, "；"),
	}, "\n")
	if containsUnsupportedConcreteOutcome(text) {
		return baziViolationError(baziViolationUnsupportedConcreteOutcome, "canonical", "", "canonical synthesis includes unsupported concrete outcome", nil, nil)
	}
	return nil
}

// projectCanonicalSynthesis maps the model's minimal judgment into the existing
// renderer structs. The projection is one-way; legacy fields are no longer model
// source-of-truth.
func projectCanonicalSynthesis(state baziCharterState, canonical baziCanonicalSynthesis) (baziStaticSynthesis, baziDynamicSynthesis) {
	static := projectCanonicalStaticSynthesis(state, canonical)
	dynamic := projectCanonicalDynamicSynthesis(state, canonical, static)
	return static, dynamic
}

func projectCanonicalStaticSynthesis(state baziCharterState, c baziCanonicalSynthesis) baziStaticSynthesis {
	tierVerdict, tierBoundary, tierWithheld := canonicalTierText(state, c.Tier)
	reasoningSteps := c.ReasoningSteps
	if len(reasoningSteps) == 0 {
		reasoningSteps = []string{
			"先核对排盘、月令与日主强弱的可复算事实。",
			"再结合格局、调候和证据缺口收束主轴边界。",
			"最后只在证据覆盖范围内说明层次与岁运兑现。",
		}
	}
	patternBasis := firstNonEmptyTrim(c.Pattern.Boundary, staticPatternFactSummary(state.Input), "本轮只保留可复算结构事实。")
	patternOutcome := firstNonEmptyTrim(c.Pattern.Verdict, c.MainAxis.Verdict, "本轮未形成格局成败裁断。")
	counterEvidence := firstNonEmptyTrim(strings.Join(c.Limitations, "；"), c.MainAxis.Boundary, c.Pattern.Boundary, "本轮未形成额外反证。")
	axisConsistency := firstNonEmptyTrim(c.MainAxis.Boundary, c.Pattern.Boundary, "主轴边界以已覆盖证据为准。")
	strengthBalance := firstNonEmptyTrim(c.Strength.Verdict, strengthEvidenceSummary(state.Input.Yongshen), "本轮未形成强弱裁断。")
	tiaohouAnchor := firstNonEmptyTrim(c.Tiaohou.Verdict, "本轮只确认季节环境与调候边界。")
	tiaohouConstraint := firstNonEmptyTrim(c.Tiaohou.Boundary, "调候先后需以已覆盖规则材料为准。")
	claimStrength := canonicalConfidence(c.MainAxis.Confidence)
	static := baziStaticSynthesis{
		Source:                  firstNonEmptyTrim(c.Source, "model"),
		RuleProfile:             strings.TrimSpace(state.Input.RuleProfile.ID),
		MainAxis:                firstNonEmptyTrim(c.MainAxis.Verdict, "本轮未形成主轴裁断。"),
		ClaimStrength:           claimStrength,
		SupportLevel:            canonicalSupportLevel(claimStrength),
		LimitationLevel:         canonicalLimitationLevel(counterEvidence),
		WordingCap:              canonicalWordingCap(claimStrength),
		ConsistencyFlags:        []string{"仅作结构观察"},
		AxisLevel:               "方向成立",
		EffectOnTiaohou:         "中性",
		EffectOnCoreDisease:     "中性",
		EffectOnJiShenDirection: "中性",
		AxisCeiling:             "受限路线",
		ConflictReasons:         filterNonEmpty([]string{counterEvidence}),
		PatternBasis:            patternBasis,
		PatternOutcome:          patternOutcome,
		CounterEvidence:         counterEvidence,
		AxisConsistency:         axisConsistency,
		TiaohouConstraint:       tiaohouConstraint,
		TiaohouAnchor:           tiaohouAnchor,
		StrengthBalance:         strengthBalance,
		Strength: baziStrengthJudgment{
			Conclusion: canonicalStrengthConclusion(c.Strength.Verdict),
			Reasoning:  firstNonEmptyTrim(c.Strength.Boundary, strengthBalance),
			Boundary:   firstNonEmptyTrim(c.Strength.Boundary, "扶抑结论不自动等于格局取用或调候用神。"),
		},
		Usage: baziUsageLayers{
			Fuyi:     firstNonEmptyTrim(c.Strength.Boundary, "扶抑取用以强弱证据边界为准。"),
			Pattern:  patternOutcome,
			Tiaohou:  tiaohouConstraint,
			Priority: "以主轴、调候和证据覆盖共同收束，不作单一维度硬推。",
		},
		PatternAdjudication: buildProjectedPatternAdjudication(state),
		PatternAndQingZhuo:  firstNonEmptyTrim(c.Pattern.Boundary, c.Pattern.Verdict, "本轮仅作结构观察。"),
		QiShiOrCongHua:      "本轮不以气势从化另立主轴。",
		TierJudgment:        tierVerdict,
		TierBasis:           tierBoundary,
		ReasoningSummary:    firstNonEmptyTrim(c.MainAxis.Verdict+"；"+c.Pattern.Verdict+"；"+tierBoundary, c.MainAxis.Verdict),
		ReasoningSteps:      reasoningSteps,
		TopicDirectAnswer:   "",
		TopicFocusAnswer:    "",
		Advantages:          canonicalListOrDefault(c.Advantages, []string{firstNonEmptyTrim(c.MainAxis.Verdict, "结构主轴可观察。")}),
		Risks:               canonicalListOrDefault(c.Risks, []string{counterEvidence}),
		Citations:           mergeCitations(c.Citations, state.EvidenceBundle.Citations...),
		ContractAudit:       c.ContractAudit,
		FieldAudit:          append([]string{}, c.FieldAudit...),
	}
	if tierWithheld {
		static.FieldAudit = append(static.FieldAudit, "canonical_tier_withheld_by_runtime")
	}
	static = sanitizeMinorStaticProjection(state, static)
	static.Assertions = buildProjectedStaticAssertions(state, static, c, tierWithheld)
	return normalizeStaticSynthesis(static)
}

// sanitizeMinorStaticProjection keeps the code-owned legacy projection inside
// the minor age-domain contract. The canonical model may mention adult outcome
// domains while explaining risk; static renderer fields must drop those concrete
// outcomes and retain only structure, growth, care routine or observable change.
func sanitizeMinorStaticProjection(state baziCharterState, static baziStaticSynthesis) baziStaticSynthesis {
	context := buildBaziSubjectContext(state.Input)
	if context.AgeBand != "infant" && context.AgeBand != "child" && context.AgeBand != "adolescent" {
		return static
	}
	fallback := "本轮仅按结构、成长环境、照护节奏和可观察发展观察，不展开成人现实落点。"
	static.MainAxis = minorOutcomeSafeText(static.MainAxis, "本轮仅作命局结构观察。")
	static.PatternBasis = minorOutcomeSafeText(static.PatternBasis, fallback)
	static.PatternOutcome = minorOutcomeSafeText(static.PatternOutcome, fallback)
	static.CounterEvidence = minorOutcomeSafeText(static.CounterEvidence, fallback)
	static.AxisConsistency = minorOutcomeSafeText(static.AxisConsistency, fallback)
	static.TiaohouConstraint = minorOutcomeSafeText(static.TiaohouConstraint, fallback)
	static.TiaohouAnchor = minorOutcomeSafeText(static.TiaohouAnchor, fallback)
	static.TierJudgment = minorOutcomeSafeText(static.TierJudgment, "命格层次中等（保守定位）")
	static.TierBasis = minorOutcomeSafeText(static.TierBasis, fallback)
	static.ReasoningSummary = minorOutcomeSafeText(static.ReasoningSummary, fallback)
	static.ReasoningSteps = minorOutcomeSafeList(static.ReasoningSteps, []string{
		"先看命局结构、季节环境和已覆盖证据。",
		"再把结论限制在成长环境、照护节奏和可观察发展内。",
	})
	static.Advantages = minorOutcomeSafeList(static.Advantages, []string{"结构主轴可观察。"})
	static.Risks = minorOutcomeSafeList(static.Risks, []string{fallback})
	static.Strength.Conclusion = minorOutcomeSafeText(static.Strength.Conclusion, "强弱仅作受力估计。")
	static.Strength.Reasoning = minorOutcomeSafeText(static.Strength.Reasoning, "强弱为受力估计，不直接推出具体应事。")
	static.Strength.Boundary = minorOutcomeSafeText(static.Strength.Boundary, "扶抑结论不自动等于格局取用或现实落点。")
	static.Usage.Fuyi = minorOutcomeSafeText(static.Usage.Fuyi, "扶抑取用以强弱证据边界为准。")
	static.Usage.Pattern = minorOutcomeSafeText(static.Usage.Pattern, fallback)
	static.Usage.Tiaohou = minorOutcomeSafeText(static.Usage.Tiaohou, fallback)
	static.Usage.Priority = minorOutcomeSafeText(static.Usage.Priority, "以结构和证据覆盖共同收束。")
	return static
}

func minorOutcomeSafeText(text, fallback string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return fallback
	}
	if domain, _ := firstUnauthorizedMinorOutcomeSignal(text); domain != "" {
		return fallback
	}
	return text
}

func minorOutcomeSafeList(items, fallback []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if domain, _ := firstUnauthorizedMinorOutcomeSignal(item); domain != "" {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return append([]string{}, fallback...)
	}
	return out
}

func projectCanonicalDynamicSynthesis(state baziCharterState, c baziCanonicalSynthesis, static baziStaticSynthesis) baziDynamicSynthesis {
	if !state.AnalysisPlan.NeedDynamic {
		return baziDynamicSynthesis{}
	}
	facts := buildFactsOnlyDynamicSynthesis(state.Input, static, "")
	judgments := projectCanonicalDayunJudgments(state, c)
	dayunPath := facts.DayunPath
	if len(judgments) > 0 {
		dayunPath = mergeCanonicalDayunPath(dayunPath, judgments)
	}
	claimStrength := canonicalConfidence(firstNonEmptyTrim(c.DayunOverview.Confidence, c.Liunian.Confidence))
	out := baziDynamicSynthesis{
		Source:            firstNonEmptyTrim(c.Source, "model"),
		CurrentTrend:      firstNonEmptyTrim(c.DayunOverview.Verdict, facts.CurrentTrend),
		ClaimStrength:     claimStrength,
		SupportLevel:      canonicalSupportLevel(claimStrength),
		LimitationLevel:   canonicalLimitationLevel(strings.Join(append(c.Limitations, c.Risks...), "；")),
		WordingCap:        canonicalWordingCap(claimStrength),
		ConsistencyFlags:  []string{dynamicFlagStructureOnly},
		DayunPath:         dayunPath,
		DayunJudgments:    judgments,
		CurrentDayunIndex: facts.CurrentDayunIndex,
		LiunianFocus:      firstNonEmptyTrim(c.Liunian.Verdict, facts.LiunianFocus),
		WindowLevel:       canonicalWindowLevel(c.Liunian.Verdict),
		TriggerSignals:    canonicalTriggerSignals(state, c),
		KeyWindows:        []string{firstNonEmptyTrim(c.DayunOverview.Verdict, "本轮只作结构观察。")},
		Risks:             canonicalListOrDefault(c.Risks, facts.Risks),
		ReasoningSummary:  firstNonEmptyTrim(c.DayunOverview.Verdict+"；"+c.Liunian.Verdict, facts.ReasoningSummary),
		ReasoningSteps:    canonicalListOrDefault(c.ReasoningSteps, facts.ReasoningSteps),
		OutcomeDomains:    canonicalOutcomeDomains(state),
		ContractAudit:     c.ContractAudit,
		FieldAudit:        append([]string{}, c.FieldAudit...),
	}
	out.Assertions = buildProjectedDynamicAssertions(state, out)
	return normalizeDynamicSynthesis(out)
}

func canonicalTierText(state baziCharterState, tier baziCanonicalUnit) (string, string, bool) {
	missing := tierMissingTopics(state, tier)
	if len(missing) > 0 {
		return boundedTierJudgment(tier), boundedTierBasis(missing), true
	}
	return firstNonEmptyTrim(tier.Verdict, "仅作结构观察"), firstNonEmptyTrim(tier.Boundary, "层次判断以已覆盖证据为边界。"), false
}

// boundedTierJudgment gives a visible rank even when tier evidence is
// incomplete. Missing A-tier topics cap the result; they do not remove the
// user's requested level judgment.
func boundedTierJudgment(tier baziCanonicalUnit) string {
	text := strings.TrimSpace(tier.Verdict + "；" + tier.Boundary)
	if containsAnyText([]string{text}, []string{"下等", "偏下", "低"}) {
		return "命格层次中等偏下（保守定位）"
	}
	return "命格层次中等（保守定位）"
}

// boundedTierBasis explains the rank cap in domain language instead of
// exposing an internal no-tier state to the user.
func boundedTierBasis(missing []string) string {
	labels := evidenceTopicLabels(missing)
	if len(labels) == 0 {
		labels = []string{"关键主证"}
	}
	return "按保守定级标准：已按主轴、强弱与格局信号给出层次；" +
		strings.Join(labels, "、") + "链条尚未完全闭合，层次封顶为中等，不上推中上或上等。"
}

// evidenceTopicLabels converts internal evidence topic keys into product copy
// so renderer fields do not leak fixture or contract identifiers.
func evidenceTopicLabels(topics []string) []string {
	labels := []string{}
	for _, topic := range topics {
		switch strings.TrimSpace(topic) {
		case "geju":
			labels = append(labels, "格局成败")
		case "tiaohou":
			labels = append(labels, "调候")
		case "bingyao":
			labels = append(labels, "病药救应")
		case "qingzhuo":
			labels = append(labels, "清浊")
		case "dayun":
			labels = append(labels, "大运兑现")
		default:
			if strings.TrimSpace(topic) != "" {
				labels = append(labels, strings.TrimSpace(topic))
			}
		}
	}
	return labels
}

func tierMissingTopics(state baziCharterState, tier baziCanonicalUnit) []string {
	if len(state.EvidenceQuality.MissingTopics) == 0 {
		return nil
	}
	required := state.EvidenceQuality.RequiredTopics
	if len(required) == 0 {
		required = tier.EvidenceTopics
	}
	missing := []string{}
	for _, topic := range required {
		if containsString(state.EvidenceQuality.MissingTopics, topic) {
			missing = append(missing, topic)
		}
	}
	return missing
}

func canonicalEvidenceStatus(state baziCharterState, unit baziCanonicalUnit) string {
	if len(unit.EvidenceTopics) == 0 {
		return baziEvidenceSupported
	}
	for _, topic := range unit.EvidenceTopics {
		if containsString(state.EvidenceQuality.MissingTopics, topic) {
			return baziEvidenceWithheld
		}
	}
	return baziEvidenceSupported
}

func buildProjectedStaticAssertions(state baziCharterState, static baziStaticSynthesis, c baziCanonicalSynthesis, tierWithheld bool) []baziAssertion {
	main := canonicalAssertion(state, "static.main_axis", baziAssertionMainAxis, "chart", c.MainAxis)
	strength := canonicalAssertion(state, "static.strength", baziAssertionStrength, "day_master", c.Strength)
	tiaohou := canonicalAssertion(state, "static.tiaohou", baziAssertionTiaohou, "chart", c.Tiaohou)
	pattern := canonicalAssertion(state, "static.pattern", baziAssertionPatternUsage, "chart", c.Pattern)
	tier := canonicalAssertion(state, "static.tier", baziAssertionTier, "chart", c.Tier)
	if tierWithheld {
		tier.Verdict = static.TierJudgment
		tier.Boundary = static.TierBasis
		tier.EvidenceStatus = baziEvidenceWithheld
	}
	return []baziAssertion{main, strength, tiaohou, pattern, tier}
}

func buildProjectedDynamicAssertions(state baziCharterState, dynamic baziDynamicSynthesis) []baziAssertion {
	return ensureDynamicAssertions(state, dynamic).Assertions
}

func canonicalAssertion(state baziCharterState, id string, kind baziAssertionKind, subject string, unit baziCanonicalUnit) baziAssertion {
	return baziAssertion{
		ID:             id,
		Kind:           kind,
		Subject:        subject,
		Verdict:        firstNonEmptyTrim(unit.Verdict, "仅作结构观察"),
		FactRefs:       stringsToFactRefs(unit.FactRefs),
		ClaimRefs:      stringsToClaimRefs(unit.ClaimRefs),
		EvidenceTopics: append([]string{}, unit.EvidenceTopics...),
		EvidenceStatus: canonicalEvidenceStatus(state, unit),
		Confidence:     canonicalConfidence(unit.Confidence),
		Boundary:       firstNonEmptyTrim(unit.Boundary, "仅解释已覆盖证据内的结构，不展开具体应事。"),
	}
}

func stringsToFactRefs(items []string) []baziFactRef {
	out := make([]baziFactRef, 0, len(items))
	for _, item := range filterNonEmpty(items) {
		out = append(out, baziFactRef(item))
	}
	return out
}

func stringsToClaimRefs(items []string) []baziClaimRef {
	out := make([]baziClaimRef, 0, len(items))
	for _, item := range filterNonEmpty(items) {
		out = append(out, baziClaimRef(item))
	}
	return out
}

func buildProjectedPatternAdjudication(state baziCharterState) baziPatternAdjudication {
	candidateName := firstNonEmptyTrim(stringValue(state.Input.Yongshen["geju_candidate"]), "月令候选")
	return baziPatternAdjudication{
		MonthCommandCandidateID:  "month_command",
		SelectedAxisCandidateIDs: []string{"month_command"},
		Candidates: []baziPatternCandidate{{
			ID:             "month_command",
			Name:           candidateName,
			Origin:         "month_command",
			Visibility:     "以工具返回透藏事实为准",
			Role:           "selected_axis",
			FactRefs:       []baziFactRef{"yongshen.geju_candidate", "chart.month_branch"},
			EvidenceTopics: []string{"geju"},
			ComparisonDimensions: []string{
				"visibility",
				"hidden_stem_tier",
				"root_support",
				"season_support",
				"structural_closure",
				"counter_evidence",
			},
		}},
	}
}

func projectCanonicalDayunJudgments(state baziCharterState, c baziCanonicalSynthesis) []baziDayunJudgment {
	periods := dayunPeriods(state.Input.Dayun)
	if len(periods) == 0 {
		return nil
	}
	periodIndexByGanZhi := make(map[string]int, len(periods))
	for i, period := range periods {
		if ganZhi := strings.TrimSpace(stringValue(period["ganZhi"])); ganZhi != "" {
			periodIndexByGanZhi[ganZhi] = i
		}
	}
	byIndex := map[int]baziCanonicalDayunUnit{}
	for _, unit := range c.DayunPeriods {
		if strings.TrimSpace(unit.Verdict) == "" {
			continue
		}
		index := -1
		if ganZhi := strings.TrimSpace(unit.GanZhi); ganZhi != "" {
			if matched, ok := periodIndexByGanZhi[ganZhi]; ok {
				index = matched
			}
		}
		if index < 0 && unit.Index != nil {
			index = *unit.Index
		}
		if index >= 0 && index < len(periods) {
			byIndex[index] = unit
		}
	}
	if len(byIndex) == 0 {
		return nil
	}
	out := make([]baziDayunJudgment, 0, len(periods))
	for i, period := range periods {
		ganZhi := strings.TrimSpace(stringValue(period["ganZhi"]))
		unit, ok := byIndex[i]
		if !ok {
			out = append(out, baziDayunJudgment{
				GanZhi:         ganZhi,
				Trend:          "仅作结构观察",
				Interpretation: "本步大运未被模型列为重点窗口；仅展示工具事实。",
				Evidence:       relationTextList(period["dayun_chonghe"]),
				OutcomeDomains: canonicalOutcomeDomains(state),
			})
			continue
		}
		out = append(out, baziDayunJudgment{
			GanZhi:         firstNonEmptyTrim(unit.GanZhi, ganZhi),
			Trend:          firstNonEmptyTrim(unit.Verdict, "仅作结构观察"),
			Interpretation: firstNonEmptyTrim(unit.Boundary, "只解释结构触发，不展开具体生活事件。"),
			Evidence:       canonicalListOrDefault(unit.FactRefs, relationTextList(period["dayun_chonghe"])),
			OutcomeDomains: canonicalOutcomeDomains(state),
		})
	}
	return out
}

func mergeCanonicalDayunPath(facts []string, judgments []baziDayunJudgment) []string {
	if len(facts) == 0 {
		return renderDayunJudgmentLines(judgments)
	}
	out := append([]string{}, facts...)
	for i, judgment := range judgments {
		if i >= len(out) || strings.TrimSpace(judgment.Interpretation) == "" || strings.Contains(judgment.Interpretation, "未被模型列为重点窗口") {
			continue
		}
		out[i] = strings.TrimSpace(out[i]) + "\n- **结构裁断**：" + judgment.Trend + "；" + judgment.Interpretation
	}
	return out
}

func canonicalConfidence(value string) string {
	switch strings.TrimSpace(value) {
	case "明确成立", "倾向成立", "封顶判断":
		if strings.TrimSpace(value) == "封顶判断" {
			return "明确成立"
		}
		return strings.TrimSpace(value)
	default:
		return "保守判断"
	}
}

func canonicalSupportLevel(confidence string) string {
	switch canonicalConfidence(confidence) {
	case "明确成立":
		return "有气"
	case "倾向成立":
		return "有根"
	default:
		return "出现"
	}
}

func canonicalLimitationLevel(text string) string {
	if containsAnyText([]string{text}, []string{"核心", "明显", "不足", "受限", "缺失", "暂不"}) {
		return "明显"
	}
	return "轻微"
}

func canonicalWordingCap(confidence string) string {
	switch canonicalConfidence(confidence) {
	case "明确成立":
		return "明确"
	case "倾向成立":
		return "中性"
	default:
		return "保守"
	}
}

func canonicalStrengthConclusion(text string) string {
	for _, allowed := range []string{"中和偏强", "中和偏弱", "中和附近", "偏强", "偏弱"} {
		if strings.Contains(text, allowed) {
			return allowed
		}
	}
	return "中和附近"
}

func canonicalWindowLevel(text string) string {
	switch {
	case strings.Contains(text, "转折"):
		return "转折年"
	case strings.Contains(text, "承压"):
		return "承压年"
	case strings.Contains(text, "窗口"):
		return "窗口年"
	default:
		return "扰动年"
	}
}

func canonicalOutcomeDomains(state baziCharterState) []string {
	ctx := buildBaziSubjectContext(state.Input)
	if containsString(ctx.AllowedOutcomeDomains, "structure") {
		return []string{"structure"}
	}
	return []string{"structure"}
}

func canonicalTriggerSignals(state baziCharterState, c baziCanonicalSynthesis) []string {
	items := []string{}
	items = append(items, c.DayunOverview.FactRefs...)
	items = append(items, c.Liunian.FactRefs...)
	items = append(items, relationTextList(state.Input.Liunian["liunian_chonghe"])...)
	return canonicalListOrDefault(items, []string{"本轮仅按已计算大运、流年和关系事实观察。"})
}

func canonicalListOrDefault(items, fallback []string) []string {
	items = filterNonEmpty(items)
	if len(items) > 0 {
		return items
	}
	return filterNonEmpty(fallback)
}

func annotateCanonicalSynthesis(ctx context.Context, canonical baziCanonicalSynthesis) {
	withheld := 0
	for _, unit := range []baziCanonicalUnit{canonical.MainAxis, canonical.Strength, canonical.Tiaohou, canonical.Pattern, canonical.Tier, canonical.DayunOverview, canonical.Liunian} {
		if strings.Contains(unit.Verdict, "证据不足") || strings.Contains(unit.Boundary, "缺失") {
			withheld++
		}
	}
	tracing.SetTraceAttributes(ctx, map[string]any{
		"bazi.synthesis.schema":         "canonical_v1",
		"bazi.canonical.unit_count":     7 + len(canonical.DayunPeriods),
		"bazi.canonical.withheld_units": withheld,
		"bazi.projection.source":        "canonical",
	})
}

func canonicalFailureFactsOnly(state baziCharterState, err error, auditCode, fallback string) (baziStaticSynthesis, baziDynamicSynthesis) {
	reason := recoveryReasonText(err, fallback)
	static := buildFactsOnlyStaticSynthesis(state.Input, reason)
	dynamic := buildFactsOnlyDynamicSynthesis(state.Input, static, reason)
	// Facts-only output is runtime-generated after model text is discarded, so
	// the display contract is clean even though the recovery reason remains
	// visible in FieldAudit and trace attributes.
	static.ContractAudit = baziContractAudit{Compliant: true}
	dynamic.ContractAudit = baziContractAudit{Compliant: true}
	static.FieldAudit = append(static.FieldAudit, auditCode)
	dynamic.FieldAudit = append(dynamic.FieldAudit, auditCode)
	return static, dynamic
}

func canonicalDynamicFailureFactsOnly(state baziCharterState, static baziStaticSynthesis, err error) baziDynamicSynthesis {
	reason := recoveryReasonText(err, "最小裁断动态投影越过授权领域，已降级展示可复算大运与流年事实。")
	dynamic := buildFactsOnlyDynamicSynthesis(state.Input, static, reason)
	dynamic.ContractAudit = baziContractAudit{Compliant: true}
	dynamic.FieldAudit = append(dynamic.FieldAudit, "canonical_dynamic_projection_facts_only")
	if failure, ok := baziContractFailureFromError("dynamic_projection", err); ok {
		dynamic.FieldAudit = append(dynamic.FieldAudit,
			"contract_failure_class:"+failure.Class,
			"recovery_policy:"+failure.RecoveryPolicy,
		)
	}
	return dynamic
}

func canonicalSynthesisRuntimeFailure(cause error) error {
	return &RuntimeFailure{
		Class:       failureClassModelContractViolation,
		Stage:       "canonical_synthesis",
		Domain:      "bazi",
		Code:        "BAZI_CANONICAL_SYNTHESIS_FAILED",
		Retryable:   false,
		Degraded:    true,
		UserVisible: true,
		Message:     "本轮专业综合未通过，已停止展示不稳定裁断，仅保留可复算事实。",
		Cause:       fmt.Errorf("canonical synthesis failed: %w", cause),
	}
}
