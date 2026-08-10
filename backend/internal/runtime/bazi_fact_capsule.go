// Package runtime 包含 Manager 拥有的八字运行时合同。
//
// 本文件只把 runtime 的图状态转换为 Bazi domain 的窄事实输入；
// 事实计算、可读投影和证据覆盖规则由 domain 负责，不承担模型或图编排。
package runtime

import bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"

// BaziFactCapsule 保留 runtime 的兼容名称，实际类型由 Bazi domain 所有。
type BaziFactCapsule = bazidomain.FactCapsule

// baziTierEvidenceTopics 保留 runtime 合同校验使用的稳定主题顺序。
var baziTierEvidenceTopics = bazidomain.TierEvidenceTopics()

// buildBaziFactCapsule 把已选月令、当前大运和证据覆盖传给 domain 构建事实胶囊。
func buildBaziFactCapsule(state baziCharterState) BaziFactCapsule {
	return bazidomain.BuildFactCapsule(baziFactInput(state))
}

// buildBaziFactCapsulePromptView 把 domain 事实胶囊投影成模型可读的中文字段。
func buildBaziFactCapsulePromptView(state baziCharterState, includeDynamic bool) map[string]any {
	return bazidomain.BuildPromptView(baziFactInput(state), includeDynamic)
}

// baziFactInput 提取 domain 不应依赖的 runtime 状态与当前大运选择。
func baziFactInput(state baziCharterState) bazidomain.FactInput {
	currentPeriod := map[string]any{}
	periods := dayunPeriods(state.Input.Dayun)
	if index := currentDayunIndexForInput(state.Input); index >= 0 && index < len(periods) {
		currentPeriod = periods[index]
	}
	return bazidomain.FactInput{
		BaziResult:       state.Input.BaziResult,
		Yongshen:         state.Input.Yongshen,
		MonthCommand:     monthBranchForEvidenceQuery(state.Input),
		CurrentPeriodRef: currentDayunPeriodRef(state),
		CurrentPeriod:    currentPeriod,
		CoveredTopics:    state.EvidenceQuality.CoveredTopics,
	}
}

// capsuleOfficialDisplay 保留 runtime 文案调用点并委托给 domain。
func capsuleOfficialDisplay(capsule BaziFactCapsule) string {
	return bazidomain.OfficialDisplay(capsule)
}

// capsuleFireDisplay 保留 runtime 文案调用点并委托给 domain。
func capsuleFireDisplay(capsule BaziFactCapsule) string {
	return bazidomain.FireDisplay(capsule)
}

// capsuleTiaohouDisplay 保留 runtime 文案调用点并委托给 domain。
func capsuleTiaohouDisplay(capsule BaziFactCapsule) string {
	return bazidomain.TiaohouDisplay(capsule)
}

// capsuleTierEvidenceDisplay 保留 runtime 文案调用点并委托给 domain。
func capsuleTierEvidenceDisplay(capsule BaziFactCapsule) string {
	return bazidomain.TierEvidenceDisplay(capsule)
}

// tierEvidenceComplete 将 runtime 的证据审计输入交给 domain 判断覆盖是否完整。
func tierEvidenceComplete(state baziCharterState) bool {
	return bazidomain.TierEvidenceComplete(state.EvidenceQuality.CoveredTopics)
}

// tierEvidenceMissing 返回 runtime 当前证据审计尚未覆盖的独立主题。
func tierEvidenceMissing(state baziCharterState) []string {
	return bazidomain.TierEvidenceMissing(state.EvidenceQuality.CoveredTopics)
}
