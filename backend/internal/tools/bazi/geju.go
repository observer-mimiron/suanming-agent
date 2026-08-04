// This file belongs to the BaZi deterministic calculation layer.
// It owns BaZi pattern calculation for this package.
// It computes reproducible BaZi facts; it must not generate narrative readings.
package bazi

import (
	"fmt"
	"strings"
)

// geju.go 格局判定：月令取格、伤官细分、变格检测、成败规则。
// 按《子平真诠》月令藏干透干取格法，参考 knowledge/wiki/ref-bazi-geju.md。
// 复用 shishen.go 的 tenGodOf + tables.go 的 ke 表，避免重复定义。

// gejuQingZhuoReason 保存格局清浊的结构化证据。
// 清浊不是单看月令透不透干，还要结合混杂、救应与主轴稳定度综合判断。
type gejuQingZhuoReason struct {
	Label    string         `json:"label"`
	Summary  string         `json:"summary"`
	Evidence []string       `json:"evidence"`
	Signals  map[string]any `json:"signals"`
}

// evaluateGejuQingZhuo 在既有取格与成败结果之上，补一层“清浊结构”判断。
// 规则有意保持克制：determineGeju 继续负责“取什么格”，这里仅判断该格是否清纯、
// 杂而有救，或已进入变格口径，避免把整个格局系统重写成黑箱评分器。
func evaluateGejuQingZhuo(gejuName, baseQingZhuo, gejuStatus, gejuDetail, gejuCombination string) (string, gejuQingZhuoReason) {
	if baseQingZhuo == "变" {
		reason := gejuQingZhuoReason{
			Label:    "变",
			Summary:  "此盘已按变格口径处理，不再用正格的清浊尺子硬套。",
			Evidence: []string{"已触发变格判断：" + gejuName},
			Signals: map[string]any{
				"month_order_revealed": false,
				"core_god_pure":        false,
				"has_mixture":          false,
				"has_relief":           false,
				"main_axis_stable":     true,
			},
		}
		return reason.Label, reason
	}

	monthOrderRevealed := baseQingZhuo == "清"
	hasMixture := gejuStatus == "破格" ||
		strings.Contains(gejuDetail, "忌神") ||
		strings.Contains(gejuDetail, "混杂") ||
		strings.Contains(gejuDetail, "见官") ||
		strings.Contains(gejuDetail, "破格")
	hasRelief := gejuStatus == "成格" ||
		gejuStatus == "成败参半" ||
		strings.Contains(gejuDetail, "喜神") ||
		strings.Contains(gejuDetail, "制化") ||
		strings.Contains(gejuDetail, "配合") ||
		strings.Contains(gejuCombination, "[主]") ||
		strings.Contains(gejuCombination, "[次]")
	mainAxisStable := gejuStatus != "破格" || hasRelief
	coreGodPure := monthOrderRevealed && !hasMixture

	label := "浊"
	switch {
	case coreGodPure && (gejuStatus == "成格" || gejuStatus == "基本成立"):
		label = "清"
	case monthOrderRevealed && mainAxisStable:
		label = "偏清"
	case !monthOrderRevealed && hasRelief && mainAxisStable:
		label = "浊中有清"
	case !monthOrderRevealed && mainAxisStable:
		label = "半清半浊"
	default:
		label = "浊"
	}

	evidence := make([]string, 0, 4)
	if monthOrderRevealed {
		evidence = append(evidence, "月令真神透干，主气较显。")
	} else {
		evidence = append(evidence, "月令真神未透，先天纯度不足。")
	}
	if hasMixture {
		evidence = append(evidence, "存在混杂或破损信号："+gejuDetail)
	}
	if hasRelief {
		reliefBasis := gejuDetail
		if gejuCombination != "" && gejuCombination != "无明显组合关系" {
			reliefBasis = gejuCombination
		}
		evidence = append(evidence, "仍见制化、配合或主轴支撑："+reliefBasis)
	}
	if gejuStatus != "" {
		evidence = append(evidence, "成败状态："+gejuStatus)
	}

	summary := map[string]string{
		"清":    "月令主气明确，杂气轻，主轴稳定，可按较清的正格理解。",
		"偏清":   "主轴已立，但仍夹少量杂气，宜按偏清而非纯清处理。",
		"浊中有清": "先天不够清纯，但命局另有救应，属于杂中有序、浊中见清。",
		"半清半浊": "主轴虽可辨认，但清浊拉扯并存，层次取中。",
		"浊":    "杂气偏重且救应不足，不宜拔高为清纯格局。",
	}[label]

	reason := gejuQingZhuoReason{
		Label:    label,
		Summary:  summary,
		Evidence: evidence,
		Signals: map[string]any{
			"month_order_revealed": monthOrderRevealed,
			"core_god_pure":        coreGodPure,
			"has_mixture":          hasMixture,
			"has_relief":           hasRelief,
			"main_axis_stable":     mainAxisStable,
		},
	}
	return label, reason
}

// determineGeju 按“子平本气优先”口径判定主格。
//
// 主格先取月令本气，再看是否透干来判断“真神显露”还是“本气未透”。
// 中气、余气可以作为成败、配合、清浊的辅助证据，但不替代月令本气的主格地位。
// 特殊规则：月令本气为比肩时取建禄格；月令本气为劫财时区分月刃格与月劫格。
func determineGeju(monthZhi, dayGan, dayWx string, allGan []string,
	stemWx map[string]string, branchHidegan map[string][]string, generates map[string]string) (string, string, string) {

	hidegans := branchHidegan[monthZhi]
	if len(hidegans) == 0 {
		return "未知", "月令无藏干数据", "浊"
	}

	gejuNameOf := func(tenGod string) string {
		switch tenGod {
		case "正官":
			return "正官格"
		case "七杀":
			return "七杀格"
		case "正财":
			return "正财格"
		case "偏财":
			return "偏财格"
		case "正印":
			return "正印格"
		case "偏印":
			return "偏印格"
		case "食神":
			return "食神格"
		case "伤官":
			return "伤官格"
		case "比肩":
			return "建禄格"
		case "劫财":
			return "月劫格"
		default:
			return "未知格"
		}
	}

	posNameOf := func(idx int) string {
		switch idx {
		case 0:
			return "本气"
		case 1:
			return "中气"
		default:
			return "余气"
		}
	}

	ganSet := map[string]bool{}
	for _, g := range allGan {
		if g == dayGan {
			continue
		}
		ganSet[g] = true
	}

	renTable := map[string]string{
		"甲": "卯", "丙": "午", "戊": "午", "庚": "酉", "壬": "子",
	}

	mainHidden := hidegans[0]
	mainTenGod := tenGodOf(mainHidden, dayGan, stemWx[mainHidden], dayWx, generates)
	mainGeju := gejuNameOf(mainTenGod)

	switch mainTenGod {
	case "比肩":
		mainGeju = "建禄格"
	case "劫财":
		if monthZhi == renTable[dayGan] {
			mainGeju = "月刃格"
		} else {
			mainGeju = "月劫格"
		}
	}

	if ganSet[mainHidden] {
		return mainGeju, fmt.Sprintf("月令%s本气%s透出天干，先以本气十神%s定主格为%s", monthZhi, mainHidden, mainTenGod, mainGeju), "清"
	}

	for i := 1; i < len(hidegans); i++ {
		hg := hidegans[i]
		if !ganSet[hg] {
			continue
		}
		return mainGeju, fmt.Sprintf("月令%s本气%s未透，仍先以本气十神%s定主格为%s；另见%s%s%s透干，可作辅格与成败配合参考", monthZhi, mainHidden, mainTenGod, mainGeju, monthZhi, posNameOf(i), hg), "浊"
	}

	return mainGeju, fmt.Sprintf("月令%s本气%s未透，先以本气十神%s定主格为%s", monthZhi, mainHidden, mainTenGod, mainGeju), "浊"
}

// refineShangGuan 仅记录官星是否透干，避免把“官未透”误写成“命盘无官星”。
// 藏官是否影响伤尽属于流派裁断，必须保留给上游 rule profile。
func refineShangGuan(gejuName, gejuBasis, dayGan string, allGan []string, stemWx map[string]string, generates map[string]string) (string, string) {
	dayWx := stemWx[dayGan]
	for _, g := range allGan {
		if g == dayGan {
			continue
		}
		if ke[stemWx[g]] == dayWx {
			return "伤官格(见官)", gejuBasis + "；命盘有官星透出，伤官见官，需注意破格风险"
		}
	}
	return "伤官格(官未透)", gejuBasis + "；官星未透干，伤尽与否仍须结合藏官与所选流派裁断"
}

// checkBianGe 检测变格（从格、专旺格、两神成象格）。
// 日主弱极无依时取从格，日主旺极且五行偏聚时取专旺格，命盘仅两种五行时取两神成象格。
// 返回变格名称，空串表示非变格。
// 注意：本函数 local generates 为"印星方向"（"木":"水" 即水生木），与 dayun.go 的
// "食伤方向" generates 语义相反——保持 local 避免命名冲突。
func checkBianGe(strength string, dayGan string, dayWx string, allGan []string, allZhi []string, stemWx map[string]string, branchHidegan map[string][]string) string {
	wxCount := map[string]int{"木": 0, "火": 0, "土": 0, "金": 0, "水": 0}
	for _, g := range allGan {
		wxCount[stemWx[g]]++
	}
	for _, z := range allZhi {
		for _, hg := range branchHidegan[z] {
			wxCount[stemWx[hg]]++
		}
	}
	total := 0
	for _, c := range wxCount {
		total += c
	}
	if total == 0 {
		return ""
	}

	generates := map[string]string{"木": "水", "火": "木", "土": "火", "金": "土", "水": "金"}
	support := wxCount[dayWx] + wxCount[generates[dayWx]]

	if strength == "身弱极" && support <= 1 {
		maxWx, maxCnt := "", 0
		for wx, cnt := range wxCount {
			if wx == dayWx || wx == generates[dayWx] {
				continue
			}
			if cnt > maxCnt {
				maxCnt = cnt
				maxWx = wx
			}
		}
		if maxCnt > 0 && maxCnt >= total/2 {
			switch {
			case ke[dayWx] == maxWx:
				return "从财格"
			case ke[maxWx] == dayWx:
				return "从杀格"
			case generates[maxWx] == dayWx:
				return "从儿格"
			default:
				return "从格"
			}
		}
	}

	if strength == "身旺极" {
		guanCount := 0
		for wx, cnt := range wxCount {
			if ke[wx] == dayWx {
				guanCount += cnt
			}
		}
		if guanCount == 0 && support >= total-1 {
			if wxCount[dayWx] >= total-1 {
				switch dayWx {
				case "木":
					return "曲直格"
				case "火":
					return "炎上格"
				case "土":
					return "稼穑格"
				case "金":
					return "从革格"
				case "水":
					return "润下格"
				}
			}
			return "从强格"
		}
	}

	nonZeroWx := 0
	for _, cnt := range wxCount {
		if cnt > 0 {
			nonZeroWx++
		}
	}
	if nonZeroWx == 2 {
		return "两神成象格"
	}

	return ""
}

// gejuChengbaiRule 定义一种格局的成败规则。
type gejuChengbaiRule struct {
	xi      []string
	ji      []string
	success string
	fail    string
}

// gejuRules 是《子平真诠》十种正格的成败规则表。
var gejuRules = map[string]gejuChengbaiRule{
	"正官格": {xi: []string{"正印", "偏财"}, ji: []string{"伤官", "七杀"}, success: "官逢财印，清纯无混", fail: "伤官透出或七杀混杂"},
	"七杀格": {xi: []string{"食神", "正印"}, ji: []string{"正财", "伤官"}, success: "食制或印化，杀有驾驭", fail: "财生杀或无制化"},
	"正财格": {xi: []string{"食神", "正官"}, ji: []string{"七杀", "比肩", "劫财"}, success: "财旺有食生，官护财", fail: "比劫夺财或七杀破局"},
	"偏财格": {xi: []string{"食神", "正官"}, ji: []string{"七杀", "比肩", "劫财"}, success: "偏财得食生，无比劫夺", fail: "比劫群起夺财"},
	"食神格": {xi: []string{"偏财"}, ji: []string{"偏印"}, success: "食神生财，无枭夺", fail: "偏印出现夺食"},
	"伤官格": {xi: []string{"正财", "正印"}, ji: []string{"正官"}, success: "伤官生财或佩印", fail: "见正官，伤官见官"},
	"正印格": {xi: []string{"正官", "七杀"}, ji: []string{"正财", "伤官"}, success: "官印双全或杀印相生", fail: "财重印轻，贪财坏印"},
	"偏印格": {xi: []string{"七杀", "正官"}, ji: []string{"食神"}, success: "偏印逢官杀有化", fail: "食神被夺，格局无力"},
	"建禄格": {xi: []string{"正官", "正财"}, ji: []string{"七杀"}, success: "透官逢财印，或透财逢食伤", fail: "无财官或透煞无制"},
	"月劫格": {xi: []string{"正官", "正财"}, ji: []string{"七杀"}, success: "透官逢财印，或透财逢食伤", fail: "无财官或透煞无制"},
	"月刃格": {xi: []string{"正官", "七杀"}, ji: []string{}, success: "官或杀制刃，有制伏", fail: "无官无杀，刃气无制"},
}

// checkGejuChengbai 按《子平真诠》格局成败规则判断成格/破格。
func checkGejuChengbai(gejuName, dayGan string, allGan []string,
	stemWx map[string]string, generates map[string]string) (string, string) {

	baseGeju := gejuName
	if idx := strings.Index(gejuName, "("); idx > 0 {
		baseGeju = gejuName[:idx]
	}
	rule, ok := gejuRules[baseGeju]
	if !ok {
		return "成格", "变格或特殊格局，不适用正格成败规则"
	}

	visibleTenGods := map[string]bool{}
	for _, g := range allGan {
		if g == dayGan {
			continue
		}
		visibleTenGods[tenGodOf(g, dayGan, stemWx[g], stemWx[dayGan], generates)] = true
	}

	jiFound := []string{}
	for _, j := range rule.ji {
		if visibleTenGods[j] {
			jiFound = append(jiFound, j)
		}
	}
	xiFound := []string{}
	for _, x := range rule.xi {
		if visibleTenGods[x] {
			xiFound = append(xiFound, x)
		}
	}

	switch {
	case len(jiFound) > 0 && len(xiFound) == 0:
		return "破格", fmt.Sprintf("忌神%s出现且无喜神制化 → %s", strings.Join(jiFound, "、"), rule.fail)
	case len(jiFound) > 0 && len(xiFound) > 0:
		return "成败参半", fmt.Sprintf("喜神%s制化忌神%s，格局有瑕 → %s", strings.Join(xiFound, "、"), strings.Join(jiFound, "、"), rule.success)
	case len(xiFound) > 0:
		return "成格", fmt.Sprintf("喜神%s配合 → %s", strings.Join(xiFound, "、"), rule.success)
	default:
		if baseGeju == "月刃格" {
			return "破格", "无官无杀，刃气无制"
		}
		return "基本成立", "喜忌均不显，格局基本成立"
	}
}
