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
		"gender":            gender,
	}, nil
}
