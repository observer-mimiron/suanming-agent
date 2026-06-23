package bazi

import "context"

// DayunAnalyzer 大运分析工具。根据日主和用神喜忌，对每步大运标注十神类型和吉凶评价。
type DayunAnalyzer struct{}

func (t *DayunAnalyzer) Name() string        { return "dayun_analyzer" }
func (t *DayunAnalyzer) Description() string { return "分析每个大运的吉凶和十神类型" }

func (t *DayunAnalyzer) Label() string { return "大运分析" }

func (t *DayunAnalyzer) Execute(_ context.Context, params map[string]any) (any, error) {
	dayun, _ := params["dayun"].([]map[string]any)

	// 兜底：dayun 经 JSON 序列化/反序列化后类型变为 []interface{}，而非 []map[string]any。
	if dayun == nil {
		if di, ok := params["dayun"].([]interface{}); ok {
			dayun = make([]map[string]any, 0, len(di))
			for _, item := range di {
				if dm, ok := item.(map[string]interface{}); ok {
					dm2 := make(map[string]any, len(dm))
					for k, v := range dm {
						dm2[k] = v
					}
					dayun = append(dayun, dm2)
				}
			}
		}
	}
	baziResult, _ := params["bazi_result"].(map[string]any)
	yongshen, _ := baziResult["yongshen"].(map[string]any)

	dayGan, _ := baziResult["dayGan"].(string)
	yongList := toStringSlice(yongshen["yong_shen"])
	jiList := toStringSlice(yongshen["ji_shen"])

	// 十神速查表：以日干和目标天干为键，返回十神名称
	shiShenTable := map[string]map[string]string{
		"甲": {"甲": "比肩", "乙": "劫财", "丙": "食神", "丁": "伤官", "戊": "偏财", "己": "正财", "庚": "七杀", "辛": "正官", "壬": "偏印", "癸": "正印"},
		"乙": {"甲": "劫财", "乙": "比肩", "丙": "伤官", "丁": "食神", "戊": "正财", "己": "偏财", "庚": "正官", "辛": "七杀", "壬": "正印", "癸": "偏印"},
		"丙": {"甲": "偏印", "乙": "正印", "丙": "比肩", "丁": "劫财", "戊": "食神", "己": "伤官", "庚": "偏财", "辛": "正财", "壬": "七杀", "癸": "正官"},
		"丁": {"甲": "正印", "乙": "偏印", "丙": "劫财", "丁": "比肩", "戊": "伤官", "己": "食神", "庚": "正财", "辛": "偏财", "壬": "正官", "癸": "七杀"},
		"戊": {"甲": "七杀", "乙": "正官", "丙": "偏印", "丁": "正印", "戊": "比肩", "己": "劫财", "庚": "食神", "辛": "伤官", "壬": "偏财", "癸": "正财"},
		"己": {"甲": "正官", "乙": "七杀", "丙": "正印", "丁": "偏印", "戊": "劫财", "己": "比肩", "庚": "伤官", "辛": "食神", "壬": "正财", "癸": "偏财"},
		"庚": {"甲": "偏财", "乙": "正财", "丙": "七杀", "丁": "正官", "戊": "偏印", "己": "正印", "庚": "比肩", "辛": "劫财", "壬": "食神", "癸": "伤官"},
		"辛": {"甲": "正财", "乙": "偏财", "丙": "正官", "丁": "七杀", "戊": "正印", "己": "偏印", "庚": "劫财", "辛": "比肩", "壬": "伤官", "癸": "食神"},
		"壬": {"甲": "食神", "乙": "伤官", "丙": "偏财", "丁": "正财", "戊": "七杀", "己": "正官", "庚": "偏印", "辛": "正印", "壬": "比肩", "癸": "劫财"},
		"癸": {"甲": "伤官", "乙": "食神", "丙": "正财", "丁": "偏财", "戊": "正官", "己": "七杀", "庚": "正印", "辛": "偏印", "壬": "劫财", "癸": "比肩"},
	}

	// 五行相生：key 生 value（木生火 等）
	generates := map[string]string{"木": "火", "火": "土", "土": "金", "金": "水", "水": "木"}

	// 将日主的五行用神映射到十神类别
	dayWx, _ := baziResult["day_master_wuxing"].(string)
	yongCategories := map[string]bool{}
	jiCategories := map[string]bool{}
	for _, y := range yongList {
		if y == dayWx {
			yongCategories["同我"] = true
		} else if generates[dayWx] == y { // dayWx generates y = 食伤,泄
			yongCategories["泄"] = true
		} else if generates[y] == dayWx { // y generates dayWx = 印星,生
			yongCategories["生"] = true
		} else if y != dayWx {
			yongCategories["耗"] = true // 财星
			yongCategories["克"] = true // 官杀
		}
	}
	for _, j := range jiList {
		if generates[dayWx] == j {
			jiCategories["泄"] = true
		} else if generates[j] == dayWx {
			jiCategories["生"] = true
		} else if j == dayWx {
			jiCategories["同我"] = true
		} else {
			jiCategories["耗"] = true
			jiCategories["克"] = true
		}
	}

	tenGodWuxing := map[string]string{"比肩": "同我", "劫财": "同我", "食神": "泄", "伤官": "泄", "偏财": "耗", "正财": "耗", "七杀": "克", "正官": "克", "偏印": "生", "正印": "生"}

	annotated := make([]map[string]any, 0, len(dayun))
	for _, dy := range dayun {
		// 跳过空条目
		if dy["startAge"] == nil || dy["endAge"] == nil {
			continue
		}
		gz, _ := dy["ganZhi"].(string)
		if gz == "" {
			continue
		}
		runes := []rune(gz)
		if len(runes) < 1 {
			continue
		}
		dyGan := string(runes[0])

		tenGod, ok := shiShenTable[dayGan][dyGan]
		if !ok {
			continue
		}
		godType := tenGodWuxing[tenGod]

		var quality string
		switch {
		case yongCategories[godType]:
			quality = "大吉"
		case jiCategories[godType]:
			quality = "凶"
		default:
			quality = "平"
		}

		annotated = append(annotated, map[string]any{
			"startAge": dy["startAge"], "endAge": dy["endAge"],
			"ganZhi": gz, "tenGod": tenGod, "tenGodType": godType,
			"quality": quality,
		})
	}

	return map[string]any{"dayun_analyzed": annotated}, nil
}

func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	arr, ok := v.([]string)
	if !ok {
		if arr2, ok2 := v.([]interface{}); ok2 {
			for _, x := range arr2 {
				if s, ok3 := x.(string); ok3 {
					arr = append(arr, s)
				}
			}
		}
	}
	return arr
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
