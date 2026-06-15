package tools

import (
	"context"
	"fmt"

	"github.com/6tail/lunar-go/calendar"
	"github.com/cloudwego/eino/schema"
)

// BaziCalcTool 八字排盘工具。根据出生年月日时和性别，计算四柱八字（年柱、月柱、日柱、时柱）
// 包含十神、纳音、旬空、地支藏干、地势等信息，并统计五行分布和大运。
type BaziCalcTool struct{}

func (t *BaziCalcTool) Name() string        { return "bazi_calc" }
func (t *BaziCalcTool) Description() string { return "计算八字排盘，输入出生年月日时+性别" }
func (t *BaziCalcTool) EinoToolInfo() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: t.Description(),
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"year":   {Type: schema.Number, Desc: "出生年，1900-2100", Required: true},
			"month":  {Type: schema.Number, Desc: "出生月，1-12", Required: true},
			"day":    {Type: schema.Number, Desc: "出生日，1-31", Required: true},
			"hour":   {Type: schema.Number, Desc: "出生小时，0-23", Required: true},
			"gender": {Type: schema.String, Desc: "性别，男或女", Enum: []string{"男", "女"}, Required: true},
		}),
	}
}

func (t *BaziCalcTool) Execute(_ context.Context, params map[string]any) (any, error) {
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
		}
	}

	// 五行统计：将天干地支映射为五行（木火土金水），分别对四柱统计各五行出现次数
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

	// 大运计算：lunar-go 根据性别阴阳自动推算起运时间和各步大运干支
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
		"pillars":    pillars,
		"dayGan":     ec.GetDayGan(),
		"dayGanWuxing": ec.GetDayWuXing(),
		"wuxing":     wuxing,
		"dayun":      dayun,
		"gender":     gender,
		"birthday":   fmt.Sprintf("%d-%02d-%02d %02d:00", y, m, d, h),
		"mingGong":   ec.GetMingGong(),
		"mingGongNaYin": ec.GetMingGongNaYin(),
		"shenGong":   ec.GetShenGong(),
		"shenGongNaYin": ec.GetShenGongNaYin(),
		"taiYuan":    ec.GetTaiYuan(),
		"taiYuanNaYin": ec.GetTaiYuanNaYin(),
		"lunarDate":  fmt.Sprintf("%s年%s月%s日", lunar.GetYearInGanZhi(), lunar.GetMonthInGanZhi(), lunar.GetDayInGanZhi()),
	}, nil
}
