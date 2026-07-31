package runtime

import (
	"context"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/guidance"
	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// ShouldEnterGuidance 判断本轮消息是否允许进入或继续 guidance。
// router 非空时优先用 router；nil 走 regex 兜底（受 Confidence 守卫）。
func ShouldEnterGuidance(router intent.Router, message string, route policy.ApprovedRoute, st *state.SessionState) bool {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return false
	}

	signal := guidance.Sniff(trimmed)

	if st.Guidance != nil {
		if anyHardNegative(router, trimmed, route, signal) {
			return false
		}
		if route.PolicyHints.QimenMode == "primary" || intent.HasTimingFocus(trimmed) {
			return false
		}
		return true
	}

	if anyHardNegativeForNewEntry(router, trimmed, route, signal) {
		return false
	}

	if signal.ShouldOfferConsult() || signal.ShouldChooseTopic() {
		return true
	}

	return false
}

// anyHardNegative 检查硬性不进入/断开 guidance 的条件。
// router 非空时优先用 router；nil 或 err 走 regex 兜底（受 Confidence 守卫）。
func anyHardNegative(router intent.Router, msg string, route policy.ApprovedRoute, signal guidance.Signal) bool {
	if intent.ContainsBirthInfo(msg) {
		return true
	}
	if intent.ContainsExplicitDivinationAction(msg) {
		return true
	}

	// 术数方法提及——router 优先
	if router != nil {
		result, err := router.Match(context.Background(), msg)
		if err == nil {
			if result.Decision == intent.DecisionPositive {
				return true
			}
			if result.Decision == intent.DecisionNegative || result.Decision == intent.DecisionNone {
				return false // negative/none 不断——避免被 regex 击穿
			}
		}
		// err 落到 regex 兜底
	}

	// regex 兜底（router nil 或 err），受 Confidence 守卫
	if route.Confidence >= 0.7 {
		return false // 高置信，禁用 dumb regex，信任 LLM
	}
	if intent.MentionsQimenMethod(msg) || intent.MentionsZiweiMethod(msg) || intent.MentionsBaziMethod(msg) {
		return true
	}
	return false
}

// anyHardNegativeForNewEntry 检查新入场的额外 hard negative。
func anyHardNegativeForNewEntry(router intent.Router, msg string, route policy.ApprovedRoute, signal guidance.Signal) bool {
	if anyHardNegative(router, msg, route, signal) {
		return true
	}
	hasGuidanceSignal := signal.ShouldOfferConsult() || signal.ShouldChooseTopic()
	if hasGuidanceSignal {
		return false
	}
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
