package ziwei

import "github.com/6tail/lunar-go/calendar"

// GetChangSheng12 安长生十二神。长生十二神反映命盘中每个宫位的生命周期阶段。
//
// 起点规则：水二局起申宫、木三局起亥宫、金四局起巳宫、土五局起申宫、火六局起寅宫。
// 顺逆规则：阳男阴女顺行（顺时针），阴男阳女逆行（逆时针）。
// 十二神顺序：长生→沐浴→冠带→临官→帝旺→衰→病→死→墓→绝→胎→养。
// 帝旺所在宫位为该生年最强的宫位。
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

// GetBoShi12 安博士十二神。博士十二神用于辅助判断流年小运细节。
//
// 起点规则：从禄存所在宫位起博士。
// 顺逆规则：阳男阴女顺行（顺时针），阴男阳女逆行（逆时针）。
// 十二神顺序：博士→力士→青龙→小耗→将军→奏书→飞廉→喜神→病符→大耗→伏兵→官府。
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
