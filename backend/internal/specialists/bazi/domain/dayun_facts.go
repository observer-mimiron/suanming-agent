// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责从工具载荷归一大运目录及其已验收判断的展示事实；
// 不读取会话，不调用模型、检索、追踪或输出传输。
package domain

import (
	"fmt"
	"strings"
)

// DayunJudgment 是对一条确定性大运事实的已验收解读。
// 它不生成趋势；调用方必须提供经合同校验的模型结果。
type DayunJudgment struct {
	GanZhi         string   `json:"gan_zhi"`
	Trend          string   `json:"trend"`
	Interpretation string   `json:"interpretation"`
	Evidence       []string `json:"evidence,omitempty"`
	OutcomeDomains []string `json:"outcome_domains,omitempty"`
}

// DayunPeriods 提取工具返回的大运目录，并兼容既有嵌套载荷形状。
func DayunPeriods(dayun map[string]any) []map[string]any {
	raw := dayun["dayun_analyzed"]
	if wrapper, ok := raw.(map[string]any); ok {
		raw = wrapper["dayun_analyzed"]
	}
	return dayunMapSlice(raw)
}

// DayunPeriodDisplayLabel 将已计算的大运事实归一为展示标签。
func DayunPeriodDisplayLabel(period map[string]any) string {
	ganZhi := strings.TrimSpace(dayunString(period["ganZhi"]))
	if ganZhi == "" {
		return ""
	}
	parts := []string{}
	startAge := strings.TrimSpace(fmt.Sprint(period["startAge"]))
	endAge := strings.TrimSpace(fmt.Sprint(period["endAge"]))
	if startAge != "" && endAge != "" && startAge != "<nil>" && endAge != "<nil>" {
		parts = append(parts, startAge+"-"+endAge+"岁")
	}
	startAt := ShortPeriodTime(period["startAt"])
	endAt := ShortPeriodTime(period["endAtExclusive"])
	if startAt != "" && endAt != "" {
		parts = append(parts, startAt+"至"+endAt+"前")
	}
	if len(parts) == 0 {
		return ganZhi + "运"
	}
	return ganZhi + "运（" + strings.Join(parts, "；") + "）"
}

// ShortPeriodTime 截断工具时间到既有展示粒度。
func ShortPeriodTime(raw any) string {
	text := strings.TrimSpace(fmt.Sprint(raw))
	if text == "" || text == "<nil>" {
		return ""
	}
	if len(text) >= len("2006-01-02 15:04") {
		return text[:len("2006-01-02 15:04")]
	}
	return text
}

// RenderDayunJudgmentLines 将已验收判断投影为稳定文本行，供合同检查与展示复用。
func RenderDayunJudgmentLines(judgments []DayunJudgment) []string {
	lines := make([]string, 0, len(judgments))
	for _, judgment := range judgments {
		ganZhi := strings.TrimSpace(judgment.GanZhi)
		trend := strings.TrimSpace(judgment.Trend)
		interpretation := strings.TrimSpace(judgment.Interpretation)
		if ganZhi == "" || trend == "" || interpretation == "" {
			continue
		}
		parts := []string{fmt.Sprintf("### %s：%s", ganZhi, trend), "**解读**：" + interpretation}
		for _, evidence := range NonEmptyStrings(judgment.Evidence) {
			parts = append(parts, "- **依据**："+evidence)
		}
		lines = append(lines, strings.Join(parts, "\n"))
	}
	return lines
}

// dayunMapSlice 兼容工具常见的大运数组解码形态。
func dayunMapSlice(raw any) []map[string]any {
	switch items := raw.(type) {
	case []map[string]any:
		return items
	case []map[string]string:
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			value := make(map[string]any, len(item))
			for key, field := range item {
				value[key] = field
			}
			out = append(out, value)
		}
		return out
	case []any:
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			switch value := item.(type) {
			case map[string]any:
				out = append(out, value)
			case map[string]string:
				converted := make(map[string]any, len(value))
				for key, field := range value {
					converted[key] = field
				}
				out = append(out, converted)
			}
		}
		return out
	default:
		return nil
	}
}

// dayunString 只读取原始字符串字段，避免隐式格式化业务事实。
func dayunString(raw any) string {
	value, _ := raw.(string)
	return value
}
