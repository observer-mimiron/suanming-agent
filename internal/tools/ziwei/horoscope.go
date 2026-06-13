package ziwei

import "github.com/6tail/lunar-go/calendar"

// GetChangSheng12 安长生十二神
//
// 五行局决定起位：水二局起申，木三局起亥，金四局起巳，土五局起申，火六局起寅
// 阳男阴女顺行，阴男阳女逆行
func GetChangSheng12(fiveElemNum int, gender string, yearBranch string) [12]string {
	var startIdx int
	switch fiveElemNum {
	case Water2:
		startIdx = FixEarthlyBranchIndex("申")
	case Wood3:
		startIdx = FixEarthlyBranchIndex("亥")
	case Metal4:
		startIdx = FixEarthlyBranchIndex("巳")
	case Earth5:
		startIdx = FixEarthlyBranchIndex("申")
	case Fire6:
		startIdx = FixEarthlyBranchIndex("寅")
	}

	branchYin := BranchYinYang[branchIndex(yearBranch)]
	isMale := gender == "男"
	forward := (isMale && branchYin == 0) || (!isMale && branchYin == 1)

	var result [12]string
	for i := 0; i < 12; i++ {
		var idx int
		if forward {
			idx = FixIndex12(startIdx + i)
		} else {
			idx = FixIndex12(startIdx - i)
		}
		result[idx] = ChangSheng12Names[i]
	}
	return result
}

// GetBoShi12 安博士十二神
//
// 禄存起博士，阳男阴女顺行，阴男阳女逆行
func GetBoShi12(solar *calendar.Solar, gender, yearStem, yearBranch string) [12]string {
	lu, _, _, _ := GetLuYangTuoMaIndex(yearStem, yearBranch)

	branchYin := BranchYinYang[branchIndex(yearBranch)]
	isMale := gender == "男"
	forward := (isMale && branchYin == 0) || (!isMale && branchYin == 1)

	var result [12]string
	for i := 0; i < 12; i++ {
		var idx int
		if forward {
			idx = FixIndex12(lu + i)
		} else {
			idx = FixIndex12(lu - i)
		}
		result[idx] = BoShi12Names[i]
	}
	return result
}
