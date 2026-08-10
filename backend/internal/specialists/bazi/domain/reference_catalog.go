// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责把引用 ID 和已计算关系文本投影为稳定的模型输入目录；
// 不读取 SessionState、不执行合同校验，也不决定模型是否可重试。
package domain

import (
	"sort"
	"strings"
)

// ReferenceCatalogInput 是构建模型引用目录所需的窄输入。
// runtime 负责从本轮状态提取 ID、规则分类和关系文本，本包只负责稳定投影。
type ReferenceCatalogInput struct {
	FactIDs         []string
	ClaimCategories map[string]string
	RelationTexts   map[string]string
}

// ReferenceCatalogEntry 是模型选择引用时使用的只读索引项。
// Hint 仅帮助定位输入事实，模型回传时仍只能使用 ID。
type ReferenceCatalogEntry struct {
	ID   string `json:"id"`
	Hint string `json:"hint"`
}

// BuildReferenceCatalogView 将引用目录投影为稳定顺序的模型输入 DTO。
func BuildReferenceCatalogView(input ReferenceCatalogInput) map[string]any {
	return map[string]any{
		"fact_refs":     referenceEntries(input.FactIDs, factHint),
		"claim_refs":    referenceEntries(sortedKeys(input.ClaimCategories), func(id string) string { return claimHint(input.ClaimCategories[id]) }),
		"relation_refs": referenceEntries(sortedKeys(input.RelationTexts), func(id string) string { return relationHint(id, input.RelationTexts[id]) }),
	}
}

// referenceEntries 保持稳定顺序，避免 map 遍历影响模型输入和回归快照。
func referenceEntries(ids []string, hint func(string) string) []ReferenceCatalogEntry {
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	entries := make([]ReferenceCatalogEntry, 0, len(sorted))
	for _, id := range sorted {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		entries = append(entries, ReferenceCatalogEntry{ID: id, Hint: hint(id)})
	}
	return entries
}

// sortedKeys 返回稳定排序的非空 map key。
func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// factHint 将事实 ID 映射到模型输入中的稳定字段位置，不生成命理结论。
func factHint(id string) string {
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

// claimHint 将规则引用定位到已声明的规则分类，不扩写规则结论。
func claimHint(category string) string {
	return "输入 selected_rule_profile：" + firstNonEmpty(category, "已声明规则 claim")
}

// relationHint 将已计算关系文本投影为短定位说明，空文本只保留关系来源提示。
func relationHint(id, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		switch {
		case strings.HasPrefix(id, "relation.natal."):
			return "输入 core_chart：原局已计算关系"
		case strings.HasPrefix(id, "relation.liunian."):
			return "输入 dynamic_facts.liunian：已计算关系"
		case strings.HasPrefix(id, "relation.dayun."):
			return "输入 dynamic_facts.dayun：对应大运已计算关系"
		}
		return "输入中的已计算关系"
	}
	return "已计算关系：" + value
}
