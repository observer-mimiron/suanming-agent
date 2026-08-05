// Package runtime 包含 Manager 拥有的八字静态反馈与校验入口。
//
// 本文件负责静态综合的字段级反馈、确定性校验和 retry 后降级；
// 不拥有最终答复权，也不让 specialist 绕过 Manager 流程。
package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
)

// runStaticSynthesisWithFeedback 在静态综合未通过合同时给模型一次结构化反馈。
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
		payload["static_feedback"] = buildStaticSynthesisFeedback(chartState, output, err)
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
		if recovered, recoverErr := recoverStaticSynthesisAfterRetry(chartState, output, err); recoverErr == nil {
			return recovered, nil
		}
		return baziStaticSynthesis{}, err
	}
	output.Source = "model"
	return output, nil
}

// recoverStaticSynthesisAfterRetry 用确定性 facts-only 替换二次仍不安全的静态候选。
// 候选文本必须丢弃，避免把未通过合同的模型判断交给 renderer。
func recoverStaticSynthesisAfterRetry(chartState baziCharterState, output baziStaticSynthesis, cause error) (baziStaticSynthesis, error) {
	failure, ok := baziContractFailureFromError("static_synthesis", cause)
	if !ok || failure.RecoveryPolicy != baziRecoveryPolicyStaticFactsOnly {
		return baziStaticSynthesis{}, cause
	}
	recovered := recoverStaticSynthesis(chartState, output, cause)
	recovered.ContractAudit = output.ContractAudit
	recovered.FieldAudit = append(recovered.FieldAudit,
		"contract_failure_class:"+failure.Class,
		"recovery_policy:"+failure.RecoveryPolicy,
	)
	if err := validateStaticSynthesisResult(chartState, recovered); err != nil {
		return baziStaticSynthesis{}, err
	}
	return recovered, nil
}

// acceptPartialStaticSynthesisAfterRetry 只接受 renderer 可省略字段的二次缺漏。
// 事实冲突、方法合同和核心裁断缺失仍返回错误，不能用模型文本冒充安全输出。
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

// validateStaticSynthesisWithAudit 先跑确定性校验，再让语义审计评估候选。
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

// validateStaticStrengthAgainstEvidence 防止模型反写 runtime 已计算的强弱方向。
// 中和附近仍交给综合判断；只有“偏强/偏弱”显式反转才进入机器可读恢复口。
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
			return baziViolationError(baziViolationFactConflict, "static.strength_balance", "", fmt.Sprintf("static strength reverses balance evidence: %s", strength), nil, []string{strength})
		}
	case "偏强":
		if strings.Contains(reading, "偏弱") || strings.Contains(reading, "身弱") {
			return baziViolationError(baziViolationFactConflict, "static.strength_balance", "", fmt.Sprintf("static strength reverses balance evidence: %s", strength), nil, []string{strength})
		}
	}
	return nil
}

func buildStaticSynthesisFeedback(chartState baziCharterState, output baziStaticSynthesis, cause error) string {
	lines := []string{
		"请严格按当前结构化裁定重写静态综合，不得保留与 axis_level / axis_ceiling 冲突的升级措辞。",
		"若某条路线只到“结构信号”或“受限路线”，main_axis、pattern_outcome、tier_basis 就不得再写成“化杀为权”“贵格已成”“可以拔高”或同级升级结论。",
		"若存在调候冲突、病点放大或忌神方向放大，必须把限制写进 pattern_outcome、counter_evidence、tier_basis，明确落成“方向成立但力度受限”“不宜拔高”“只能作受限路线参考”。",
	}
	if failure, ok := baziContractFailureFromError("static_synthesis", cause); ok && failure.Class == baziContractFailureEvidenceOverclaim {
		lines = append(lines,
			fmt.Sprintf("本轮 evidence_quality.enough=%t；missing_topics=%s。缺失主题不能支持确定性病药、清浊或命格层次硬断。", chartState.EvidenceQuality.Enough, strings.Join(chartState.EvidenceQuality.MissingTopics, "、")),
			"若问题字段是 static.tier 或 kind=tier assertion：必须同时重写 tier_judgment、tier_basis、assertions[].verdict、assertions[].boundary。",
			"允许写法：tier_judgment 写“命格层次中等（保守定位）”或更低；tier_basis 写明缺失主题与保守封顶标准。",
			"禁止写法：不得输出“中上”“上等”“中等偏上”“可以拔高”等正向高层次，也不得写“暂不定级”回避用户要的层次判断。",
		)
		if failure.Field != "" {
			lines = append(lines, "本次证据越权字段："+failure.Field)
		}
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
