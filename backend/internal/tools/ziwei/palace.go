// This file belongs to the Zi Wei deterministic calculation layer.
// It owns Zi Wei palace calculation for this package.
// It computes reproducible Zi Wei facts; it must not compose user-facing readings.
package ziwei

import "github.com/6tail/lunar-go/calendar"

// SoulAndBody 命宫身宫计算结果的容器。SoulIndex/BodyIndex 为命宫身在十二宫中的索引（0-11），
// HeavenlyStemOfSoul/EarthlyBranchOfSoul 为命宫对应的干支。
type SoulAndBody struct {
	SoulIndex           int
	BodyIndex           int
	HeavenlyStemOfSoul  string
	EarthlyBranchOfSoul string
}

// GetSoulAndBody 安命宫、身宫。紫微斗数排盘的第一步，确定命宫和身宫所在宫位。
//
// 命宫算法：从寅宫起正月，顺数到出生月，然后从该宫位逆数到出生时辰，所得宫位为命宫。
// 身宫算法：从寅宫起正月，顺数到出生月，然后从该宫位顺数到出生时辰，所得宫位为身宫。
// 命宫代表先天命运格局，身宫代表后天努力方向。
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
		SoulIndex:           soulIndex,
		BodyIndex:           bodyIndex,
		HeavenlyStemOfSoul:  HeavenStems[heavenlyStemOfSoulIdx],
		EarthlyBranchOfSoul: EarthBranches[earthlyBranchOfSoulIdx],
	}
}

// fixLunarMonthIndex 调整农历月份索引（以寅=0，即正月=0）。紫微斗数以寅宫为正月起点。
// 若 fixLeap=true 且为闰月且当日超过15日，则视为下一个月处理，这是紫微斗数处理闰月的常见规则。
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

// GetFiveElementsClass 定五行局。以命宫干支纳音定五行局类型，五行局决定起运年龄和安长生十二神起点。
//
// 算法：天干取数（甲乙=1、丙丁=2、戊己=3、庚辛=4、壬癸=5），
// 地支取数（子丑午未=1、寅申卯酉=2、辰戌巳亥=3），
// 干支数值相加，超过5则减5，最终差值对应五行局：1=木三局、2=金四局、3=水二局、4=火六局、5=土五局。
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

// GetPalaceNames 获取命盘中旋转后的十二宫名。根据命宫所在索引将标准十二宫名映射到实际宫位。
// 如命宫在迁移宫位置（索引6），则迁移宫变为命宫，其余依次旋转。
func GetPalaceNames(soulIndex int) [12]string {
	var names [12]string
	for i := 0; i < 12; i++ {
		idx := FixIndex12(i - soulIndex)
		names[i] = PalaceNames[idx]
	}
	return names
}

// GetHoroscope 起大限（定大限起始年龄和走势）。大限是紫微斗数中十年一运的区间判断依据。
//
// 规则：阳男阴女顺行（从命宫起顺时针每步进一宫），阴男阳女逆行（逆时针每步退一宫）。
// 起运年龄由五行局数值决定：水二局2岁起运、木三局3岁、金四局4岁、土五局5岁、火六局6岁，每步大限增加10年。
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
