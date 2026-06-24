package bazi

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/6tail/lunar-go/calendar"
)

// YongShenTool 用神推算工具。基于八字四柱的天干地支，从月令得时、通根、生扶三个维度
// 综合评定日主强弱（身旺/身弱/中和），并根据身强喜克泄耗、身弱喜生扶的原则推荐用神、喜神和忌神。
type YongShenTool struct{}

func (t *YongShenTool) Name() string        { return "yongshen" }
func (t *YongShenTool) Description() string { return "分析日主强弱并推荐用神喜忌" }

func (t *YongShenTool) Label() string { return "用神分析" }

func (t *YongShenTool) Execute(_ context.Context, params map[string]any) (any, error) {
	year, _ := params["year"].(float64)
	month, _ := params["month"].(float64)
	day, _ := params["day"].(float64)
	hour, _ := params["hour"].(float64)
	gender, _ := params["gender"].(string)

	if year < 1900 || year > 2100 {
		return nil, fmt.Errorf("year out of range")
	}

	y, m, d, h := int(year), int(month), int(day), int(hour)

	// 真太阳时校正（与 bazi_calc 保持一致的逻辑）：基于经度偏差（差1度约4分钟）修正当地时间，
	// 以确保排盘时月柱和日柱的正确性，特别是跨日界的情况（如新疆23:30钟表时可能为次日凌晨）
	minute := 0
	if mv, hasMin := params["minute"].(float64); hasMin {
		minute = int(mv)
	}
	if lng, hasLng := params["longitude"].(float64); hasLng && lng >= -180 && lng <= 180 {
		offsetMinutes := int((lng - 120.0) * 4)
		solarMinutes := h*60 + minute + offsetMinutes
		for solarMinutes < 0 {
			solarMinutes += 24 * 60
			d--
		}
		for solarMinutes >= 24*60 {
			solarMinutes -= 24 * 60
			d++
		}
		h = solarMinutes / 60
	}

	solar := calendar.NewSolar(y, m, d, h, 0, 0)
	lunar := solar.GetLunar()
	ec := lunar.GetEightChar()
	ec.SetSect(1) // 启用晚子时处理

	dayGan := ec.GetDayGan()
	dayZhi := ec.GetDayZhi()
	monthZhi := ec.GetMonthZhi()

	// 构建完整的四柱干支列表用于后续分析
	allGan := []string{ec.GetYearGan(), ec.GetMonthGan(), dayGan, ec.GetTimeGan()}
	allZhi := []string{ec.GetYearZhi(), monthZhi, dayZhi, ec.GetTimeZhi()}

	// 将天干地支映射到五行
	stemWx := map[string]string{
		"甲": "木", "乙": "木", "丙": "火", "丁": "火", "戊": "土",
		"己": "土", "庚": "金", "辛": "金", "壬": "水", "癸": "水",
	}
	// 地支藏干表（简化版主气）
	branchHidegan := map[string][]string{
		"子": {"癸"}, "丑": {"己", "辛", "癸"}, "寅": {"甲", "丙", "戊"},
		"卯": {"乙"}, "辰": {"戊", "乙", "癸"}, "巳": {"丙", "戊", "庚"},
		"午": {"丁", "己"}, "未": {"己", "丁", "乙"}, "申": {"庚", "壬", "戊"},
		"酉": {"辛"}, "戌": {"戊", "辛", "丁"}, "亥": {"壬", "甲"},
	}
	// 生某五行的元素（印星）
	generates := map[string]string{"木": "水", "火": "木", "土": "火", "金": "土", "水": "金"}

	dayWx := stemWx[dayGan] // 如"土"

	// 判断月令旺衰状态
	monthSeasons := map[string]string{
		"寅": "春", "卯": "春", "辰": "春",
		"巳": "夏", "午": "夏", "未": "夏",
		"申": "秋", "酉": "秋", "戌": "秋",
		"亥": "冬", "子": "冬", "丑": "冬",
	}
	season := monthSeasons[monthZhi]

	// 五行四时旺相表：各季节中旺相的元素
	seasonWang := map[string]string{"春": "木", "夏": "火", "秋": "金", "冬": "水", "土": "土"}
	seasonXiang := map[string]string{"春": "火", "夏": "土", "秋": "水", "冬": "木"}

	monthScore := 0
	if dayWx == seasonWang[season] {
		monthScore = 3 // 得时
	} else if dayWx == seasonXiang[season] {
		monthScore = 2 // 相
	} else {
		monthScore = 1 // 休囚死
	}

	// 统计通根数量
	rootCount := 0
	for _, z := range allZhi {
		for _, hg := range branchHidegan[z] {
			if stemWx[hg] == dayWx {
				rootCount++
			}
		}
	}
	// 统计同五行天干数
	sameElementCount := 0
	for _, g := range allGan {
		if stemWx[g] == dayWx {
			sameElementCount++
		}
	}
	// 统计印星生扶数
	generateCount := 0
	for _, g := range allGan {
		if stemWx[g] == generates[dayWx] {
			generateCount++
		}
	}
	for _, z := range allZhi {
		for _, hg := range branchHidegan[z] {
			if stemWx[hg] == generates[dayWx] {
				generateCount++
			}
		}
	}

	// 日主强弱判定
	totalSupport := monthScore + rootCount + generateCount
	var strength string
	switch {
	case totalSupport >= 7:
		strength = "身旺极"
	case totalSupport >= 5:
		strength = "身强"
	case totalSupport >= 3:
		strength = "中和"
	default:
		strength = "身弱"
	}
	if totalSupport <= 1 {
		strength = "身弱极"
	}

	// 定用神
	var yongShen, xiShen, jiShen []string
	wx := []string{"木", "火", "土", "金", "水"}
	switch strength {
	case "身强", "身旺极":
		// 身强需克泄耗
		// 用神：食伤（泄）、财星（耗）、官杀（克）
		for _, e := range wx {
			for _, g := range wx {
				if stemWx[g] == dayWx {
					// 食伤 = 日主生 x
					if generates[e] == dayWx {
						yongShen = append(yongShen, e)
					}
					// 财星 = 日主克 x
					if generates[dayWx] == e {
						yongShen = append(yongShen, e)
					}
				}
			}
		}
		// 官杀 = 克日主
		for _, e := range wx {
			if generates[e] == dayWx && dayWx != e {
				yongShen = append(yongShen, e)
			}
		}
	default: // 身弱, 身弱极, 中和
		// 身弱需生扶
		yongShen = append(yongShen, generates[dayWx])        // 印星
		yongShen = append(yongShen, dayWx)                    // 比劫
	}

	// 判断调候需求（季节性调整）
	tiaoHouNeed := ""
	if season == "冬" {
		tiaoHouNeed = "需火调候暖局"
	} else if season == "夏" {
		tiaoHouNeed = "需水调候润局"
	}

	// 去重
	dedup := func(s []string) []string {
		seen := map[string]bool{}
		var r []string
		for _, v := range s {
			if !seen[v] {
				r = append(r, v)
				seen[v] = true
			}
		}
		return r
	}
	yongShen = dedup(yongShen)

	// 计算忌神（用神的对立面）
	yongSet := map[string]bool{}
	for _, y := range yongShen {
		yongSet[y] = true
	}
	for _, e := range wx {
		if !yongSet[e] && e != dayWx {
			jiShen = append(jiShen, e)
		}
	}

	// 喜神（生扶用神的五行）
	for _, y := range yongShen {
		g := generates[y]
		if g != "" && g != y {
			xiShen = append(xiShen, g)
		}
	}
	xiShen = dedup(xiShen)

	// 格局判定（按《子平真诠》月令藏干透干取格法）。
	// 月令地支藏干中透出天干者，以该天干与日主的十神关系命名格局。
	// 本气透出优先，次看中气、余气。若均未透出，取月令本气暗格。
	gejuName, gejuBasis, qingZhuo := determineGeju(monthZhi, dayGan, dayWx, allGan, stemWx, branchHidegan, generates)
	// 伤官格细分：判断是伤尽（无官星，为贵）还是见官（有官星，破格风险）
	if gejuName == "伤官格" {
		gejuName, gejuBasis = refineShangGuan(gejuName, gejuBasis, dayGan, allGan, stemWx, generates)
	}
	// 变格检测：身极旺或极弱时优先取变格（子平真诠："从格之法，以日主弱极无依为最"）
	if bgName := checkBianGe(strength, dayGan, dayWx, allGan, allZhi, stemWx, branchHidegan); bgName != "" {
		gejuName = bgName
		gejuBasis = "变格：日主" + strength + "，符合" + bgName + "条件"
		qingZhuo = "变"
	}
	gejuStatus, gejuDetail := checkGejuChengbai(gejuName, dayGan, allGan, stemWx, generates)
	// 组合关系检测：检测杀印相生、伤官佩印、食神制杀等组合条件及阻碍因素。
	// 代码只做条件检测，力量判断和最终成立与否由 LLM 分析。
	gejuCombination := detectGejuCombination(dayGan, dayWx, strength, allGan, allZhi, stemWx, branchHidegan, generates)

	return map[string]any{
		"day_master":        dayGan,
		"day_master_wuxing": dayWx,
		"strength":          strength,
		"month_score":       monthScore,
		"root_count":        rootCount,
		"same_element":      sameElementCount,
		"generate_count":    generateCount,
		"total_support":     totalSupport,
		"season":            season,
		"yong_shen":         yongShen,
		"xi_shen":           xiShen,
		"ji_shen":           jiShen,
		"tiao_hou":          tiaoHouNeed,
		"geju":              gejuName,
		"geju_basis":        gejuBasis,
		"geju_status":       gejuStatus,
		"geju_detail":       gejuDetail,
		"geju_qing_zhuo":    qingZhuo,
		"geju_combination":  gejuCombination,
		"gender":            gender,
	}, nil
}

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

	// 十神关系判定（闭包）
	tenGodOf := func(stem string) string {
		stemW := stemWx[stem]
		if stemW == dayWx {
			if isYangStem(dayGan) == isYangStem(stem) {
				return "比肩"
			}
			return "劫财"
		}
		if generates[dayWx] == stemW {
			if isYangStem(dayGan) == isYangStem(stem) {
				return "偏印"
			}
			return "正印"
		}
		if generates[stemW] == dayWx {
			if isYangStem(dayGan) == isYangStem(stem) {
				return "食神"
			}
			return "伤官"
		}
		ke := map[string]string{"木": "土", "土": "水", "水": "火", "火": "金", "金": "木"}
		if ke[dayWx] == stemW {
			if isYangStem(dayGan) == isYangStem(stem) {
				return "偏财"
			}
			return "正财"
		}
		if ke[stemW] == dayWx {
			if isYangStem(dayGan) == isYangStem(stem) {
				return "七杀"
			}
			return "正官"
		}
		return "未知"
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
			tg := tenGodOf(hg)
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
	tg := tenGodOf(hidegans[0])
	if tg == "比肩" && monthZhi != luTable[dayGan] {
		if len(hidegans) > 1 {
			tg = tenGodOf(hidegans[1])
			usedIdx = 1
		}
	}
	if tg == "劫财" && monthZhi != renTable[dayGan] {
		if len(hidegans) > 1 {
			tg = tenGodOf(hidegans[1])
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
	ke := map[string]string{"木": "土", "土": "水", "水": "火", "火": "金", "金": "木"}

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
	ke := map[string]string{"木": "土", "土": "水", "水": "火", "火": "金", "金": "木"}

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

// isYangStem 判断天干是否为阳干。
func isYangStem(stem string) bool {
	switch stem {
	case "甲", "丙", "戊", "庚", "壬":
		return true
	default:
		return false
	}
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

// tenGodOf 计算某天干对日主的十神关系。
func tenGodOf(stem, dayGan, stemW, dayW string, generates map[string]string) string {
	if stemW == dayW {
		if isYangStem(dayGan) == isYangStem(stem) {
			return "比肩"
		}
		return "劫财"
	}
	if generates[dayW] == stemW {
		if isYangStem(dayGan) == isYangStem(stem) {
			return "偏印"
		}
		return "正印"
	}
	if generates[stemW] == dayW {
		if isYangStem(dayGan) == isYangStem(stem) {
			return "食神"
		}
		return "伤官"
	}
	ke := map[string]string{"木": "土", "土": "水", "水": "火", "火": "金", "金": "木"}
	if ke[dayW] == stemW {
		if isYangStem(dayGan) == isYangStem(stem) {
			return "偏财"
		}
		return "正财"
	}
	if ke[stemW] == dayW {
		if isYangStem(dayGan) == isYangStem(stem) {
			return "七杀"
		}
		return "正官"
	}
	return "未知"
}

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

	// 收集命盘中所有十神（天干透出 + 地支藏干），记录来源位置
	type tenGodInfo struct {
		god       string
		source    string // "年干"/"月干"/"时干" or "年支"/"月支"/"日支"/"时支"
		isGan     bool
		pillarIdx int // 0=年, 1=月, 2=日, 3=时
	}
	allGods := []tenGodInfo{}
	posNames := []string{"年", "月", "日", "时"}
	for i, g := range allGan {
		if g == dayGan {
			continue // 排除日主
		}
		tg := tenGodOf(g, dayGan, stemWx[g], dayWx, generates)
		allGods = append(allGods, tenGodInfo{god: tg, source: posNames[i] + "干", isGan: true, pillarIdx: i})
	}
	for i, z := range allZhi {
		for _, hg := range branchHidegan[z] {
			if hg == dayGan {
				continue
			}
			tg := tenGodOf(hg, dayGan, stemWx[hg], dayWx, generates)
			allGods = append(allGods, tenGodInfo{god: tg, source: posNames[i] + "支", isGan: false, pillarIdx: i})
		}
	}

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
