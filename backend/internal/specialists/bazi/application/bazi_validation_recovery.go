// Package application 包含八字用例层的合同辅助函数。
//
// 本文件只保留共享的文本选择、事实摘要和错误分类辅助；
// 不负责改写模型综合结果、接受 partial 输出或执行恢复状态机。
package application

import (
	"fmt"
	"strings"

	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
)

const (
	dynamicFlagMixedConstraint     = "吉中有阻"
	dynamicFlagVolatileOpportunity = "机会伴随强变动"
	dynamicFlagLimitationRemains   = "限制仍在"
	dynamicFlagStructureOnly       = "仅作结构观察"
)

var allowedDynamicConsistencyFlags = []string{
	dynamicFlagMixedConstraint,
	dynamicFlagVolatileOpportunity,
	dynamicFlagLimitationRemains,
	dynamicFlagStructureOnly,
}

// strengthEvidenceSummary 把确定性强弱证据压成 renderer 可读的摘要，不替模型作综合判断。
func strengthEvidenceSummary(yongshen map[string]any) string {
	strength := strings.TrimSpace(stringValue(yongshen["strength"]))
	evidence, _ := yongshen["strength_evidence"].(map[string]any)
	support := intValue(evidence["support_score"])
	pressure := intValue(evidence["pressure_score"])
	month := intValue(yongshen["month_score"])
	root := intValue(yongshen["root_count"])
	sameElement := intValue(yongshen["same_element"])
	generate := intValue(yongshen["generate_count"])
	if strength == "" {
		return "整体受力仍需保守判断。"
	}
	return fmt.Sprintf("日主%s；月令受力 %d，通根 %d 处，同类透干 %d，印星生扶 %d；扶身合计 %d，食伤泄身、财耗与官杀克合计 %d。", strength, month, root, sameElement, generate, support, pressure)
}

// recoveryReasonText 把内部失败原因转为恢复记录；没有原因时使用固定降级说明。
func recoveryReasonText(cause error, fallback string) string {
	if cause == nil || strings.TrimSpace(cause.Error()) == "" {
		return fallback
	}
	return cause.Error()
}

// hasDynamicHardBoundary 识别动态文本中的高风险具体应事，供严格 validator 使用。
func hasDynamicHardBoundary(text string) bool {
	return bazidomain.HasUnsupportedConcreteOutcome(text) || containsAnyText([]string{text}, []string{"投资", "投资建议"})
}

// normalizeByAlias 只服务分析计划兼容值，不改写静态或动态综合结果。
func normalizeByAlias(value string, aliases map[string]string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if canonical, ok := aliases[value]; ok {
		return canonical
	}
	return value
}

// NormalizeByAlias maps a planner compatibility value to its canonical enum.
func NormalizeByAlias(value string, aliases map[string]string) string {
	return normalizeByAlias(value, aliases)
}

// firstNonEmptyTrim 返回第一个非空文本，供 runtime 投影选择确定性回退值。
func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
