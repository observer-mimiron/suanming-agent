package bazi

import (
	"context"
	"fmt"

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
	// 与 bazi_calc 保持同一“子正换日”口径，避免日主与时柱来自两套不同规则。
	ec.SetSect(calendar.EightCharSectStrictZiZheng)

	dayGan := ec.GetDayGan()
	dayZhi := ec.GetDayZhi()
	monthZhi := ec.GetMonthZhi()

	// 构建完整的四柱干支列表用于后续分析
	allGan := []string{ec.GetYearGan(), ec.GetMonthGan(), dayGan, ec.GetTimeGan()}
	allZhi := []string{ec.GetYearZhi(), monthZhi, dayZhi, ec.GetTimeZhi()}

	// stemWx/branchHidegan 引用 tables.go 包级表。
	// generatedBy 为“印星方向”（木←水，即水生木），drainsTo 为“食伤方向”（木→火）。
	// 两个方向在后续用神推导里都会用到，显式拆开可避免把五行值误当成天干键。
	generatedBy := map[string]string{"木": "水", "火": "木", "土": "火", "金": "土", "水": "金"}
	drainsTo := map[string]string{"木": "火", "火": "土", "土": "金", "金": "水", "水": "木"}

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
		if stemWx[g] == generatedBy[dayWx] {
			generateCount++
		}
	}
	for _, z := range allZhi {
		for _, hg := range branchHidegan[z] {
			if stemWx[hg] == generatedBy[dayWx] {
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

	// 定用神：拆成“平衡主用 / 调候先务 / 条件性辅助”三层，避免把条件性信号误写成稳定主用。
	var yongShen, xiShen, jiShen []string
	var balanceYongShen, tiaohouYongShen, conditionalYongShen []string
	wx := []string{"木", "火", "土", "金", "水"}
	switch strength {
	case "身强", "身旺极":
		// 身强先取泄与耗为平衡主用；官杀虽可克身，但同时转生印星，通常保留为条件性辅助。
		balanceYongShen = append(balanceYongShen, drainsTo[dayWx])
		balanceYongShen = append(balanceYongShen, ke[dayWx])
		for _, e := range wx {
			if ke[e] == dayWx {
				conditionalYongShen = append(conditionalYongShen, e)
				break
			}
		}
	default: // 身弱, 身弱极, 中和
		// 身弱需生扶
		balanceYongShen = append(balanceYongShen, generatedBy[dayWx]) // 印星
		balanceYongShen = append(balanceYongShen, dayWx)              // 比劫
	}

	// 判断调候需求（季节性调整）
	tiaoHouNeed := ""
	if season == "冬" {
		tiaoHouNeed = "需火调候暖局"
		tiaohouYongShen = append(tiaohouYongShen, "火")
	} else if season == "夏" {
		tiaoHouNeed = "需水调候润局"
		tiaohouYongShen = append(tiaohouYongShen, "水")
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
	balanceYongShen = dedup(balanceYongShen)
	tiaohouYongShen = dedup(tiaohouYongShen)
	conditionalYongShen = dedup(conditionalYongShen)
	yongShen = append(yongShen, tiaohouYongShen...)
	yongShen = append(yongShen, balanceYongShen...)
	yongShen = append(yongShen, conditionalYongShen...)
	yongShen = dedup(yongShen)

	// 计算忌神（用神的对立面）
	yongSet := map[string]bool{}
	for _, y := range yongShen {
		yongSet[y] = true
	}
	for _, e := range wx {
		if !yongSet[e] {
			jiShen = append(jiShen, e)
		}
	}

	// 喜神只取“与用神同方向、且不会脱离用神集合”的辅助承接。
	// 这样可避免把本已判为忌神的方向，又因“生扶用神”被机械推回喜神。
	for _, y := range yongShen {
		g := generatedBy[y]
		if g != "" && g != y && yongSet[g] {
			xiShen = append(xiShen, g)
		}
	}
	xiShen = dedup(xiShen)
	if len(xiShen) == 0 {
		xiShen = append(xiShen, yongShen...)
	}

	// 格局判定（按《子平真诠》月令藏干透干取格法）。
	// 月令地支藏干中透出天干者，以该天干与日主的十神关系命名格局。
	// 本气透出优先，次看中气、余气。若均未透出，取月令本气暗格。
	gejuName, gejuBasis, qingZhuo := determineGeju(monthZhi, dayGan, dayWx, allGan, stemWx, branchHidegan, generatedBy)
	// 伤官格细分：判断是伤尽（无官星，为贵）还是见官（有官星，破格风险）
	if gejuName == "伤官格" {
		gejuName, gejuBasis = refineShangGuan(gejuName, gejuBasis, dayGan, allGan, stemWx, generatedBy)
	}
	// 变格检测：身极旺或极弱时优先取变格（子平真诠："从格之法，以日主弱极无依为最"）
	if bgName := checkBianGe(strength, dayGan, dayWx, allGan, allZhi, stemWx, branchHidegan); bgName != "" {
		gejuName = bgName
		gejuBasis = "变格：日主" + strength + "，符合" + bgName + "条件"
		qingZhuo = "变"
	}
	gejuStatus, gejuDetail := checkGejuChengbai(gejuName, dayGan, allGan, stemWx, generatedBy)
	// 组合关系检测：检测杀印相生、伤官佩印、食神制杀等组合条件及阻碍因素。
	// 代码只做条件检测，力量判断和最终成立与否由 LLM 分析。
	gejuCombination := detectGejuCombination(dayGan, dayWx, strength, allGan, allZhi, stemWx, branchHidegan, generatedBy)
	qingZhuo, qingZhuoReason := evaluateGejuQingZhuo(gejuName, qingZhuo, gejuStatus, gejuDetail, gejuCombination)
	// 冲合刑害检测（六冲/三合/三会/相刑/相害）和十神力量统计：算法只产证据字段，
	// 不下命格层次结论，由 LLM 在 interpret.md 命格层次约束清单下综合判断。
	chongheResult := computeChonghe(allZhi)
	shiShenPower := computeShiShenPower(dayGan, dayWx, allGan, allZhi, stemWx, generatedBy, branchHidegan)

	return map[string]any{
		"day_master":            dayGan,
		"day_master_wuxing":     dayWx,
		"strength":              strength,
		"month_score":           monthScore,
		"root_count":            rootCount,
		"same_element":          sameElementCount,
		"generate_count":        generateCount,
		"total_support":         totalSupport,
		"season":                season,
		"balance_yong_shen":     balanceYongShen,
		"tiaohou_yong_shen":     tiaohouYongShen,
		"conditional_yong_shen": conditionalYongShen,
		"yong_shen":             yongShen,
		"xi_shen":               xiShen,
		"ji_shen":               jiShen,
		"tiao_hou":              tiaoHouNeed,
		"geju":                  gejuName,
		"geju_basis":            gejuBasis,
		"geju_status":           gejuStatus,
		"geju_detail":           gejuDetail,
		"geju_qing_zhuo":        qingZhuo,
		"geju_qing_zhuo_reason": map[string]any{
			"label":    qingZhuoReason.Label,
			"summary":  qingZhuoReason.Summary,
			"evidence": qingZhuoReason.Evidence,
			"signals":  qingZhuoReason.Signals,
		},
		"geju_combination": gejuCombination,
		"chonghe":          chongheResult,
		"shi_shen_power":   shiShenPower,
		"gender":           gender,
	}, nil
}
