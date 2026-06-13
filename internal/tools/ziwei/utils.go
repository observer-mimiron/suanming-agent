package ziwei

// FixIndex 将索引锁定在 0~max-1 范围内，支持负数循环
func FixIndex(index int, max int) int {
	if index < 0 {
		return FixIndex(index+max, max)
	}
	if index > max-1 {
		return FixIndex(index-max, max)
	}
	return index
}

// FixIndex12 默认 max=12 的 FixIndex（宫位和地支都用12）
func FixIndex12(index int) int {
	return FixIndex(index, 12)
}

// FixIndex10 默认 max=10 的 FixIndex（天干用10）
func FixIndex10(index int) int {
	return FixIndex(index, 10)
}

// FixEarthlyBranchIndex 地支名 → 宫位索引（寅=0）
func FixEarthlyBranchIndex(branch string) int {
	idx := branchIndex(branch)
	yinIdx := branchIndex("寅")
	return FixIndex12(idx - yinIdx)
}

// branchIndex 地支名 → 0-11 索引
func branchIndex(branch string) int {
	for i, b := range EarthBranches {
		if b == branch {
			return i
		}
	}
	return -1
}

// stemIndex 天干名 → 0-9 索引
func stemIndex(stem string) int {
	for i, s := range HeavenStems {
		if s == stem {
			return i
		}
	}
	return -1
}

// TimeToIndex 小时 → 时辰索引（0=早子时, 1-11=丑-亥, 12=晚子时）
func TimeToIndex(hour int) int {
	if hour == 0 {
		return 0
	}
	if hour == 23 {
		return 12
	}
	return (hour + 1) / 2
}

// GetBrightness 获取星曜在指定宫位的亮度
func GetBrightness(starName string, palaceIndex int) string {
	brightness, ok := StarBrightness[starName]
	if !ok {
		return ""
	}
	idx := FixIndex12(palaceIndex)
	return brightness[idx]
}

// GetMutagen 判断星曜在生年天干下是否有四化
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

// GetAgeIndex 获取小限起始宫位索引（按年支）
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
