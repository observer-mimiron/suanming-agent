package orchestrator

import "github.com/wikiglobal/suanming-agent/internal/policy"

// bridgeDecision 将策略批准的路线转换为旧版基于动作的路由元组。
// 必须接收 ApprovedRoute（策略门后），而非原始 SupervisorDecision，
// 以确保低置信度澄清、领域降级和其他策略覆盖在实时代理流程中生效。
func bridgeDecision(route policy.ApprovedRoute, msg string) (action string, patch map[string]any, question string, needsQimen bool, rawBazi []string) {
	patch = route.Slots.Profile
	if patch == nil {
		patch = map[string]any{}
	}
	question = route.Slots.QuestionText
	if question == "" {
		question = msg
	}
	needsQimen = route.PolicyHints.NeedsQimen

	// 如果策略门强制要求澄清，短路到 incomplete 状态。
	if route.NeedsClarification {
		return "incomplete", patch, question, needsQimen, rawBazi
	}

	switch route.TaskIntent {
	case "collect_profile":
		action = "new_profile"
	case "amend_profile":
		action = "update_profile"
	case "direct_bazi":
		action = "bazi_input"
		rawBazi = extractBaziPillars(msg)
	case "interpret_chart":
		action = "followup"
	case "fortune_followup":
		action = "followup"
		needsQimen = true
	case "timing_followup", "cross_domain_consult":
		action = "followup"
		needsQimen = true
	default:
		action = "followup"
	}

	return action, patch, question, needsQimen, rawBazi
}
