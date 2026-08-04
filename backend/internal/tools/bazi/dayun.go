// This file belongs to the BaZi deterministic calculation layer.
// It owns DaYun calculation for this package.
// It computes reproducible BaZi facts; it must not generate narrative readings.
package bazi

import (
	"context"
	"strings"
)

// DayunAnalyzer 大运分析工具。它只计算大运的十神与已声明的干支关系。
// 大运吉凶必须由带版本的 rule profile 结合静态裁断给出，不能由本工具线性评分。
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
	dayGan, _ := baziResult["dayGan"].(string)

	// 命局四柱地支，用于大运-命局冲合应期分析（复用 liunian.go 的提取 helper）
	_, allZhi := extractGanZhiFromPillars(baziResult["pillars"])

	// shiShenTable 引用 tables.go 包级十神速查表（与 liunian.go 共享）

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

		// 大运地支——冲合刑害应期分析（大运为客，命局为主）
		dyZhi := ""
		if len(runes) >= 2 {
			dyZhi = string(runes[1])
		}
		branchHiddenStems := append([]string{}, branchHidegan[dyZhi]...)
		branchTenGods := computeSubShiShen(dayGan, branchHiddenStems)
		branchMainTenGod := ""
		if len(branchTenGods) > 0 {
			branchMainTenGod = branchTenGods[0]
		}
		chonghe := computeDayunChonghe(dyZhi, allZhi)
		annotated = append(annotated, map[string]any{
			"startAge": dy["startAge"], "endAge": dy["endAge"],
			// 日期边界属于大运事实。保留它们使动态层在流年缓存缺失时
			// 仍能按真实交运时刻定位当前大运，而不是退回到任意一条运。
			"startAt": dy["startAt"], "endAtExclusive": dy["endAtExclusive"],
			"ganZhi": gz, "tenGod": tenGod, "tenGodType": godType,
			"branch": dyZhi, "branchHiddenStems": branchHiddenStems,
			"branchTenGods": branchTenGods, "branchMainTenGod": branchMainTenGod,
			"quality":      "待profile裁断",
			"quality_base": "待profile裁断",
			"quality_reason": map[string]any{
				"summary": "仅报告大运十神与已声明的地支关系；吉凶待 rule profile 结合原局裁断。",
				"signals": map[string]any{
					"base_quality":         "待profile裁断",
					"base_layer":           "unclassified",
					"has_branch_conflict":  hasNegativeDayunRelation(chonghe),
					"has_branch_support":   hasPositiveDayunRelation(chonghe),
					"touches_core_pillars": touchesCorePillars(chonghe),
				},
			},
			"dayun_chonghe": chonghe,
		})
	}

	return map[string]any{"dayun_analyzed": annotated}, nil
}

func hasNegativeDayunRelation(chonghe []map[string]string) bool {
	for _, item := range chonghe {
		switch item["type"] {
		case "六冲", "相刑", "相害":
			return true
		}
	}
	return false
}

func hasPositiveDayunRelation(chonghe []map[string]string) bool {
	for _, item := range chonghe {
		switch item["type"] {
		case "三合", "三会":
			return true
		}
	}
	return false
}

func touchesCorePillars(chonghe []map[string]string) bool {
	for _, item := range chonghe {
		pillars := item["pillars"]
		if strings.Contains(pillars, "月柱") || strings.Contains(pillars, "日柱") {
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
