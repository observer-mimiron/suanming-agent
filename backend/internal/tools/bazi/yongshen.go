package bazi

import (
	"context"
	"fmt"
	"time"

	"github.com/6tail/lunar-go/calendar"
)

// YongShenTool 提供八字受力、月令与十神位置事实。
// 强弱、用忌、格局成败和调候优先级由 runtime 的 selected rule profile 裁断，
// 本工具不得把工程估计伪装成命理 verdict。
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

	// 与 bazi_calc 复用同一真太阳时函数，避免命盘与受力证据在子正边界使用不同日柱。
	minute := 0
	if mv, hasMin := params["minute"].(float64); hasMin {
		minute = int(mv)
	}
	if lng, hasLng := params["longitude"].(float64); hasLng && lng >= -180 && lng <= 180 {
		corrected := time.Date(y, time.Month(m), d, h, minute, 0, 0, time.UTC).
			Add(time.Duration(TrueSolarOffsetMinutes(y, m, d, lng)) * time.Minute)
		correctedYear, correctedMonth, correctedDay := corrected.Date()
		y, m, d = correctedYear, int(correctedMonth), correctedDay
		h, minute, _ = corrected.Clock()
	}

	// Keep the same corrected instant as bazi_calc. Using :00 here made the
	// evidence tool diverge from the chart whenever a supplied minute mattered.
	solar := calendar.NewSolar(y, m, d, h, minute, 0)
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
		monthScore = 8 // 得令。月令是旺衰权重核心，不能与普通藏干同级。
	} else if dayWx == seasonXiang[season] {
		monthScore = 4 // 得相。
	} else {
		monthScore = 1 // 休囚死
	}

	// 统计通根数量和根气层级。本气根与余气根不能等权，
	// 否则亥、子这种强水根会被巳、未中的杂气机械压倒。
	rootCount := 0
	rootScore := 0
	for _, z := range allZhi {
		for idx, hg := range branchHidegan[z] {
			if stemWx[hg] == dayWx {
				rootCount++
				rootScore += hiddenRootWeight(idx)
			}
		}
	}
	// 统计同五行天干数。sameElementCount 保留原始事实计数；
	// sameStemScore 只把日主以外的同类透干计入扶身侧。
	sameElementCount := 0
	sameStemScore := 0
	for _, g := range allGan {
		if stemWx[g] == dayWx {
			sameElementCount++
		}
		if g != dayGan && stemWx[g] == dayWx {
			sameStemScore += visibleStemWeight()
		}
	}
	// 统计印星生扶数和生扶力度。透干印比藏支印更直接；
	// 藏支印只作辅助资源，避免“藏庚”被放大成强印。
	generateCount := 0
	generateScore := 0
	for _, g := range allGan {
		if stemWx[g] == generatedBy[dayWx] {
			generateCount++
			generateScore += visibleStemWeight()
		}
	}
	for _, z := range allZhi {
		for idx, hg := range branchHidegan[z] {
			if stemWx[hg] == generatedBy[dayWx] {
				generateCount++
				generateScore += hiddenGenerationWeight(idx)
			}
		}
	}

	// 同时统计食伤泄、财耗和官杀克，供上游按流派裁断；不输出极旺/极弱。
	// 这里保留的是可解释的受力证据，不是完整的格局、调候或层次 verdict。
	totalSupport := monthScore + rootScore + sameStemScore + generateScore
	drainWx := drainsTo[dayWx]
	wealthWx := ke[dayWx]
	officerWx := ""
	for element, controlled := range ke {
		if controlled == dayWx {
			officerWx = element
			break
		}
	}
	pressureCount := 0
	for _, g := range allGan {
		if g == dayGan {
			continue
		}
		if wx := stemWx[g]; wx == drainWx || wx == wealthWx || wx == officerWx {
			pressureCount += visibleStemWeight()
		}
	}
	for _, z := range allZhi {
		for idx, hg := range branchHidegan[z] {
			if wx := stemWx[hg]; wx == drainWx || wx == wealthWx || wx == officerWx {
				pressureCount += hiddenPressureWeight(idx)
			}
		}
	}
	strengthMethod := "balance_evidence_v1"
	var strength string
	switch {
	case totalSupport-pressureCount >= 4:
		strength = "偏强"
	case pressureCount-totalSupport >= 4:
		strength = "偏弱"
	default:
		strength = "中和附近"
	}

	// 当前只保留季节性提示。它不是《穷通宝鉴》的逐日主、逐月令调候规则表。
	seasonalTiaohouHint := ""
	if season == "冬" {
		seasonalTiaohouHint = "冬令寒暖需由调候 profile 结合日主与月令裁断"
	} else if season == "夏" {
		seasonalTiaohouHint = "夏令燥湿需由调候 profile 结合日主与月令裁断"
	}

	// 月令主格候选（按子平月令取格口径）。候选不是成格 verdict。
	// 月令地支藏干中透出天干者，以该天干与日主的十神关系命名格局。
	// 本气透出优先，次看中气、余气。若均未透出，取月令本气暗格。
	gejuName, gejuBasis, _ := determineGeju(monthZhi, dayGan, dayWx, allGan, stemWx, branchHidegan, generatedBy)
	// 伤官候选仅记录官星是否透干；伤尽与否保留给 profile。
	if gejuName == "伤官格" {
		gejuName, gejuBasis = refineShangGuan(gejuName, gejuBasis, dayGan, allGan, stemWx, generatedBy)
	}
	// 组合只报告候选位置；不得输出主次、成格、富贵、化杀或为忌。
	gejuCombination := detectGejuCombination(dayGan, dayWx, strength, allGan, allZhi, stemWx, branchHidegan, generatedBy)
	// 冲合刑害检测（六冲/三合/三会/相刑/相害）和十神力量统计：算法只产证据字段，
	// 不下命格层次结论，由 LLM 在 interpret.md 命格层次约束清单下综合判断。
	chongheResult := computeChonghe(allZhi)
	shiShenPower := computeShiShenPower(dayGan, dayWx, allGan, allZhi, stemWx, generatedBy, branchHidegan)
	officialVisibility := tenGodVisibility("正官", dayGan, dayWx, allGan, allZhi, stemWx, generatedBy, branchHidegan)
	balanceStatus := "仅受力估计，待profile裁断"

	return map[string]any{
		"day_master":        dayGan,
		"day_master_wuxing": dayWx,
		"strength":          strength,
		"strength_method":   strengthMethod,
		"strength_evidence": map[string]any{
			"support_score":    totalSupport,
			"pressure_score":   pressureCount,
			"support_signals":  []string{"月令受力", "通根", "同类透干", "印星生扶"},
			"pressure_signals": []string{"食伤泄", "财耗", "官杀克"},
		},
		"balance_status":        balanceStatus,
		"official_visibility":   officialVisibility,
		"month_score":           monthScore,
		"root_count":            rootCount,
		"same_element":          sameElementCount,
		"generate_count":        generateCount,
		"total_support":         totalSupport,
		"season":                season,
		"seasonal_tiaohou_hint": seasonalTiaohouHint,
		"balance_yong_shen":     []string{},
		"tiaohou_yong_shen":     []string{},
		"conditional_yong_shen": []string{},
		"yong_shen":             []string{},
		"xi_shen":               []string{},
		"ji_shen":               []string{},
		"tiao_hou":              "待 qiongtong_tiaohou_v1 规则表实现",
		"geju":                  gejuName + "候选",
		"geju_candidate":        gejuName,
		"geju_basis":            gejuBasis,
		"geju_status":           "待profile裁断",
		"geju_detail":           "仅已输出月令与透干候选，成败、清浊和变格待 selected rule profile 裁断",
		"geju_qing_zhuo":        "待profile裁断",
		"geju_qing_zhuo_reason": map[string]any{
			"label":    "待profile裁断",
			"summary":  "清浊需要格局、成败、救应与反证共同裁断。",
			"evidence": []string{gejuBasis},
			"signals":  map[string]any{},
		},
		"geju_combination": gejuCombination,
		"chonghe":          chongheResult,
		"shi_shen_power":   shiShenPower,
		"gender":           gender,
	}, nil
}

func visibleStemWeight() int {
	return 2
}

func hiddenRootWeight(index int) int {
	switch index {
	case 0:
		return 3
	case 1:
		return 2
	default:
		return 1
	}
}

func hiddenGenerationWeight(index int) int {
	if index == 0 {
		return 2
	}
	return 1
}

func hiddenPressureWeight(index int) int {
	if index == 0 {
		return 2
	}
	return 1
}

// tenGodVisibility 将十神的透干和藏支位置分开输出，避免“官未透”被误写成“无官星”。
func tenGodVisibility(targetGod, dayGan, dayWx string, allGan, allZhi []string,
	stemWx, generates map[string]string, branchHidegan map[string][]string) map[string]any {
	visible := []map[string]string{}
	hidden := []map[string]string{}
	for _, info := range collectTenGods(dayGan, dayWx, allGan, allZhi, stemWx, generates, branchHidegan) {
		if info.god != targetGod {
			continue
		}
		if info.isGan {
			visible = append(visible, map[string]string{"pillar": info.source, "stem": allGan[info.pillarIdx]})
			continue
		}
		hidden = append(hidden, map[string]string{
			"pillar": info.source,
			"branch": info.branch,
			"stem":   info.hiddenGan,
			"tier":   hiddenStemTierName(info.hiddenIdx),
		})
	}
	return map[string]any{"visible": visible, "hidden": hidden}
}
