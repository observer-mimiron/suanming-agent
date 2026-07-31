package ziwei

import (
	"context"
	"fmt"

	"github.com/6tail/lunar-go/calendar"
)

// LiuNianMutagen 流年四化标记。表示某一年的流年四化情况，即该年哪些星曜产生了化禄/化权/化科/化忌。
// 流年四化用于判断该年的吉凶应事领域。
type LiuNianMutagen struct {
	Star    string `json:"star"`    // 星名
	Mutagen string `json:"mutagen"` // 化禄/化权/化科/化忌
}

// LiuNianInfo 流年信息。包含流年的干支、四化情况和小限所在宫位。
// 流年用于推算特定年份的运势走向，与命盘大限配合解读。
type LiuNianInfo struct {
	Year       int               `json:"year"`        // 流年（公历）
	YearStem   string            `json:"year_stem"`   // 流年天干
	YearBranch string            `json:"year_branch"` // 流年地支
	Mutagens   [4]LiuNianMutagen `json:"mutagens"`    // 流年四化
	AgePalace  string            `json:"age_palace"`  // 小限宫位名
}

// GetLiuNian 计算流年信息。基于本命盘推算指定公历年份的流年干支、四化和小限宫位。
// baseChart 为本命盘（由 BuildChart 生成），targetYear 为流年公历年份，currentAge 为虚岁年龄。
// 流年分析是紫微斗数咨询中最常用的动态推运方法，与大限配合使用。
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
	yearBranchBirth := baseChart.FourPillars["年柱"][3:]            // 年柱第2字符开始是地支
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

// ApplyLiuNianMutagens 将流年四化映射到命盘星曜。返回星名到四化的映射表（如"紫微"→"化权"），
// 不修改原命盘数据，适用于展示流年四化信息。
func ApplyLiuNianMutagens(chart *ZiWeiChart, liuNian *LiuNianInfo) map[string]string {
	result := make(map[string]string)
	for _, m := range liuNian.Mutagens {
		result[m.Star] = m.Mutagen
	}
	return result
}

// GetLiuNianByYear 便捷函数：给定出生年月日时和性别，直接排本命盘并计算流年信息。
// 适用于一次性的流年查询场景，内部委托 BuildChart 和 GetLiuNian。
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

// ZiWeiLiuNianTool 紫微斗数流年分析工具。
// 根据出生信息和目标年份推算流年干支、四化和小限宫位。
type ZiWeiLiuNianTool struct{}

func (t *ZiWeiLiuNianTool) Name() string { return "ziwei_liunian" }

func (t *ZiWeiLiuNianTool) Description() string {
	return "紫微斗数流年分析，输入出生年月日时+性别+目标年份+虚岁年龄，返回流年干支、四化、小限宫位"
}

func (t *ZiWeiLiuNianTool) Label() string { return "流年分析" }

func (t *ZiWeiLiuNianTool) Execute(_ context.Context, params map[string]any) (any, error) {
	year, _ := params["year"].(float64)
	month, _ := params["month"].(float64)
	day, _ := params["day"].(float64)
	hour, _ := params["hour"].(float64)
	gender, _ := params["gender"].(string)
	targetYear, _ := params["target_year"].(float64)
	age, _ := params["age"].(float64)

	solar, timeIndex := correctedBirthSolar(int(year), int(month), int(day), int(hour), params)
	chart, err := BuildChart(solar, timeIndex, gender)
	if err != nil {
		return nil, fmt.Errorf("流年分析失败: %w", err)
	}
	ln := GetLiuNian(chart, int(targetYear), int(age))

	return map[string]any{
		"year":        ln.Year,
		"year_stem":   ln.YearStem,
		"year_branch": ln.YearBranch,
		"mutagens":    ln.Mutagens,
		"age_palace":  ln.AgePalace,
	}, nil
}
