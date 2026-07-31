// Package bazi 提供八字四柱排盘、大运分析、神煞推算和用神分析的核心算法。
// 基于 lunar-go 库实现天文历法计算，支持真太阳时校正和晚子时处理。
package bazi

import (
	"context"
	"fmt"
	"math"
	"time"

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
	// 中国标准时间基于 120°E。真太阳时除了经度地方时，还必须叠加
	// 当天均时差；只做经度修正会漏掉约 16 分钟的季节性差异，并在子正边界
	// 错过实际跨日。先统一修正出生时刻，再交由子正换日口径计算四柱。
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
	// ---- 太阳时校正结束 ----

	// The minute is part of the birth instant. It affects true-solar-time
	// correction and must remain available to lunar-go's start-luck boundary.
	solar := calendar.NewSolar(y, m, d, h, minute, 0)
	lunar := solar.GetLunar()
	ec := lunar.GetEightChar()
	// sect=3 启用项目采用的“子正换日”口径：
	// 23:00-23:59 仍属当日，00:00-00:59 才进次日，且子时时干与当日日干保持联动。
	ec.SetSect(calendar.EightCharSectStrictZiZheng)

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
			"name":       names[i],
			"stem":       gan[i],
			"branch":     zhi[i],
			"shiShen":    shiShenGan[i],
			"naYin":      naYin[i],
			"xunKong":    xunKong[i],
			"diShi":      diShi[i],
			"xun":        xun[i],
			"hideGan":    hideGan[i],
			"subShiShen": computeSubShiShen(ec.GetDayGan(), hideGan[i]),
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
	byPillar, pillars = mergeShenshaForDisplay(byPillar, updatedPillars)

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
	startSolar := yun.GetStartSolar()
	dayun := make([]map[string]any, 0)
	for _, dy := range yun.GetDaYun() {
		startAt := startSolar
		if dy.GetIndex() > 0 {
			startAt = startSolar.NextYear((dy.GetIndex() - 1) * 10)
		}
		endAtExclusive := startSolar.NextYear(dy.GetIndex() * 10)
		if dy.GetIndex() == 0 {
			startAt = solar
			endAtExclusive = startSolar
		}
		dayun = append(dayun, map[string]any{
			"sequence":       dy.GetIndex(),
			"phase":          dayunPhase(dy.GetIndex()),
			"startAge":       dy.GetStartAge(),
			"endAge":         dy.GetEndAge(),
			"startYear":      dy.GetStartYear(),
			"endYear":        dy.GetEndYear(),
			"startAt":        startAt.ToYmdHms(),
			"endAtExclusive": endAtExclusive.ToYmdHms(),
			"ganZhi":         dy.GetGanZhi(),
		})
	}

	// lunarDate 面向展示时必须与当前四柱口径完全一致，
	// 否则子时边界会出现“四柱已换日、农历日仍停留前一日”的自相矛盾。
	return map[string]any{
		"calendar_rule_version": CalendarRuleVersion,
		"pillars":               pillars,
		"dayGan":                ec.GetDayGan(),
		// 这里展示的是“日主五行”，应只取日干五行，不取日柱复合五行。
		"dayGanWuxing": stemWx[ec.GetDayGan()],
		"wuxing":       wuxing,
		"dayun":        dayun,
		"dayun_metadata": map[string]any{
			"direction":          dayunDirection(yun.IsForward()),
			"direction_basis":    dayunDirectionBasis(ec.GetYearGan(), gender, yun.IsForward()),
			"calculation_method": "lunar_go_yun_sect1",
			"start_offset": map[string]any{
				"years":  yun.GetStartYear(),
				"months": yun.GetStartMonth(),
				"days":   yun.GetStartDay(),
				"hours":  yun.GetStartHour(),
			},
			"start_at": startSolar.ToYmdHms(),
		},
		"gender":          gender,
		"birthday":        fmt.Sprintf("%d-%02d-%02d %02d:%02d", y, m, d, h, minute),
		"mingGong":        ec.GetMingGong(),
		"mingGongNaYin":   ec.GetMingGongNaYin(),
		"shenGong":        ec.GetShenGong(),
		"shenGongNaYin":   ec.GetShenGongNaYin(),
		"taiYuan":         ec.GetTaiYuan(),
		"taiYuanNaYin":    ec.GetTaiYuanNaYin(),
		"lunarDate":       fmt.Sprintf("%s年%s月%s日", lunar.GetYearInGanZhi(), lunar.GetMonthInGanZhi(), ec.GetDay()),
		"shensha_summary": shenshaSummary,
	}, nil
}

// TrueSolarOffsetMinutes returns the total minute offset from China Standard
// Time to apparent solar time for a date and longitude.
func TrueSolarOffsetMinutes(year, month, day int, longitude float64) int {
	// NOAA 的均时差近似式以分钟返回“视太阳时 - 平太阳时”。日期粒度下
	// 误差远小于当前排盘输入的分钟精度，且避免为单一排盘工具引入外部天文依赖。
	dayOfYear := time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC).YearDay()
	gamma := 2 * math.Pi / 365 * (float64(dayOfYear) - 1)
	equationOfTime := 229.18 * (0.000075 + 0.001868*math.Cos(gamma) - 0.032077*math.Sin(gamma) - 0.014615*math.Cos(2*gamma) - 0.040849*math.Sin(2*gamma))
	longitudeCorrection := (longitude - 120.0) * 4
	return int(math.Round(longitudeCorrection + equationOfTime))
}

func dayunPhase(index int) string {
	if index == 0 {
		return "childhood"
	}
	return "dayun"
}

func dayunDirection(forward bool) string {
	if forward {
		return "forward"
	}
	return "reverse"
}

func dayunDirectionBasis(yearStem, gender string, forward bool) string {
	yinYang := "阳"
	if yearStem == "乙" || yearStem == "丁" || yearStem == "己" || yearStem == "辛" || yearStem == "癸" {
		yinYang = "阴"
	}
	direction := "顺行"
	if !forward {
		direction = "逆行"
	}
	return "年干" + yearStem + "为" + yinYang + "；" + gender + "命，按" + yinYang + "年" + gender + "命" + direction
}
