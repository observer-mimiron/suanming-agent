package tools

import (
	"fmt"

	"github.com/6tail/lunar-go/calendar"
)

type BaziCalcTool struct{}

func (t *BaziCalcTool) Name() string        { return "bazi_calc" }
func (t *BaziCalcTool) Description() string { return "计算八字排盘，输入出生年月日时+性别" }

func (t *BaziCalcTool) Execute(params map[string]any) (any, error) {
	// 参数校验
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

	solar := calendar.NewSolar(y, m, d, h, 0, 0)
	lunar := solar.GetLunar()
	ec := lunar.GetEightChar()

	// 四柱：天干 + 地支 + 十神（十神用 ShiShenGan 方法，计算天干相对日干的十神）
	gan := []string{ec.GetYearGan(), ec.GetMonthGan(), ec.GetDayGan(), ec.GetTimeGan()}
	zhi := []string{ec.GetYearZhi(), ec.GetMonthZhi(), ec.GetDayZhi(), ec.GetTimeZhi()}
	shiShen := []string{ec.GetYearShiShenGan(), ec.GetMonthShiShenGan(), ec.GetDayShiShenGan(), ec.GetTimeShiShenGan()}
	names := []string{"年柱", "月柱", "日柱", "时柱"}

	pillars := make([]map[string]string, 4)
	for i := 0; i < 4; i++ {
		pillars[i] = map[string]string{
			"name":    names[i],
			"stem":    gan[i],
			"branch":  zhi[i],
			"shiShen": shiShen[i],
		}
	}

	// 五行：按四柱天干地支共 8 个字统计
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

	// 大运 (1=男, 0=女)
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
		"pillars":  pillars,
		"dayGan":   ec.GetDayGan(),
		"wuxing":   wuxing,
		"dayun":    dayun,
		"gender":   gender,
		"birthday": fmt.Sprintf("%d-%02d-%02d %02d:00", y, m, d, h),
	}, nil
}
