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

	// 命局四柱地支，用于大运-命局冲合应期分析（复用 liunian.go 的提取 helper）
	_, allZhi := extractGanZhiFromPillars(baziResult["pillars"])

	// shiShenTable 引用 tables.go 包级十神速查表（与 liunian.go 共享）

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

		// 大运地支——冲合刑害应期分析（大运为客，命局为主）
		dyZhi := ""
		if len(runes) >= 2 {
			dyZhi = string(runes[1])
		}
		chonghe := computeDayunChonghe(dyZhi, allZhi)

		annotated = append(annotated, map[string]any{
			"startAge": dy["startAge"], "endAge": dy["endAge"],
			"ganZhi": gz, "tenGod": tenGod, "tenGodType": godType,
			"quality":     quality,
			"dayun_chonghe": chonghe,
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

// computeDayunChonghe 检测大运地支与命局四支的冲合刑害关系。
// 大运为客、命局为主——只报告包含大运地支的关系，纯命局间关系由 computeChonghe 产出。
// 复用 chonghe.go 包级配对表（liuChongPairs/haiPairs/ziXingSet/sanHeCombos/sanHuiCombos/sanXingCombos）。
func computeDayunChonghe(dayunZhi string, allZhi []string) []map[string]string {
	items := []map[string]string{}
	if dayunZhi == "" {
		return items
	}
	pillarNames := []string{"年柱", "月柱", "日柱", "时柱"}

	// 六冲
	for _, pair := range liuChongPairs {
		a := string([]rune(pair)[0])
		b := string([]rune(pair)[1])
		if dayunZhi != a && dayunZhi != b {
			continue
		}
		otherZhi := b
		if dayunZhi == b {
			otherZhi = a
		}
		for i, z := range allZhi {
			if z == otherZhi {
				items = append(items, makeDayunChongheItem("六冲", pair, pillarNames[i]+otherZhi, "冲"))
			}
		}
	}

	// 相害
	for _, pair := range haiPairs {
		a := string([]rune(pair)[0])
		b := string([]rune(pair)[1])
		if dayunZhi != a && dayunZhi != b {
			continue
		}
		otherZhi := b
		if dayunZhi == b {
			otherZhi = a
		}
		for i, z := range allZhi {
			if z == otherZhi {
				items = append(items, makeDayunChongheItem("相害", pair, pillarNames[i]+otherZhi, "害"))
			}
		}
	}

	// 自刑：大运支与命局同地支（辰午酉亥）
	if ziXingSet[dayunZhi] {
		for i, z := range allZhi {
			if z == dayunZhi {
				items = append(items, makeDayunChongheItem("相刑", dayunZhi+dayunZhi, pillarNames[i]+dayunZhi, "自刑"))
			}
		}
	}

	// 三合/三会/三刑：大运支参与的组合，需命局凑齐其余 2 个地支
	for combo, elem := range sanHeCombos {
		if !comboContainsRune(combo, dayunZhi) {
			continue
		}
		if pillars, ok := findTriplePillarsWithTarget(combo, dayunZhi, "大运", allZhi, pillarNames); ok {
			items = append(items, map[string]string{
				"type":        "三合",
				"zhi":         combo,
				"pillars":     pillars,
				"description": "大运参与" + combo + "合" + elem + "局",
			})
		}
	}
	for combo, elem := range sanHuiCombos {
		if !comboContainsRune(combo, dayunZhi) {
			continue
		}
		if pillars, ok := findTriplePillarsWithTarget(combo, dayunZhi, "大运", allZhi, pillarNames); ok {
			items = append(items, map[string]string{
				"type":        "三会",
				"zhi":         combo,
				"pillars":     pillars,
				"description": "大运参与" + combo + "会" + elem + "局",
			})
		}
	}
	for _, combo := range sanXingCombos {
		if !comboContainsRune(combo, dayunZhi) {
			continue
		}
		if pillars, ok := findTriplePillarsWithTarget(combo, dayunZhi, "大运", allZhi, pillarNames); ok {
			items = append(items, map[string]string{
				"type":        "相刑",
				"zhi":         combo,
				"pillars":     pillars,
				"description": "大运参与" + combo + "三刑",
			})
		}
	}

	return items
}

// makeDayunChongheItem 构造大运冲合条目。description 格式："大运Xsuffix otherPillar"。
func makeDayunChongheItem(typ, zhi, otherPillar, suffix string) map[string]string {
	return map[string]string{
		"type":        typ,
		"zhi":         zhi,
		"pillars":     "大运" + otherPillar,
		"description": "大运" + zhi + suffix + otherPillar,
	}
}

// findTriplePillarsWithTarget 检查 combo 三地支是否在 allZhi 中齐备。
// pillars 按 combo 顺序拼接，targetZhi 标记为 targetLabel（如"大运"）。
func findTriplePillarsWithTarget(combo, targetZhi, targetLabel string, allZhi, pillarNames []string) (string, bool) {
	pillars := ""
	for _, r := range []rune(combo) {
		z := string(r)
		if z == targetZhi {
			pillars += targetLabel
			continue
		}
		idx := -1
		for i, ez := range allZhi {
			if ez == z {
				idx = i
				break
			}
		}
		if idx < 0 {
			return "", false
		}
		pillars += pillarNames[idx]
	}
	return pillars, true
}
