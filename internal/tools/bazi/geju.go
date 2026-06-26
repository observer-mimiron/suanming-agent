package bazi

import (
	"fmt"
	"strings"
)

// geju.go 格局判定：月令取格、伤官细分、变格检测、成败规则。
// 按《子平真诠》月令藏干透干取格法，参考 knowledge/wiki/ref-bazi-geju.md。
// 复用 shishen.go 的 tenGodOf + tables.go 的 ke 表，避免重复定义。

// determineGeju 按《子平真诠》月令藏干透干取格法判定格局。
//
// 月令地支藏干按本气→中气→余气顺序检查是否透出天干。
// 透出者取格，以该天干与日主的十神关系命名（如正官格、七杀格、伤官格等）。
// 若无藏干透出，取月令本气的十神关系作为暗格。
// 特殊规则：月令为比劫（日主同五行）时，按建禄/月刃/月劫处理。
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
			return "月刃格"
		default:
			return "未知格"
		}
	}

	ganSet := map[string]bool{}
	for _, g := range allGan {
		if g == dayGan {
			continue
		}
		ganSet[g] = true
	}

	luTable := map[string]string{
		"甲": "寅", "乙": "卯", "丙": "巳", "戊": "巳",
		"丁": "午", "己": "午", "庚": "申", "辛": "酉",
		"壬": "亥", "癸": "子",
	}
	renTable := map[string]string{
		"甲": "卯", "丙": "午", "戊": "午", "庚": "酉", "壬": "子",
	}

	for i, hg := range hidegans {
		if ganSet[hg] {
			tg := tenGodOf(hg, dayGan, stemWx[hg], dayWx, generates)
			if tg == "比肩" && monthZhi != luTable[dayGan] {
				continue
			}
			if tg == "劫财" && monthZhi != renTable[dayGan] {
				continue
			}
			pos := "本气"
			if i == 1 {
				pos = "中气"
			} else if i >= 2 {
				pos = "余气"
			}
			return gejuNameOf(tg), fmt.Sprintf("月令%s藏干%s透出天干，十神为%s，取%s", monthZhi+pos, hg, tg, gejuNameOf(tg)), "清"
		}
	}

	usedIdx := 0
	tg := tenGodOf(hidegans[0], dayGan, stemWx[hidegans[0]], dayWx, generates)
	if tg == "比肩" && monthZhi != luTable[dayGan] {
		if len(hidegans) > 1 {
			tg = tenGodOf(hidegans[1], dayGan, stemWx[hidegans[1]], dayWx, generates)
			usedIdx = 1
		}
	}
	if tg == "劫财" && monthZhi != renTable[dayGan] {
		if len(hidegans) > 1 {
			tg = tenGodOf(hidegans[1], dayGan, stemWx[hidegans[1]], dayWx, generates)
			usedIdx = 1
		}
	}
	usedPos := "本气"
	if usedIdx == 1 {
		usedPos = "中气"
	} else if usedIdx >= 2 {
		usedPos = "余气"
	}
	return gejuNameOf(tg), fmt.Sprintf("月令%s%s%s未透出，以%s十神%s取暗格%s", monthZhi, usedPos, hidegans[usedIdx], usedPos, tg, gejuNameOf(tg)), "浊"
}

// refineShangGuan 细分伤官格为伤尽或见官。
// 子平真诠：伤官最忌见官，四柱无官星则伤尽为贵；有官星透出则伤官见官，破格风险大。
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
	return "伤官格(伤尽)", gejuBasis + "；命盘无官星，伤官伤尽为贵"
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
	"正官格":  {xi: []string{"正印", "偏财"}, ji: []string{"伤官", "七杀"}, success: "官逢财印，清纯无混", fail: "伤官透出或七杀混杂"},
	"七杀格":  {xi: []string{"食神", "正印"}, ji: []string{"正财", "伤官"}, success: "食制或印化，杀有驾驭", fail: "财生杀或无制化"},
	"正财格":  {xi: []string{"食神", "正官"}, ji: []string{"七杀", "比肩", "劫财"}, success: "财旺有食生，官护财", fail: "比劫夺财或七杀破局"},
	"偏财格":  {xi: []string{"食神", "正官"}, ji: []string{"七杀", "比肩", "劫财"}, success: "偏财得食生，无比劫夺", fail: "比劫群起夺财"},
	"食神格":  {xi: []string{"偏财"}, ji: []string{"偏印"}, success: "食神生财，无枭夺", fail: "偏印出现夺食"},
	"伤官格":  {xi: []string{"正财", "正印"}, ji: []string{"正官"}, success: "伤官生财或佩印", fail: "见正官，伤官见官"},
	"正印格":  {xi: []string{"正官", "七杀"}, ji: []string{"正财", "伤官"}, success: "官印双全或杀印相生", fail: "财重印轻，贪财坏印"},
	"偏印格":  {xi: []string{"七杀", "正官"}, ji: []string{"食神"}, success: "偏印逢官杀有化", fail: "食神被夺，格局无力"},
	"建禄格":  {xi: []string{"正官", "正财"}, ji: []string{"七杀"}, success: "透官逢财印，或透财逢食伤", fail: "无财官或透煞无制"},
	"月刃格":  {xi: []string{"正官", "七杀"}, ji: []string{}, success: "官或杀制刃，有制伏", fail: "无官无杀，刃气无制"},
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
