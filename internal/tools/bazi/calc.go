// Package bazi 提供八字四柱排盘、大运分析、神煞推算和用神分析的核心算法。
// 基于 lunar-go 库实现天文历法计算，支持真太阳时校正和晚子时处理。
package bazi

import (
	"context"
	"fmt"

	"github.com/6tail/lunar-go/calendar"
)

// CalcTool 八字排盘核心工具。根据出生年月日时和性别，计算四柱八字，包含十神、纳音、旬空、
// 藏干、地势、五行统计、神煞、大运等完整命盘信息。支持经度参数做真太阳时校正。
type CalcTool struct{}

func (t *CalcTool) Name() string { return "bazi_calc" }
func (t *CalcTool) Description() string {
	return "计算八字排盘，输入出生年月日时+性别，可选longitude(经度)做真太阳时校正"
}

func (t *CalcTool) Label() string { return "八字排盘" }

func (t *CalcTool) Execute(_ context.Context, params map[string]any) (any, error) {
	year, ok := params["year"].(float64)
	if !ok || year < 1900 || year > 2100 {
		return nil, fmt.Errorf("year out of range")
	}
	month, ok := params["month"].(float64)
	if !ok || month < 1 || month > 12 {
		return nil, fmt.Errorf("month out of range")
	}
	day, ok := params["day"].(float64)
	if !ok || day < 1 || day > 31 {
		return nil, fmt.Errorf("day out of range")
	}
	hour, ok := params["hour"].(float64)
	if !ok || hour < 0 || hour > 23 {
		return nil, fmt.Errorf("hour out of range")
	}
	gender, ok := params["gender"].(string)
	if !ok || (gender != "男" && gender != "女") {
		return nil, fmt.Errorf("gender must be 男/女")
	}

	y, m, d, h := int(year), int(month), int(day), int(hour)

	// ---- 真太阳时校正 ----
	// 中国标准时间基于 120°E。如果用户提供出生地经度，做太阳时校正。
	// 每差 1° 经度，太阳时差约 4 分钟。>120°E 加时间，<120°E 减时间。
	// 校正可能跨越日界（如新疆 23:30 钟表时间 → 太阳时次日 00:xx）。
	origHour := h
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
		minute = solarMinutes % 60
		if h != origHour {
			// 已校正 — 下方生日字符串使用修正后的日期时辰
		}
	}
	// ---- 太阳时校正结束 ----

	solar := calendar.NewSolar(y, m, d, h, 0, 0)
	lunar := solar.GetLunar()
	ec := lunar.GetEightChar()
	// sect=1 启用晚子时处理：23:00-23:59 的日柱算次日
	// lunar-go 默认 sect=2（不处理晚子时），需要手动设为 1
	ec.SetSect(1)

	gan := []string{ec.GetYearGan(), ec.GetMonthGan(), ec.GetDayGan(), ec.GetTimeGan()}
	zhi := []string{ec.GetYearZhi(), ec.GetMonthZhi(), ec.GetDayZhi(), ec.GetTimeZhi()}
	shiShenGan := []string{ec.GetYearShiShenGan(), ec.GetMonthShiShenGan(), ec.GetDayShiShenGan(), ec.GetTimeShiShenGan()}
	naYin := []string{ec.GetYearNaYin(), ec.GetMonthNaYin(), ec.GetDayNaYin(), ec.GetTimeNaYin()}
	xunKong := []string{ec.GetYearXunKong(), ec.GetMonthXunKong(), ec.GetDayXunKong(), ec.GetTimeXunKong()}
	diShi := []string{ec.GetYearDiShi(), ec.GetMonthDiShi(), ec.GetDayDiShi(), ec.GetTimeDiShi()}
	xun := []string{ec.GetYearXun(), ec.GetMonthXun(), ec.GetDayXun(), ec.GetTimeXun()}
	hideGan := [][]string{ec.GetYearHideGan(), ec.GetMonthHideGan(), ec.GetDayHideGan(), ec.GetTimeHideGan()}
	names := []string{"年柱", "月柱", "日柱", "时柱"}

	pillars := make([]map[string]any, 4)
	for i := 0; i < 4; i++ {
		pillars[i] = map[string]any{
			"name":    names[i],
			"stem":    gan[i],
			"branch":  zhi[i],
			"shiShen": shiShenGan[i],
			"naYin":   naYin[i],
			"xunKong": xunKong[i],
			"diShi":   diShi[i],
			"xun":     xun[i],
			"hideGan": hideGan[i],
		}
	}

	// 五行统计
	stemWuxing := map[string]string{
		"甲": "木", "乙": "木", "丙": "火", "丁": "火", "戊": "土",
		"己": "土", "庚": "金", "辛": "金", "壬": "水", "癸": "水",
	}
	branchWuxing := map[string]string{
		"子": "水", "丑": "土", "寅": "木", "卯": "木", "辰": "土", "巳": "火",
		"午": "火", "未": "土", "申": "金", "酉": "金", "戌": "土", "亥": "水",
	}
	wuxing := map[string]int{"木": 0, "火": 0, "土": 0, "金": 0, "水": 0}
	for _, g := range gan {
		if w, ok := stemWuxing[g]; ok {
			wuxing[w]++
		}
	}
	for _, z := range zhi {
		if w, ok := branchWuxing[z]; ok {
			wuxing[w]++
		}
	}

	// 神煞
	byPillar, updatedPillars := computeShensha(gan, zhi, xunKong, pillars)
	pillars = updatedPillars

	allNames := make([]string, 0)
	seenSS := map[string]bool{}
	for _, items := range byPillar {
		for _, item := range items {
			if !seenSS[item.Name] {
				allNames = append(allNames, item.Name)
				seenSS[item.Name] = true
			}
		}
	}
	shenshaSummary := map[string]any{
		"all":       allNames,
		"by_pillar": byPillar,
	}

	// 大运
	genderInt := 0
	if gender == "男" {
		genderInt = 1
	}
	yun := ec.GetYun(genderInt)
	dayun := make([]map[string]any, 0)
	for _, dy := range yun.GetDaYun() {
		dayun = append(dayun, map[string]any{
			"startAge": dy.GetStartAge(),
			"endAge":   dy.GetEndAge(),
			"ganZhi":   dy.GetGanZhi(),
		})
	}

	return map[string]any{
		"pillars":         pillars,
		"dayGan":          ec.GetDayGan(),
		"dayGanWuxing":    ec.GetDayWuXing(),
		"wuxing":          wuxing,
		"dayun":           dayun,
		"gender":          gender,
		"birthday":        fmt.Sprintf("%d-%02d-%02d %02d:%02d", y, m, d, h, minute),
		"mingGong":        ec.GetMingGong(),
		"mingGongNaYin":   ec.GetMingGongNaYin(),
		"shenGong":        ec.GetShenGong(),
		"shenGongNaYin":   ec.GetShenGongNaYin(),
		"taiYuan":         ec.GetTaiYuan(),
		"taiYuanNaYin":    ec.GetTaiYuanNaYin(),
		"lunarDate":       fmt.Sprintf("%s年%s月%s日", lunar.GetYearInGanZhi(), lunar.GetMonthInGanZhi(), lunar.GetDayInGanZhi()),
		"shensha_summary": shenshaSummary,
	}, nil
}
