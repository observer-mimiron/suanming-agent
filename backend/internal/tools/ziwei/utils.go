// This file belongs to the Zi Wei deterministic calculation layer.
// It owns Zi Wei calculation helpers for this package.
// It computes reproducible Zi Wei facts; it must not compose user-facing readings.
package ziwei

// FixIndex 循环索引修正。将任意整数锁定在 0~(max-1) 范围内，支持正负多轮循环。
// 例如 max=12, index=15 返回 3；index=-1 返回 11。用于紫微斗数中宫位/地支索引的循环处理。
func FixIndex(index int, max int) int {
	if index < 0 {
		return FixIndex(index+max, max)
	}
	if index > max-1 {
		return FixIndex(index-max, max)
	}
	return index
}

// FixIndex12 基于12模的循环索引修正。紫微斗数中宫位和地支都使用12循环体系（十二地支、十二宫），
// 内部委托给 FixIndex(index, 12)。
func FixIndex12(index int) int {
	return FixIndex(index, 12)
}

// FixIndex10 基于10模的循环索引修正。天干使用10循环体系（十天干），
// 内部委托给 FixIndex(index, 10)。
func FixIndex10(index int) int {
	return FixIndex(index, 10)
}

// FixEarthlyBranchIndex 地支名转宫位索引。紫微斗数以寅宫为起点（索引0），将十二地支映射到0-11范围。
// 如"寅"返回0，"卯"返回1，...，"丑"返回11。
func FixEarthlyBranchIndex(branch string) int {
	idx := branchIndex(branch)
	yinIdx := branchIndex("寅")
	return FixIndex12(idx - yinIdx)
}

// branchIndex 地支名称转序数索引。子=0、丑=1、寅=2、...、亥=11。
// 这是地支的自然顺序（从子开始），与紫微斗数中从寅开始的索引不同。
func branchIndex(branch string) int {
	for i, b := range EarthBranches {
		if b == branch {
			return i
		}
	}
	return -1
}

// stemIndex 天干名称转序数索引。甲=0、乙=1、丙=2、...、癸=9。
func stemIndex(stem string) int {
	for i, s := range HeavenStems {
		if s == stem {
			return i
		}
	}
	return -1
}

// TimeToIndex 小时转时辰索引。紫微斗数将一天分为12+1个时辰：
// 0=早子时（00:00-00:59）、1=丑时、...、10=戌时、11=亥时、12=晚子时（23:00-23:59）。
// 晚子时的四柱处理与早子时不同（日柱算次日）。
func TimeToIndex(hour int) int {
	if hour == 0 {
		return 0
	}
	if hour == 23 {
		return 12
	}
	return (hour + 1) / 2
}

// GetBrightness 获取星曜在指定宫位的亮度等级。亮度反映星曜在该宫位的影响力：
// 庙（影响力最强）、旺、得、利、平、陷（影响力最弱）、不（不发光）。
// 吉星逢庙旺则吉上加吉，凶星逢庙旺则凶性显露。
func GetBrightness(starName string, palaceIndex int) string {
	brightness, ok := StarBrightness[starName]
	if !ok {
		return ""
	}
	idx := FixIndex12(palaceIndex)
	return brightness[idx]
}

// GetMutagen 判断星曜在生年天干下是否产生四化。四化是指定星曜因生年天干触发"化禄、化权、化科、化忌"四种变化，
// 是紫微斗数中判断吉凶应期的关键要素。如甲年廉贞化禄、破军化权、武曲化科、太阳化忌。
func GetMutagen(yearStemIndex int, starName string) string {
	mutagens := MutagenTable[yearStemIndex]
	for i, star := range mutagens {
		if star == starName {
			switch i {
			case 0:
				return "化禄"
			case 1:
				return "化权"
			case 2:
				return "化科"
			case 3:
				return "化忌"
			}
		}
	}
	return ""
}

// GetAgeIndex 获取小限起始宫位索引（按年支）。小限是每年一宫的流年运势判断方法：
// 寅午戌年生人从辰宫起、申子辰年生人从戌宫起、巳酉丑年生人从未宫起、亥卯未年生人从丑宫起。
// 男顺女逆逐岁推移。
func GetAgeIndex(yearBranch string) int {
	bi := branchIndex(yearBranch)
	switch {
	case bi == 2 || bi == 6 || bi == 10: // 寅午戌
		return FixEarthlyBranchIndex("辰")
	case bi == 8 || bi == 0 || bi == 4: // 申子辰
		return FixEarthlyBranchIndex("戌")
	case bi == 5 || bi == 9 || bi == 1: // 巳酉丑
		return FixEarthlyBranchIndex("未")
	case bi == 11 || bi == 3 || bi == 7: // 亥卯未
		return FixIndex12(FixEarthlyBranchIndex("丑"))
	}
	return -1
}
