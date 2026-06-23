package runtime

import (
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/guidance"
	"github.com/wikiglobal/suanming-agent/internal/intent"
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

	// ── Active guidance: check break conditions, then allow continuation ──
	if st.Guidance != nil {
		if anyHardNegative(trimmed, route, signal) {
			return false // break guidance: birth info / explicit method / explicit action
		}
		// qimen primary timing or sniff timing focus → break guidance
		if route.PolicyHints.QimenMode == "primary" || intent.HasTimingFocus(trimmed) {
			return false
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

// anyHardNegative 检查硬性不进入/断开 guidance 的条件。
// 这些条件无论是否有 active guidance 都生效（如 birth info / explicit method）。
func anyHardNegative(msg string, route policy.ApprovedRoute, signal guidance.Signal) bool {
	if intent.ContainsBirthInfo(msg) {
		return true
	}
	if intent.MentionsQimenMethod(msg) || intent.MentionsZiweiMethod(msg) || intent.MentionsBaziMethod(msg) {
		return true
	}
	if intent.ContainsExplicitDivinationAction(msg) {
		return true
	}
	return false
}

// anyHardNegativeForNewEntry 检查新入场（无 active guidance 时）的额外 hard negative。
// sniff 的 fate-adjacent / broad-intent 信号优先于模型路由（qimen primary / collect_profile 等），
// 防止模型误路由阻止 guidance 入场。
func anyHardNegativeForNewEntry(msg string, route policy.ApprovedRoute, signal guidance.Signal) bool {
	if anyHardNegative(msg, route, signal) {
		return true
	}
	// sniff 明确命中引导信号 → 允许入场，不阻
	hasGuidanceSignal := signal.ShouldOfferConsult() || signal.ShouldChooseTopic()
	if hasGuidanceSignal {
		return false
	}
	// 无引导信号时：qimen primary timing 和 collect_profile 不进场
	if route.PolicyHints.QimenMode == "primary" {
		return true
	}
	if intent.HasTimingFocus(msg) {
		return true
	}
	if route.TaskIntent == "collect_profile" || route.TaskIntent == "amend_profile" {
		return true
	}
	return false
}

