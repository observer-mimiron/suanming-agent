package ziwei

import "github.com/6tail/lunar-go/calendar"

// SoulAndBody 命宫身宫计算结果
type SoulAndBody struct {
	SoulIndex         int
	BodyIndex         int
	HeavenlyStemOfSoul string
	EarthlyBranchOfSoul string
}

// GetSoulAndBody 安命宫、身宫
//
// 寅起正月，顺数至生月，逆数生时为命宫。
// 寅起正月，顺数至生月，顺数生时为身宫。
func GetSoulAndBody(solar *calendar.Solar, timeIndex int, fixLeap bool) SoulAndBody {
	lunar := solar.GetLunar()
	monthIndex := fixLunarMonthIndex(lunar, timeIndex, fixLeap)

	ec := lunar.GetEightChar()
	ec.SetSect(1)

	timeBranchIdx := branchIndex(ec.GetTimeZhi())

	// 命宫：顺时针数到生月，再逆时针数到生时
	soulIndex := FixIndex12(monthIndex - timeBranchIdx)
	// 身宫：顺时针数到生月，再顺时针数到生时
	bodyIndex := FixIndex12(monthIndex + timeBranchIdx)

	yearStemIdx := stemIndex(ec.GetYearGan())
	startHeavenlyStemIdx := TigerRule[yearStemIdx]
	heavenlyStemOfSoulIdx := FixIndex10(startHeavenlyStemIdx + soulIndex)
	earthlyBranchOfSoulIdx := FixIndex12(soulIndex + branchIndex("寅"))

	return SoulAndBody{
		SoulIndex:          soulIndex,
		BodyIndex:          bodyIndex,
		HeavenlyStemOfSoul:  HeavenStems[heavenlyStemOfSoulIdx],
		EarthlyBranchOfSoul: EarthBranches[earthlyBranchOfSoulIdx],
	}
}

// fixLunarMonthIndex 调整农历月份索引（以寅=0）
// 正月建寅。若 fixLeap=true 且闰月且 day>15，按下个月算。
func fixLunarMonthIndex(lunar *calendar.Lunar, timeIndex int, fixLeap bool) int {
	lunarMonth := lunar.GetMonth()
	isLeap := lunarMonth < 0
	if isLeap {
		lunarMonth = -lunarMonth
	}

	needAdd := isLeap && fixLeap && lunar.GetDay() > 15 && timeIndex != 12
	m := lunarMonth
	if needAdd {
		m++
	}
	return FixIndex12(m - 1)
}

// GetFiveElementsClass 定五行局（以命宫干支纳音而定）
//
// 天干取数：甲乙1 丙丁2 戊己3 庚辛4 壬癸5
// 地支取数：子丑午未1 寅申卯酉2 辰戌巳亥3
// 干支数相加，超过5减5，差1木 差2金 差3水 差4火 差5土
func GetFiveElementsClass(heavenlyStem, earthlyBranch string) (string, int) {
	si := stemIndex(heavenlyStem)
	bi := branchIndex(earthlyBranch)

	hsNum := si/2 + 1
	ebNum := (bi%6)/2 + 1
	sum := hsNum + ebNum
	for sum > 5 {
		sum -= 5
	}

	// [木三局, 金四局, 水二局, 火六局, 土五局]
	classNames := [5]string{"木三局", "金四局", "水二局", "火六局", "土五局"}
	classNums := [5]int{Wood3, Metal4, Water2, Fire6, Earth5}

	return classNames[sum-1], classNums[sum-1]
}

// GetPalaceNames 获取从寅宫开始的十二宫名（以命宫为始旋转）
func GetPalaceNames(soulIndex int) [12]string {
	var names [12]string
	for i := 0; i < 12; i++ {
		idx := FixIndex12(i - soulIndex)
		names[i] = PalaceNames[idx]
	}
	return names
}

// GetHoroscope 起大限
//
// 大限由命宫起，阳男阴女顺行；阴男阳女逆行，每十年过一宫限。
// 起运年龄由五行局定：水二局2岁起，木三局3岁起...
func GetHoroscope(solar *calendar.Solar, timeIndex int, gender string, fixLeap bool, soulIndex int, heavenlyStemOfSoul, earthlyBranchOfSoul string, yearStemIdx int, yearBranchIdx int) [12]DecadalInfo {
	var decadals [12]DecadalInfo

	_, fiveElemNum := GetFiveElementsClass(heavenlyStemOfSoul, earthlyBranchOfSoul)

	// 阳男阴女顺行，阴男阳女逆行
	branchYin := BranchYinYang[yearBranchIdx] // 0=阳 1=阴
	isMale := gender == "男"

	var forward bool
	if isMale {
		forward = branchYin == 0 // 阳男顺行
	} else {
		forward = branchYin == 1 // 阴女顺行
	}

	startStemIdx := TigerRule[yearStemIdx]

	for i := 0; i < 12; i++ {
		var idx int
		if forward {
			idx = FixIndex12(soulIndex + i)
		} else {
			idx = FixIndex12(soulIndex - i)
		}

		startAge := fiveElemNum + 10*i
		stemIndex := FixIndex10(startStemIdx + idx)
		branchIdx := FixIndex12(branchIndex("寅") + idx)

		decadals[idx] = DecadalInfo{
			StartAge:      startAge,
			EndAge:        startAge + 9,
			HeavenlyStem:  HeavenStems[stemIndex],
			EarthlyBranch: EarthBranches[branchIdx],
		}
	}

	return decadals
}
