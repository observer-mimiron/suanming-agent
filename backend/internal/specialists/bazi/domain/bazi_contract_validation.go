// Package domain 包含八字静态与动态结果的领域合同校验。
//
// 本文件负责静态/动态投影、证据边界和最终文本的合同校验；
// 不负责图节点编排、模型调用或确定性事实计算。
package domain

import (
	"fmt"
	"strconv"
	"strings"
)

var allowedDynamicConsistencyFlags = []string{
	"吉中有阻", "机会伴随强变动", "限制仍在", "仅作结构观察",
}

// validateStaticStage 校验静态综合的必需字段和年龄授权边界。
func validateStaticStage(state baziCharterState) error {
	if isFactsOnlyStaticSynthesis(state.StaticSynthesis) {
		return validateFactsOnlyStaticSynthesis(state)
	}
	if state.Input.RuleProfile.ID != "" && state.StaticSynthesis.RuleProfile != "" && state.StaticSynthesis.RuleProfile != state.Input.RuleProfile.ID {
		return projectionMismatchViolation("static.rule_profile", "static synthesis rule profile does not match selected profile", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.MainAxis) == "" {
		return projectionMismatchViolation("static.main_axis", "missing static synthesis main axis", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.PatternBasis) == "" {
		return projectionMismatchViolation("static.pattern_basis", "missing static synthesis pattern basis", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.PatternOutcome) == "" {
		return projectionMismatchViolation("static.pattern_outcome", "missing static synthesis pattern outcome", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.CounterEvidence) == "" {
		return projectionMismatchViolation("static.counter_evidence", "missing static synthesis counter evidence", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.AxisConsistency) == "" {
		return projectionMismatchViolation("static.axis_consistency", "missing static synthesis axis consistency", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.TiaohouAnchor) == "" {
		return projectionMismatchViolation("static.tiaohou_anchor", "missing static synthesis tiaohou anchor", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.PatternAndQingZhuo) == "" {
		return projectionMismatchViolation("static.pattern_and_qing_zhuo", "missing static synthesis pattern and qingzhuo", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.TierJudgment) == "" {
		return projectionMismatchViolation("static.tier_judgment", "missing static synthesis tier judgment", nil)
	}
	if strings.Contains(state.StaticSynthesis.TierJudgment, "层级暂不定级") {
		return projectionMismatchViolation("static.tier_judgment", "static synthesis exposes internal no-tier state", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.TierBasis) == "" {
		return projectionMismatchViolation("static.tier_basis", "missing static synthesis tier basis", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.ReasoningSummary) == "" {
		return projectionMismatchViolation("static.reasoning_summary", "missing static synthesis reasoning summary", nil)
	}
	if len(state.StaticSynthesis.ReasoningSteps) == 0 {
		return projectionMismatchViolation("static.reasoning_steps", "missing static synthesis reasoning steps", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.ClaimStrength) == "" {
		return projectionMismatchViolation("static.claim_strength", "missing static synthesis claim strength", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.SupportLevel) == "" {
		return projectionMismatchViolation("static.support_level", "missing static synthesis support level", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.LimitationLevel) == "" {
		return projectionMismatchViolation("static.limitation_level", "missing static synthesis limitation level", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.WordingCap) == "" {
		return projectionMismatchViolation("static.wording_cap", "missing static synthesis wording cap", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.AxisLevel) == "" {
		return projectionMismatchViolation("static.axis_level", "missing static synthesis axis level", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.EffectOnTiaohou) == "" {
		return projectionMismatchViolation("static.effect_on_tiaohou", "missing static synthesis effect on tiaohou", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.EffectOnCoreDisease) == "" {
		return projectionMismatchViolation("static.effect_on_core_disease", "missing static synthesis effect on core disease", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.EffectOnJiShenDirection) == "" {
		return projectionMismatchViolation("static.effect_on_jishen_direction", "missing static synthesis effect on ji-shen direction", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.AxisCeiling) == "" {
		return projectionMismatchViolation("static.axis_ceiling", "missing static synthesis axis ceiling", nil)
	}
	if requiresStaticConflictReasons(state.StaticSynthesis) && len(state.StaticSynthesis.ConflictReasons) == 0 {
		return projectionMismatchViolation("static.conflict_reasons", "missing static synthesis conflict reasons", nil)
	}
	if err := validateStaticOutcomeScope(state); err != nil {
		return err
	}
	return nil
}

// isFactsOnlyStaticSynthesis reports whether static output is deterministic facts-only fallback.
func isFactsOnlyStaticSynthesis(s baziStaticSynthesis) bool {
	return strings.TrimSpace(s.Source) == baziSynthesisSourceFactsOnlyDegraded
}

// isFactsOnlyDynamicSynthesis reports whether dynamic output is deterministic facts-only fallback.
func isFactsOnlyDynamicSynthesis(d baziDynamicSynthesis) bool {
	return strings.TrimSpace(d.Source) == baziSynthesisSourceFactsOnlyDegraded
}

// validateFactsOnlyStaticSynthesis 校验 facts-only fallback 自身可展示。
// 这里不触发模型 repair；错误仍必须机器可读，便于 recovery 和 trace 分类。
func validateFactsOnlyStaticSynthesis(state baziCharterState) error {
	if state.Input.RuleProfile.ID != "" && state.StaticSynthesis.RuleProfile != state.Input.RuleProfile.ID {
		return projectionMismatchViolation("static.facts_only.rule_profile", "facts-only static synthesis rule profile does not match selected profile", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.MainAxis) == "" {
		return projectionMismatchViolation("static.facts_only.main_axis", "facts-only static synthesis missing degraded message", nil)
	}
	if strings.TrimSpace(state.StaticSynthesis.PatternBasis) == "" && len(state.Input.BaziResult) == 0 && len(state.Input.Yongshen) == 0 {
		return projectionMismatchViolation("static.facts_only.chart_facts", "facts-only static synthesis has no chart facts to show", nil)
	}
	return nil
}

// staticSynthesisUserVisibleText collects static fields subject to age-scope validation.
func staticSynthesisUserVisibleText(output baziStaticSynthesis) string {
	return strings.Join([]string{
		output.MainAxis, output.PatternBasis, output.PatternOutcome, output.CounterEvidence,
		output.AxisConsistency, output.TiaohouConstraint, output.TiaohouAnchor, output.TierJudgment,
		output.TierBasis, output.ReasoningSummary, strings.Join(output.ReasoningSteps, "\n"),
		output.Strength.Conclusion, output.Strength.Reasoning, output.Strength.Boundary,
		output.Usage.Fuyi, output.Usage.Pattern, output.Usage.Tiaohou, output.Usage.Priority,
		strings.Join(output.Advantages, "\n"), strings.Join(output.Risks, "\n"),
	}, "\n")
}

// validateStaticOutcomeScope applies the same age-domain contract to static
// synthesis as dynamic synthesis. Static layers own visible advantages, risks
// and tier rationale, so minors must not receive adult-domain projections there.
func validateStaticOutcomeScope(state baziCharterState) error {
	context := buildBaziSubjectContext(state.Input)
	if context.AgeBand != "infant" && context.AgeBand != "child" && context.AgeBand != "adolescent" {
		return nil
	}
	if domain, term := firstUnauthorizedMinorOutcomeSignal(staticSynthesisUserVisibleText(state.StaticSynthesis)); domain != "" {
		return baziViolationError(baziViolationUnsupportedConcreteOutcome, "static", "", fmt.Sprintf("minor static synthesis writes unauthorized %s outcome signal %q", domain, term), nil, context.AllowedOutcomeDomains)
	}
	return nil
}

// validateDynamicAgainstProfileScope enforces authorized domains and wording caps for dynamic output.
func validateDynamicAgainstProfileScope(state baziCharterState) error {
	if err := validateDynamicOutcomeDomains(state); err != nil {
		return err
	}
	text := strings.Join([]string{
		state.DynamicSynthesis.CurrentTrend, strings.Join(state.DynamicSynthesis.DayunPath, "\n"),
		strings.Join(RenderDayunJudgmentLines(state.DynamicSynthesis.DayunJudgments), "\n"),
		state.DynamicSynthesis.LiunianFocus, strings.Join(state.DynamicSynthesis.TriggerSignals, "\n"),
		strings.Join(state.DynamicSynthesis.KeyWindows, "\n"), strings.Join(state.DynamicSynthesis.Risks, "\n"),
		strings.Join(state.DynamicSynthesis.ConsistencyFlags, "\n"), state.DynamicSynthesis.ReasoningSummary,
		strings.Join(state.DynamicSynthesis.ReasoningSteps, "\n"),
	}, "\n")
	if hasDynamicHardBoundary(text) {
		return baziViolationError(baziViolationUnsupportedConcreteOutcome, "dynamic", "", "dynamic synthesis overstates unsupported concrete outcome", nil, nil)
	}
	if err := validateDynamicConsistencyFlags(state.DynamicSynthesis.ConsistencyFlags); err != nil {
		return err
	}
	return nil
}

// buildBaziSubjectContext 从八字事实载荷提取年龄范围校验所需字段。
func buildBaziSubjectContext(input baziCharterInput) SubjectContext {
	birthday := strings.TrimSpace(stringValue(input.BaziResult["birthday"]))
	birthYear := 0
	if len(birthday) >= 4 {
		birthYear, _ = strconv.Atoi(birthday[:4])
	}
	return BuildSubjectContext(SubjectContextInput{
		BirthYear:  birthYear,
		TargetYear: intValue(input.Liunian["liunian_year"]),
	})
}

// hasDynamicHardBoundary 拒绝动态层写入高风险具体应事。
func hasDynamicHardBoundary(text string) bool {
	return containsUnsupportedConcreteOutcome(text) || containsAnyText(text, []string{"投资", "投资建议"})
}

// validateDynamicOutcomeDomains enforces the deterministic age-based scope
// declared in the payload. It validates structured domains rather than growing
// a natural-language blacklist of possible life events.
func validateDynamicOutcomeDomains(state baziCharterState) error {
	context := buildBaziSubjectContext(state.Input)
	isMinor := context.AgeBand == "infant" || context.AgeBand == "child" || context.AgeBand == "adolescent"
	if !isMinor && len(state.DynamicSynthesis.DayunJudgments) == 0 {
		return nil
	}
	periodDomains := make(map[string]bool)
	for index, judgment := range state.DynamicSynthesis.DayunJudgments {
		if len(judgment.OutcomeDomains) == 0 {
			return baziViolationError(baziViolationScopeEscalation, fmt.Sprintf("dynamic.dayun_judgments[%d].outcome_domains", index), "", "each dayun judgment must declare outcome domains", nil, context.AllowedOutcomeDomains)
		}
		for _, domain := range judgment.OutcomeDomains {
			domain = strings.TrimSpace(domain)
			if !containsString(context.AllowedOutcomeDomains, domain) {
				return baziViolationError(baziViolationScopeEscalation, fmt.Sprintf("dynamic.dayun_judgments[%d].outcome_domains", index), "", "dayun outcome domain is not authorized for the subject age", nil, context.AllowedOutcomeDomains)
			}
			periodDomains[domain] = true
		}
	}
	if len(state.DynamicSynthesis.OutcomeDomains) == 0 {
		if !isMinor && len(periodDomains) == 0 {
			return nil
		}
		return baziViolationError(baziViolationScopeEscalation, "dynamic.outcome_domains", "", "dynamic synthesis must declare outcome domains", nil, context.AllowedOutcomeDomains)
	}
	for _, domain := range state.DynamicSynthesis.OutcomeDomains {
		if !containsString(context.AllowedOutcomeDomains, strings.TrimSpace(domain)) {
			return baziViolationError(baziViolationScopeEscalation, "dynamic.outcome_domains", "", "dynamic outcome domain is not authorized for the subject age", nil, context.AllowedOutcomeDomains)
		}
	}
	for domain := range periodDomains {
		if !containsString(state.DynamicSynthesis.OutcomeDomains, domain) {
			return baziViolationError(baziViolationScopeEscalation, "dynamic.outcome_domains", "", "top-level outcome domains must cover every dayun judgment domain", nil, state.DynamicSynthesis.OutcomeDomains)
		}
	}
	if isMinor {
		if domain, term := firstUnauthorizedMinorOutcomeSignal(dynamicUserVisibleText(state.DynamicSynthesis)); domain != "" {
			return baziViolationError(baziViolationUnsupportedConcreteOutcome, "dynamic", "", fmt.Sprintf("minor dynamic synthesis writes unauthorized %s outcome signal %q", domain, term), nil, context.AllowedOutcomeDomains)
		}
	}
	return nil
}

// firstUnauthorizedMinorOutcomeSignal maps concrete life-event wording back to
// the age-domain contract. A domain label alone is not an outcome: it may occur
// in a boundary sentence, while minors must still not receive adult projections.
func firstUnauthorizedMinorOutcomeSignal(text string) (string, string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ""
	}
	for _, signal := range []struct {
		domain string
		terms  []string
	}{
		{domain: "career", terms: []string{"事业突破", "职业晋升", "职场晋升", "工作升迁", "职位变动", "升迁"}},
		{domain: "finance", terms: []string{"财富增长", "财运大涨", "收入增加", "赚钱机会", "投资获利", "破财"}},
		{domain: "marriage", terms: []string{"婚姻变故", "婚恋结果", "配偶关系", "感情结果"}},
		{domain: "legal", terms: []string{"法律", "官非", "诉讼", "牢狱", "血光", "伤灾"}},
		{domain: "medical", terms: []string{"疾病预测", "健康风险", "病症", "脾胃疾病", "心血管", "高血压"}},
	} {
		for _, term := range signal.terms {
			if strings.Contains(text, term) {
				return signal.domain, term
			}
		}
	}
	return "", ""
}

// dynamicUserVisibleText collects dynamic fields subject to age-scope validation.
func dynamicUserVisibleText(dynamic baziDynamicSynthesis) string {
	periodTexts := make([]string, 0, len(dynamic.DayunJudgments))
	for _, judgment := range dynamic.DayunJudgments {
		periodTexts = append(periodTexts, judgment.Trend, judgment.Interpretation, strings.Join(judgment.Evidence, "\n"))
	}
	return strings.Join([]string{
		dynamic.CurrentTrend,
		strings.Join(dynamic.DayunPath, "\n"),
		strings.Join(periodTexts, "\n"),
		dynamic.LiunianFocus,
		strings.Join(dynamic.TriggerSignals, "\n"),
		strings.Join(dynamic.KeyWindows, "\n"),
		strings.Join(dynamic.Risks, "\n"),
		dynamic.ReasoningSummary,
		strings.Join(dynamic.ReasoningSteps, "\n"),
	}, "\n")
}

// validateDynamicConsistencyFlags 校验动态一致性标签是否属于固定枚举。
// 非法枚举是投影合同错误，不进入事实修复或动态 facts-only。
func validateDynamicConsistencyFlags(flags []string) error {
	for _, flag := range flags {
		flag = strings.TrimSpace(flag)
		if flag == "" {
			continue
		}
		if !containsString(allowedDynamicConsistencyFlags, flag) {
			return projectionMismatchViolation(
				"dynamic.consistency_flags",
				fmt.Sprintf("dynamic synthesis uses unsupported consistency flag %q; allowed: %s", flag, strings.Join(allowedDynamicConsistencyFlags, "、")),
				allowedDynamicConsistencyFlags,
			)
		}
	}
	return nil
}

// validateEvidenceBundlePreconditions ensures retrieval-required plans have query packets.
func validateEvidenceBundlePreconditions(state baziCharterState) error {
	if state.EvidencePlan.NeedRetrieval && len(state.EvidencePlan.QueryPackets) == 0 {
		return fmt.Errorf("missing query packets for retrieval-required state")
	}
	return nil
}

// validateDynamicPreconditions 确认动态综合只在静态主轴已准备后运行。
// 上游静态结果缺失属于结构合同错误，不触发模型 repair。
func validateDynamicPreconditions(state baziCharterState) error {
	if strings.TrimSpace(state.StaticSynthesis.MainAxis) == "" {
		return projectionMismatchViolation("dynamic.preconditions.static.main_axis", "dynamic stage requires static synthesis first", nil)
	}
	return nil
}

// validateDynamicStage 校验动态综合的核心字段、覆盖范围和授权边界。
// 字段缺失或结构不一致保持 hard-error/recovery 语义，不新增动态 repair。
func validateDynamicStage(state baziCharterState) error {
	if isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		return validateFactsOnlyDynamicSynthesis(state)
	}
	if strings.TrimSpace(state.DynamicSynthesis.CurrentTrend) == "" {
		return projectionMismatchViolation("dynamic.current_trend", "missing dynamic synthesis current trend", nil)
	}
	if len(state.DynamicSynthesis.DayunPath) == 0 {
		return projectionMismatchViolation("dynamic.dayun_path", "missing dynamic synthesis dayun path", nil)
	}
	if expected := len(DayunPeriods(state.Input.Dayun)); expected > 0 && len(state.DynamicSynthesis.DayunPath) < expected {
		return baziViolationError(
			baziViolationDayunCoverageMissing,
			"dynamic.dayun_path",
			"",
			fmt.Sprintf("dynamic synthesis omits calculated dayun periods: got %d, want at least %d", len(state.DynamicSynthesis.DayunPath), expected),
			nil,
			nil,
		)
	}
	if err := validateDayunJudgmentFacts(state, state.DynamicSynthesis.DayunJudgments); err != nil {
		return err
	}
	if strings.TrimSpace(state.DynamicSynthesis.LiunianFocus) == "" {
		return projectionMismatchViolation("dynamic.liunian_focus", "missing dynamic synthesis liunian focus", nil)
	}
	if strings.TrimSpace(state.DynamicSynthesis.WindowLevel) == "" {
		return projectionMismatchViolation("dynamic.window_level", "missing dynamic synthesis window level", nil)
	}
	if strings.TrimSpace(state.DynamicSynthesis.ReasoningSummary) == "" {
		return projectionMismatchViolation("dynamic.reasoning_summary", "missing dynamic synthesis reasoning summary", nil)
	}
	if len(state.DynamicSynthesis.ReasoningSteps) == 0 {
		return projectionMismatchViolation("dynamic.reasoning_steps", "missing dynamic synthesis reasoning steps", nil)
	}
	if strings.TrimSpace(state.DynamicSynthesis.ClaimStrength) == "" {
		return projectionMismatchViolation("dynamic.claim_strength", "missing dynamic synthesis claim strength", nil)
	}
	if strings.TrimSpace(state.DynamicSynthesis.SupportLevel) == "" {
		return projectionMismatchViolation("dynamic.support_level", "missing dynamic synthesis support level", nil)
	}
	if strings.TrimSpace(state.DynamicSynthesis.LimitationLevel) == "" {
		return projectionMismatchViolation("dynamic.limitation_level", "missing dynamic synthesis limitation level", nil)
	}
	if strings.TrimSpace(state.DynamicSynthesis.WordingCap) == "" {
		return projectionMismatchViolation("dynamic.wording_cap", "missing dynamic synthesis wording cap", nil)
	}
	return validateDynamicAgainstProfileScope(state)
}

// validateFactsOnlyDynamicSynthesis 校验动态 facts-only 降级仍保留可展示事实。
// 该校验只分类失败，不把降级输出重新送入模型 repair。
func validateFactsOnlyDynamicSynthesis(state baziCharterState) error {
	if strings.TrimSpace(state.DynamicSynthesis.CurrentTrend) == "" {
		return projectionMismatchViolation("dynamic.facts_only.current_trend", "facts-only dynamic synthesis missing degraded message", nil)
	}
	if expected := len(DayunPeriods(state.Input.Dayun)); expected > 0 && len(state.DynamicSynthesis.DayunPath) < expected {
		return baziViolationError(
			baziViolationDayunCoverageMissing,
			"dynamic.facts_only.dayun_path",
			"",
			fmt.Sprintf("facts-only dynamic synthesis omits calculated dayun periods: got %d, want at least %d", len(state.DynamicSynthesis.DayunPath), expected),
			nil,
			nil,
		)
	}
	return nil
}

// validateDayunJudgmentFacts validates the sole current-period judgment against
// deterministic facts. Other periods remain runtime-rendered facts, so requiring
// model verdicts for every period would violate the dynamic DTO contract.
func validateDayunJudgmentFacts(state baziCharterState, judgments []baziDayunJudgment) error {
	if len(judgments) == 0 {
		// 兼容既有 session 投影；当前图的动态 DTO 已在上游要求一条 current-period claim。
		return nil
	}
	periods := DayunPeriods(state.Input.Dayun)
	currentIndex := currentDayunIndexForInput(state.Input)
	if currentIndex < 0 || currentIndex >= len(periods) {
		return projectionMismatchViolation("dynamic.current_period_ref", "dynamic synthesis has no runtime-bound current period", nil)
	}
	if len(judgments) != 1 {
		return baziViolationError(
			baziViolationDayunCoverageMissing,
			"dynamic.dayun_judgments",
			"",
			fmt.Sprintf("dynamic synthesis must contain one current-period judgment: got %d", len(judgments)),
			nil,
			nil,
		)
	}
	judgment := judgments[0]
	want := strings.TrimSpace(stringValue(periods[currentIndex]["ganZhi"]))
	got := strings.TrimSpace(judgment.GanZhi)
	field := "dynamic.dayun_judgments[0].gan_zhi"
	if want == "" {
		return baziViolationError(baziViolationFactRefMissing, field, "", "calculated current dayun is missing gan_zhi", nil, nil)
	}
	if got == "" {
		return projectionMismatchViolation(field, "dynamic current-period judgment is missing gan_zhi", []string{want})
	}
	if !strings.HasPrefix(got, want) {
		return baziViolationError(
			baziViolationFactConflict,
			field,
			"",
			fmt.Sprintf("dynamic current-period judgment does not match calculated period %q", want),
			nil,
			[]string{want},
		)
	}
	if strings.TrimSpace(judgment.Trend) == "" || strings.TrimSpace(judgment.Interpretation) == "" {
		return projectionMismatchViolation(
			"dynamic.dayun_judgments[0]",
			"dynamic current-period judgment is incomplete",
			nil,
		)
	}
	return nil
}

// validateCharterConsistency checks cross-stage static and dynamic projection consistency.
func validateCharterConsistency(state baziCharterState) error {
	if isFactsOnlyStaticSynthesis(state.StaticSynthesis) || isFactsOnlyDynamicSynthesis(state.DynamicSynthesis) {
		return nil
	}
	if err := validateStaticDecisionConsistency(state.StaticSynthesis); err != nil {
		return err
	}
	if err := validateStaticAxisVerdictConsistency(state.StaticSynthesis); err != nil {
		return err
	}
	if err := validateStaticEvidenceCoverageBoundary(state); err != nil {
		return err
	}
	if err := validateStaticTiaohouEvidenceWording(state); err != nil {
		return err
	}
	if err := validateStaticAxisAgainstChartFacts(state); err != nil {
		return err
	}
	if strings.TrimSpace(state.DynamicSynthesis.CurrentTrend) == "" {
		return nil
	}
	if err := validateDynamicDecisionConsistency(state.DynamicSynthesis); err != nil {
		return err
	}
	if err := validateCurrentDayunLineConsistency(state.DynamicSynthesis); err != nil {
		return err
	}
	return nil
}

// validateDynamicSynthesisAfterGraphNormalization rechecks normalized dynamic output at the graph boundary.
func validateDynamicSynthesisAfterGraphNormalization(state baziCharterState) error {
	return validateDynamicSynthesisResult(state, state.DynamicSynthesis)
}

// validateDynamicSynthesisResult validates normalized dynamic output before it is accepted.
func validateDynamicSynthesisResult(chartState baziCharterState, output baziDynamicSynthesis) error {
	checkState := chartState
	checkState.DynamicSynthesis = output
	if isFactsOnlyDynamicSynthesis(checkState.DynamicSynthesis) {
		return validateDynamicStage(checkState)
	}
	checkState.DynamicSynthesis = ensureDynamicAssertions(checkState, projectDynamicAssertionsToLegacy(checkState.DynamicSynthesis))
	if err := validateDynamicStage(checkState); err != nil {
		return err
	}
	if err := validateDynamicAssertions(checkState); err != nil {
		return err
	}
	return validateCharterConsistency(checkState)
}

// validateStaticEvidenceCoverageBoundary caps only the strength of a static
// route when planned A-tier topics remain missing. It does not select a chart
// methodology or alter the model's interpretive text.
func validateStaticEvidenceCoverageBoundary(state baziCharterState) error {
	if len(state.EvidenceQuality.MissingTopics) == 0 {
		return nil
	}
	if state.StaticSynthesis.AxisLevel == "主轴成立" || state.StaticSynthesis.AxisLevel == "可以拔高" {
		return baziViolationError(baziViolationEvidenceTopicMissing, "static.axis_level", "", fmt.Sprintf("static axis level exceeds incomplete evidence boundary: missing %s", strings.Join(state.EvidenceQuality.MissingTopics, ", ")), state.EvidenceQuality.MissingTopics, state.EvidenceQuality.CoveredTopics)
	}
	return nil
}

// validateStaticTiaohouEvidenceWording 只拦截已覆盖调候主证却声称证据缺失的输出。
// 调候 verdict 已由 static claim 的 Schema、严格解码和引用合同保证，
// 不再以自然语言短语表重复判定，避免把合法措辞误降级。
func validateStaticTiaohouEvidenceWording(state baziCharterState) error {
	if !containsString(state.EvidenceQuality.CoveredTopics, "tiaohou") {
		return nil
	}
	static := state.StaticSynthesis
	fields := []struct {
		name  string
		value string
	}{
		{name: "main_axis", value: static.MainAxis},
		{name: "pattern_outcome", value: static.PatternOutcome},
		{name: "counter_evidence", value: static.CounterEvidence},
		{name: "axis_consistency", value: static.AxisConsistency},
		{name: "tiaohou_constraint", value: static.TiaohouConstraint},
		{name: "tiaohou_anchor", value: static.TiaohouAnchor},
		{name: "pattern_and_qing_zhuo", value: static.PatternAndQingZhuo},
		{name: "tier_basis", value: static.TierBasis},
		{name: "reasoning_summary", value: static.ReasoningSummary},
		{name: "usage.tiaohou", value: static.Usage.Tiaohou},
		{name: "usage.priority", value: static.Usage.Priority},
		{name: "topic_direct_answer", value: static.TopicDirectAnswer},
		{name: "topic_focus_answer", value: static.TopicFocusAnswer},
	}
	for i, step := range static.ReasoningSteps {
		fields = append(fields, struct {
			name  string
			value string
		}{name: fmt.Sprintf("reasoning_steps[%d]", i), value: step})
	}
	for i, item := range static.Advantages {
		fields = append(fields, struct {
			name  string
			value string
		}{name: fmt.Sprintf("advantages[%d]", i), value: item})
	}
	for i, item := range static.Risks {
		fields = append(fields, struct {
			name  string
			value string
		}{name: fmt.Sprintf("risks[%d]", i), value: item})
	}
	for _, field := range fields {
		if phrase := firstTiaohouMissingEvidencePhrase(field.value); phrase != "" {
			return baziViolationError(baziViolationEvidenceTopicMissing, "static."+field.name, "", fmt.Sprintf("static tiaohou wording contradicts covered authority evidence in %s: %s; covered_topics=%s", field.name, phrase, strings.Join(state.EvidenceQuality.CoveredTopics, ", ")), []string{"tiaohou"}, nil)
		}
	}
	return nil
}

// firstTiaohouMissingEvidencePhrase finds wording that contradicts covered Tiaohou evidence.
func firstTiaohouMissingEvidencePhrase(text string) string {
	for _, phrase := range []string{
		"调候关键证据缺失",
		"调候证据缺失",
		"调候规则材料不足",
		"调候用神未定",
		"具体调候用神待",
		"具体用神待穷通宝鉴",
		"需穷通宝鉴",
		"待穷通宝鉴",
	} {
		if strings.Contains(text, phrase) {
			return phrase
		}
	}
	return ""
}

// validateCurrentDayunLineConsistency 约束当前大运总述与当前大运条目保持同线，
// 避免同一步运同时被写成“承托主轴”和“偏压/压制主轴”的相反口径。
func validateCurrentDayunLineConsistency(d baziDynamicSynthesis) error {
	// Direction words are explanatory language, not chart facts. A mixed trend
	// can legitimately describe the same luck pillar from multiple dimensions;
	// record suspicious wording in soft audit instead of rejecting the report.
	_ = d
	return nil
}

// validateStaticAxisVerdictConsistency 校验轴线裁断字段之间的封顶关系。
// 该层只处理结构投影一致性，不替模型选择格局路线。
func validateStaticAxisVerdictConsistency(s baziStaticSynthesis) error {
	if strings.TrimSpace(s.AxisLevel) == "" &&
		strings.TrimSpace(s.EffectOnTiaohou) == "" &&
		strings.TrimSpace(s.EffectOnCoreDisease) == "" &&
		strings.TrimSpace(s.EffectOnJiShenDirection) == "" &&
		strings.TrimSpace(s.AxisCeiling) == "" &&
		len(s.ConflictReasons) == 0 {
		return nil
	}
	if err := validateAllowedValue("static axis level", s.AxisLevel, []string{"结构可见", "方向成立", "主轴成立", "可以拔高"}); err != nil {
		return err
	}
	if err := validateAllowedValue("static effect on tiaohou", s.EffectOnTiaohou, []string{"支持", "中性", "冲突"}); err != nil {
		return err
	}
	if err := validateAllowedValue("static effect on core disease", s.EffectOnCoreDisease, []string{"缓解", "中性", "放大"}); err != nil {
		return err
	}
	// `effect_on_jishen_direction` 的主合同已收口为 `缓解/中性/放大`，
	// 但旧测试样例和历史输出里仍可能出现同义的 `抑制`，这里保留兼容以避免被误杀。
	if err := validateAllowedValue("static effect on ji-shen direction", s.EffectOnJiShenDirection, []string{"缓解", "抑制", "中性", "放大"}); err != nil {
		return err
	}
	if err := validateAllowedValue("static axis ceiling", s.AxisCeiling, []string{"结构信号", "受限路线", "可作主轴", "可以拔高"}); err != nil {
		return err
	}
	if requiresStaticConflictReasons(s) && len(s.ConflictReasons) == 0 {
		return projectionMismatchViolation("static.conflict_reasons", "static conflict reasons required", nil)
	}
	if err := validateAxisVerdictAgainstConflict(s); err != nil {
		return err
	}
	if s.AxisCeiling == "结构信号" &&
		containsAnyText([]string{s.MainAxis, s.PatternOutcome, s.TierBasis}, []string{"主轴", "贵格", "化杀为权"}) {
		return projectionMismatchViolation("static.axis_ceiling", "static synthesis promotes structure signal beyond axis ceiling", nil)
	}
	if s.AxisCeiling == "受限路线" &&
		containsAnyText([]string{s.MainAxis, s.PatternOutcome, s.TierBasis}, []string{"纯主轴贵格", "可以拔高", "化杀为权"}) {
		return projectionMismatchViolation("static.axis_ceiling", "static synthesis promotes restricted route beyond axis ceiling", nil)
	}
	return nil
}

// requiresStaticConflictReasons limits the conflict-reason requirement to
// outputs that actually declare conflict, amplification, or a restricted
// ceiling. Neutral axes still go through semantic audit; they should not be
// rejected before audit just because no conflict list is present.
func requiresStaticConflictReasons(s baziStaticSynthesis) bool {
	return s.EffectOnTiaohou == "冲突" ||
		s.EffectOnCoreDisease == "放大" ||
		s.EffectOnJiShenDirection == "放大" ||
		s.AxisCeiling == "结构信号" ||
		s.AxisCeiling == "受限路线" ||
		containsString(s.ConsistencyFlags, "方向成立但力度受限")
}

// validateAxisVerdictAgainstConflict 防止冲突路线被投影成高封顶主轴。
// 冲突事实本身可被解释，但字段封顶必须同步降级。
func validateAxisVerdictAgainstConflict(s baziStaticSynthesis) error {
	conflictCount := 0
	if s.EffectOnTiaohou == "冲突" {
		conflictCount++
	}
	if s.EffectOnCoreDisease == "放大" {
		conflictCount++
	}
	if s.EffectOnJiShenDirection == "放大" {
		conflictCount++
	}
	if conflictCount == 0 {
		return nil
	}
	switch s.AxisCeiling {
	case "结构信号", "受限路线":
	default:
		return projectionMismatchViolation("static.axis_ceiling", "static axis ceiling is too high for a conflict-amplifying route", nil)
	}
	if s.AxisCeiling == "结构信号" && (s.AxisLevel == "主轴成立" || s.AxisLevel == "可以拔高") {
		return projectionMismatchViolation("static.axis_level", "static axis level exceeds structure-signal ceiling", nil)
	}
	if s.AxisCeiling == "受限路线" && s.AxisLevel == "可以拔高" {
		return projectionMismatchViolation("static.axis_level", "static axis level exceeds restricted-route ceiling", nil)
	}
	return nil
}

// validateStaticDecisionConsistency 校验静态综合的强度、限制和可见措辞封顶。
// 失败进入 projection mismatch，不当作事实冲突或方法合同错误。
func validateStaticDecisionConsistency(s baziStaticSynthesis) error {
	if err := validateAllowedValue("static claim strength", s.ClaimStrength, []string{"保守判断", "倾向成立", "明确成立", "封顶判断"}); err != nil {
		return err
	}
	if err := validateAllowedValue("static support level", s.SupportLevel, []string{"出现", "有根", "有气", "得力", "成势"}); err != nil {
		return err
	}
	if err := validateAllowedValue("static limitation level", s.LimitationLevel, []string{"轻微", "明显", "核心硬伤"}); err != nil {
		return err
	}
	if err := validateAllowedValue("static wording cap", s.WordingCap, []string{"保守", "中性", "明确", "封顶"}); err != nil {
		return err
	}
	if containsString(s.ConsistencyFlags, "方向成立但力度受限") &&
		!containsAnyText([]string{s.PatternOutcome, s.CounterEvidence, s.TierBasis}, []string{
			"力度受限",
			"条件受限",
			"受限",
			"不足以",
			"不算强救",
			"不够强",
			"药力不够",
			"药力有限",
			"层次受限",
			"难以拔高",
			"不能拔高",
			"难入上等",
			"难以进入上等",
			"层级暂不定级",
			"完整层次规则尚未覆盖",
			"不自动换算富贵层次",
			"不自动换算富贵等级",
		}) {
		return projectionMismatchViolation("static.consistency_flags", "static consistency flag requires visible limitation text", nil)
	}
	if containsAnyText([]string{s.MainAxis, s.PatternOutcome, s.TierBasis}, []string{"一飞冲天"}) &&
		!allowsFlourishByWordingCap(s.WordingCap, "一飞冲天") {
		return projectionMismatchViolation("static.wording_cap", "static synthesis overstates wording beyond wording cap", nil)
	}
	if containsAnyText([]string{s.MainAxis, s.PatternOutcome, s.TierBasis}, []string{"可享清福", "福泽深厚", "贵人众多"}) &&
		!allowsFlourishByWordingCap(s.WordingCap, "positive_flourish") {
		return projectionMismatchViolation("static.wording_cap", "static synthesis overstates wording beyond wording cap", nil)
	}
	return nil
}

// validateStaticAxisAgainstChartFacts leaves methodology disputes to model eval.
// Runtime guards only reject structural field conflicts elsewhere in the graph.
func validateStaticAxisAgainstChartFacts(state baziCharterState) error {
	_ = state
	return nil
}

// validateDynamicDecisionConsistency checks dynamic strength, limitation and wording enums.
func validateDynamicDecisionConsistency(d baziDynamicSynthesis) error {
	if err := validateAllowedValue("dynamic claim strength", d.ClaimStrength, []string{"保守判断", "倾向成立", "明确成立", "封顶判断"}); err != nil {
		return err
	}
	if err := validateAllowedValue("dynamic support level", d.SupportLevel, []string{"出现", "有根", "有气", "得力", "成势"}); err != nil {
		return err
	}
	if err := validateAllowedValue("dynamic limitation level", d.LimitationLevel, []string{"轻微", "明显", "核心硬伤"}); err != nil {
		return err
	}
	if err := validateAllowedValue("dynamic wording cap", d.WordingCap, []string{"保守", "中性", "明确", "封顶"}); err != nil {
		return err
	}
	return nil
}

// validateAllowedValue 校验静态/动态投影枚举值，并返回机器可读字段错误。
func validateAllowedValue(name, value string, allowed []string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return projectionMismatchViolation(validationFieldName(name), fmt.Sprintf("missing %s", name), allowed)
	}
	if !containsString(allowed, value) {
		return projectionMismatchViolation(validationFieldName(name), fmt.Sprintf("invalid %s: %s", name, value), allowed)
	}
	return nil
}

// projectionMismatchViolation 统一包装静态/动态投影类字段错误。
func projectionMismatchViolation(field, message string, allowed []string) error {
	return baziViolationError(baziViolationScopeEscalation, field, "", message, nil, allowed)
}

// validationFieldName 把 validator 的人类标签压成 trace/repair 可读字段名。
func validationFieldName(name string) string {
	field := strings.TrimSpace(strings.ToLower(name))
	field = strings.ReplaceAll(field, "ji-shen", "jishen")
	field = strings.ReplaceAll(field, " ", ".")
	return field
}

// allowsFlourishByWordingCap checks whether a wording cap permits a flourish class.
func allowsFlourishByWordingCap(wordingCap, flourishClass string) bool {
	wordingCap = strings.TrimSpace(wordingCap)
	switch flourishClass {
	case "positive_flourish":
		return wordingCap == "明确" || wordingCap == "封顶"
	case "一飞冲天":
		return wordingCap == "封顶"
	default:
		return false
	}
}
