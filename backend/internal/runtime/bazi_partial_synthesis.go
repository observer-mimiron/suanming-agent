package runtime

import "strings"

const baziSynthesisSourceModelPartial = "model_partial"

// isPartialSynthesisSource reports whether the model output was accepted after
// retry with only display-owned omissions. Validators still reject fact,
// methodology, scope, and semantic audit failures before this state is reached.
func isPartialSynthesisSource(source string) bool {
	return strings.TrimSpace(source) == baziSynthesisSourceModelPartial
}

// isOmittableStaticSynthesisError allows only missing presentation details to
// become partial output. Core static judgments remain fatal because hiding them
// would make the reading look complete without a valid method spine.
func isOmittableStaticSynthesisError(cause error) bool {
	message := strings.TrimSpace(errorText(cause))
	if !strings.HasPrefix(message, "missing static synthesis ") {
		return false
	}
	for _, field := range []string{
		"pattern basis",
		"tiaohou anchor",
		"pattern and qingzhuo",
		"tier basis",
		"reasoning summary",
		"reasoning steps",
		"claim strength",
		"support level",
		"limitation level",
		"wording cap",
		"axis level",
		"effect on tiaohou",
		"effect on core disease",
		"effect on ji-shen direction",
		"axis ceiling",
		"conflict reasons",
	} {
		if strings.Contains(message, field) {
			return true
		}
	}
	return false
}

// validatePartialStaticSynthesis reruns the fatal static contracts while
// allowing renderer-owned detail omissions. This prevents a missing display
// field from masking fact conflicts, method-contract failures, or missing core
// judgments after the retry.
func validatePartialStaticSynthesis(chartState baziCharterState, output baziStaticSynthesis) error {
	checkState := chartState
	checkState.StaticSynthesis = ensureStaticAssertions(chartState, projectStaticAssertionsToLegacy(normalizeStaticSynthesis(output)))
	if err := validatePartialStaticCore(checkState); err != nil {
		return err
	}
	if err := validateStaticAssertions(checkState); err != nil {
		return err
	}
	if err := validateStaticStrengthAgainstEvidence(checkState); err != nil {
		return err
	}
	return validatePartialStaticConsistency(checkState)
}

func validatePartialStaticCore(state baziCharterState) error {
	static := state.StaticSynthesis
	if state.Input.RuleProfile.ID != "" && static.RuleProfile != "" && static.RuleProfile != state.Input.RuleProfile.ID {
		return synthesisCoreError("static synthesis rule profile does not match selected profile")
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "main axis", value: static.MainAxis},
		{name: "pattern outcome", value: static.PatternOutcome},
		{name: "counter evidence", value: static.CounterEvidence},
		{name: "axis consistency", value: static.AxisConsistency},
		{name: "tier judgment", value: static.TierJudgment},
	} {
		if strings.TrimSpace(field.value) == "" {
			return missingSynthesisCoreError("static", field.name)
		}
	}
	if strings.Contains(static.TierJudgment, "层级暂不定级") {
		return missingSynthesisCoreError("static", "valid tier judgment")
	}
	return nil
}

func validatePartialStaticConsistency(state baziCharterState) error {
	static := state.StaticSynthesis
	if err := validateAllowedValueIfPresent("static claim strength", static.ClaimStrength, []string{"保守判断", "倾向成立", "明确成立", "封顶判断"}); err != nil {
		return err
	}
	if err := validateAllowedValueIfPresent("static support level", static.SupportLevel, []string{"出现", "有根", "有气", "得力", "成势"}); err != nil {
		return err
	}
	if err := validateAllowedValueIfPresent("static limitation level", static.LimitationLevel, []string{"轻微", "明显", "核心硬伤"}); err != nil {
		return err
	}
	if err := validateAllowedValueIfPresent("static wording cap", static.WordingCap, []string{"保守", "中性", "明确", "封顶"}); err != nil {
		return err
	}
	if err := validateAllowedValueIfPresent("static axis level", static.AxisLevel, []string{"结构可见", "方向成立", "主轴成立", "可以拔高"}); err != nil {
		return err
	}
	if err := validateAllowedValueIfPresent("static effect on tiaohou", static.EffectOnTiaohou, []string{"支持", "中性", "冲突"}); err != nil {
		return err
	}
	if err := validateAllowedValueIfPresent("static effect on core disease", static.EffectOnCoreDisease, []string{"缓解", "中性", "放大"}); err != nil {
		return err
	}
	if err := validateAllowedValueIfPresent("static effect on ji-shen direction", static.EffectOnJiShenDirection, []string{"缓解", "抑制", "中性", "放大"}); err != nil {
		return err
	}
	if err := validateAllowedValueIfPresent("static axis ceiling", static.AxisCeiling, []string{"结构信号", "受限路线", "可作主轴", "可以拔高"}); err != nil {
		return err
	}
	if containsString(static.ConsistencyFlags, "方向成立但力度受限") &&
		!containsAnyText([]string{static.PatternOutcome, static.CounterEvidence, static.TierBasis}, []string{"力度受限", "条件受限", "受限", "不足以", "不算强救", "不够强", "药力不够", "药力有限", "层次受限", "难以拔高", "不能拔高", "难入上等", "难以进入上等"}) {
		return missingSynthesisCoreError("static", "visible limitation text")
	}
	if strings.TrimSpace(static.AxisLevel) != "" {
		if err := validateStaticEvidenceCoverageBoundary(state); err != nil {
			return err
		}
	}
	if static.AxisCeiling == "结构信号" &&
		containsAnyText([]string{static.MainAxis, static.PatternOutcome, static.TierBasis}, []string{"主轴", "贵格", "化杀为权"}) {
		return synthesisCoreError("static partial synthesis promotes structure signal beyond axis ceiling")
	}
	if static.AxisCeiling == "受限路线" &&
		containsAnyText([]string{static.MainAxis, static.PatternOutcome, static.TierBasis}, []string{"纯主轴贵格", "可以拔高", "化杀为权"}) {
		return synthesisCoreError("static partial synthesis promotes restricted route beyond axis ceiling")
	}
	if staticHasAxisVerdict(static) {
		relaxed := static
		if requiresStaticConflictReasons(relaxed) && len(relaxed.ConflictReasons) == 0 {
			relaxed.ConflictReasons = []string{"partial omission accepted after retry"}
		}
		if err := validateStaticAxisVerdictConsistency(relaxed); err != nil {
			return err
		}
	}
	if err := validateStaticAxisAgainstChartFacts(state); err != nil {
		return err
	}
	return nil
}

func staticHasAxisVerdict(static baziStaticSynthesis) bool {
	return strings.TrimSpace(static.AxisLevel) != "" &&
		strings.TrimSpace(static.EffectOnTiaohou) != "" &&
		strings.TrimSpace(static.EffectOnCoreDisease) != "" &&
		strings.TrimSpace(static.EffectOnJiShenDirection) != "" &&
		strings.TrimSpace(static.AxisCeiling) != ""
}

// isOmittableDynamicSynthesisError mirrors the static rule for dynamic text:
// missing explanatory metadata can be omitted, while period coverage, fact
// conflicts, adult-domain overreach, and semantic audit failures stay fatal.
func isOmittableDynamicSynthesisError(cause error) bool {
	message := strings.TrimSpace(errorText(cause))
	if !strings.HasPrefix(message, "missing dynamic synthesis ") {
		return false
	}
	for _, field := range []string{
		"window level",
		"reasoning summary",
		"reasoning steps",
		"claim strength",
		"support level",
		"limitation level",
		"wording cap",
	} {
		if strings.Contains(message, field) {
			return true
		}
	}
	return false
}

// acceptPartialDynamicSynthesisAfterRetry keeps dynamic text only when the
// retry is missing display metadata but still satisfies period, fact, scope,
// assertion, and static-ceiling contracts.
func acceptPartialDynamicSynthesisAfterRetry(chartState baziCharterState, output baziDynamicSynthesis, cause error) (baziDynamicSynthesis, bool) {
	if !isOmittableDynamicSynthesisError(cause) {
		return baziDynamicSynthesis{}, false
	}
	output = ensureDynamicAssertions(chartState, projectDynamicAssertionsToLegacy(normalizeDynamicSynthesis(output)))
	if err := validatePartialDynamicSynthesis(chartState, output); err != nil {
		return baziDynamicSynthesis{}, false
	}
	output.Source = baziSynthesisSourceModelPartial
	output.RecoveryReason = recoveryReasonText(cause, "动态综合存在局部缺漏，已省略对应展示块。")
	output.FieldAudit = append(output.FieldAudit, "dynamic_partial_omitted:"+partialSynthesisReason(cause))
	return output, true
}

// validatePartialDynamicSynthesis mirrors validateDynamicSynthesisResult while
// allowing only presentation metadata omissions. It still rejects missing
// periods, forged relations, unauthorized outcome domains, and hard contract
// violations.
func validatePartialDynamicSynthesis(chartState baziCharterState, output baziDynamicSynthesis) error {
	checkState := chartState
	checkState.DynamicSynthesis = ensureDynamicAssertions(chartState, projectDynamicAssertionsToLegacy(normalizeDynamicSynthesis(output)))
	if err := validatePartialDynamicCore(checkState); err != nil {
		return err
	}
	if err := validateDynamicAssertions(checkState); err != nil {
		return err
	}
	if err := validateDynamicAgainstProfileScope(checkState); err != nil {
		return err
	}
	if err := validatePartialDynamicConsistency(checkState.DynamicSynthesis); err != nil {
		return err
	}
	if err := validateCurrentDayunLineConsistency(checkState.DynamicSynthesis); err != nil {
		return err
	}
	return validateDynamicAgainstStaticCeiling(checkState.StaticSynthesis, checkState.DynamicSynthesis)
}

func validatePartialDynamicCore(state baziCharterState) error {
	dynamic := state.DynamicSynthesis
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "current trend", value: dynamic.CurrentTrend},
		{name: "liunian focus", value: dynamic.LiunianFocus},
	} {
		if strings.TrimSpace(field.value) == "" {
			return missingSynthesisCoreError("dynamic", field.name)
		}
	}
	if len(dynamic.DayunPath) == 0 {
		return missingSynthesisCoreError("dynamic", "dayun path")
	}
	if expected := len(dayunPeriods(state.Input.Dayun)); expected > 0 && len(dynamic.DayunPath) < expected {
		return missingSynthesisCoreError("dynamic", "complete dayun path")
	}
	return validateDayunJudgmentFacts(state.Input.Dayun, dynamic.DayunJudgments)
}

func validatePartialDynamicConsistency(dynamic baziDynamicSynthesis) error {
	if err := validateAllowedValueIfPresent("dynamic claim strength", dynamic.ClaimStrength, []string{"保守判断", "倾向成立", "明确成立", "封顶判断"}); err != nil {
		return err
	}
	if err := validateAllowedValueIfPresent("dynamic support level", dynamic.SupportLevel, []string{"出现", "有根", "有气", "得力", "成势"}); err != nil {
		return err
	}
	if err := validateAllowedValueIfPresent("dynamic limitation level", dynamic.LimitationLevel, []string{"轻微", "明显", "核心硬伤"}); err != nil {
		return err
	}
	return validateAllowedValueIfPresent("dynamic wording cap", dynamic.WordingCap, []string{"保守", "中性", "明确", "封顶"})
}

func validateAllowedValueIfPresent(label, value string, allowed []string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return validateAllowedValue(label, value, allowed)
}

func missingSynthesisCoreError(stage, field string) error {
	return synthesisCoreError(stage + " partial synthesis missing required " + field)
}

func partialSynthesisReason(cause error) string {
	message := strings.TrimSpace(errorText(cause))
	if message == "" {
		return "unknown"
	}
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\t", " ")
	return message
}

func errorText(cause error) string {
	if cause == nil {
		return ""
	}
	return cause.Error()
}

type synthesisCoreError string

func (e synthesisCoreError) Error() string { return string(e) }
