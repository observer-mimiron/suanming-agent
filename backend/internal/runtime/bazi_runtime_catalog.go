// Package runtime 包含 Manager 拥有的八字运行时事实引用目录。
//
// 本文件从确定性输入和规则材料派生模型可引用的 ID，并提供只读定位说明；
// 不推断关系、不生成 claim、不决定修复，也不渲染输出。
package runtime

import (
	"fmt"
	"sort"
	"strings"
)

// baziRuntimeCatalog is the per-turn allow-list for model provenance references.
type baziRuntimeCatalog struct {
	Facts     map[string]struct{}
	Claims    map[string]struct{}
	Relations map[string]struct{}
}

// baziCatalogEntry 是给模型选择引用的只读索引。hint 仅帮助定位输入事实，
// 输出合同仍只允许回传 ID，不能把 hint 当作事实字段回传。
type baziCatalogEntry struct {
	ID   string `json:"id"`
	Hint string `json:"hint"`
}

// buildBaziRuntimeCatalog derives all allowed reference IDs before model output is validated.
func buildBaziRuntimeCatalog(state baziCharterState) baziRuntimeCatalog {
	catalog := baziRuntimeCatalog{
		Facts: knownBaziFactRefs(state), Claims: knownBaziClaimRefs(state.Input.RuleProfile), Relations: map[string]struct{}{},
	}
	for index, period := range dayunPeriods(state.Input.Dayun) {
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

// validateBaziReferenceCatalog rejects unregistered IDs before semantic validators run.
func validateBaziReferenceCatalog(state baziCharterState, assertions []baziAssertion) error {
	return validateBaziReferenceCatalogAgainst(assertions, buildBaziRuntimeCatalog(state))
}

// validateStaticBaziReferenceCatalog accepts only the deterministic facts that
// belong to a natal judgment. Dynamic period facts are intentionally absent.
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
	for _, assertion := range assertions {
		for _, ref := range assertion.FactRefs {
			if !isKnownBaziFactRef(ref, catalog.Facts) {
				return baziViolationError(baziViolationUndeclaredFactClaim, "assertions.fact_refs", assertion.ID, "fact_ref is not declared in this runtime catalog", []string{string(ref)}, sortedCatalogIDs(catalog.Facts))
			}
		}
		for _, ref := range assertion.ClaimRefs {
			if _, ok := catalog.Claims[strings.TrimSpace(string(ref))]; !ok {
				return baziViolationError(baziViolationUndeclaredFactClaim, "assertions.claim_refs", assertion.ID, "claim_ref is not declared in this runtime catalog", []string{string(ref)}, sortedCatalogIDs(catalog.Claims))
			}
		}
		for _, ref := range assertion.RelationRefs {
			id := strings.TrimSpace(string(ref))
			if _, ok := catalog.Relations[id]; !ok {
				return baziViolationError(baziViolationUndeclaredFactClaim, "assertions.relation_refs", assertion.ID, "relation_ref is not declared in this runtime catalog", []string{id}, sortedCatalogIDs(catalog.Relations))
			}
		}
	}
	return nil
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

// baziRuntimeCatalogView 将本轮允许的引用以 ID 加定位说明暴露给模型。
// 只给裸 ID 会迫使模型猜编号；hint 来自同轮确定性输入，输出校验仍只接受 ID。
func baziRuntimeCatalogView(state baziCharterState) map[string]any {
	return baziRuntimeCatalogViewFor(state, buildBaziRuntimeCatalog(state))
}

// baziStaticRuntimeCatalogView exposes the exact static allow-list used by the
// static validator. Relation selectors are intentionally absent from this DTO.
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
	periods := dayunPeriods(state.Input.Dayun)
	refs := make([]string, len(periods))
	for index := range periods {
		refs[index] = fmt.Sprintf("dayun[%d]", index)
	}
	return refs
}

// currentDayunPeriodRef exposes the runtime-selected period without replacing
// the complete period directory used for deterministic display.
func currentDayunPeriodRef(state baziCharterState) string {
	if index := currentDayunIndexForInput(state.Input); index >= 0 {
		return fmt.Sprintf("dayun[%d]", index)
	}
	return ""
}

// baziRuntimeCatalogViewFor 将指定 allow-list 以 ID 和定位说明投影给模型。
func baziRuntimeCatalogViewFor(state baziCharterState, catalog baziRuntimeCatalog) map[string]any {
	return map[string]any{
		"fact_refs":     baziCatalogEntries(sortedCatalogIDs(catalog.Facts), func(id string) string { return baziFactRefHint(state, id) }),
		"claim_refs":    baziCatalogEntries(sortedCatalogIDs(catalog.Claims), func(id string) string { return baziClaimRefHint(state, id) }),
		"relation_refs": baziCatalogEntries(sortedCatalogIDs(catalog.Relations), func(id string) string { return baziRelationRefHint(state, id) }),
	}
}

// baziCatalogEntries 保持稳定顺序，避免模型输入和回归快照受 map 遍历影响。
func baziCatalogEntries(ids []string, hint func(string) string) []baziCatalogEntry {
	entries := make([]baziCatalogEntry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, baziCatalogEntry{ID: id, Hint: hint(id)})
	}
	return entries
}

// baziFactRefHint 将事实 ID 对应到输入中的稳定字段位置，不创建额外命理结论。
func baziFactRefHint(state baziCharterState, id string) string {
	switch id {
	case "chart.pillars":
		return "输入 core_chart.pillars：四柱"
	case "chart.day_master":
		return "输入 core_chart.day_master：日主"
	case "chart.month_branch", "chart.month_pillar":
		return "输入 core_chart.month_pillar：月令/月柱"
	case "yongshen.strength", "yongshen.strength_evidence", "yongshen.balance_status":
		return "输入 core_chart：强弱与受力证据"
	case "yongshen.season", "yongshen.seasonal_tiaohou_hint", "yongshen.tiao_hou", "yongshen.tiaohou_yong_shen":
		return "输入 core_chart：季节与调候材料"
	case "yongshen.geju", "yongshen.geju_basis", "yongshen.geju_candidate", "yongshen.geju_detail", "yongshen.geju_status":
		return "输入 core_chart：格局候选与依据"
	case "fact_capsule.month_command", "fact_capsule.root_positions", "fact_capsule.visible_same_element_stems",
		"fact_capsule.month_score", "fact_capsule.root_count", "fact_capsule.same_element_count",
		"fact_capsule.resource_support_count", "fact_capsule.support_score", "fact_capsule.pressure_score",
		"fact_capsule.support_signals", "fact_capsule.pressure_signals", "fact_capsule.official_visible",
		"fact_capsule.official_hidden", "fact_capsule.fire_present", "fact_capsule.fire_visible",
		"fact_capsule.fire_effective", "fact_capsule.fire_effectiveness_known", "fact_capsule.core_facts_ready",
		"fact_capsule.tier_evidence_complete", "fact_capsule.tier_evidence_missing":
		return "输入 fact_capsule：已计算的裁断前提"
	case "liunian.year", "liunian.gan_zhi", "liunian.stem", "liunian.branch", "liunian.shi_shen", "liunian.current_dayun":
		return "输入 dynamic_facts.liunian：本轮流年计算事实"
	}
	if strings.HasPrefix(id, "dayun[") {
		return "输入 dynamic_facts.dayun：对应序号大运的计算事实"
	}
	return "输入 core_chart 或 dynamic_facts 的已计算字段：" + id
}

// baziClaimRefHint 将规则 claim 定位到 selected_rule_profile，不暴露或扩写规则结论。
func baziClaimRefHint(state baziCharterState, id string) string {
	for _, claim := range state.Input.RuleProfile.Claims {
		if id == claim.ID || id == claim.RuleID {
			return "输入 selected_rule_profile：" + firstNonEmptyTrim(claim.Category, "已声明规则 claim")
		}
	}
	return "输入 selected_rule_profile 的已声明规则 claim"
}

// baziRelationRefHint 将关系 ID 定位到同轮已计算关系文本，只供模型选择 ID。
func baziRelationRefHint(state baziCharterState, id string) string {
	if index, ok := baziRelationIndex(id, "relation.natal."); ok {
		return firstNonEmptyTrim(relationHint(relationTextList(state.Input.Yongshen["chonghe"]), index), "输入 core_chart：原局已计算关系")
	}
	if index, ok := baziRelationIndex(id, "relation.liunian."); ok {
		return firstNonEmptyTrim(relationHint(relationTextList(state.Input.Liunian["liunian_chonghe"]), index), "输入 dynamic_facts.liunian：已计算关系")
	}
	if periodIndex, relationIndex, ok := baziDayunRelationIndex(id); ok {
		periods := dayunPeriods(state.Input.Dayun)
		if periodIndex < len(periods) {
			return firstNonEmptyTrim(relationHint(relationTextList(periods[periodIndex]["dayun_chonghe"]), relationIndex), "输入 dynamic_facts.dayun：对应大运已计算关系")
		}
	}
	return "输入中的已计算关系"
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

// relationHint 返回已计算关系的短说明，空值交由上层回退为字段定位。
func relationHint(relations []string, index int) string {
	if index < 0 || index >= len(relations) {
		return ""
	}
	return "已计算关系：" + strings.TrimSpace(relations[index])
}
