package runtime

import (
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// guardFinalAnswer 在最终回答输出前做结果验收，避免模型跳过关键排盘仍直接下结论。
func guardFinalAnswer(route policy.ApprovedRoute, st *state.SessionState, finalText string) (turnType string, text string) {
	if route.PrimaryDomain == "qimen" && !st.HasQimenResult() {
		return "guardrail_blocked", "本轮问题已判定为奇门主链，但运行时没有拿到奇门盘结果，所以已拦截本轮结论输出。请重试；若再次出现，请检查 `qimen_dunjia` 是否真正被调用。"
	}
	if route.PrimaryDomain == "ziwei" && !st.HasZiWeiResult() {
		return "guardrail_blocked", "本轮问题已判定为紫微主链，但运行时没有拿到紫微命盘结果，所以已拦截本轮结论输出。请重试；若再次出现，请检查 `ziwei_calc` 是否真正被调用。"
	}
	return "agent_reading", finalText
}

func shouldBufferFinalAnswer(route policy.ApprovedRoute) bool {
	switch route.PrimaryDomain {
	case "qimen", "ziwei":
		return true
	default:
		return false
	}
}
