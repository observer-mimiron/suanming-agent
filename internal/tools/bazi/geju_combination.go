package bazi

import (
	"fmt"
	"sort"
	"strings"
)

// geju_combination.go 13 种十神组合关系检测。
// 检测食神制杀、杀印相生、伤官佩印、官印相生、财滋杀、伤官生财、食神生财、
// 财官相生、食神吐秀、以劫合杀、枭神夺食、伤官见官、贪财坏印等组合。
// 复用 shishen.go 的 collectTenGods 收集十神位置，避免重复实现。

// detectGejuCombination 检测命盘中的组合关系条件及阻碍因素。
//
// 检测 13 种组合关系，每个组合检查三个维度：
//   1. 存在性：十神在天干透出或地支藏干中出现
//   2. 阻碍：有无破坏组合的因素（财破印、枭夺食等）
//   3. 位置力量：十神之间的距离（贴身 vs 遥隔）、半合局加持、身强身弱影响
//
// 代码提供事实和建议优先级，最终主次判断由 LLM 基于这些事实完成。
// 优先级标注规则：[主]核心组合 > [次]重要辅助 > [辅]辅助/条件不足 > [忌]破坏因素
func detectGejuCombination(dayGan, dayWx, strength string, allGan, allZhi []string,
	stemWx map[string]string, branchHidegan map[string][]string, generates map[string]string) string {

	allGods := collectTenGods(dayGan, dayWx, allGan, allZhi, stemWx, generates, branchHidegan)

	hasGod := func(god string) bool {
		for _, info := range allGods {
			if info.god == god {
				return true
			}
		}
		return false
	}
	hasGanGod := func(god string) bool {
		for _, info := range allGods {
			if info.god == god && info.isGan {
				return true
			}
		}
		return false
	}
	godSources := func(god string) []string {
		sources := []string{}
		for _, info := range allGods {
			if info.god == god {
				sources = append(sources, info.source)
			}
		}
		return sources
	}
	// 印星是否当令有力：月令地支藏干为印星五行即为当令
	yinStrongEnough := func() bool {
		monthHidegans := branchHidegan[allZhi[1]] // 月令地支
		for _, hg := range monthHidegans {
			hgWx := stemWx[hg]
			// 印星五行 = 生日主的五行
			if generates[dayWx] == hgWx {
				return true // 印星当令
			}
		}
		// 或印星通根：地支中有印星五行
		yinWx := generates[dayWx]
		for _, z := range allZhi {
			for _, hg := range branchHidegan[z] {
				if stemWx[hg] == yinWx {
					return true
				}
			}
		}
		return false
	}
	// 财破印阻碍：财星透出天干会克印
	hasCaiBreakYin := func() bool {
		return hasGanGod("正财") || hasGanGod("偏财")
	}
	// minDistance 返回两个十神之间的最小柱距（0=同柱，1=贴身，2+=隔柱）
	minDistance := func(god1, god2 string) int {
		minD := 99
		for _, a := range allGods {
			if a.god != god1 {
				continue
			}
			for _, b := range allGods {
				if b.god != god2 {
					continue
				}
				d := a.pillarIdx - b.pillarIdx
				if d < 0 {
					d = -d
				}
				if d < minD {
					minD = d
				}
			}
		}
		return minD
	}
	// shaYinHalfCombo 检查七杀所在分支与印星所在分支是否形成半合局。
	// 半合局（三合局中任意两支）：申子→水, 寅午→火, 巳酉→金, 亥卯→木
	// 返回（合化五行, 七杀位置, 印星位置, 是否成立）
	shaYinHalfCombo := func() (string, string, string, bool) {
		halfCombos := map[string]string{
			"申子": "水", "子申": "水",
			"寅午": "火", "午寅": "火",
			"巳酉": "金", "酉巳": "金",
			"亥卯": "木", "卯亥": "木",
		}
		branchPosNames := []string{"年支", "月支", "日支", "时支"}
		for _, a := range allGods {
			if a.isGan || a.god != "七杀" {
				continue
			}
			for _, b := range allGods {
				if b.isGan || (b.god != "正印" && b.god != "偏印") {
					continue
				}
				if a.pillarIdx == b.pillarIdx {
					continue
				}
				key := allZhi[a.pillarIdx] + allZhi[b.pillarIdx]
				if elem, ok := halfCombos[key]; ok {
					return elem, branchPosNames[a.pillarIdx], branchPosNames[b.pillarIdx], true
				}
			}
		}
		return "", "", "", false
	}

	isStrong := strength == "身强" || strength == "身旺极"

	type comboResult struct {
		priority string // 主/次/辅/忌
		desc     string
	}
	var results []comboResult

	// --- 食神制杀：食神 + 七杀，直接制杀为用 ---
	// 优先级受距离和半合影响：贴身制杀为主，遥隔或杀印半合时降为辅
	if hasGod("食神") && hasGod("七杀") {
		shiSrc := godSources("食神")
		shaSrc := godSources("七杀")
		if hasGanGod("偏印") {
			results = append(results, comboResult{"辅", fmt.Sprintf("食神制杀受阻（食神在%s，七杀在%s，偏印透出夺食）", strings.Join(shiSrc, "、"), strings.Join(shaSrc, "、"))})
		} else {
			dist := minDistance("食神", "七杀")
			desc := fmt.Sprintf("食神制杀成立（食神在%s，七杀在%s）", strings.Join(shiSrc, "、"), strings.Join(shaSrc, "、"))
			if dist <= 1 {
				desc += "，贴身制杀"
			} else {
				desc += fmt.Sprintf("，隔%d柱遥隔，制杀力弱", dist-1)
			}
			// 杀印半合贴身相生时，制杀降为辅助
			if elem, shaPos, yinPos, found := shaYinHalfCombo(); found {
				desc += fmt.Sprintf("；%s与%s半合%s局，杀印贴身相生，制杀降为辅助", shaPos, yinPos, elem)
				results = append(results, comboResult{"辅", desc})
			} else if dist >= 2 {
				results = append(results, comboResult{"辅", desc})
			} else {
				results = append(results, comboResult{"主", desc})
			}
		}
	}

	// --- 杀印相生：七杀 + 印星，化杀为权 ---
	// 优先级受半合、距离、身强影响：半合贴身或身强时升为主
	if hasGod("七杀") && (hasGod("正印") || hasGod("偏印")) {
		shaSrc := godSources("七杀")
		yinType := "印星"
		if hasGod("正印") && !hasGod("偏印") {
			yinType = "正印"
		} else if hasGod("偏印") && !hasGod("正印") {
			yinType = "偏印"
		}
		yinSrc := godSources("正印")
		yinSrc = append(yinSrc, godSources("偏印")...)
		yinStrong := yinStrongEnough()
		caiBreak := hasCaiBreakYin()
		// 距离检测：取七杀与印星（正印或偏印）的最小柱距
		dist := minDistance("七杀", "正印")
		if d2 := minDistance("七杀", "偏印"); d2 < dist {
			dist = d2
		}
		distNote := ""
		if dist <= 1 {
			distNote = "，贴身相生"
		} else if dist >= 2 {
			distNote = fmt.Sprintf("，隔%d柱相生力减", dist-1)
		}
		// 半合检测
		halfComboNote := ""
		hasHalfCombo := false
		if elem, shaPos, yinPos, found := shaYinHalfCombo(); found {
			hasHalfCombo = true
			halfComboNote = fmt.Sprintf("；%s与%s半合%s局，贴身化杀为权", shaPos, yinPos, elem)
		}
		// 身强判断
		strengthNote := ""
		if isStrong {
			strengthNote = "；身强，印化杀为第一要务"
		}
		switch {
		case !yinStrong:
			results = append(results, comboResult{"辅", fmt.Sprintf("杀印相生条件不足（七杀在%s，%s在%s，印星未当令/无根，化杀力弱）", strings.Join(shaSrc, "、"), yinType, strings.Join(yinSrc, "、"))})
		case caiBreak:
			results = append(results, comboResult{"辅", fmt.Sprintf("杀印相生受阻（七杀在%s，%s在%s，财星透出破印）", strings.Join(shaSrc, "、"), yinType, strings.Join(yinSrc, "、"))})
		default:
			desc := fmt.Sprintf("杀印相生成立（七杀在%s，%s在%s，印星当令有力，无财破印%s%s%s）",
				strings.Join(shaSrc, "、"), yinType, strings.Join(yinSrc, "、"), distNote, halfComboNote, strengthNote)
			// 半合贴身 或 身强 → [主]；无食神 → [主]；有食神且无半合 → [次]
			if hasHalfCombo || isStrong || !hasGod("食神") {
				results = append(results, comboResult{"主", desc})
			} else {
				results = append(results, comboResult{"次", desc})
			}
		}
	}

	// --- 伤官佩印：伤官 + 印星，印制伤官泄秀 ---
	// 身强时伤官为泄秀非病，佩印降为辅
	if hasGod("伤官") && (hasGod("正印") || hasGod("偏印")) {
		sgSrc := godSources("伤官")
		yinType := "印星"
		if hasGod("正印") && !hasGod("偏印") {
			yinType = "正印"
		} else if hasGod("偏印") && !hasGod("正印") {
			yinType = "偏印"
		}
		yinSrc := godSources("正印")
		yinSrc = append(yinSrc, godSources("偏印")...)
		yinStrong := yinStrongEnough()
		caiBreak := hasCaiBreakYin()
		strengthNote := ""
		if isStrong {
			strengthNote = "；身强，伤官为泄秀非病，佩印非首要"
		}
		switch {
		case !yinStrong:
			results = append(results, comboResult{"辅", fmt.Sprintf("伤官佩印条件不足（伤官在%s，%s在%s，印星未当令/无根，制伤力弱）", strings.Join(sgSrc, "、"), yinType, strings.Join(yinSrc, "、"))})
		case caiBreak:
			results = append(results, comboResult{"辅", fmt.Sprintf("伤官佩印受阻（伤官在%s，%s在%s，财星透出破印）", strings.Join(sgSrc, "、"), yinType, strings.Join(yinSrc, "、"))})
		default:
			desc := fmt.Sprintf("伤官佩印成立（伤官在%s，%s在%s，印星当令有力，无财破印%s）",
				strings.Join(sgSrc, "、"), yinType, strings.Join(yinSrc, "、"), strengthNote)
			// 身强时降为辅
			if isStrong {
				results = append(results, comboResult{"辅", desc})
			} else {
				results = append(results, comboResult{"主", desc})
			}
		}
	}

	// --- 官印相生（次）：正官 + 印星，官生印印生身 ---
	if hasGod("正官") && (hasGod("正印") || hasGod("偏印")) && !hasGod("七杀") {
		guanSrc := godSources("正官")
		yinSrc := godSources("正印")
		yinSrc = append(yinSrc, godSources("偏印")...)
		if hasCaiBreakYin() {
			results = append(results, comboResult{"辅", fmt.Sprintf("官印相生受阻（正官在%s，印星在%s，财星透出破印）", strings.Join(guanSrc, "、"), strings.Join(yinSrc, "、"))})
		} else {
			results = append(results, comboResult{"次", fmt.Sprintf("官印相生成立（正官在%s，印星在%s）", strings.Join(guanSrc, "、"), strings.Join(yinSrc, "、"))})
		}
	}

	// --- 财滋杀（忌）：财星 + 七杀，无食制无印化时财生杀攻身 ---
	if (hasGod("正财") || hasGod("偏财")) && hasGod("七杀") && !hasGod("食神") && !(hasGod("正印") || hasGod("偏印")) {
		caiSrc := godSources("正财")
		caiSrc = append(caiSrc, godSources("偏财")...)
		shaSrc := godSources("七杀")
		results = append(results, comboResult{"忌", fmt.Sprintf("财滋杀成立（财星在%s，七杀在%s），无食制无印化，杀攻身为忌", strings.Join(caiSrc, "、"), strings.Join(shaSrc, "、"))})
	}

	// --- 伤官生财（主）：伤官 + 财星，无印时泄秀生财，富格 ---
	if hasGod("伤官") && (hasGod("正财") || hasGod("偏财")) && !(hasGod("正印") || hasGod("偏印")) {
		sgSrc := godSources("伤官")
		caiSrc := godSources("正财")
		caiSrc = append(caiSrc, godSources("偏财")...)
		results = append(results, comboResult{"主", fmt.Sprintf("伤官生财成立（伤官在%s，财星在%s），泄秀生财，富格", strings.Join(sgSrc, "、"), strings.Join(caiSrc, "、"))})
	}

	// --- 食神生财（主）：食神 + 财星，无杀时泄秀生财，富格 ---
	if hasGod("食神") && (hasGod("正财") || hasGod("偏财")) && !hasGod("七杀") {
		shiSrc := godSources("食神")
		caiSrc := godSources("正财")
		caiSrc = append(caiSrc, godSources("偏财")...)
		if hasGanGod("偏印") {
			results = append(results, comboResult{"辅", fmt.Sprintf("食神生财受阻（食神在%s，财星在%s，偏印透出夺食）", strings.Join(shiSrc, "、"), strings.Join(caiSrc, "、"))})
		} else {
			results = append(results, comboResult{"主", fmt.Sprintf("食神生财成立（食神在%s，财星在%s），泄秀生财，富格", strings.Join(shiSrc, "、"), strings.Join(caiSrc, "、"))})
		}
	}

	// --- 财官相生（次）：财星 + 正官，无杀混杂时财生官，贵格 ---
	if (hasGod("正财") || hasGod("偏财")) && hasGod("正官") && !hasGod("七杀") {
		caiSrc := godSources("正财")
		caiSrc = append(caiSrc, godSources("偏财")...)
		guanSrc := godSources("正官")
		results = append(results, comboResult{"次", fmt.Sprintf("财官相生成立（财星在%s，正官在%s），财生官，贵格", strings.Join(caiSrc, "、"), strings.Join(guanSrc, "、"))})
	}

	// --- 食神吐秀（辅）：身强有食神，无杀无财时食神纯泄秀为用 ---
	if hasGod("食神") && !hasGod("七杀") && !hasGod("正财") && !hasGod("偏财") {
		shiSrc := godSources("食神")
		if hasGanGod("偏印") {
			results = append(results, comboResult{"忌", fmt.Sprintf("枭神夺食（偏印透出，食神在%s），夺食破格", strings.Join(shiSrc, "、"))})
		} else {
			results = append(results, comboResult{"辅", fmt.Sprintf("食神吐秀成立（食神在%s），身强泄秀为用", strings.Join(shiSrc, "、"))})
		}
	}

	// --- 以劫合杀（次）：劫财 + 七杀，无食制无印化时以劫合杀化凶 ---
	if hasGod("劫财") && hasGod("七杀") && !hasGod("食神") && !(hasGod("正印") || hasGod("偏印")) {
		jieSrc := godSources("劫财")
		shaSrc := godSources("七杀")
		results = append(results, comboResult{"次", fmt.Sprintf("以劫合杀成立（劫财在%s，七杀在%s），合杀化凶为吉", strings.Join(jieSrc, "、"), strings.Join(shaSrc, "、"))})
	}

	// --- 枭神夺食（忌）：偏印透干 + 食神 → 夺食破格 ---
	// 仅当偏印透出天干时才算夺食（藏支偏印力量不足以夺食）
	if hasGanGod("偏印") && hasGod("食神") {
		xiaoSrc := godSources("偏印")
		shiSrc := godSources("食神")
		// 如果食神制杀/生财已检测到夺食阻碍，此处不重复标注
		alreadyNoted := false
		for _, r := range results {
			if r.priority == "辅" && strings.Contains(r.desc, "偏印透出夺食") {
				alreadyNoted = true
				break
			}
		}
		if !alreadyNoted {
			results = append(results, comboResult{"忌", fmt.Sprintf("枭神夺食（偏印在%s，食神在%s），偏印夺食破格", strings.Join(xiaoSrc, "、"), strings.Join(shiSrc, "、"))})
		}
	}

	// --- 伤官见官（忌）：伤官 + 正官 → 为祸百端；有印化解则降为辅 ---
	if hasGod("伤官") && hasGod("正官") {
		sgSrc := godSources("伤官")
		guanSrc := godSources("正官")
		if hasGod("正印") || hasGod("偏印") {
			yinSrc := godSources("正印")
			yinSrc = append(yinSrc, godSources("偏印")...)
			results = append(results, comboResult{"辅", fmt.Sprintf("伤官见官有印化解（伤官在%s，正官在%s，印星在%s），凶中有救", strings.Join(sgSrc, "、"), strings.Join(guanSrc, "、"), strings.Join(yinSrc, "、"))})
		} else {
			results = append(results, comboResult{"忌", fmt.Sprintf("伤官见官（伤官在%s，正官在%s），无印化解，为祸百端", strings.Join(sgSrc, "、"), strings.Join(guanSrc, "、"))})
		}
	}

	// --- 贪财坏印（忌）：财星透干 + 印星 → 财破印；印太重时反为弃印就财（吉） ---
	if (hasGanGod("正财") || hasGanGod("偏财")) && (hasGod("正印") || hasGod("偏印")) {
		caiSrc := godSources("正财")
		caiSrc = append(caiSrc, godSources("偏财")...)
		yinSrc := godSources("正印")
		yinSrc = append(yinSrc, godSources("偏印")...)
		// 严格判断：只有月令本气为印星才算"印当令"，才能弃印就财
		yinWx := generates[dayWx]
		monthZhiHidegans := branchHidegan[allZhi[1]]
		yinIsLingzhu := len(monthZhiHidegans) > 0 && stemWx[monthZhiHidegans[0]] == yinWx
		if yinIsLingzhu {
			results = append(results, comboResult{"辅", fmt.Sprintf("弃印就财（财星在%s，印星在%s），印当令有力，弃印就财反为吉", strings.Join(caiSrc, "、"), strings.Join(yinSrc, "、"))})
		} else {
			results = append(results, comboResult{"忌", fmt.Sprintf("贪财坏印（财星在%s，印星在%s），财透干克印为忌", strings.Join(caiSrc, "、"), strings.Join(yinSrc, "、"))})
		}
	}

	if len(results) == 0 {
		return "无明显组合关系"
	}
	// 按优先级排序：主 > 次 > 辅 > 忌
	priOrder := map[string]int{"主": 0, "次": 1, "辅": 2, "忌": 3}
	sort.SliceStable(results, func(i, j int) bool {
		return priOrder[results[i].priority] < priOrder[results[j].priority]
	})
	parts := make([]string, len(results))
	for i, r := range results {
		parts[i] = fmt.Sprintf("[%s]%s", r.priority, r.desc)
	}
	return strings.Join(parts, "；")
}
