// Package domain 包含八字事实引用目录与允许列表。
//
// 本文件从确定性输入和规则材料派生模型可引用的 ID，并把 runtime 状态适配为目录 DTO；
// 不生成模型提示、不推断关系、不生成 claim、不决定修复，也不渲染输出。
package domain

import (
	"fmt"
	"sort"
	"strings"
)

// baziRuntimeCatalog 是单轮模型引用的 allow-list；它只用于 runtime 合同校验。
type baziRuntimeCatalog = ReferenceCatalog

// buildBaziRuntimeCatalog 在模型输出校验前派生本轮全部允许引用 ID。
func buildBaziRuntimeCatalog(state baziCharterState) baziRuntimeCatalog {
	catalog := baziRuntimeCatalog{
		Facts: knownBaziFactRefs(state), Claims: knownBaziClaimRefs(state.Input.RuleProfile), Relations: map[string]struct{}{},
	}
	for index, period := range DayunPeriods(state.Input.Dayun) {
		for relationIndex := range relationTextList(period["dayun_chonghe"]) {
			catalog.Relations[fmt.Sprintf("relation.dayun.%d.%d", index, relationIndex)] = struct{}{}
		}
	}
	for relationIndex := range relationTextList(state.Input.Liunian["liunian_chonghe"]) {
		catalog.Relations[fmt.Sprintf("relation.liunian.%d", relationIndex)] = struct{}{}
	}
	for relationIndex := range relationTextList(state.Input.Yongshen["chonghe"]) {
		catalog.Relations[fmt.Sprintf("relation.natal.%d", relationIndex)] = struct{}{}
	}
	return catalog
}

// validateBaziReferenceCatalog 在语义校验前拒绝未注册的引用 ID。
func validateBaziReferenceCatalog(state baziCharterState, assertions []baziAssertion) error {
	return validateBaziReferenceCatalogAgainst(assertions, buildBaziRuntimeCatalog(state))
}

// validateStaticBaziReferenceCatalog 只接受本命裁断所属的确定性事实，主动排除动态事实。
func validateStaticBaziReferenceCatalog(state baziCharterState, assertions []baziAssertion) error {
	return validateBaziReferenceCatalogAgainst(assertions, buildStaticBaziRuntimeCatalog(state))
}

// validateDynamicBaziReferenceCatalog 只接受动态节点本轮实际拿到的引用集合。
// 这使 prompt 清单与 Go 门禁一致，避免未注入的静态 ID 成为成功旁路。
func validateDynamicBaziReferenceCatalog(state baziCharterState, assertions []baziAssertion) error {
	return validateBaziReferenceCatalogAgainst(assertions, buildDynamicBaziRuntimeCatalog(state))
}

// validateBaziReferenceCatalogAgainst 用同一份节点 allow-list 校验模型回传的三类引用。
func validateBaziReferenceCatalogAgainst(assertions []baziAssertion, catalog baziRuntimeCatalog) error {
	return ValidateReferenceCatalog(assertions, catalog)
}

// buildDynamicBaziRuntimeCatalog 只保留动态裁断可直接引用的岁运、流年与关系事实。
// 静态主轴作为输入前提传入，不再要求动态节点重复选择静态事实或规则 claim。
func buildDynamicBaziRuntimeCatalog(state baziCharterState) baziRuntimeCatalog {
	full := buildBaziRuntimeCatalog(state)
	catalog := baziRuntimeCatalog{Facts: map[string]struct{}{}, Claims: map[string]struct{}{}, Relations: map[string]struct{}{}}
	for id := range full.Facts {
		if strings.HasPrefix(id, "dayun[") || strings.HasPrefix(id, "liunian.") {
			catalog.Facts[id] = struct{}{}
		}
	}
	for id := range full.Relations {
		if strings.HasPrefix(id, "relation.dayun.") || strings.HasPrefix(id, "relation.liunian.") {
			catalog.Relations[id] = struct{}{}
		}
	}
	return catalog
}

// buildStaticBaziRuntimeCatalog removes dynamic facts and all relation IDs from
// a natal judgment. Static relations do not decide a claim slot and previously
// invited the model to invent ordinal IDs instead of using chart facts.
func buildStaticBaziRuntimeCatalog(state baziCharterState) baziRuntimeCatalog {
	full := buildBaziRuntimeCatalog(state)
	catalog := baziRuntimeCatalog{Facts: map[string]struct{}{}, Claims: map[string]struct{}{}, Relations: map[string]struct{}{}}
	for id := range full.Facts {
		if strings.HasPrefix(id, "dayun[") || strings.HasPrefix(id, "liunian.") {
			continue
		}
		catalog.Facts[id] = struct{}{}
	}
	for id := range full.Claims {
		catalog.Claims[id] = struct{}{}
	}
	return catalog
}

func sortedCatalogIDs(catalog map[string]struct{}) []string {
	ids := make([]string, 0, len(catalog))
	for id := range catalog {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// knownBaziClaimRefs 返回规则 profile 已声明、模型可引用的 claim ID。
func knownBaziClaimRefs(profile baziRuleProfile) map[string]struct{} {
	out := map[string]struct{}{}
	for _, claim := range profile.Claims {
		if strings.TrimSpace(claim.ID) != "" {
			out[claim.ID] = struct{}{}
		}
		if strings.TrimSpace(claim.RuleID) != "" {
			out[claim.RuleID] = struct{}{}
		}
	}
	for _, verdict := range profile.Verdicts {
		if strings.TrimSpace(verdict.RuleID) != "" {
			out[verdict.RuleID] = struct{}{}
		}
	}
	return out
}

// knownBaziFactRefs 返回本轮模型可引用的确定性事实 ID。
// 事实胶囊字段与命盘字段同时保留，避免模型用自由别名替代稳定引用。
func knownBaziFactRefs(state baziCharterState) map[string]struct{} {
	out := map[string]struct{}{
		"chart.day_gan": {}, "chart.day_master": {}, "chart.day_master_wuxing": {}, "chart.month_branch": {},
		"chart.month_pillar": {}, "chart.pillars": {}, "chart.wuxing": {},
		"fact_capsule.month_command": {}, "fact_capsule.root_positions": {}, "fact_capsule.visible_same_element_stems": {},
		"fact_capsule.month_score": {}, "fact_capsule.root_count": {}, "fact_capsule.same_element_count": {},
		"fact_capsule.resource_support_count": {}, "fact_capsule.support_score": {}, "fact_capsule.pressure_score": {},
		"fact_capsule.support_signals": {}, "fact_capsule.pressure_signals": {},
		"fact_capsule.official_visible": {}, "fact_capsule.official_hidden": {},
		"fact_capsule.fire_present": {}, "fact_capsule.fire_visible": {}, "fact_capsule.fire_effective": {},
		"fact_capsule.fire_effectiveness_known": {}, "fact_capsule.core_facts_ready": {},
		"fact_capsule.tier_evidence_complete": {}, "fact_capsule.tier_evidence_missing": {},
		"yongshen.balance_status": {}, "yongshen.balance_yong_shen": {}, "yongshen.conditional_yong_shen": {},
		"yongshen.day_master": {}, "yongshen.day_master_wuxing": {}, "yongshen.geju": {},
		"yongshen.geju_basis": {}, "yongshen.geju_candidate": {}, "yongshen.geju_combination": {},
		"yongshen.geju_detail": {}, "yongshen.geju_qing_zhuo": {}, "yongshen.geju_qing_zhuo_reason": {},
		"yongshen.geju_status": {}, "yongshen.ji_shen": {}, "yongshen.official_visibility": {},
		"yongshen.official_visibility.visible": {}, "yongshen.official_visibility.hidden": {},
		"yongshen.season": {}, "yongshen.seasonal_tiaohou_hint": {}, "yongshen.shi_shen_power": {},
		"yongshen.strength": {}, "yongshen.strength_evidence": {}, "yongshen.strength_method": {},
		"yongshen.tiao_hou": {}, "yongshen.tiaohou_yong_shen": {}, "yongshen.xi_shen": {}, "yongshen.yong_shen": {},
		"liunian.branch": {}, "liunian.current_dayun": {}, "liunian.gan_zhi": {}, "liunian.relations": {},
		"liunian.shen_sha": {}, "liunian.shi_shen": {}, "liunian.stem": {}, "liunian.year": {},
	}
	for i := range DayunPeriods(state.Input.Dayun) {
		for _, field := range []string{
			"end_age", "end_at_exclusive", "branch", "branch_hidden_stems", "branch_ten_gods",
			"branch_main_ten_god", "gan_zhi", "period_id", "relations", "sequence", "start_age",
			"start_at", "ten_god",
		} {
			out[fmt.Sprintf("dayun[%d].%s", i, field)] = struct{}{}
		}
	}
	return out
}

// isKnownBaziFactRef 只接受本轮 catalog 中逐字声明的完整 fact ID。
func isKnownBaziFactRef(ref baziFactRef, known map[string]struct{}) bool {
	_, ok := known[strings.TrimSpace(string(ref))]
	return ok
}

// baziRuntimeCatalogView 将本轮允许的引用以 ID 加定位说明暴露给模型。
// 只给裸 ID 会迫使模型猜编号；hint 来自同轮确定性输入，输出校验仍只接受 ID。
func baziRuntimeCatalogView(state baziCharterState) map[string]any {
	return baziRuntimeCatalogViewFor(state, buildBaziRuntimeCatalog(state))
}

// baziStaticRuntimeCatalogView 暴露静态校验使用的精确 allow-list，主动移除关系选择器。
func baziStaticRuntimeCatalogView(state baziCharterState) map[string]any {
	view := baziRuntimeCatalogViewFor(state, buildStaticBaziRuntimeCatalog(state))
	delete(view, "relation_refs")
	return view
}

// baziDynamicRuntimeCatalogView 将动态节点唯一允许的引用集合注入 prompt。
func baziDynamicRuntimeCatalogView(state baziCharterState) map[string]any {
	view := baziRuntimeCatalogViewFor(state, buildDynamicBaziRuntimeCatalog(state))
	view["period_refs"] = baziDynamicPeriodRefs(state)
	view["current_period_ref"] = currentDayunPeriodRef(state)
	return view
}

// baziDynamicPeriodRefs 返回本轮动态节点可选择的确定性大运定位符。
// period_ref 只负责选择 catalog 项，干支、年龄和日期仍由 runtime 生成。
func baziDynamicPeriodRefs(state baziCharterState) []string {
	periods := DayunPeriods(state.Input.Dayun)
	refs := make([]string, len(periods))
	for index := range periods {
		refs[index] = fmt.Sprintf("dayun[%d]", index)
	}
	return refs
}

// currentDayunPeriodRef 暴露 runtime 选中的当前大运，不替换用于确定性展示的完整目录。
func currentDayunPeriodRef(state baziCharterState) string {
	if index := currentDayunIndexForInput(state.Input); index >= 0 {
		return fmt.Sprintf("dayun[%d]", index)
	}
	return ""
}

// baziRuntimeCatalogViewFor 将 runtime 状态适配为领域目录 DTO，再交给领域包稳定投影。
func baziRuntimeCatalogViewFor(state baziCharterState, catalog baziRuntimeCatalog) map[string]any {
	return BuildReferenceCatalogView(ReferenceCatalogInput{
		FactIDs:         sortedCatalogIDs(catalog.Facts),
		ClaimCategories: baziClaimCategories(state, catalog.Claims),
		RelationTexts:   baziRelationTexts(state, catalog.Relations),
	})
}

// baziClaimCategories 从规则 profile 提取目录中允许的分类，不把完整规则结论带入 DTO。
func baziClaimCategories(state baziCharterState, allowed map[string]struct{}) map[string]string {
	categories := make(map[string]string, len(allowed))
	for id := range allowed {
		for _, claim := range state.Input.RuleProfile.Claims {
			if id == claim.ID || id == claim.RuleID {
				categories[id] = strings.TrimSpace(claim.Category)
				break
			}
		}
		if _, ok := categories[id]; !ok {
			categories[id] = ""
		}
	}
	return categories
}

// baziRelationTexts 从同轮确定性输入提取关系文本，空值交给领域 DTO 使用来源回退。
func baziRelationTexts(state baziCharterState, allowed map[string]struct{}) map[string]string {
	texts := make(map[string]string, len(allowed))
	for id := range allowed {
		texts[id] = baziRelationText(state, id)
	}
	return texts
}

// baziRelationText 将关系 ID 定位到同轮已计算关系文本。
func baziRelationText(state baziCharterState, id string) string {
	if index, ok := baziRelationIndex(id, "relation.natal."); ok {
		return relationTextAt(state.Input.Yongshen["chonghe"], index)
	}
	if index, ok := baziRelationIndex(id, "relation.liunian."); ok {
		return relationTextAt(state.Input.Liunian["liunian_chonghe"], index)
	}
	if periodIndex, relationIndex, ok := baziDayunRelationIndex(id); ok {
		periods := DayunPeriods(state.Input.Dayun)
		if periodIndex < len(periods) {
			return relationTextAt(periods[periodIndex]["dayun_chonghe"], relationIndex)
		}
	}
	return ""
}

// baziRelationIndex 解析单层关系 ID 的末尾序号。
func baziRelationIndex(id, prefix string) (int, bool) {
	if !strings.HasPrefix(id, prefix) {
		return 0, false
	}
	var index int
	if _, err := fmt.Sscanf(strings.TrimPrefix(id, prefix), "%d", &index); err != nil {
		return 0, false
	}
	return index, true
}

// baziDayunRelationIndex 解析 relation.dayun.<period>.<relation> 的确定性定位。
func baziDayunRelationIndex(id string) (int, int, bool) {
	var periodIndex, relationIndex int
	if _, err := fmt.Sscanf(id, "relation.dayun.%d.%d", &periodIndex, &relationIndex); err != nil {
		return 0, 0, false
	}
	return periodIndex, relationIndex, true
}

// relationTextAt 返回指定关系文本，越界时保留空值让领域投影使用通用来源提示。
func relationTextAt(raw any, index int) string {
	relations := relationTextList(raw)
	if index < 0 || index >= len(relations) {
		return ""
	}
	return strings.TrimSpace(relations[index])
}
