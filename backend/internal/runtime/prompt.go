// This file belongs to the manager-owned runtime layer.
// It owns runtime prompt construction for this package.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"fmt"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// buildProfileSection 构建出生资料文本。同时给出原始出生时间和系统已计算的权威四柱。
func buildProfileSection(st *state.SessionState) string {
	year, _ := st.Profile["year"]
	month, _ := st.Profile["month"]
	day, _ := st.Profile["day"]
	hour, _ := st.Profile["hour"]
	minute, _ := st.Profile["minute"]
	gender, _ := st.Profile["gender"].(string)
	birthplace, _ := st.Profile["birthplace"].(string)

	var lines []string

	// 命盘归属
	if st.Subject != "" {
		lines = append(lines, "命盘归属："+st.Subject)
	}

	// 原始出生时间
	timeStr := fmt.Sprintf("%v年%v月%v日%v时", year, month, day, hour)
	if minute != nil {
		timeStr = fmt.Sprintf("%v年%v月%v日%v:%v", year, month, day, hour, minute)
	}
	meta := []string{timeStr}
	if gender != "" {
		meta = append(meta, gender)
	}
	if birthplace != "" {
		meta = append(meta, birthplace)
	}
	lines = append(lines, "出生时间："+strings.Join(meta, "，"))

	// 系统计算的权威四柱（已做真太阳时校正和晚子时处理）
	if st.HasBaziResult() {
		if pillars, ok := st.BaziResult["pillars"].([]map[string]any); ok && len(pillars) == 4 {
			lines = append(lines, fmt.Sprintf(
				"**系统排盘结果（权威，你必须使用此结果，不得自行推算）：%s%s年 %s%s月 %s%s日 %s%s时**",
				pillars[0]["stem"], pillars[0]["branch"],
				pillars[1]["stem"], pillars[1]["branch"],
				pillars[2]["stem"], pillars[2]["branch"],
				pillars[3]["stem"], pillars[3]["branch"],
			))
		}
		if dayGan, ok := st.BaziResult["dayGan"].(string); ok && dayGan != "" {
			lines = append(lines, fmt.Sprintf("日主%s", dayGan))
		}
		if birthday, ok := st.BaziResult["birthday"].(string); ok && birthday != "" {
			lines = append(lines, fmt.Sprintf("（系统校正后时间：%s）", birthday))
		}
	}

	return strings.Join(lines, "\n") + "\n"
}
