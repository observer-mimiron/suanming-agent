// 本文件属于紫微 adapter 层的确定性排盘实现。
// 本文件负责命宫、身宫、五行局和十二宫大限的基础计算。
// 不负责模型、Session、trace、SSE 或最终文本。
package adapter

import (
	"github.com/6tail/lunar-go/calendar"
	ziweidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/ziwei/domain"
)

// SoulAndBody 命宫身宫计算结果的容器。SoulIndex/BodyIndex 为命宫身在十二宫中的索引（0-11），
// HeavenlyStemOfSoul/EarthlyBranchOfSoul 为命宫对应的干支。
type SoulAndBody struct {
	SoulIndex           int
	BodyIndex           int
	HeavenlyStemOfSoul  string
	EarthlyBranchOfSoul string
}

// GetSoulAndBody 从 lunar-go 提取四柱和归一农历月序，再委托 domain 计算命宫身宫。
// 返回值保留既有 adapter 兼容类型；闰月和晚子时处理仍由本层负责。
func GetSoulAndBody(solar *calendar.Solar, timeIndex int, fixLeap bool) SoulAndBody {
	lunar := solar.GetLunar()
	monthIndex := fixLunarMonthIndex(lunar, timeIndex, fixLeap)

	ec := lunar.GetEightChar()
	ec.SetSect(1)

	timeBranchIdx := branchIndex(ec.GetTimeZhi())
	yearStemIdx := stemIndex(ec.GetYearGan())
	indices := ziweidomain.GetSoulAndBody(monthIndex, timeBranchIdx, yearStemIdx)

	return SoulAndBody{
		SoulIndex:           indices.SoulIndex,
		BodyIndex:           indices.BodyIndex,
		HeavenlyStemOfSoul:  HeavenStems[indices.HeavenlyStemIndex],
		EarthlyBranchOfSoul: EarthBranches[indices.EarthlyBranchIndex],
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

// GetFiveElementsClass 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetFiveElementsClass(heavenlyStem, earthlyBranch string) (string, int) {
	return ziweidomain.GetFiveElementsClass(heavenlyStem, earthlyBranch)
}

// GetPalaceNames 将既有 adapter 调用转发到 domain 唯一宫名旋转规则。
func GetPalaceNames(soulIndex int) [12]string {
	return ziweidomain.GetPalaceNames(soulIndex)
}

// GetHoroscope 起大限（定大限起始年龄和走势）。大限是紫微斗数中十年一运的区间判断依据。
//
// 规则：阳男阴女顺行（从命宫起顺时针每步进一宫），阴男阳女逆行（逆时针每步退一宫）。
// 起运年龄由五行局数值决定：水二局2岁起运、木三局3岁、金四局4岁、土五局5岁、火六局6岁，每步大限增加10年。
func GetHoroscope(solar *calendar.Solar, timeIndex int, gender string, fixLeap bool, soulIndex int, heavenlyStemOfSoul, earthlyBranchOfSoul string, yearStemIdx int, yearBranchIdx int) [12]DecadalInfo {
	_, fiveElemNum := GetFiveElementsClass(heavenlyStemOfSoul, earthlyBranchOfSoul)
	intervals := ziweidomain.GetDecadalIntervals(gender, soulIndex, fiveElemNum, yearStemIdx, yearBranchIdx)
	var decadals [12]DecadalInfo
	for i, interval := range intervals {
		decadals[i] = DecadalInfo{
			StartAge:      interval.StartAge,
			EndAge:        interval.EndAge,
			HeavenlyStem:  interval.HeavenlyStem,
			EarthlyBranch: interval.EarthlyBranch,
		}
	}
	return decadals
}
