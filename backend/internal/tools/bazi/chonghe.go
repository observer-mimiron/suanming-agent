// This file belongs to the BaZi deterministic calculation layer.
// It owns BaZi relation calculation for this package.
// It computes reproducible BaZi facts; it must not generate narrative readings.
package bazi

// chonghe.go 四柱地支之间的冲合刑害关系检测。
// 配对表为包级 var，liunian.go（流年冲合）复用同一套表。

// liuChongPairs 六冲对（命理学传统书写顺序，如"子午"而非"午子"）。
var liuChongPairs = []string{"子午", "丑未", "寅申", "卯酉", "辰戌", "巳亥"}

// sanHeCombos 三合局：申子辰水、寅午戌火、巳酉丑金、亥卯未木。
var sanHeCombos = map[string]string{
	"申子辰": "水", "寅午戌": "火", "巳酉丑": "金", "亥卯未": "木",
}

// sanHuiCombos 三会局：寅卯辰木、巳午未火、申酉戌金、亥子丑水。
var sanHuiCombos = map[string]string{
	"寅卯辰": "木", "巳午未": "火", "申酉戌": "金", "亥子丑": "水",
}

// sanXingCombos 三刑：寅巳申三刑 + 丑戌未三刑。
var sanXingCombos = []string{"寅巳申", "丑戌未"}

// ziXingSet 自刑地支集合：辰午酉亥 重复出现即自刑。
var ziXingSet = map[string]bool{"辰": true, "午": true, "酉": true, "亥": true}

// haiPairs 相害对（命理学传统书写顺序）。
var haiPairs = []string{"子未", "丑午", "寅巳", "卯辰", "申亥", "酉戌"}

// computeChonghe 计算四柱地支之间的冲合刑害关系。
//
// 检测五类关系：六冲、三合局、三会局、相刑（三刑+子卯+自刑）、相害。
// 同一对地支可能同时匹配多种关系（如寅+巳既是相刑又是相害），分别报告。
// 返回 []map[string]string，每个 item 含 type/zhi/pillars/description 四个 key。
// 无任何关系时返回空 slice（非 nil），便于上层 JSON 序列化。
//
// zhi 字段按命理学传统书写顺序（如"子午"而非"午子"），便于 LLM 识别。
// pillars 字段按柱位顺序（年/月/日/时 索引升序），便于定位涉及柱位。
func computeChonghe(allZhi []string) []map[string]string {
	pillarNames := []string{"年", "月", "日", "时"}
	items := []map[string]string{}

	// 六冲
	for _, pair := range liuChongPairs {
		a := string([]rune(pair)[0])
		b := string([]rune(pair)[1])
		aIdx, bIdx := branchIndex(a, allZhi), branchIndex(b, allZhi)
		if aIdx >= 0 && bIdx >= 0 {
			items = append(items, makeChongheItem("六冲", pair, aIdx, bIdx, pillarNames, "冲"))
		}
	}

	// 三合局
	for combo, elem := range sanHeCombos {
		if pillars, ok := findTriplePillarsInOrder(combo, allZhi, pillarNames); ok {
			items = append(items, map[string]string{
				"type":        "三合",
				"zhi":         combo,
				"pillars":     pillars,
				"description": pillars + combo + "合" + elem + "局",
			})
		}
	}

	// 三会局
	for combo, elem := range sanHuiCombos {
		if pillars, ok := findTriplePillarsInOrder(combo, allZhi, pillarNames); ok {
			items = append(items, map[string]string{
				"type":        "三会",
				"zhi":         combo,
				"pillars":     pillars,
				"description": pillars + combo + "会" + elem + "局",
			})
		}
	}

	// 相刑：三刑（寅巳申 / 丑戌未）+ 子卯相刑 + 自刑（辰午酉亥 重复≥2 次）
	for _, combo := range sanXingCombos {
		if pillars, ok := findTriplePillarsInOrder(combo, allZhi, pillarNames); ok {
			items = append(items, map[string]string{
				"type":        "相刑",
				"zhi":         combo,
				"pillars":     pillars,
				"description": pillars + combo + "三刑",
			})
		}
	}
	ziIdx, maoIdx := branchIndex("子", allZhi), branchIndex("卯", allZhi)
	if ziIdx >= 0 && maoIdx >= 0 {
		items = append(items, makeChongheItem("相刑", "子卯", ziIdx, maoIdx, pillarNames, "相刑"))
	}
	zhiIdxs := map[string][]int{}
	for i, z := range allZhi {
		zhiIdxs[z] = append(zhiIdxs[z], i)
	}
	for z, idxs := range zhiIdxs {
		if ziXingSet[z] && len(idxs) >= 2 {
			pillars := ""
			for _, i := range idxs {
				pillars += pillarNames[i]
			}
			items = append(items, map[string]string{
				"type":        "相刑",
				"zhi":         z + z,
				"pillars":     pillars,
				"description": pillars + z + z + "自刑",
			})
		}
	}

	// 相害
	for _, pair := range haiPairs {
		a := string([]rune(pair)[0])
		b := string([]rune(pair)[1])
		aIdx, bIdx := branchIndex(a, allZhi), branchIndex(b, allZhi)
		if aIdx >= 0 && bIdx >= 0 {
			items = append(items, makeChongheItem("相害", pair, aIdx, bIdx, pillarNames, "害"))
		}
	}

	return items
}

// branchIndex 返回目标地支在 allZhi 中首次出现的索引，未找到返回 -1。
func branchIndex(target string, allZhi []string) int {
	for i, z := range allZhi {
		if z == target {
			return i
		}
	}
	return -1
}

// makeChongheItem 构造二元关系的 chonghe 条目。pillars 按柱位索引升序拼接。
// 注意：本函数假设 aIdx/bIdx 都在 pillarNames 长度范围内（4 柱），
// 不适用于"流年"或"大运"作为参与方的情况——liunian.go 用 makeLiunianChongheItem。
func makeChongheItem(typ string, zhi string, aIdx, bIdx int, pillarNames []string, suffix string) map[string]string {
	p1, p2 := pillarNames[aIdx], pillarNames[bIdx]
	if aIdx > bIdx {
		p1, p2 = pillarNames[bIdx], pillarNames[aIdx]
	}
	return map[string]string{
		"type":        typ,
		"zhi":         zhi,
		"pillars":     p1 + p2,
		"description": p1 + p2 + zhi + suffix,
	}
}

// findTriplePillarsInOrder 检查 combo（如"申子辰"）三个地支是否全部出现在四柱中。
// pillars 按 combo 中地支的出现顺序拼接（非柱位顺序），便于描述时对齐地支顺序。
func findTriplePillarsInOrder(combo string, allZhi []string, pillarNames []string) (string, bool) {
	pillars := ""
	for _, r := range []rune(combo) {
		idx := branchIndex(string(r), allZhi)
		if idx < 0 {
			return "", false
		}
		pillars += pillarNames[idx]
	}
	return pillars, true
}
