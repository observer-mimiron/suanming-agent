package ziwei

import (
	"fmt"

	"github.com/6tail/lunar-go/calendar"
)

// BuildChart 紫微斗数排盘主函数。整合安命宫身宫、定五行局、安紫微天府、安主星辅星、
// 长生博士十二神、起大限、安杂曜等全部步骤，组装成完整的十二宫 ZiWeiChart 命盘。
// 算法遵循传统紫微斗数规则（如五虎遁起寅宫天干、五鼠遁定时辰天干）。
func BuildChart(solar *calendar.Solar, timeIndex int, gender string) (*ZiWeiChart, error) {
	lunar := solar.GetLunar()
	ec := lunar.GetEightChar()
	ec.SetSect(1)

	yearStem := ec.GetYearGan()
	yearBranch := ec.GetYearZhi()
	monthStem := ec.GetMonthGan()
	monthBranch := ec.GetMonthZhi()
	dayStem := ec.GetDayGan()
	dayBranch := ec.GetDayZhi()
	timeStem := ec.GetTimeGan()
	timeBranch := ec.GetTimeZhi()

	yearStemIdx := stemIndex(yearStem)
	yearBranchIdx := branchIndex(yearBranch)

	// 人命宫、身宫
	sab := GetSoulAndBody(solar, timeIndex, true)

	// 五行局
	fiveElemName, fiveElemNum := GetFiveElementsClass(sab.HeavenlyStemOfSoul, sab.EarthlyBranchOfSoul)

	// 十二宫名
	palaceNames := GetPalaceNames(sab.SoulIndex)

	// 紫微、天府定位
	ziweiIdx, tianfuIdx := GetStartIndex(solar, timeIndex, sab.HeavenlyStemOfSoul, sab.EarthlyBranchOfSoul)

	// 主星
	majorStars := GetMajorStar(ziweiIdx, tianfuIdx, yearStemIdx)

	// 辅星
	lunarMonth := lunar.GetMonth()
	minorStars := GetMinorStar(yearStem, yearBranch, timeIndex, lunarMonth)

	// 杂曜
	yearlyIdx := GetYearlyStarIndex(solar, timeIndex, gender, yearStem, yearBranch, sab.SoulIndex, sab.BodyIndex)
	monthlyIdx := GetMonthlyStarIndex(lunar, timeIndex)
	dailyIdx := GetDailyStarIndex(lunar, timeIndex)
	timelyIdx := GetTimelyStarIndex(timeIndex)
	hongluan, tianxi := GetLuanXiIndex(yearBranch)
	adjStars := GetAdjectiveStar(yearlyIdx, monthlyIdx, dailyIdx, timelyIdx, hongluan, tianxi)

	// 大限
	decadals := GetHoroscope(solar, timeIndex, gender, false, sab.SoulIndex, sab.HeavenlyStemOfSoul, sab.EarthlyBranchOfSoul, yearStemIdx, yearBranchIdx)

	// 长生十二神
	changsheng := GetChangSheng12(fiveElemNum, gender, yearBranch)

	// 博士十二神
	boshi := GetBoShi12(solar, gender, yearStem, yearBranch)

	// 组装十二宫
	palaces := make([]ZiWeiPalace, 12)
	for i := 0; i < 12; i++ {
		// 宫干：从命宫干支起，顺时针推算
		hsIdx := FixIndex10(stemIndex(sab.HeavenlyStemOfSoul) - sab.SoulIndex + i)
		ebIdx := FixIndex12(2 + i) // 寅=2

		isBodyPalace := sab.BodyIndex == i
		isOriginalPalace := ebIdx != 0 && ebIdx != 1 && HeavenStems[hsIdx] == yearStem // 来因宫

		palaces[i] = ZiWeiPalace{
			Index:            i,
			Name:             palaceNames[i],
			HeavenlyStem:     HeavenStems[hsIdx],
			EarthlyBranch:    EarthBranches[ebIdx],
			IsBodyPalace:     isBodyPalace,
			IsOriginalPalace: isOriginalPalace,
			MajorStars:       majorStars[i],
			MinorStars:       minorStars[i],
			AdjectiveStars:   adjStars[i],
			ChangSheng12:     changsheng[i],
			BoShi12:          boshi[i],
			Decadal:          decadals[i],
		}
	}

	// 命主、身主
	soulPalaceBranch := EarthBranches[FixIndex12(sab.SoulIndex+2)]
	bodyPalaceBranch := EarthBranches[FixIndex12(sab.BodyIndex+2)]
	soulMaster := SoulMaster[branchIndex(soulPalaceBranch)]
	bodyMaster := BodyMaster[yearBranchIdx]

	solarStr := fmt.Sprintf("%d-%02d-%02d", solar.GetYear(), solar.GetMonth(), solar.GetDay())
	lunarStr := fmt.Sprintf("%d年%d月%d日", lunar.GetYear(), lunar.GetMonth(), lunar.GetDay())

	return &ZiWeiChart{
		Gender:               gender,
		SolarDate:            solarStr,
		LunarDate:            lunarStr,
		FourPillars:          map[string]string{
			"年柱": yearStem + yearBranch,
			"月柱": monthStem + monthBranch,
			"日柱": dayStem + dayBranch,
			"时柱": timeStem + timeBranch,
		},
		SoulPalaceBranch:     soulPalaceBranch,
		BodyPalaceBranch:     bodyPalaceBranch,
		SoulPalaceGanZhi:     sab.HeavenlyStemOfSoul + sab.EarthlyBranchOfSoul,
		FiveElementsClass:    fiveElemName,
		FiveElementsClassNum: fiveElemNum,
		SoulMaster:           soulMaster,
		BodyMaster:           bodyMaster,
		Palaces:              palaces,
	}, nil
}
