package ziwei

import (
	"github.com/6tail/lunar-go/calendar"
)

// GetStartIndex 起紫微星诀
//
// 局数除日数，商数宫前走；若见数无余，便要起虎口；日数小於局，还直宫中守。
// 返回紫微星和天府星的宫位索引（寅=0）
func GetStartIndex(solar *calendar.Solar, timeIndex int, heavenlyStemOfSoul, earthlyBranchOfSoul string) (int, int) {
	lunar := solar.GetLunar()
	lunarDay := lunar.GetDay()

	_, fiveElemNum := GetFiveElementsClass(heavenlyStemOfSoul, earthlyBranchOfSoul)

	day := lunarDay
	if timeIndex == 12 {
		day++ // 晚子时加一日
	}

	// 循环找余数为0的商
	offset := 0
	quotient := 0
	for {
		divisor := day + offset
		quotient = divisor / fiveElemNum
		remainder := divisor % fiveElemNum
		if remainder == 0 {
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
	tianfuIndex := FixIndex12(12 - ziweiIndex)

	return ziweiIndex, tianfuIndex
}

// GetLuYangTuoMaIndex 获取禄存、擎羊、陀罗、天马索引
func GetLuYangTuoMaIndex(yearStem, yearBranch string) (lu, yang, tuo, ma int) {
	si := stemIndex(yearStem)
	bi := branchIndex(yearBranch)

	// 禄存：按年干
	switch si {
	case 0:
		lu = FixEarthlyBranchIndex("寅")
	case 1:
		lu = FixEarthlyBranchIndex("卯")
	case 2, 4:
		lu = FixEarthlyBranchIndex("巳")
	case 3, 5:
		lu = FixEarthlyBranchIndex("午")
	case 6:
		lu = FixEarthlyBranchIndex("申")
	case 7:
		lu = FixEarthlyBranchIndex("酉")
	case 8:
		lu = FixEarthlyBranchIndex("亥")
	case 9:
		lu = FixEarthlyBranchIndex("子")
	}

	// 天马：按年支，只出现在四马地
	switch {
	case bi == 2 || bi == 6 || bi == 10: // 寅午戌 → 申
		ma = FixEarthlyBranchIndex("申")
	case bi == 8 || bi == 0 || bi == 4: // 申子辰 → 寅
		ma = FixEarthlyBranchIndex("寅")
	case bi == 5 || bi == 9 || bi == 1: // 巳酉丑 → 亥
		ma = FixEarthlyBranchIndex("亥")
	case bi == 11 || bi == 3 || bi == 7: // 亥卯未 → 巳
		ma = FixEarthlyBranchIndex("巳")
	}

	yang = FixIndex12(lu + 1)
	tuo = FixIndex12(lu - 1)

	return
}

// GetKuiYueIndex 获取天魁天钺索引（按年干）
func GetKuiYueIndex(yearStem string) (kui, yue int) {
	si := stemIndex(yearStem)
	switch si {
	case 0, 4, 6: // 甲戊庚 → 丑未
		kui = FixEarthlyBranchIndex("丑")
		yue = FixEarthlyBranchIndex("未")
	case 1, 5: // 乙己 → 子申
		kui = FixEarthlyBranchIndex("子")
		yue = FixEarthlyBranchIndex("申")
	case 7: // 辛 → 午寅
		kui = FixEarthlyBranchIndex("午")
		yue = FixEarthlyBranchIndex("寅")
	case 2, 3: // 丙丁 → 亥酉
		kui = FixEarthlyBranchIndex("亥")
		yue = FixEarthlyBranchIndex("酉")
	case 8, 9: // 壬癸 → 卯巳
		kui = FixEarthlyBranchIndex("卯")
		yue = FixEarthlyBranchIndex("巳")
	}
	return
}

// GetZuoYouIndex 获取左辅右弼索引（按生月）
func GetZuoYouIndex(lunarMonth int) (zuo, you int) {
	zuo = FixIndex12(FixEarthlyBranchIndex("辰") + (lunarMonth - 1))
	you = FixIndex12(FixEarthlyBranchIndex("戌") - (lunarMonth - 1))
	return
}

// GetChangQuIndex 获取文昌文曲索引（按时支）
func GetChangQuIndex(timeIndex int) (chang, qu int) {
	ti := FixIndex12(timeIndex)
	chang = FixIndex12(FixEarthlyBranchIndex("戌") - ti)
	qu = FixIndex12(FixEarthlyBranchIndex("辰") + ti)
	return
}

// GetKongJieIndex 获取地空地劫索引（按时支）
func GetKongJieIndex(timeIndex int) (kong, jie int) {
	ti := FixIndex12(timeIndex)
	haiIdx := FixEarthlyBranchIndex("亥")
	kong = FixIndex12(haiIdx - ti)
	jie = FixIndex12(haiIdx + ti)
	return
}

// GetHuoLingIndex 获取火星铃星索引（按年支+时支）
func GetHuoLingIndex(yearBranch string, timeIndex int) (huo, ling int) {
	ti := FixIndex12(timeIndex)
	bi := branchIndex(yearBranch)
	switch {
	case bi == 2 || bi == 6 || bi == 10: // 寅午戌
		huo = FixIndex12(FixEarthlyBranchIndex("丑") + ti)
		ling = FixIndex12(FixEarthlyBranchIndex("卯") + ti)
	case bi == 8 || bi == 0 || bi == 4: // 申子辰
		huo = FixIndex12(FixEarthlyBranchIndex("寅") + ti)
		ling = FixIndex12(FixEarthlyBranchIndex("戌") + ti)
	case bi == 5 || bi == 9 || bi == 1: // 巳酉丑
		huo = FixIndex12(FixEarthlyBranchIndex("卯") + ti)
		ling = FixIndex12(FixEarthlyBranchIndex("戌") + ti)
	case bi == 11 || bi == 3 || bi == 7: // 亥卯未
		huo = FixIndex12(FixEarthlyBranchIndex("酉") + ti)
		ling = FixIndex12(FixEarthlyBranchIndex("戌") + ti)
	}
	return
}

// GetLuanXiIndex 获取红鸾天喜索引（按年支）
func GetLuanXiIndex(yearBranch string) (hongluan, tianxi int) {
	bi := branchIndex(yearBranch)
	hongluan = FixIndex12(FixEarthlyBranchIndex("卯") - bi)
	tianxi = FixIndex12(hongluan + 6)
	return
}

// GetHuagaiXianchiIndex 获取华盖咸池索引（按年支）
func GetHuagaiXianchiIndex(yearBranch string) (huagai, xianchi int) {
	bi := branchIndex(yearBranch)
	switch {
	case bi == 2 || bi == 6 || bi == 10: // 寅午戌
		huagai = FixEarthlyBranchIndex("戌")
		xianchi = FixEarthlyBranchIndex("卯")
	case bi == 8 || bi == 0 || bi == 4: // 申子辰
		huagai = FixEarthlyBranchIndex("辰")
		xianchi = FixEarthlyBranchIndex("酉")
	case bi == 5 || bi == 9 || bi == 1: // 巳酉丑
		huagai = FixEarthlyBranchIndex("丑")
		xianchi = FixEarthlyBranchIndex("午")
	case bi == 11 || bi == 3 || bi == 7: // 亥卯未
		huagai = FixEarthlyBranchIndex("未")
		xianchi = FixEarthlyBranchIndex("子")
	}
	return
}

// GetGuGuaIndex 获取孤辰寡宿索引（按年支）
func GetGuGuaIndex(yearBranch string) (guchen, guasu int) {
	bi := branchIndex(yearBranch)
	switch {
	case bi == 2 || bi == 3 || bi == 4: // 寅卯辰
		guchen = FixEarthlyBranchIndex("巳")
		guasu = FixEarthlyBranchIndex("丑")
	case bi == 5 || bi == 6 || bi == 7: // 巳午未
		guchen = FixEarthlyBranchIndex("申")
		guasu = FixEarthlyBranchIndex("辰")
	case bi == 8 || bi == 9 || bi == 10: // 申酉戌
		guchen = FixEarthlyBranchIndex("亥")
		guasu = FixEarthlyBranchIndex("未")
	case bi == 11 || bi == 0 || bi == 1: // 亥子丑
		guchen = FixEarthlyBranchIndex("寅")
		guasu = FixEarthlyBranchIndex("戌")
	}
	return
}

// GetJieshaIndex 获取劫杀索引（按年支）
func GetJieshaIndex(yearBranch string) int {
	bi := branchIndex(yearBranch)
	switch {
	case bi == 8 || bi == 0 || bi == 4: // 申子辰 → 巳(3)
		return 3
	case bi == 11 || bi == 3 || bi == 7: // 亥卯未 → 申(6)
		return 6
	case bi == 2 || bi == 6 || bi == 10: // 寅午戌 → 亥(9)
		return 9
	case bi == 5 || bi == 9 || bi == 1: // 巳酉丑 → 寅(0)
		return 0
	}
	return -1
}

// GetDahaoIndex 获取大耗索引（按年支）
func GetDahaoIndex(yearBranch string) int {
	bi := branchIndex(yearBranch)
	// 大耗位于年支对宫，阳顺阴逆移一位
	matched := [12]string{"未", "午", "酉", "申", "亥", "戌", "丑", "子", "卯", "寅", "巳", "辰"}
	return FixIndex12(FixEarthlyBranchIndex(matched[bi]))
}

// GetNianjieIndex 获取年解索引（按年支）
func GetNianjieIndex(yearBranch string) int {
	bi := branchIndex(yearBranch)
	// 解神从戌上起子，逆数至当生年太岁上
	matched := [12]string{"戌", "酉", "申", "未", "午", "巳", "辰", "卯", "寅", "丑", "子", "亥"}
	return FixIndex12(FixEarthlyBranchIndex(matched[bi]))
}

// YearlyStarIndex 年系星索引
type YearlyStarIndex struct {
	Xianchi, Huagai, Guchen, Guasu             int
	Tiancai, Tianshou, Tianchu, Posui, Feilian int
	Longchi, Fengge, Tianku, Tianxu            int
	Tianguan, Tianfu, Tiande, Yuede, Tiankong  int
	Jielu, Kongwang, Xunkong, Jiekong          int
	Tianshi, Tianshang                         int
	Jiesha, Nianjie, Dahao                     int
}

// GetYearlyStarIndex 获取所有年系星索引
func GetYearlyStarIndex(solar *calendar.Solar, timeIndex int, gender, yearStem, yearBranch string, soulIndex, bodyIndex int) YearlyStarIndex {
	si := stemIndex(yearStem)
	bi := branchIndex(yearBranch)

	huagai, xianchi := GetHuagaiXianchiIndex(yearBranch)
	guchen, guasu := GetGuGuaIndex(yearBranch)

	// 天才: 命宫起子，顺行至生年支
	tiancai := FixIndex12(soulIndex + bi)
	// 天寿: 身宫起子，顺行至生年支
	tianshou := FixIndex12(bodyIndex + bi)

	// 天厨: 按年干
	tianchuStems := [10]string{"巳", "午", "子", "巳", "午", "申", "寅", "午", "酉", "亥"}
	tianchu := FixIndex12(FixEarthlyBranchIndex(tianchuStems[si]))

	// 破碎: 按年支三合
	posuiBranches := [3]string{"巳", "丑", "酉"}
	posui := FixIndex12(FixEarthlyBranchIndex(posuiBranches[bi%3]))

	// 蜚蠊
	feilianBranches := [12]string{"申", "酉", "戌", "巳", "午", "未", "寅", "卯", "辰", "亥", "子", "丑"}
	feilian := FixIndex12(FixEarthlyBranchIndex(feilianBranches[bi]))

	// 龙池: 辰起子，顺数至年支
	longchi := FixIndex12(FixEarthlyBranchIndex("辰") + bi)
	// 凤阁: 戌起子，逆数至年支
	fengge := FixIndex12(FixEarthlyBranchIndex("戌") - bi)

	// 天哭: 午起子，逆数至年支
	tianku := FixIndex12(FixEarthlyBranchIndex("午") - bi)
	// 天虚: 午起子，顺数至年支
	tianxu := FixIndex12(FixEarthlyBranchIndex("午") + bi)

	// 天官: 按年干
	tianguanBranches := [10]string{"未", "辰", "巳", "寅", "卯", "酉", "亥", "酉", "戌", "午"}
	tianguan := FixIndex12(FixEarthlyBranchIndex(tianguanBranches[si]))

	// 天福: 按年干
	tianfuBranches := [10]string{"酉", "申", "子", "亥", "卯", "寅", "午", "巳", "午", "巳"}
	tianfu := FixIndex12(FixEarthlyBranchIndex(tianfuBranches[si]))

	// 天德: 酉起子，顺数至年支
	tiande := FixIndex12(FixEarthlyBranchIndex("酉") + bi)
	// 月德: 巳起子，顺数至年支
	yuede := FixIndex12(FixEarthlyBranchIndex("巳") + bi)

	// 天空: 生年支前一位
	tiankong := FixIndex12(FixEarthlyBranchIndex(yearBranch) + 1)

	// 截路空亡: 按年干
	jieluBranches := [5]string{"申", "午", "辰", "寅", "子"}
	jielu := FixIndex12(FixEarthlyBranchIndex(jieluBranches[si%5]))
	// 空亡
	kongwangBranches := [5]string{"酉", "未", "巳", "卯", "丑"}
	kongwang := FixIndex12(FixEarthlyBranchIndex(kongwangBranches[si%5]))

	// 旬空
	xunkong := FixIndex12(FixEarthlyBranchIndex(yearBranch) + stemIndex("癸") - si + 1)
	if bi%2 != xunkong%2 {
		xunkong = FixIndex12(xunkong + 1)
	}

	// 截空（中州派用，取截路或空亡之一）
	jiekong := kongwang
	if bi%2 == 0 {
		jiekong = jielu
	}

	// 劫杀
	jiesha := GetJieshaIndex(yearBranch)

	// 年解
	nianjie := GetNianjieIndex(yearBranch)

	// 大耗
	dahao := GetDahaoIndex(yearBranch)

	// 天使天伤
	tianshi, tianshang := GetTianshiTianshangIndex(gender, yearBranch, soulIndex)

	return YearlyStarIndex{
		Xianchi: xianchi, Huagai: huagai, Guchen: guchen, Guasu: guasu,
		Tiancai: tiancai, Tianshou: tianshou, Tianchu: tianchu, Posui: posui, Feilian: feilian,
		Longchi: longchi, Fengge: fengge, Tianku: tianku, Tianxu: tianxu,
		Tianguan: tianguan, Tianfu: tianfu, Tiande: tiande, Yuede: yuede, Tiankong: tiankong,
		Jielu: jielu, Kongwang: kongwang, Xunkong: xunkong, Jiekong: jiekong,
		Tianshi: tianshi, Tianshang: tianshang,
		Jiesha: jiesha, Nianjie: nianjie, Dahao: dahao,
	}
}

// GetTianshiTianshangIndex 获取天使天伤索引
func GetTianshiTianshangIndex(gender, yearBranch string, soulIndex int) (tianshi, tianshang int) {
	bi := branchIndex(yearBranch)
	yinyang := bi % 2 // 0=阳 1=阴
	isMale := gender == "男"
	sameYinYang := yinyang == 0 && isMale || yinyang == 1 && !isMale

	// 天使居疾厄，天伤居交友
	tianshi = FixIndex12(6 + soulIndex) // 疾厄 = index 6 in PalaceNames
	tianshang = FixIndex12(5 + soulIndex) // 交友 = index 5

	if !sameYinYang {
		// 阴男阳女交换
		tianshi, tianshang = tianshang, tianshi
	}
	return
}

// MonthlyStarIndex 月系星索引
type MonthlyStarIndex struct {
	Yuejie, Tianyao, Tianxing, Yinsha, Tianyue, Tianwu int
}

// GetMonthlyStarIndex 获取月系星索引
func GetMonthlyStarIndex(lunar *calendar.Lunar, timeIndex int) MonthlyStarIndex {
	monthIndex := fixLunarMonthIndex(lunar, timeIndex, false) // 0-based

	// 月解
	yuejieBranches := [6]string{"申", "戌", "子", "寅", "辰", "午"}
	yuejie := FixIndex12(FixEarthlyBranchIndex(yuejieBranches[monthIndex/2]))

	// 天姚: 丑起正月顺数
	tianyao := FixIndex12(FixEarthlyBranchIndex("丑") + monthIndex)

	// 天刑: 酉起正月顺数
	tianxing := FixIndex12(FixEarthlyBranchIndex("酉") + monthIndex)

	// 阴煞
	yinshaBranches := [6]string{"寅", "子", "戌", "申", "午", "辰"}
	yinsha := FixIndex12(FixEarthlyBranchIndex(yinshaBranches[monthIndex%6]))

	// 天月
	tianyueBranches := [12]string{"戌", "巳", "辰", "寅", "未", "卯", "亥", "未", "寅", "午", "戌", "寅"}
	tianyue := FixIndex12(FixEarthlyBranchIndex(tianyueBranches[monthIndex]))

	// 天巫
	tianwuBranches := [4]string{"巳", "申", "寅", "亥"}
	tianwu := FixIndex12(FixEarthlyBranchIndex(tianwuBranches[monthIndex%4]))

	return MonthlyStarIndex{
		Yuejie:  yuejie,
		Tianyao: tianyao,
		Tianxing: tianxing,
		Yinsha:  yinsha,
		Tianyue: tianyue,
		Tianwu:  tianwu,
	}
}

// DailyStarIndex 日系星索引
type DailyStarIndex struct {
	Santai, Bazuo, Enguang, Tiangui int
}

// GetDailyStarIndex 获取日系星索引
func GetDailyStarIndex(lunar *calendar.Lunar, timeIndex int) DailyStarIndex {
	lunarDay := lunar.GetDay()
	monthIndex := fixLunarMonthIndex(lunar, timeIndex, false)

	zuo, you := GetZuoYouIndex(monthIndex + 1) // monthIndex 0-based, lunar month 1-based
	chang, qu := GetChangQuIndex(timeIndex)

	dayIndex := lunarDay - 1
	if timeIndex >= 12 {
		dayIndex = lunarDay // 晚子时不变
	}

	santai := FixIndex12(zuo + dayIndex)
	bazuo := FixIndex12(you - dayIndex)
	enguang := FixIndex12(chang + dayIndex - 1)
	tiangui := FixIndex12(qu + dayIndex - 1)

	return DailyStarIndex{Santai: santai, Bazuo: bazuo, Enguang: enguang, Tiangui: tiangui}
}

// TimelyStarIndex 时系星索引
type TimelyStarIndex struct {
	Taifu, Fenggao int
}

// GetTimelyStarIndex 获取时系星索引
func GetTimelyStarIndex(timeIndex int) TimelyStarIndex {
	ti := FixIndex12(timeIndex)
	taifu := FixIndex12(FixEarthlyBranchIndex("午") + ti)
	fenggao := FixIndex12(FixEarthlyBranchIndex("寅") + ti)
	return TimelyStarIndex{Taifu: taifu, Fenggao: fenggao}
}
