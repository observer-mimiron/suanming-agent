package bazi

// shishen.go 十神相关计算：天干阴阳、十神关系、十神力量统计。
// 共享 collectTenGods 收集逻辑，供 detectGejuCombination（geju_combination.go）
// 和 computeShiShenPower 复用。

// isYangStem 判断天干是否为阳干。甲丙戊庚壬为阳，乙丁己辛癸为阴。
func isYangStem(stem string) bool {
	switch stem {
	case "甲", "丙", "戊", "庚", "壬":
		return true
	default:
		return false
	}
}

// tenGodOf 计算某天干对日主的十神关系。
// 同五行→比肩/劫财；生我→正印/偏印；我生→食神/伤官；我克→正财/偏财；克我→正官/七杀。
// 阴阳同→偏，阴阳异→正。
// generates 为印星方向（"木":"水" 意为水生木），ke 引用 tables.go 包级表。
func tenGodOf(stem, dayGan, stemW, dayW string, generates map[string]string) string {
	if stemW == dayW {
		if isYangStem(dayGan) == isYangStem(stem) {
			return "比肩"
		}
		return "劫财"
	}
	if generates[dayW] == stemW {
		if isYangStem(dayGan) == isYangStem(stem) {
			return "偏印"
		}
		return "正印"
	}
	if generates[stemW] == dayW {
		if isYangStem(dayGan) == isYangStem(stem) {
			return "食神"
		}
		return "伤官"
	}
	if ke[dayW] == stemW {
		if isYangStem(dayGan) == isYangStem(stem) {
			return "偏财"
		}
		return "正财"
	}
	if ke[stemW] == dayW {
		if isYangStem(dayGan) == isYangStem(stem) {
			return "七杀"
		}
		return "正官"
	}
	return "未知"
}

// tenGodInfo 命盘中某个十神的出现位置记录。
// detectGejuCombination 用此做距离/位置判断，computeShiShenPower 聚合为 gan/zhi count。
type tenGodInfo struct {
	god       string
	source    string // "年干"/"月干"/"时干" or "年支"/"月支"/"日支"/"时支"
	isGan     bool
	pillarIdx int // 0=年, 1=月, 2=日, 3=时
	branch    string
	hiddenGan string
	hiddenIdx int
}

func hiddenStemTierName(idx int) string {
	switch idx {
	case 0:
		return "本气"
	case 1:
		return "中气"
	case 2:
		return "余气"
	default:
		return "杂气"
	}
}

func (info tenGodInfo) displaySource() string {
	if info.isGan || info.branch == "" || info.hiddenGan == "" {
		return info.source
	}
	return info.source + info.branch + "中" + hiddenStemTierName(info.hiddenIdx) + info.hiddenGan
}

func (info tenGodInfo) sourceWeight() int {
	if info.isGan {
		return 100
	}
	switch info.hiddenIdx {
	case 0:
		return 60
	case 1:
		return 40
	case 2:
		return 20
	default:
		return 10
	}
}

func (info tenGodInfo) distanceToDayPillar() int {
	d := info.pillarIdx - 2
	if d < 0 {
		return -d
	}
	return d
}

// collectTenGods 遍历命盘所有十神（天干透出 + 地支藏干，排除日主），返回每个出现位置的记录。
// detectGejuCombination 和 computeShiShenPower 共用此收集逻辑，避免重复实现。
func collectTenGods(dayGan, dayWx string, allGan, allZhi []string,
	stemWx, generates map[string]string,
	branchHidegan map[string][]string) []tenGodInfo {
	posNames := []string{"年", "月", "日", "时"}
	out := []tenGodInfo{}
	for i, g := range allGan {
		if g == dayGan {
			continue
		}
		tg := tenGodOf(g, dayGan, stemWx[g], dayWx, generates)
		out = append(out, tenGodInfo{god: tg, source: posNames[i] + "干", isGan: true, pillarIdx: i})
	}
	for i, z := range allZhi {
		for j, hg := range branchHidegan[z] {
			if hg == dayGan {
				continue
			}
			tg := tenGodOf(hg, dayGan, stemWx[hg], dayWx, generates)
			out = append(out, tenGodInfo{
				god:       tg,
				source:    posNames[i] + "支",
				isGan:     false,
				pillarIdx: i,
				branch:    z,
				hiddenGan: hg,
				hiddenIdx: j,
			})
		}
	}
	return out
}

// computeShiShenPower 统计命盘中各十神的力量分布。
//   - gan_count：天干透出次数
//   - zhi_count：地支藏干次数
//   - total：gan_count + zhi_count
//   - weighted：天干透出权重 1.0 + 地支藏干权重 0.5（透出力量高于藏支）
//
// 收集逻辑通过 collectTenGods 共享。算法只产证据字段，不下"十神力量强/弱"结论，
// 由 LLM 结合 strength/yong_shen 综合判断。
func computeShiShenPower(dayGan, dayWx string, allGan, allZhi []string,
	stemWx, generates map[string]string,
	branchHidegan map[string][]string) map[string]map[string]float64 {
	type counts struct {
		gan, zhi int
	}
	agg := map[string]*counts{}
	for _, info := range collectTenGods(dayGan, dayWx, allGan, allZhi, stemWx, generates, branchHidegan) {
		if agg[info.god] == nil {
			agg[info.god] = &counts{}
		}
		if info.isGan {
			agg[info.god].gan++
		} else {
			agg[info.god].zhi++
		}
	}
	out := make(map[string]map[string]float64, len(agg))
	for god, c := range agg {
		total := c.gan + c.zhi
		weighted := float64(c.gan)*1.0 + float64(c.zhi)*0.5
		out[god] = map[string]float64{
			"gan_count": float64(c.gan),
			"zhi_count": float64(c.zhi),
			"total":     float64(total),
			"weighted":  weighted,
		}
	}
	return out
}

// computeSubShiShen 将某柱藏干按日主换算为副星十神列表。
// 这里保留藏干原有顺序，便于前端与藏干逐项对照显示。
func computeSubShiShen(dayGan string, hideGan []string) []string {
	if len(hideGan) == 0 {
		return []string{}
	}
	row, ok := shiShenTable[dayGan]
	if !ok {
		return []string{}
	}

	out := make([]string, 0, len(hideGan))
	for _, stem := range hideGan {
		if god, ok := row[stem]; ok && god != "" {
			out = append(out, god)
		}
	}
	return out
}
