// Package domain 包含八字事实胶囊与模型视图合同。
//
// 本文件只把 runtime 的图状态转换为 Bazi domain 的窄事实输入；
// 事实计算、可读投影和证据覆盖规则由 domain 负责，不承担模型或图编排。
package domain

import "strings"

// BaziFactCapsule 保留 runtime 的兼容名称，实际类型由 Bazi domain 所有。
type BaziFactCapsule = FactCapsule

// buildBaziFactCapsule 把已选月令和当前大运传给 domain 构建事实胶囊。
func buildBaziFactCapsule(state baziCharterState) BaziFactCapsule {
	return BuildFactCapsule(baziFactInput(state))
}

// buildBaziFactCapsulePromptView 把 domain 事实胶囊投影成模型可读的中文字段。
func buildBaziFactCapsulePromptView(state baziCharterState, includeDynamic bool) map[string]any {
	return BuildPromptView(baziFactInput(state), includeDynamic)
}

// baziFactInput 提取 domain 不应依赖的 runtime 状态与当前大运选择。
func baziFactInput(state baziCharterState) FactInput {
	currentPeriod := map[string]any{}
	periods := DayunPeriods(state.Input.Dayun)
	if index := currentDayunIndexForInput(state.Input); index >= 0 && index < len(periods) {
		currentPeriod = periods[index]
	}
	return FactInput{
		BaziResult:       state.Input.BaziResult,
		Yongshen:         state.Input.Yongshen,
		MonthCommand:     MonthBranchForEvidenceQuery(state.Input),
		CurrentPeriodRef: currentDayunPeriodRef(state),
		CurrentPeriod:    currentPeriod,
	}
}

// capsuleOfficialDisplay 保留 runtime 文案调用点并委托给 domain。
func capsuleOfficialDisplay(capsule BaziFactCapsule) string {
	return OfficialDisplay(capsule)
}

// capsuleFireDisplay 保留 runtime 文案调用点并委托给 domain。
func capsuleFireDisplay(capsule BaziFactCapsule) string {
	return FireDisplay(capsule)
}

// capsuleTiaohouDisplay 保留 runtime 文案调用点并委托给 domain。
func capsuleTiaohouDisplay(capsule BaziFactCapsule) string {
	return TiaohouDisplay(capsule)
}

// MonthBranchForEvidenceQuery 从月柱或扶抑事实中读取月令地支。
func MonthBranchForEvidenceQuery(input baziCharterInput) string {
	if pillar := ExtractMonthPillar(input.BaziResult["pillars"]); len(pillar) > 0 {
		if branch := strings.TrimSpace(stringValue(pillar["branch"])); branch != "" {
			return branch
		}
	}
	return firstNonEmpty(stringValue(input.Yongshen["month_branch"]), stringValue(input.Yongshen["month_zhi"]))
}
