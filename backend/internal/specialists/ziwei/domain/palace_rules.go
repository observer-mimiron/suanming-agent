// 本文件属于紫微 domain 层。
// 本文件负责命宫身宫、五行局、十二宫旋转和起紫微索引等不依赖历法库的纯规则。
// 不负责 lunar-go、命盘组装、工具参数、Session、模型、trace、SSE 或最终文本。
package domain

// SoulAndBody 表示命宫身宫及命宫干支的纯索引结果，不携带历法或传输对象。
type SoulAndBody struct {
	SoulIndex          int
	BodyIndex          int
	HeavenlyStemIndex  int
	EarthlyBranchIndex int
}

// GetSoulAndBody 根据已归一的农历月序、时支索引和年干索引计算命宫身宫。
// 月序以寅宫正月为0，时支和年干使用自然干支索引；闰月和晚子时归一必须由 adapter 完成。
func GetSoulAndBody(monthIndex, timeBranchIndex, yearStemIndex int) SoulAndBody {
	soulIndex := FixIndex12(monthIndex - timeBranchIndex)
	bodyIndex := FixIndex12(monthIndex + timeBranchIndex)

	return SoulAndBody{
		SoulIndex:          soulIndex,
		BodyIndex:          bodyIndex,
		HeavenlyStemIndex:  FixIndex10(TigerRule[yearStemIndex] + soulIndex),
		EarthlyBranchIndex: FixIndex12(soulIndex + BranchIndex("寅")),
	}
}

// GetFiveElementsClass 根据命宫干支确定五行局名称和起运数值。
// 输入必须是现有紫微干支名称；返回值保持既有命盘和大限计算合同。
func GetFiveElementsClass(heavenlyStem, earthlyBranch string) (string, int) {
	stemIndex := StemIndex(heavenlyStem)
	branchIndex := BranchIndex(earthlyBranch)

	heavenlyStemNumber := stemIndex/2 + 1
	earthlyBranchNumber := (branchIndex%6)/2 + 1
	sum := heavenlyStemNumber + earthlyBranchNumber
	for sum > 5 {
		sum -= 5
	}

	classNames := [5]string{"木三局", "金四局", "水二局", "火六局", "土五局"}
	classNumbers := [5]int{Wood3, Metal4, Water2, Fire6, Earth5}
	return classNames[sum-1], classNumbers[sum-1]
}

// GetPalaceNames 根据命宫索引旋转标准十二宫名，返回实际宫位顺序。
func GetPalaceNames(soulIndex int) [12]string {
	var names [12]string
	for i := 0; i < 12; i++ {
		idx := FixIndex12(i - soulIndex)
		names[i] = PalaceNames[idx]
	}
	return names
}

// GetZiweiStartIndex 根据农历日和五行局数计算紫微、天府所在宫位索引。
// 晚子时的日数修正由 adapter 在取得 lunar-go 日数后完成；本函数只处理纯索引规则。
func GetZiweiStartIndex(lunarDay, fiveElemNum int) (int, int) {
	// 循环找余数为0的商，保持既有“商数宫前走、余数取虎口”的算法。
	offset := 0
	quotient := 0
	for {
		divisor := lunarDay + offset
		quotient = divisor / fiveElemNum
		if divisor%fiveElemNum == 0 {
			break
		}
		offset++
	}

	quotient %= 12
	ziweiIndex := quotient - 1
	if offset%2 == 0 {
		ziweiIndex += offset
	} else {
		ziweiIndex -= offset
	}

	ziweiIndex = FixIndex12(ziweiIndex)
	return ziweiIndex, FixIndex12(12 - ziweiIndex)
}
