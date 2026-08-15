// 本文件属于紫微 adapter 层的确定性排盘实现。
// 本文件负责主星、辅星和杂曜的宫位定位输入计算。
// 不负责模型、Session、trace、SSE 或最终文本。
package adapter

import (
	"github.com/6tail/lunar-go/calendar"
	ziweidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/ziwei/domain"
)

// GetStartIndex 从 lunar-go 提取农历日并调用 domain 起紫微规则，保留既有 adapter 签名。
func GetStartIndex(solar *calendar.Solar, timeIndex int, heavenlyStemOfSoul, earthlyBranchOfSoul string) (int, int) {
	lunar := solar.GetLunar()
	lunarDay := lunar.GetDay()

	_, fiveElemNum := GetFiveElementsClass(heavenlyStemOfSoul, earthlyBranchOfSoul)
	if timeIndex == 12 {
		lunarDay++ // 晚子时加一日，历法边界仍由 adapter 负责。
	}
	return ziweidomain.GetZiweiStartIndex(lunarDay, fiveElemNum)
}

// GetLuYangTuoMaIndex 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetLuYangTuoMaIndex(yearStem, yearBranch string) (lu, yang, tuo, ma int) {
	return ziweidomain.GetLuYangTuoMaIndex(yearStem, yearBranch)
}

// GetKuiYueIndex 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetKuiYueIndex(yearStem string) (kui, yue int) {
	return ziweidomain.GetKuiYueIndex(yearStem)
}

// GetZuoYouIndex 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetZuoYouIndex(lunarMonth int) (zuo, you int) {
	return ziweidomain.GetZuoYouIndex(lunarMonth)
}

// GetChangQuIndex 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetChangQuIndex(timeIndex int) (chang, qu int) {
	return ziweidomain.GetChangQuIndex(timeIndex)
}

// GetKongJieIndex 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetKongJieIndex(timeIndex int) (kong, jie int) {
	return ziweidomain.GetKongJieIndex(timeIndex)
}

// GetHuoLingIndex 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetHuoLingIndex(yearBranch string, timeIndex int) (huo, ling int) {
	return ziweidomain.GetHuoLingIndex(yearBranch, timeIndex)
}

// GetLuanXiIndex 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetLuanXiIndex(yearBranch string) (hongluan, tianxi int) {
	return ziweidomain.GetLuanXiIndex(yearBranch)
}

// GetHuagaiXianchiIndex 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetHuagaiXianchiIndex(yearBranch string) (huagai, xianchi int) {
	return ziweidomain.GetHuagaiXianchiIndex(yearBranch)
}

// GetGuGuaIndex 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetGuGuaIndex(yearBranch string) (guchen, guasu int) {
	return ziweidomain.GetGuGuaIndex(yearBranch)
}

// GetJieshaIndex 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetJieshaIndex(yearBranch string) int { return ziweidomain.GetJieshaIndex(yearBranch) }

// GetDahaoIndex 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetDahaoIndex(yearBranch string) int { return ziweidomain.GetDahaoIndex(yearBranch) }

// GetNianjieIndex 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetNianjieIndex(yearBranch string) int { return ziweidomain.GetNianjieIndex(yearBranch) }

// YearlyStarIndex 是 domain 年系索引结果的 adapter 兼容别名。
type YearlyStarIndex = ziweidomain.YearlyStarIndex

// GetYearlyStarIndex 保留旧 adapter 签名，转发到不依赖 lunar-go 的 domain 规则。
func GetYearlyStarIndex(_ *calendar.Solar, _ int, gender, yearStem, yearBranch string, soulIndex, bodyIndex int) YearlyStarIndex {
	return ziweidomain.GetYearlyStarIndex(gender, yearStem, yearBranch, soulIndex, bodyIndex)
}

// GetTianshiTianshangIndex 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetTianshiTianshangIndex(gender, yearBranch string, soulIndex int) (tianshi, tianshang int) {
	return ziweidomain.GetTianshiTianshangIndex(gender, yearBranch, soulIndex)
}

// TimelyStarIndex 是 domain 时系索引结果的 adapter 兼容别名。
type TimelyStarIndex = ziweidomain.TimelyStarIndex

// GetTimelyStarIndex 将既有 adapter 调用转发到 domain 唯一规则实现。
func GetTimelyStarIndex(timeIndex int) TimelyStarIndex {
	return ziweidomain.GetTimelyStarIndex(timeIndex)
}

// MonthlyStarIndex 是 domain 月系索引结果的 adapter 兼容别名。
type MonthlyStarIndex = ziweidomain.MonthlyStarIndex

// GetMonthlyStarIndex 获取月系杂曜索引。按月支推算月解、天姚（桃花星）、天刑（刑伤星）、
// 阴煞（阴性煞星）、天月、天巫（宗教星）的位置。
func GetMonthlyStarIndex(lunar *calendar.Lunar, timeIndex int) MonthlyStarIndex {
	monthIndex := fixLunarMonthIndex(lunar, timeIndex, false) // 从0开始
	return ziweidomain.GetMonthlyStarIndex(monthIndex)
}

// DailyStarIndex 是 domain 日系索引结果的 adapter 兼容别名。
type DailyStarIndex = ziweidomain.DailyStarIndex

// GetDailyStarIndex 获取日系杂曜索引。按日支推算三台、八座、恩光、天贵的位置，
// 这些星曜与左辅右弼和文昌文曲的位置关系密切。
func GetDailyStarIndex(lunar *calendar.Lunar, timeIndex int) DailyStarIndex {
	lunarDay := lunar.GetDay()
	monthIndex := fixLunarMonthIndex(lunar, timeIndex, false)
	return ziweidomain.GetDailyStarIndex(lunarDay, monthIndex, timeIndex)
}
