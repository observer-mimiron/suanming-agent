// Package domain 包含八字静态合同失败的领域反馈。
//
// 本文件负责校验 canonical 投影后的静态字段、断言和强弱事实一致性；
// 不负责模型重试、零值补全、partial 接受或最终答复渲染。
package domain

// validateStaticSynthesisResult 校验 canonical 投影后的静态合同，不做别名改写或零值补全。
func validateStaticSynthesisResult(chartState baziCharterState, output baziStaticSynthesis) error {
	checkState := chartState
	checkState.StaticSynthesis = output
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
	return ValidateStaticStrengthAgainstEvidence(state.Input.Yongshen, state.StaticSynthesis)
}
