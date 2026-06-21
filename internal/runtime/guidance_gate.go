package runtime

import (
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/guidance"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// ShouldEnterGuidance 判断本轮消息是否允许进入或继续 guidance。
//
// 只做 hard gate，不改 route / session / domain。
// 优先检查 hard negative，再检查 hard positive。
func ShouldEnterGuidance(message string, route policy.ApprovedRoute, st *state.SessionState) bool {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return false
	}

	signal := guidance.Sniff(trimmed)

	// ── Active guidance: check break conditions first, then allow continuation ──
	if st.Guidance != nil {
		if anyHardNegative(trimmed, route, signal) {
			return false // break guidance, hand back to execution chain
		}
		return true // continuation
	}

	// ── No active guidance: hard negative + collect_profile gate blocks entry ──
	if anyHardNegativeForNewEntry(trimmed, route, signal) {
		return false
	}

	// ── Hard positive: first-turn fate-adjacent or broad-intent ──
	if signal.ShouldOfferConsult() || signal.ShouldChooseTopic() {
		return true
	}

	return false
}

// anyHardNegative 检查 hard negative 条件（birth info / explicit method / explicit action / qimen timing / collect_profile-only）。
func anyHardNegative(msg string, route policy.ApprovedRoute, signal guidance.Signal) bool {
	if containsBirthTime(msg) {
		return true
	}
	if mentionsQimenMethod(msg) || mentionsZiweiMethod(msg) || mentionsBaziMethod(msg) {
		return true
	}
	if containsExplicitAction(msg) {
		return true
	}
	if route.PolicyHints.QimenMode == "primary" {
		return true
	}
	if signal.TimingFocus {
		return true
	}
	return false
}

// anyHardNegativeForNewEntry 检查新入场（无 active guidance 时）的 hard negative。
func anyHardNegativeForNewEntry(msg string, route policy.ApprovedRoute, signal guidance.Signal) bool {
	if anyHardNegative(msg, route, signal) {
		return true
	}
	// 无 guidance 信号时，collect_profile / amend_profile 不应进 guidance
	if (route.TaskIntent == "collect_profile" || route.TaskIntent == "amend_profile") &&
		!signal.ShouldOfferConsult() && !signal.ShouldChooseTopic() {
		return true
	}
	return false
}

// containsBirthTime 检查消息是否包含出生日期信息。
func containsBirthTime(msg string) bool {
	patterns := []string{
		"年", "月", "日", "出生", "生的",
	}
	count := 0
	for _, p := range patterns {
		if strings.Contains(msg, p) {
			count++
		}
	}
	return count >= 2
}

// mentionsQimenMethod 检查消息是否显式提及奇门。
func mentionsQimenMethod(msg string) bool {
	return strings.Contains(msg, "奇门") || strings.Contains(msg, "遁甲")
}

// mentionsZiweiMethod 检查消息是否显式提及紫微。
func mentionsZiweiMethod(msg string) bool {
	return strings.Contains(msg, "紫微") || strings.Contains(msg, "斗数")
}

// mentionsBaziMethod 检查消息是否显式提及八字。
func mentionsBaziMethod(msg string) bool {
	return strings.Contains(msg, "八字")
}

// containsExplicitAction 检查消息是否包含显式执行请求。
func containsExplicitAction(msg string) bool {
	actions := []string{"帮我算", "帮我看", "排盘", "帮我看看", "算一下", "看一下"}
	for _, a := range actions {
		if strings.Contains(msg, a) {
			return true
		}
	}
	return false
}
