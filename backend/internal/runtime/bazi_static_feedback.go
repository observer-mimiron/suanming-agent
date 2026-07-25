package runtime

import (
	"fmt"
	"strings"
)

func (e *Executor) runStaticSynthesisWithFeedback(chartState baziCharterState, run func(map[string]any) (baziStaticSynthesis, error)) (baziStaticSynthesis, error) {
	payload := buildStaticSynthesisPayload(chartState)
	output, err := run(payload)
	if err != nil {
		return baziStaticSynthesis{}, err
	}
	output = normalizeStaticSynthesis(output)
	if err := validateStaticSynthesisResult(chartState, output); err == nil {
		return output, nil
	} else {
		payload["static_feedback"] = buildStaticSynthesisFeedback(output, err)
	}

	output, err = run(payload)
	if err != nil {
		return baziStaticSynthesis{}, err
	}
	output = normalizeStaticSynthesis(output)
	if err := validateStaticSynthesisResult(chartState, output); err != nil {
		recovered := recoverStaticSynthesis(chartState, output, err)
		if recoverErr := validateStaticSynthesisResult(chartState, recovered); recoverErr != nil {
			return baziStaticSynthesis{}, recoverErr
		}
		return recovered, nil
	}
	return output, nil
}

func validateStaticSynthesisResult(chartState baziCharterState, output baziStaticSynthesis) error {
	checkState := chartState
	checkState.StaticSynthesis = normalizeStaticSynthesis(output)
	if err := validateStaticStage(checkState); err != nil {
		return err
	}
	return validateCharterConsistency(checkState)
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
	return strings.Join(lines, "\n")
}
