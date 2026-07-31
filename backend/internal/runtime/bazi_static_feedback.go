package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (e *Executor) runStaticSynthesisWithFeedback(chartState baziCharterState, run func(map[string]any) (baziStaticSynthesis, error)) (baziStaticSynthesis, error) {
	payload := buildStaticSynthesisPayload(chartState)
	output, err := run(payload)
	if err != nil {
		return baziStaticSynthesis{}, err
	}
	output = ensureStaticAssertions(chartState, projectStaticAssertionsToLegacy(normalizeStaticSynthesis(output)))
	if err := validateStaticSynthesisResult(chartState, output); err == nil {
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
	if err := validateStaticSynthesisResult(chartState, output); err != nil {
		recovered := recoverStaticSynthesis(chartState, output, err)
		if recoverErr := validateStaticSynthesisResult(chartState, recovered); recoverErr != nil {
			return baziStaticSynthesis{}, recoverErr
		}
		return recovered, nil
	}
	output.Source = "model"
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
	if errText := strings.TrimSpace(cause.Error()); errText != "" {
		lines = append(lines, "本次校验失败原因："+errText)
	}
	if violation, ok := baziViolationFromError(cause); ok {
		if raw, err := json.Marshal(violation); err == nil {
			lines = append(lines, "机器可读 violation："+string(raw))
		}
	}
	return strings.Join(lines, "\n")
}
