// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责事实胶囊及其可读模型视图；不依赖 runtime、SessionState、模型客户端、trace 或 SSE。
package domain

import (
	"fmt"
	"strings"
)

var stemElements = map[string]string{
	"甲": "木", "乙": "木", "丙": "火", "丁": "火", "戊": "土",
	"己": "土", "庚": "金", "辛": "金", "壬": "水", "癸": "水",
}

// FactInput 是构建一次事实胶囊所需的最小确定性输入。
// runtime 传入已选月令和当前大运，因此本包无需知道 ExecutionPlan 或 SessionState。
type FactInput struct {
	BaziResult       map[string]any
	Yongshen         map[string]any
	MonthCommand     string
	CurrentPeriodRef string
	CurrentPeriod    map[string]any
}

// RootPosition 记录日主通根，不把通根数量升级成格局或层次结论；层级沿用藏干确定顺序。
type RootPosition struct {
	Pillar string `json:"pillar"`
	Branch string `json:"branch"`
	Tier   string `json:"tier"`
}

// FactCapsule 是提供给语义模型节点的确定性事实与资格合同，不包含模型生成的裁断文本。
type FactCapsule struct {
	MonthCommand            string         `json:"month_command"`
	RootPositions           []RootPosition `json:"root_positions"`
	VisibleSameElementStems []string       `json:"visible_same_element_stems"`
	MonthScore              int            `json:"month_score"`
	RootCount               int            `json:"root_count"`
	SameElementCount        int            `json:"same_element_count"`
	ResourceSupportCount    int            `json:"resource_support_count"`
	SupportScore            int            `json:"support_score"`
	PressureScore           int            `json:"pressure_score"`
	SupportSignals          []string       `json:"support_signals"`
	PressureSignals         []string       `json:"pressure_signals"`
	OfficialVisible         bool           `json:"official_visible"`
	OfficialHidden          bool           `json:"official_hidden"`
	FirePresent             bool           `json:"fire_present"`
	FireVisible             bool           `json:"fire_visible"`
	FireEffective           bool           `json:"fire_effective"`
	FireEffectivenessKnown  bool           `json:"fire_effectiveness_known"`
	CoreFactsReady          bool           `json:"core_facts_ready"`
	CurrentPeriodRef        string         `json:"current_period_ref"`
	CurrentPeriodGanZhi     string         `json:"current_period_ganzhi"`
	CurrentPeriodRelations  []string       `json:"current_period_relations"`
}

// BuildFactCapsule 只从窄输入派生确定性事实，不生成命理裁断。
func BuildFactCapsule(input FactInput) FactCapsule {
	strength := mapValue(input.Yongshen, "strength_evidence")
	official := mapValue(input.Yongshen, "official_visibility")
	fire := mapValue(input.Yongshen, "fire_status")
	if len(fire) == 0 {
		fire = mapValue(input.Yongshen, "tiaohou_fire")
	}
	dayMaster := firstNonEmpty(
		stringValue(input.Yongshen["day_master"]),
		stringValue(input.BaziResult["dayGan"]),
	)
	rootPositions, sameStems, firePresent, fireVisible := capsulePillarFacts(input.BaziResult["pillars"], dayMaster)
	fireEffective, fireKnown := capsuleFireEffectiveness(fire)
	return FactCapsule{
		MonthCommand:            strings.TrimSpace(input.MonthCommand),
		RootPositions:           rootPositions,
		VisibleSameElementStems: sameStems,
		MonthScore:              intValue(input.Yongshen["month_score"]),
		RootCount:               intValue(input.Yongshen["root_count"]),
		SameElementCount:        intValue(input.Yongshen["same_element"]),
		ResourceSupportCount:    intValue(input.Yongshen["generate_count"]),
		SupportScore:            intValue(strength["support_score"]),
		PressureScore:           intValue(strength["pressure_score"]),
		SupportSignals:          stringSlice(strength["support_signals"]),
		PressureSignals:         stringSlice(strength["pressure_signals"]),
		OfficialVisible:         len(anyMapSlice(official["visible"])) > 0,
		OfficialHidden:          len(anyMapSlice(official["hidden"])) > 0,
		FirePresent:             firePresent || truthyFact(fire["present"]) || strings.TrimSpace(stringValue(fire["presence"])) != "",
		FireVisible:             fireVisible || truthyFact(fire["visible"]) || strings.TrimSpace(stringValue(fire["visible_stem"])) != "",
		FireEffective:           fireEffective,
		FireEffectivenessKnown:  fireKnown,
		CoreFactsReady:          strings.TrimSpace(input.MonthCommand) != "" && dayMaster != "" && len(input.BaziResult) > 0,
		CurrentPeriodRef:        strings.TrimSpace(input.CurrentPeriodRef),
		CurrentPeriodGanZhi:     strings.TrimSpace(stringValue(input.CurrentPeriod["ganZhi"])),
		CurrentPeriodRelations:  relationTextList(input.CurrentPeriod["dayun_chonghe"]),
	}
}

// BuildPromptView 将确定性事实胶囊投影为可读的中文字段。
func BuildPromptView(input FactInput, includeDynamic bool) map[string]any {
	capsule := BuildFactCapsule(input)
	roots := make([]string, 0, len(capsule.RootPositions))
	for _, root := range capsule.RootPositions {
		position := strings.TrimSpace(root.Pillar + root.Branch)
		if root.Tier != "" {
			position += "（" + root.Tier + "）"
		}
		roots = append(roots, position)
	}
	view := map[string]any{
		"月令":     firstNonEmpty(capsule.MonthCommand, "工具未提供"),
		"日主通根":   firstNonEmpty(strings.Join(roots, "、"), "未见可展示通根"),
		"同类透干":   firstNonEmpty(strings.Join(capsule.VisibleSameElementStems, "、"), "未见同类透干"),
		"印星生扶":   fmt.Sprintf("已计算生扶信号 %d 项", capsule.ResourceSupportCount),
		"强弱受力":   fmt.Sprintf("扶身 %d；泄耗克身 %d", capsule.SupportScore, capsule.PressureScore),
		"扶身信号":   firstNonEmpty(strings.Join(capsule.SupportSignals, "；"), "工具未提供"),
		"泄耗克身信号": firstNonEmpty(strings.Join(capsule.PressureSignals, "；"), "工具未提供"),
		"官星透藏":   OfficialDisplay(capsule),
		"火与调候状态": FireDisplay(capsule),
	}
	if includeDynamic {
		view["当前大运"] = firstNonEmpty(capsule.CurrentPeriodGanZhi, "未识别")
		view["当前大运已计算关系"] = firstNonEmpty(strings.Join(capsule.CurrentPeriodRelations, "；"), "未见已计算关系")
	}
	return view
}

// OfficialDisplay 区分官星透干与藏支，不据此生成原局结论。
func OfficialDisplay(capsule FactCapsule) string {
	switch {
	case capsule.OfficialVisible:
		return "官星透干"
	case capsule.OfficialHidden:
		return "官星藏支未透"
	default:
		return "工具未见官星透藏记录"
	}
}

// FireDisplay 将火的出现、透出和有效性保持为三个独立事实。
func FireDisplay(capsule FactCapsule) string {
	parts := []string{}
	if capsule.FirePresent {
		parts = append(parts, "有火")
	} else {
		parts = append(parts, "未见火")
	}
	if capsule.FireVisible {
		parts = append(parts, "火透出")
	} else {
		parts = append(parts, "火未透出")
	}
	switch {
	case !capsule.FireEffectivenessKnown:
		parts = append(parts, "调候有效性待确认")
	case capsule.FireEffective:
		parts = append(parts, "已确认可参与调候")
	default:
		parts = append(parts, "已确认不足以单独作为调候依据")
	}
	return strings.Join(parts, "；")
}

// TiaohouDisplay 将火事实转成受月令边界约束的调候说明。
func TiaohouDisplay(capsule FactCapsule) string {
	parts := []string{}
	if capsule.MonthCommand != "" {
		parts = append(parts, "当前先按"+capsule.MonthCommand+"月令的寒暖燥湿需求观察")
	} else {
		parts = append(parts, "当前季节需求尚未完整显示")
	}
	switch {
	case !capsule.FirePresent:
		parts = append(parts, "命局未见可直接列出的火元素条件")
	case !capsule.FireEffectivenessKnown:
		parts = append(parts, "命局虽见火，但现有材料尚不能确认其调候作用是否足够")
	case capsule.FireEffective:
		parts = append(parts, "现有材料确认火可参与调候")
	default:
		parts = append(parts, "现有材料显示火不足以单独完成调候")
	}
	if capsule.FirePresent && !capsule.FireVisible {
		parts = append(parts, "火未透出，作用仍需结合位置与时令判断")
	}
	return strings.Join(parts, "；")
}

// capsulePillarFacts 从四柱载荷提取通根层级、同类透干和火出现事实。
func capsulePillarFacts(raw any, dayMaster string) ([]RootPosition, []string, bool, bool) {
	dayElement := stemElements[dayMaster]
	roots := []RootPosition{}
	sameStems := []string{}
	firePresent := false
	fireVisible := false
	for index, pillar := range anyMapSlice(raw) {
		stem := strings.TrimSpace(stringValue(pillar["stem"]))
		if stemElements[stem] == "火" {
			firePresent, fireVisible = true, true
		}
		if stem != "" && stem != dayMaster && dayElement != "" && stemElements[stem] == dayElement {
			sameStems = append(sameStems, stem)
		}
		for hiddenIndex, hidden := range stringSlice(pillar["hideGan"]) {
			if stemElements[hidden] == "火" {
				firePresent = true
			}
			if dayElement == "" || stemElements[hidden] != dayElement {
				continue
			}
			label := strings.TrimSpace(stringValue(pillar["name"]))
			if label == "" {
				label = []string{"年柱", "月柱", "日柱", "时柱"}[minInt(index, 3)]
			}
			roots = append(roots, RootPosition{Pillar: label, Branch: strings.TrimSpace(stringValue(pillar["branch"])), Tier: hiddenStemTier(hiddenIndex)})
		}
	}
	return roots, uniqueText(sameStems), firePresent, fireVisible
}

// hiddenStemTier 将确定性藏干顺序转换为可展示的层级标签。
func hiddenStemTier(index int) string {
	switch index {
	case 0:
		return "本气"
	case 1:
		return "中气"
	default:
		return "余气"
	}
}

// capsuleFireEffectiveness 只读取确定性输入明确给出的调候有效性。
func capsuleFireEffectiveness(fire map[string]any) (bool, bool) {
	if len(fire) == 0 {
		return false, false
	}
	if _, ok := fire["effective"]; ok {
		return truthyFact(fire["effective"]), true
	}
	if value := strings.TrimSpace(stringValue(fire["effectiveness"])); value != "" {
		return value == "effective", true
	}
	return false, false
}

// mapValue 从动态载荷读取嵌套对象，类型不符时返回 nil。
func mapValue(src map[string]any, key string) map[string]any {
	value, ok := src[key]
	if !ok || value == nil {
		return nil
	}
	typed, _ := value.(map[string]any)
	return typed
}

// anyMapSlice 兼容确定性工具常见的三种对象数组解码形态。
func anyMapSlice(raw any) []map[string]any {
	switch items := raw.(type) {
	case []map[string]any:
		return items
	case []map[string]string:
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			value := make(map[string]any, len(item))
			for key, field := range item {
				value[key] = field
			}
			out = append(out, value)
		}
		return out
	case []any:
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if value, ok := item.(map[string]any); ok {
				out = append(out, value)
			}
		}
		return out
	default:
		return nil
	}
}

// relationTextList 提取已计算关系的用户可读说明。
func relationTextList(raw any) []string {
	texts := make([]string, 0)
	for _, relation := range anyMapSlice(raw) {
		if description := strings.TrimSpace(stringValue(relation["description"])); description != "" {
			texts = append(texts, description)
		}
	}
	return texts
}

// stringSlice 读取字符串数组并去除空白项。
func stringSlice(raw any) []string {
	items := []string{}
	switch values := raw.(type) {
	case []string:
		items = append(items, values...)
	case []any:
		for _, value := range values {
			items = append(items, stringValue(value))
		}
	}
	return filterNonEmpty(items)
}

// stringValue 只接受原始字符串，避免隐式格式化动态事实。
func stringValue(raw any) string {
	value, _ := raw.(string)
	return value
}

// intValue 将常见 JSON 数字形态转换为整数。
func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}

// truthyFact 兼容确定性工具返回的布尔值和字符串布尔值。
func truthyFact(value any) bool {
	if value, ok := value.(bool); ok {
		return value
	}
	return strings.EqualFold(strings.TrimSpace(stringValue(value)), "true")
}

// minInt 将柱位回退索引限制在四柱范围内。
func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

// filterNonEmpty 删除字符串列表中的空白项。
func filterNonEmpty(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// uniqueText 保持原顺序去除重复说明。
func uniqueText(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range filterNonEmpty(items) {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

// containsString 判断主题列表是否包含目标值。
func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

// firstNonEmpty 返回第一个去除空白后仍非空的字符串。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
