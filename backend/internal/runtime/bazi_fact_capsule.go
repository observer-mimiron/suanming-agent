// Package runtime 包含 Manager 拥有的八字运行时合同。
//
// 本文件把可复算命盘事实收束为裁断资格胶囊；不选择格局、不写用户文案，
// 也不承担模型修复或图编排。
package runtime

import (
	"fmt"
	"strings"
)

var baziTierEvidenceTopics = []string{"qingzhuo", "bingyao", "jiuying", "poge", "hezhizhang"}

var baziStemElements = map[string]string{
	"甲": "木", "乙": "木", "丙": "火", "丁": "火", "戊": "土",
	"己": "土", "庚": "金", "辛": "金", "壬": "水", "癸": "水",
}

// baziRootPosition records a day-master root without turning root count into a
// pattern or tier verdict. Its tier comes from the deterministic hidden-stem order.
type baziRootPosition struct {
	Pillar string `json:"pillar"`
	Branch string `json:"branch"`
	Tier   string `json:"tier"`
}

// BaziFactCapsule 是一次裁断可引用的确定性事实与资格状态。
// 它把模型不得自行推断的前置条件集中为类型化输入。
type BaziFactCapsule struct {
	MonthCommand            string             `json:"month_command"`
	RootPositions           []baziRootPosition `json:"root_positions"`
	VisibleSameElementStems []string           `json:"visible_same_element_stems"`
	MonthScore              int                `json:"month_score"`
	RootCount               int                `json:"root_count"`
	SameElementCount        int                `json:"same_element_count"`
	ResourceSupportCount    int                `json:"resource_support_count"`
	SupportScore            int                `json:"support_score"`
	PressureScore           int                `json:"pressure_score"`
	SupportSignals          []string           `json:"support_signals"`
	PressureSignals         []string           `json:"pressure_signals"`
	OfficialVisible         bool               `json:"official_visible"`
	OfficialHidden          bool               `json:"official_hidden"`
	FirePresent             bool               `json:"fire_present"`
	FireVisible             bool               `json:"fire_visible"`
	FireEffective           bool               `json:"fire_effective"`
	FireEffectivenessKnown  bool               `json:"fire_effectiveness_known"`
	CoreFactsReady          bool               `json:"core_facts_ready"`
	TierEvidenceComplete    bool               `json:"tier_evidence_complete"`
	TierEvidenceMissing     []string           `json:"tier_evidence_missing"`
	CurrentPeriodRef        string             `json:"current_period_ref"`
	CurrentPeriodGanZhi     string             `json:"current_period_ganzhi"`
	CurrentPeriodRelations  []string           `json:"current_period_relations"`
}

// buildBaziFactCapsule projects only deterministic facts required by semantic policy.
func buildBaziFactCapsule(state baziCharterState) BaziFactCapsule {
	yongshen := state.Input.Yongshen
	strength := mapValue(yongshen, "strength_evidence")
	official := mapValue(yongshen, "official_visibility")
	fire := mapValue(yongshen, "fire_status")
	if len(fire) == 0 {
		fire = mapValue(yongshen, "tiaohou_fire")
	}
	currentRef := currentDayunPeriodRef(state)
	periods := dayunPeriods(state.Input.Dayun)
	currentPeriod := map[string]any{}
	if index := currentDayunIndexForInput(state.Input); index >= 0 && index < len(periods) {
		currentPeriod = periods[index]
	}
	dayMaster := firstNonEmptyTrim(
		stringValue(yongshen["day_master"]),
		stringValue(state.Input.BaziResult["dayGan"]),
	)
	monthCommand := monthBranchForEvidenceQuery(state.Input)
	rootPositions, sameStems, firePresent, fireVisible := capsulePillarFacts(state.Input.BaziResult["pillars"], dayMaster)
	fireEffective, fireKnown := capsuleFireEffectiveness(fire)
	missing := tierEvidenceMissing(state)

	return BaziFactCapsule{
		MonthCommand:            monthCommand,
		RootPositions:           rootPositions,
		VisibleSameElementStems: sameStems,
		MonthScore:              intValue(yongshen["month_score"]),
		RootCount:               intValue(yongshen["root_count"]),
		SameElementCount:        intValue(yongshen["same_element"]),
		ResourceSupportCount:    intValue(yongshen["generate_count"]),
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
		CoreFactsReady:          monthCommand != "" && dayMaster != "" && len(state.Input.BaziResult) > 0,
		TierEvidenceComplete:    len(missing) == 0,
		TierEvidenceMissing:     missing,
		CurrentPeriodRef:        currentRef,
		CurrentPeriodGanZhi:     strings.TrimSpace(stringValue(currentPeriod["ganZhi"])),
		CurrentPeriodRelations:  relationTextList(currentPeriod["dayun_chonghe"]),
	}
}

// buildBaziFactCapsulePromptView exposes the same deterministic capsule to a
// model with Chinese display facts. Runtime identifiers remain exclusively in
// the reference catalog, so the model does not need to copy snake_case keys
// into an explanatory slot.
func buildBaziFactCapsulePromptView(state baziCharterState, includeDynamic bool) map[string]any {
	capsule := buildBaziFactCapsule(state)
	roots := make([]string, 0, len(capsule.RootPositions))
	for _, root := range capsule.RootPositions {
		position := strings.TrimSpace(root.Pillar + root.Branch)
		if root.Tier != "" {
			position += "（" + root.Tier + "）"
		}
		roots = append(roots, position)
	}
	view := map[string]any{
		"月令":       firstNonEmptyTrim(capsule.MonthCommand, "工具未提供"),
		"日主通根":     firstNonEmptyTrim(strings.Join(roots, "、"), "未见可展示通根"),
		"同类透干":     firstNonEmptyTrim(strings.Join(capsule.VisibleSameElementStems, "、"), "未见同类透干"),
		"印星生扶":     fmt.Sprintf("已计算生扶信号 %d 项", capsule.ResourceSupportCount),
		"强弱受力":     fmt.Sprintf("扶身 %d；泄耗克身 %d", capsule.SupportScore, capsule.PressureScore),
		"扶身信号":     firstNonEmptyTrim(strings.Join(capsule.SupportSignals, "；"), "工具未提供"),
		"泄耗克身信号":   firstNonEmptyTrim(strings.Join(capsule.PressureSignals, "；"), "工具未提供"),
		"官星透藏":     capsuleOfficialDisplay(capsule),
		"火与调候状态":   capsuleFireDisplay(capsule),
		"层次独立证据状态": capsuleTierEvidenceDisplay(capsule),
	}
	if includeDynamic {
		view["当前大运"] = firstNonEmptyTrim(capsule.CurrentPeriodGanZhi, "未识别")
		view["当前大运已计算关系"] = firstNonEmptyTrim(strings.Join(capsule.CurrentPeriodRelations, "；"), "未见已计算关系")
	}
	return view
}

// capsuleOfficialDisplay distinguishes visible and hidden officers without
// turning either fact into a natal-risk conclusion.
func capsuleOfficialDisplay(capsule BaziFactCapsule) string {
	switch {
	case capsule.OfficialVisible:
		return "官星透干"
	case capsule.OfficialHidden:
		return "官星藏支未透"
	default:
		return "工具未见官星透藏记录"
	}
}

// capsuleFireDisplay preserves presence, visibility and effectiveness for model input.
func capsuleFireDisplay(capsule BaziFactCapsule) string {
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

// capsuleTiaohouDisplay turns fire facts into a bounded seasonal explanation.
// A bare “有火” is only a presence flag and must not be shown as an adjustment conclusion.
func capsuleTiaohouDisplay(capsule BaziFactCapsule) string {
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

// capsuleTierEvidenceDisplay keeps evidence coverage legible without making a
// grade; the model still selects only the structured tier_assessment slots.
func capsuleTierEvidenceDisplay(capsule BaziFactCapsule) string {
	if capsule.TierEvidenceComplete {
		return "清浊、病药、救应、破格风险与何知章独立证据已覆盖"
	}
	labels := make([]string, 0, len(capsule.TierEvidenceMissing))
	for _, topic := range capsule.TierEvidenceMissing {
		labels = append(labels, map[string]string{
			"qingzhuo": "清浊", "bingyao": "病药", "jiuying": "救应", "poge": "破格风险", "hezhizhang": "何知章",
		}[topic])
	}
	return "尚缺独立证据：" + strings.Join(filterNonEmpty(labels), "、")
}

// capsulePillarFacts extracts root tiers, visible companions and fire presence
// from the existing deterministic pillar payload. It does not evaluate force.
func capsulePillarFacts(raw any, dayMaster string) ([]baziRootPosition, []string, bool, bool) {
	dayElement := baziStemElements[dayMaster]
	roots := []baziRootPosition{}
	sameStems := []string{}
	firePresent := false
	fireVisible := false
	for index, pillar := range anyMapSlice(raw) {
		stem := strings.TrimSpace(stringValue(pillar["stem"]))
		if baziStemElements[stem] == "火" {
			firePresent, fireVisible = true, true
		}
		if stem != "" && stem != dayMaster && dayElement != "" && baziStemElements[stem] == dayElement {
			sameStems = append(sameStems, stem)
		}
		for hiddenIndex, hidden := range stringSlice(pillar["hideGan"]) {
			if baziStemElements[hidden] == "火" {
				firePresent = true
			}
			if dayElement == "" || baziStemElements[hidden] != dayElement {
				continue
			}
			label := strings.TrimSpace(stringValue(pillar["name"]))
			if label == "" {
				label = []string{"年柱", "月柱", "日柱", "时柱"}[minInt(index, 3)]
			}
			roots = append(roots, baziRootPosition{
				Pillar: label,
				Branch: strings.TrimSpace(stringValue(pillar["branch"])),
				Tier:   hiddenStemTier(hiddenIndex),
			})
		}
	}
	return roots, uniqueText(sameStems), firePresent, fireVisible
}

// hiddenStemTier mirrors the deterministic pillar order only for displayable facts.
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

// capsuleFireEffectiveness treats effectiveness as unknown unless the deterministic
// input explicitly provides it. Fire presence or 午的地势 cannot stand in for it.
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

// stringSlice accepts the JSON list shapes used by deterministic tool payloads.
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

// minInt keeps the fallback pillar label bounded to the four deterministic columns.
func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

// truthyFact accepts the two JSON primitive forms emitted by deterministic tools.
func truthyFact(value any) bool {
	if value, ok := value.(bool); ok {
		return value
	}
	return strings.EqualFold(strings.TrimSpace(stringValue(value)), "true")
}

// tierEvidenceComplete reports whether every independent authority topic is covered.
// It is an upper-confidence qualification, not an instruction to withhold every grade.
func tierEvidenceComplete(state baziCharterState) bool {
	return len(tierEvidenceMissing(state)) == 0
}

// tierEvidenceMissing returns independent tier topics absent from the retrieval audit.
func tierEvidenceMissing(state baziCharterState) []string {
	missing := make([]string, 0, len(baziTierEvidenceTopics))
	for _, topic := range baziTierEvidenceTopics {
		if !containsString(state.EvidenceQuality.CoveredTopics, topic) {
			missing = append(missing, topic)
		}
	}
	return missing
}
