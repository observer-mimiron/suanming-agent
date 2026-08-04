// This file belongs to the BaZi deterministic calculation layer.
// It owns annual fortune calculation for this package.
// It computes reproducible BaZi facts; it must not generate narrative readings.
package bazi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/6tail/lunar-go/calendar"
)

// liunian.go 八字流年应期工具。
// 算法产出流年干支、流年十神、当前大运、流年与命局/大运的冲合刑害等证据字段。
// 不下"流年吉凶"结论——由 LLM 在 interpret.md:140 流年应期框架下综合判断。
//
// 复用：lunar-go 取流年干支（参考 ziwei/liunian.go:32 模式）、
// tables.go 的 shiShenTable 查流年十神、chonghe.go 的配对表查冲合关系。

// BaziLiuNianTool 八字流年分析工具。
type BaziLiuNianTool struct{}

func (t *BaziLiuNianTool) Name() string { return "bazi_liunian" }
func (t *BaziLiuNianTool) Description() string {
	return "分析八字流年干支、十神、冲合、当前大运"
}
func (t *BaziLiuNianTool) Label() string { return "流年分析" }

// Execute 接收 target_year + bazi_result，返回流年证据字段。
// bazi_result 需含 dayGan/pillars/dayun/birthday 字段（由 bazi_calc 产出）。
func (t *BaziLiuNianTool) Execute(_ context.Context, params map[string]any) (any, error) {
	targetYear, _ := params["target_year"].(float64)
	if targetYear == 0 {
		return nil, fmt.Errorf("target_year is required")
	}
	baziResult, _ := params["bazi_result"].(map[string]any)
	if baziResult == nil {
		return nil, fmt.Errorf("bazi_result is required")
	}

	dayGan, _ := baziResult["dayGan"].(string)
	allGan, allZhi := extractGanZhiFromPillars(baziResult["pillars"])
	dayunList := extractDayunList(baziResult["dayun"])
	birthYear := extractBirthYear(baziResult["birthday"])
	targetAt := time.Date(
		int(targetYear),
		time.Month(monthOrDefault(params["target_month"], 6)),
		monthOrDefault(params["target_day"], 15),
		monthOrDefault(params["target_hour"], 12),
		monthOrDefault(params["target_minute"], 0),
		0,
		0,
		time.UTC,
	)

	return computeLiuNianAt(dayGan, allGan, allZhi, dayunList, targetAt, birthYear), nil
}

// computeLiuNian 计算流年证据字段。
func computeLiuNian(dayGan string, allGan, allZhi []string, dayunList []map[string]any, targetYear, birthYear int) map[string]any {
	return computeLiuNianAt(dayGan, allGan, allZhi, dayunList, time.Date(targetYear, 6, 15, 12, 0, 0, 0, time.UTC), birthYear)
}

// computeLiuNianAt selects the current luck period using exact chart boundaries
// when present. The virtual-age fallback keeps historical chart payloads valid.
func computeLiuNianAt(dayGan string, allGan, allZhi []string, dayunList []map[string]any, targetAt time.Time, birthYear int) map[string]any {
	targetYear := targetAt.Year()
	// 1. 流年干支：年中某日取年干支（参考 ziwei/liunian.go:32）
	solar := calendar.NewSolar(targetYear, 6, 15, 12, 0, 0)
	lunar := solar.GetLunar()
	yearStem := lunar.GetYearGan()
	yearBranch := lunar.GetYearZhi()

	// 2. 流年十神：查 shiShenTable（tables.go）
	shiShen := ""
	if dayGan != "" && yearStem != "" {
		if row, ok := shiShenTable[dayGan]; ok {
			shiShen = row[yearStem]
		}
	}

	// 3. 当前大运：优先按排盘时写入的起止日期选择，旧资产才回退虚岁。
	currentAge := targetYear - birthYear + 1
	currentDayun, dayunSelection := findCurrentDayunAt(dayunList, targetAt, currentAge)

	// 4. 流年冲合：流年地支 vs 命局四支 + 当前大运支
	dayunZhi := ""
	if gz, ok := currentDayun["ganZhi"].(string); ok && len([]rune(gz)) >= 2 {
		dayunZhi = string([]rune(gz)[1])
	}
	chonghe := computeLiunianChonghe(yearBranch, allZhi, dayunZhi)

	return map[string]any{
		"liunian_year":            targetYear,
		"liunian_ganzhi":          yearStem + yearBranch,
		"liunian_stem":            yearStem,
		"liunian_branch":          yearBranch,
		"liunian_shi_shen":        shiShen,
		"liunian_target_at":       targetAt.Format("2006-01-02 15:04:05"),
		"current_dayun":           currentDayun,
		"current_dayun_selection": dayunSelection,
		"liunian_chonghe":         chonghe,
	}
}

func monthOrDefault(value any, fallback int) int {
	parsed := toInt(value)
	if parsed == 0 {
		return fallback
	}
	return parsed
}

// findCurrentDayun 遍历 dayun 列表，返回 startAge<=currentAge<=endAge 的元素。
// 未匹配返回空 map（如 age 未到起运岁数）。
// 兼容 int/float64/int64 等数值类型（lunar-go 产出 int，JSON 反序列化后变 float64）。
func findCurrentDayun(dayunList []map[string]any, currentAge int) map[string]any {
	for _, d := range dayunList {
		startAge := toInt(d["startAge"])
		endAge := toInt(d["endAge"])
		if startAge <= currentAge && currentAge <= endAge {
			return d
		}
	}
	return map[string]any{}
}

func findCurrentDayunAt(dayunList []map[string]any, targetAt time.Time, currentAge int) (map[string]any, string) {
	for _, dayun := range dayunList {
		startAt, startOK := parseDayunBoundary(dayun["startAt"])
		endAt, endOK := parseDayunBoundary(dayun["endAtExclusive"])
		if !startOK || !endOK {
			continue
		}
		if !targetAt.Before(startAt) && targetAt.Before(endAt) {
			return dayun, "date_boundary"
		}
	}
	return findCurrentDayun(dayunList, currentAge), "virtual_age_fallback"
}

func parseDayunBoundary(value any) (time.Time, bool) {
	text, ok := value.(string)
	if !ok || text == "" {
		return time.Time{}, false
	}
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", text, time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

// toInt 将 any 类型的数值转为 int，兼容 int/int64/float64。
func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

// computeLiunianChonghe 检测流年地支与命局四支+大运支的冲合刑害关系。
// 只报告包含流年地支的关系——纯命局间关系由 computeChonghe 已产出。
// 复用 chonghe.go 包级配对表（liuChongPairs/haiPairs/ziXingSet/sanHeCombos/sanHuiCombos/sanXingCombos）。
func computeLiunianChonghe(liunianZhi string, allZhi []string, dayunZhi string) []map[string]string {
	items := []map[string]string{}
	pillarNames := []string{"年柱", "月柱", "日柱", "时柱"}

	// 六冲：流年支与命局/大运支的六冲
	for _, pair := range liuChongPairs {
		a := string([]rune(pair)[0])
		b := string([]rune(pair)[1])
		if liunianZhi != a && liunianZhi != b {
			continue
		}
		otherZhi := b
		if liunianZhi == b {
			otherZhi = a
		}
		for i, z := range allZhi {
			if z == otherZhi {
				items = append(items, makeLiunianChongheItem("六冲", pair, pillarNames[i]+otherZhi, "冲"))
			}
		}
		if dayunZhi != "" && dayunZhi == otherZhi {
			items = append(items, makeLiunianChongheItem("六冲", pair, "大运"+otherZhi, "冲"))
		}
	}

	// 相害
	for _, pair := range haiPairs {
		a := string([]rune(pair)[0])
		b := string([]rune(pair)[1])
		if liunianZhi != a && liunianZhi != b {
			continue
		}
		otherZhi := b
		if liunianZhi == b {
			otherZhi = a
		}
		for i, z := range allZhi {
			if z == otherZhi {
				items = append(items, makeLiunianChongheItem("相害", pair, pillarNames[i]+otherZhi, "害"))
			}
		}
		if dayunZhi != "" && dayunZhi == otherZhi {
			items = append(items, makeLiunianChongheItem("相害", pair, "大运"+otherZhi, "害"))
		}
	}

	// 自刑：流年支与命局/大运同地支（辰午酉亥）
	if ziXingSet[liunianZhi] {
		for i, z := range allZhi {
			if z == liunianZhi {
				items = append(items, makeLiunianChongheItem("相刑", liunianZhi+liunianZhi, pillarNames[i]+liunianZhi, "自刑"))
			}
		}
		if dayunZhi != "" && dayunZhi == liunianZhi {
			items = append(items, makeLiunianChongheItem("相刑", liunianZhi+liunianZhi, "大运"+liunianZhi, "自刑"))
		}
	}

	// 三合/三会/三刑：流年支参与的组合，需命局+大运凑齐 3 个地支
	// 构造扩展地支列表（命局+大运），但不重复流年地支
	extZhi := append([]string{}, allZhi...)
	if dayunZhi != "" && dayunZhi != liunianZhi {
		extZhi = append(extZhi, dayunZhi)
	}
	extPillarNames := append([]string{}, pillarNames...)
	if dayunZhi != "" && dayunZhi != liunianZhi {
		extPillarNames = append(extPillarNames, "大运")
	}

	for combo, elem := range sanHeCombos {
		if !comboContainsRune(combo, liunianZhi) {
			continue
		}
		if pillars, ok := findTriplePillarsWithFlowYear(combo, liunianZhi, extZhi, extPillarNames); ok {
			items = append(items, map[string]string{
				"type":        "三合",
				"zhi":         combo,
				"pillars":     pillars,
				"description": "流年参与" + combo + "合" + elem + "局",
			})
		}
	}
	for combo, elem := range sanHuiCombos {
		if !comboContainsRune(combo, liunianZhi) {
			continue
		}
		if pillars, ok := findTriplePillarsWithFlowYear(combo, liunianZhi, extZhi, extPillarNames); ok {
			items = append(items, map[string]string{
				"type":        "三会",
				"zhi":         combo,
				"pillars":     pillars,
				"description": "流年参与" + combo + "会" + elem + "局",
			})
		}
	}
	for _, combo := range sanXingCombos {
		if !comboContainsRune(combo, liunianZhi) {
			continue
		}
		if pillars, ok := findTriplePillarsWithFlowYear(combo, liunianZhi, extZhi, extPillarNames); ok {
			items = append(items, map[string]string{
				"type":        "相刑",
				"zhi":         combo,
				"pillars":     pillars,
				"description": "流年参与" + combo + "三刑",
			})
		}
	}

	return items
}

// makeLiunianChongheItem 构造流年冲合条目。description 格式："流年Xsuffix otherPillar"。
// "流年"始终在前——区别于 makeChongheItem（按柱位索引排序，会越界）。
func makeLiunianChongheItem(typ, zhi, otherPillar, suffix string) map[string]string {
	return map[string]string{
		"type":        typ,
		"zhi":         zhi,
		"pillars":     "流年" + otherPillar,
		"description": "流年" + zhi + suffix + otherPillar,
	}
}

// comboContainsRune 检查 combo 字符串是否包含目标地支字符。
func comboContainsRune(combo, target string) bool {
	for _, r := range []rune(combo) {
		if string(r) == target {
			return true
		}
	}
	return false
}

// findTriplePillarsWithFlowYear 检查 combo 三地支是否在 extZhi（命局+大运）中齐备。
// pillars 按 combo 顺序拼接，流年地支标记为"流年"。
func findTriplePillarsWithFlowYear(combo, liunianZhi string, extZhi []string, extPillarNames []string) (string, bool) {
	pillars := ""
	for _, r := range []rune(combo) {
		z := string(r)
		if z == liunianZhi {
			pillars += "流年"
			continue
		}
		idx := -1
		for i, ez := range extZhi {
			if ez == z {
				idx = i
				break
			}
		}
		if idx < 0 {
			return "", false
		}
		pillars += extPillarNames[idx]
	}
	return pillars, true
}

// extractGanZhiFromPillars 从 bazi_calc 产出的 pillars 字段提取天干/地支切片。
func extractGanZhiFromPillars(pillars any) ([]string, []string) {
	p, ok := pillars.([]map[string]any)
	if !ok {
		return nil, nil
	}
	allGan, allZhi := make([]string, 0, len(p)), make([]string, 0, len(p))
	for _, pillar := range p {
		g, _ := pillar["stem"].(string)
		z, _ := pillar["branch"].(string)
		allGan = append(allGan, g)
		allZhi = append(allZhi, z)
	}
	return allGan, allZhi
}

// extractDayunList 从 bazi_calc 产出的 dayun 字段提取大运列表。
// 兼容 []map[string]any 和 []interface{}（JSON 反序列化后的类型）。
func extractDayunList(dayun any) []map[string]any {
	if d, ok := dayun.([]map[string]any); ok {
		return d
	}
	if di, ok := dayun.([]interface{}); ok {
		out := make([]map[string]any, 0, len(di))
		for _, item := range di {
			if dm, ok := item.(map[string]interface{}); ok {
				dm2 := make(map[string]any, len(dm))
				for k, v := range dm {
					dm2[k] = v
				}
				out = append(out, dm2)
			}
		}
		return out
	}
	return nil
}

// extractBirthYear 从 birthday 字符串（如"1974-04-28 16:00"）提取出生年。
func extractBirthYear(birthday any) int {
	s, _ := birthday.(string)
	if s == "" || len(s) < 4 {
		return 0
	}
	var y int
	_, err := fmt.Sscanf(s[:4], "%d", &y)
	if err != nil {
		return 0
	}
	return y
}

// toStringSlice 兼容 []string 和 []interface{} 的字符串切片转换（dayun.go 也用，保留为 helper）。
var _ = strings.Join // 保留 strings 引用，后续如有字符串拼接需求可用
