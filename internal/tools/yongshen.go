package tools

import (
	"context"
	"fmt"

	"github.com/6tail/lunar-go/calendar"
	"github.com/cloudwego/eino/schema"
)

// YongShenTool 用神推算工具。根据八字四柱的天干地支，通过月令得时、通根数量、印星生扶等维度分析日主强弱，
// 推荐用神（对日主最有利的五行）、喜神（生扶用神的五行）、忌神（克制日主的五行），并判断季节性调候需求。
type YongShenTool struct{}

func (t *YongShenTool) Name() string        { return "yongshen" }
func (t *YongShenTool) Description() string { return "分析日主强弱并推荐用神喜忌" }
func (t *YongShenTool) EinoToolInfo() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: t.Description(),
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"year":   {Type: schema.Number, Desc: "出生年，1900-2100", Required: true},
			"month":  {Type: schema.Number, Desc: "出生月，1-12", Required: true},
			"day":    {Type: schema.Number, Desc: "出生日，1-31", Required: true},
			"hour":   {Type: schema.Number, Desc: "出生小时，0-23", Required: true},
			"gender": {Type: schema.String, Desc: "性别，男或女", Enum: []string{"男", "女"}},
		}),
	}
}

func (t *YongShenTool) Execute(_ context.Context, params map[string]any) (any, error) {
	year, _ := params["year"].(float64)
	month, _ := params["month"].(float64)
	day, _ := params["day"].(float64)
	hour, _ := params["hour"].(float64)
	gender, _ := params["gender"].(string)

	if year < 1900 || year > 2100 {
		return nil, fmt.Errorf("year out of range")
	}

	solar := calendar.NewSolar(int(year), int(month), int(day), int(hour), 0, 0)
	lunar := solar.GetLunar()
	ec := lunar.GetEightChar()

	dayGan := ec.GetDayGan()
	dayZhi := ec.GetDayZhi()
	monthZhi := ec.GetMonthZhi()

	// 构建完整的四柱干支列表用于后续分析
	allGan := []string{ec.GetYearGan(), ec.GetMonthGan(), dayGan, ec.GetTimeGan()}
	allZhi := []string{ec.GetYearZhi(), monthZhi, dayZhi, ec.GetTimeZhi()}

	// 天干五行映射表：十天干对应五行（甲乙木、丙丁火、戊己土、庚辛金、壬癸水）
	stemWx := map[string]string{
		"甲": "木", "乙": "木", "丙": "火", "丁": "火", "戊": "土",
		"己": "土", "庚": "金", "辛": "金", "壬": "水", "癸": "水",
	}
	// 地支藏干表（简化版，取主气）：每个地支所含的天干，如子藏癸、丑藏己辛癸等，
	// 用于判断日主是否在地支中有"根"（通根）
	branchHidegan := map[string][]string{
		"子": {"癸"}, "丑": {"己", "辛", "癸"}, "寅": {"甲", "丙", "戊"},
		"卯": {"乙"}, "辰": {"戊", "乙", "癸"}, "巳": {"丙", "戊", "庚"},
		"午": {"丁", "己"}, "未": {"己", "丁", "乙"}, "申": {"庚", "壬", "戊"},
		"酉": {"辛"}, "戌": {"戊", "辛", "丁"}, "亥": {"壬", "甲"},
	}
	// 五行相生表：value 生 key（水 生 木、木 生 火等），用于查找日主的印星（生日主的五行）
	generates := map[string]string{"木": "水", "火": "木", "土": "火", "金": "土", "水": "金"}

	dayWx := stemWx[dayGan] // 如"土"

	// 判断月令旺衰：根据月支所在的季节，确定日主五行在当月的旺相休囚状态
	// 春木旺、夏火旺、秋金旺、冬水旺、四季末土旺
	monthSeasons := map[string]string{
		"寅": "春", "卯": "春", "辰": "春",
		"巳": "夏", "午": "夏", "未": "夏",
		"申": "秋", "酉": "秋", "戌": "秋",
		"亥": "冬", "子": "冬", "丑": "冬",
	}
	season := monthSeasons[monthZhi]

	// 五行四时旺相表：旺为当令最旺，相为次旺（被当令五行所生），休囚死为失令
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

	// 日主强弱判定：综合月令得分+通根数+印星生扶数得出总分，
	// >=7 身旺极，>=5 身强，>=3 中和，<3 身弱，<=1 身弱极
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

	// 定用神：身强喜克泄耗（官杀、食伤、财星），身弱喜生扶（印星、比劫）
	// 用神就是对日主最为有利的五行，忌神是克制日主的五行
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

	// 调候判断：冬生需火暖局（寒木向阳），夏生需水润局（调候为急）
	// 调候是八字命理中优先级别最高的取用原则
	tiaoHouNeed := ""
	if season == "冬" {
		tiaoHouNeed = "需火调候暖局"
	} else if season == "夏" {
		tiaoHouNeed = "需水调候润局"
	}

	// 用神去重：各分类计算可能重叠，去重后保留唯一集合
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

	// 忌神计算：除用神和日主自身五行外的其他五行，都是忌神
	yongSet := map[string]bool{}
	for _, y := range yongShen {
		yongSet[y] = true
	}
	for _, e := range wx {
		if !yongSet[e] && e != dayWx {
			jiShen = append(jiShen, e)
		}
	}

	// 喜神计算：生扶用神的五行即为喜神，如用神为木则喜水生木
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
