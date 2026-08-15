// 本文件属于紫微 domain 层。
// 本文件负责长生十二神和博士十二神的纯宫位排布规则。
// 不负责 lunar-go、命盘组装、工具参数、Session、模型、trace、SSE 或最终文本。
package domain

// DecadalInterval 表示一个十年大限的领域值，不携带 JSON 或传输标签。
type DecadalInterval struct {
	StartAge      int
	EndAge        int
	HeavenlyStem  string
	EarthlyBranch string
}

// GetChangSheng12 按五行局、性别和年支计算长生十二神在十二宫的分布。
func GetChangSheng12(fiveElemNum int, gender, yearBranch string) [12]string {
	var startIndex int
	switch fiveElemNum {
	case Water2:
		startIndex = FixEarthlyBranchIndex("申")
	case Wood3:
		startIndex = FixEarthlyBranchIndex("亥")
	case Metal4:
		startIndex = FixEarthlyBranchIndex("巳")
	case Earth5:
		startIndex = FixEarthlyBranchIndex("申")
	case Fire6:
		startIndex = FixEarthlyBranchIndex("寅")
	}

	branchYin := BranchYinYang[BranchIndex(yearBranch)]
	forward := (gender == "男" && branchYin == 0) || (gender != "男" && branchYin == 1)

	var result [12]string
	for i := 0; i < 12; i++ {
		index := startIndex + i
		if !forward {
			index = startIndex - i
		}
		result[FixIndex12(index)] = ChangSheng12Names[i]
	}
	return result
}

// GetBoShi12 按性别、年干和年支计算博士十二神在十二宫的分布。
func GetBoShi12(gender, yearStem, yearBranch string) [12]string {
	lu, _, _, _ := GetLuYangTuoMaIndex(yearStem, yearBranch)
	branchYin := BranchYinYang[BranchIndex(yearBranch)]
	forward := (gender == "男" && branchYin == 0) || (gender != "男" && branchYin == 1)

	var result [12]string
	for i := 0; i < 12; i++ {
		index := lu + i
		if !forward {
			index = lu - i
		}
		result[FixIndex12(index)] = BoShi12Names[i]
	}
	return result
}

// GetDecadalIntervals 根据命宫和生年信息计算十二宫大限区间。
func GetDecadalIntervals(gender string, soulIndex, fiveElemNum, yearStemIdx, yearBranchIdx int) [12]DecadalInterval {
	var intervals [12]DecadalInterval
	branchYin := BranchYinYang[yearBranchIdx]
	forward := (gender == "男" && branchYin == 0) || (gender != "男" && branchYin == 1)
	startStemIndex := TigerRule[yearStemIdx]

	for i := 0; i < 12; i++ {
		index := soulIndex + i
		if !forward {
			index = soulIndex - i
		}
		index = FixIndex12(index)
		startAge := fiveElemNum + 10*i
		intervals[index] = DecadalInterval{
			StartAge:      startAge,
			EndAge:        startAge + 9,
			HeavenlyStem:  HeavenStems[FixIndex10(startStemIndex+index)],
			EarthlyBranch: EarthBranches[FixIndex12(BranchIndex("寅")+index)],
		}
	}
	return intervals
}
