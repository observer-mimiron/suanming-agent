// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责静态综合与确定性强弱事实的一致性规则；
// 不读取会话，不调用模型、检索、repair、追踪或输出传输。
package domain

import (
	"fmt"
	"strings"
)

// ValidateStaticStrengthAgainstEvidence 拒绝静态结论反转既有偏强或偏弱事实。
// 强弱只能由强弱槽位说明，主轴和格局路线也不能借此改写确定性受力结论。
func ValidateStaticStrengthAgainstEvidence(yongshen map[string]any, synthesis StaticSynthesis) error {
	strength, _ := yongshen["strength"].(string)
	strength = strings.TrimSpace(strength)
	reading := strings.Join([]string{
		strings.TrimSpace(synthesis.MainAxis),
		strings.TrimSpace(synthesis.PatternOutcome),
		strings.TrimSpace(synthesis.Strength.Conclusion),
		strings.TrimSpace(synthesis.StrengthBalance),
	}, "\n")
	if strength == "" || reading == "" {
		return nil
	}
	switch strength {
	case "偏弱":
		if containsAnyText(reading, []string{"日主偏强", "日元偏强", "日主强旺", "日元强旺", "日主身强", "日元身强", "身强"}) {
			return NewValidationError(ViolationFactConflict, "static.strength_balance", "", fmt.Sprintf("static strength reverses balance evidence: %s", strength), nil, []string{strength})
		}
	case "偏强":
		if containsAnyText(reading, []string{"日主偏弱", "日元偏弱", "日主身弱", "日元身弱", "身弱"}) {
			return NewValidationError(ViolationFactConflict, "static.strength_balance", "", fmt.Sprintf("static strength reverses balance evidence: %s", strength), nil, []string{strength})
		}
	}
	return nil
}
