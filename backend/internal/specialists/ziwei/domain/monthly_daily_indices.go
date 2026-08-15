// 本文件属于紫微 domain 层。
// 本文件负责月系、日系杂曜的纯索引计算，只接收已归一的农历月日和时辰。
// 不负责 lunar-go、闰月/晚子时历法归一、命盘组装、工具、Session、trace、SSE 或最终文本。
package domain

// MonthlyStarIndex 表示月系杂曜的十二宫索引，不包含历法对象或输出格式。
type MonthlyStarIndex struct {
	Yuejie, Tianyao, Tianxing, Yinsha, Tianyue, Tianwu int
}

// GetMonthlyStarIndex 根据从零开始的已归一农历月序计算月系杂曜索引。
func GetMonthlyStarIndex(monthIndex int) MonthlyStarIndex {
	yuejieBranches := [6]string{"申", "戌", "子", "寅", "辰", "午"}
	yinshaBranches := [6]string{"寅", "子", "戌", "申", "午", "辰"}
	tianyueBranches := [12]string{"戌", "巳", "辰", "寅", "未", "卯", "亥", "未", "寅", "午", "戌", "寅"}
	tianwuBranches := [4]string{"巳", "申", "寅", "亥"}

	return MonthlyStarIndex{
		Yuejie:   FixIndex12(FixEarthlyBranchIndex(yuejieBranches[monthIndex/2])),
		Tianyao:  FixIndex12(FixEarthlyBranchIndex("丑") + monthIndex),
		Tianxing: FixIndex12(FixEarthlyBranchIndex("酉") + monthIndex),
		Yinsha:   FixIndex12(FixEarthlyBranchIndex(yinshaBranches[monthIndex%6])),
		Tianyue:  FixIndex12(FixEarthlyBranchIndex(tianyueBranches[monthIndex])),
		Tianwu:   FixIndex12(FixEarthlyBranchIndex(tianwuBranches[monthIndex%4])),
	}
}

// DailyStarIndex 表示日系杂曜的十二宫索引，不包含历法对象或输出格式。
type DailyStarIndex struct {
	Santai, Bazuo, Enguang, Tiangui int
}

// GetDailyStarIndex 根据农历日、已归一月序和时辰计算日系杂曜索引。
func GetDailyStarIndex(lunarDay, monthIndex, timeIndex int) DailyStarIndex {
	zuo, you := GetZuoYouIndex(monthIndex + 1)
	chang, qu := GetChangQuIndex(timeIndex)

	dayIndex := lunarDay - 1
	if timeIndex >= 12 {
		dayIndex = lunarDay
	}

	return DailyStarIndex{
		Santai:  FixIndex12(zuo + dayIndex),
		Bazuo:   FixIndex12(you - dayIndex),
		Enguang: FixIndex12(chang + dayIndex - 1),
		Tiangui: FixIndex12(qu + dayIndex - 1),
	}
}
