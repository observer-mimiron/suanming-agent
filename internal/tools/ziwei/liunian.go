package ziwei

import (
	"github.com/6tail/lunar-go/calendar"
)

// LiuNianMutagen 流年四化标记
type LiuNianMutagen struct {
	Star    string `json:"star"`    // 星名
	Mutagen string `json:"mutagen"` // 化禄/化权/化科/化忌
}

// LiuNianInfo 流年信息
type LiuNianInfo struct {
	Year       int               `json:"year"`        // 流年（公历）
	YearStem   string            `json:"year_stem"`   // 流年天干
	YearBranch string            `json:"year_branch"` // 流年地支
	Mutagens   [4]LiuNianMutagen `json:"mutagens"`    // 流年四化
	AgePalace  string            `json:"age_palace"`  // 小限宫位名
}

// GetLiuNian 计算流年信息
// baseChart 为本命盘，targetYear 为流年公历年份，currentAge 为虚岁年龄
func GetLiuNian(baseChart *ZiWeiChart, targetYear int, currentAge int) *LiuNianInfo {
	// 流年干支
	solar := calendar.NewSolar(targetYear, 6, 15, 12, 0, 0) // 年中某日取年干支
	lunar := solar.GetLunar()
	yearStem := lunar.GetYearGan()
	yearBranch := lunar.GetYearZhi()
	yearStemIdx := stemIndex(yearStem)

	// 流年四化
	ln := &LiuNianInfo{
		Year:       targetYear,
		YearStem:   yearStem,
		YearBranch: yearBranch,
	}
	mutagens := MutagenTable[yearStemIdx]
	mutagenNames := [4]string{"化禄", "化权", "化科", "化忌"}
	for i, star := range mutagens {
		ln.Mutagens[i] = LiuNianMutagen{Star: star, Mutagen: mutagenNames[i]}
	}

	// 小限：男顺女逆，起始宫位由出生年支决定
	yearBranchBirth := baseChart.FourPillars["年柱"][3:] // 年柱第2字符开始是地支
	startIdx := GetAgeIndex(string([]rune(yearBranchBirth)[0:1])) // 年支首字符

	isMale := baseChart.Gender == "男"
	age := currentAge
	if age < 1 {
		age = 1
	}

	var agePalaceIdx int
	if isMale {
		agePalaceIdx = FixIndex12(startIdx + (age - 1))
	} else {
		agePalaceIdx = FixIndex12(startIdx - (age - 1))
	}

	// 找到对应宫名
	for _, p := range baseChart.Palaces {
		if p.Index == agePalaceIdx {
			ln.AgePalace = p.Name
			break
		}
	}

	return ln
}

// ApplyLiuNianMutagens 将流年四化应用到命盘星曜上
// 返回标记了流年四化的星曜列表（不修改原命盘）
func ApplyLiuNianMutagens(chart *ZiWeiChart, liuNian *LiuNianInfo) map[string]string {
	result := make(map[string]string)
	for _, m := range liuNian.Mutagens {
		result[m.Star] = m.Mutagen
	}
	return result
}

// GetLiuNianByYear 简化版：给定出生信息和目标年，直接算流年
func GetLiuNianByYear(birthYear, birthMonth, birthDay, birthHour int, gender string, targetYear int, currentAge int) (*LiuNianInfo, *ZiWeiChart, error) {
	solar := calendar.NewSolar(birthYear, birthMonth, birthDay, birthHour, 0, 0)
	timeIndex := TimeToIndex(birthHour)
	chart, err := BuildChart(solar, timeIndex, gender)
	if err != nil {
		return nil, nil, err
	}
	ln := GetLiuNian(chart, targetYear, currentAge)
	return ln, chart, nil
}
