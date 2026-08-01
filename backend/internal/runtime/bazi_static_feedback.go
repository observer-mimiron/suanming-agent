package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
)

// runStaticSynthesisWithFeedback gives the generator one structured retry when
// deterministic or independent semantic contracts reject its first output.
func (e *Executor) runStaticSynthesisWithFeedback(chartState baziCharterState, run func(map[string]any) (baziStaticSynthesis, error), audits ...func(baziStaticSynthesis) (baziContractAudit, error)) (baziStaticSynthesis, error) {
	audit := func(baziStaticSynthesis) (baziContractAudit, error) { return baziContractAudit{Compliant: true}, nil }
	if len(audits) > 0 && audits[0] != nil {
		audit = audits[0]
	}
	payload := buildStaticSynthesisPayload(chartState)
	output, err := run(payload)
	if err != nil {
		return baziStaticSynthesis{}, err
	}
	output = ensureStaticAssertions(chartState, projectStaticAssertionsToLegacy(normalizeStaticSynthesis(output)))
	if output, err = validateStaticSynthesisWithAudit(chartState, output, audit); err == nil {
		output.Source = "model"
		return output, nil
	} else {
		payload["static_feedback"] = buildStaticSynthesisFeedback(output, err)
	}

	output, err = run(payload)
	if err != nil {
		return baziStaticSynthesis{}, err
	}
	output = ensureStaticAssertions(chartState, projectStaticAssertionsToLegacy(normalizeStaticSynthesis(output)))
	if output, err = validateStaticSynthesisWithAudit(chartState, output, audit); err != nil {
		if partial, ok := acceptPartialStaticSynthesisAfterRetry(chartState, output, err); ok {
			return partial, nil
		}
		return baziStaticSynthesis{}, err
	}
	output.Source = "model"
	return output, nil
}

// acceptPartialStaticSynthesisAfterRetry keeps a usable model reading when the
// second attempt is missing only renderer-owned detail fields. Fact conflicts,
// methodology violations, and missing core judgments still return errors so the
// runtime does not replace a failed reading with invented facts-only prose.
func acceptPartialStaticSynthesisAfterRetry(chartState baziCharterState, output baziStaticSynthesis, cause error) (baziStaticSynthesis, bool) {
	if !isOmittableStaticSynthesisError(cause) {
		return baziStaticSynthesis{}, false
	}
	output = ensureStaticAssertions(chartState, projectStaticAssertionsToLegacy(normalizeStaticSynthesis(output)))
	if err := validatePartialStaticSynthesis(chartState, output); err != nil {
		return baziStaticSynthesis{}, false
	}
	output.Source = "model_partial"
	output.RecoveryReason = recoveryReasonText(cause, "静态综合存在局部缺漏，已省略对应展示块。")
	output.FieldAudit = append(output.FieldAudit, "static_partial_omitted:"+partialSynthesisReason(cause))
	return output, true
}

// validateStaticSynthesisWithAudit runs deterministic validation first so the
// semantic reviewer only evaluates a structurally valid candidate.
func validateStaticSynthesisWithAudit(chartState baziCharterState, output baziStaticSynthesis, audit func(baziStaticSynthesis) (baziContractAudit, error)) (baziStaticSynthesis, error) {
	if err := validateStaticSynthesisResult(chartState, output); err != nil {
		return output, err
	}
	result, err := audit(output)
	output.ContractAudit = result
	if err != nil {
		return output, err
	}
	if err := validateBaziContractAudit("static", result); err != nil {
		return output, err
	}
	return output, nil
}

func validateStaticSynthesisResult(chartState baziCharterState, output baziStaticSynthesis) error {
	checkState := chartState
	checkState.StaticSynthesis = normalizeStaticSynthesis(output)
	if isFactsOnlyStaticSynthesis(checkState.StaticSynthesis) {
		return validateStaticStage(checkState)
	}
	checkState.StaticSynthesis = ensureStaticAssertions(checkState, projectStaticAssertionsToLegacy(checkState.StaticSynthesis))
	if err := validateStaticStage(checkState); err != nil {
		return err
	}
	if err := validateStaticAssertions(checkState); err != nil {
		return err
	}
	if err := validateStaticStrengthAgainstEvidence(checkState); err != nil {
		return err
	}
	return validateCharterConsistency(checkState)
}

// validateStaticStrengthAgainstEvidence prevents a model from reversing a
// decisive runtime-owned balance result. The middle band remains open to
// synthesis; only explicit "偏强" versus "偏弱" reversals are rejected.
func validateStaticStrengthAgainstEvidence(state baziCharterState) error {
	strength := strings.TrimSpace(stringValue(state.Input.Yongshen["strength"]))
	reading := strings.Join([]string{
		strings.TrimSpace(state.StaticSynthesis.Strength.Conclusion),
		strings.TrimSpace(state.StaticSynthesis.StrengthBalance),
	}, "\n")
	if strength == "" || reading == "" {
		return nil
	}
	switch strength {
	case "偏弱":
		if strings.Contains(reading, "偏强") || strings.Contains(reading, "身强") {
			return fmt.Errorf("static strength reverses balance evidence: %s", strength)
		}
	case "偏强":
		if strings.Contains(reading, "偏弱") || strings.Contains(reading, "身弱") {
			return fmt.Errorf("static strength reverses balance evidence: %s", strength)
		}
	}
	return nil
}

func buildStaticSynthesisFeedback(output baziStaticSynthesis, cause error) string {
	lines := []string{
		"请严格按当前结构化裁定重写静态综合，不得保留与 axis_level / axis_ceiling 冲突的升级措辞。",
		"若某条路线只到“结构信号”或“受限路线”，main_axis、pattern_outcome、tier_basis 就不得再写成“化杀为权”“贵格已成”“可以拔高”或同级升级结论。",
		"若存在调候冲突、病点放大或忌神方向放大，必须把限制写进 pattern_outcome、counter_evidence、tier_basis，明确落成“方向成立但力度受限”“不宜拔高”“只能作受限路线参考”。",
	}
	if ceiling := strings.TrimSpace(output.AxisCeiling); ceiling != "" {
		lines = append(lines, fmt.Sprintf("本轮你自己给出的 axis_ceiling 是“%s”，自然语言结论必须服从这一天花板。", ceiling))
	}
	if strings.TrimSpace(output.AxisLevel) != "" || strings.TrimSpace(output.AxisCeiling) != "" {
		lines = append(lines, fmt.Sprintf(
			"本轮结构字段回显：axis_level=%s；axis_ceiling=%s；effect_on_tiaohou=%s；effect_on_core_disease=%s；effect_on_jishen_direction=%s。若后三项任一为“冲突”或“放大”，axis_ceiling 只能是“结构信号”或“受限路线”；这是字段闭合要求，不是在替你选择格局。",
			firstNonEmptyTrim(output.AxisLevel, "未填"),
			firstNonEmptyTrim(output.AxisCeiling, "未填"),
			firstNonEmptyTrim(output.EffectOnTiaohou, "未填"),
			firstNonEmptyTrim(output.EffectOnCoreDisease, "未填"),
			firstNonEmptyTrim(output.EffectOnJiShenDirection, "未填"),
		))
	}
	if errText := strings.TrimSpace(cause.Error()); errText != "" {
		lines = append(lines, "本次校验失败原因："+errText)
	}
	if violation, ok := baziViolationFromError(cause); ok {
		if raw, err := json.Marshal(violation); err == nil {
			lines = append(lines, "机器可读 violation："+string(raw))
		}
	}
	if len(output.ContractAudit.Findings) > 0 {
		if raw, err := json.Marshal(output.ContractAudit.Findings); err == nil {
			lines = append(lines, "独立审计已确认的 findings（只修复这些字段，不要把检查过程当作结论）："+string(raw))
		}
	}
	return strings.Join(lines, "\n")
}
