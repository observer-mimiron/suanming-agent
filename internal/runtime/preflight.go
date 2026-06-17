package runtime

import "github.com/wikiglobal/suanming-agent/internal/policy"
import "github.com/wikiglobal/suanming-agent/internal/state"

// preflightResult 描述 preflight 检查的结果。
type preflightResult struct {
	ShortCircuit bool   // 是否短路
	TurnType     string // 转向类型标识
	Text         string // 用户可见的提示文本
}

// preflight 在执行 agent 前做确定性硬判断，确保模型只能在批准边界内行动。
func preflight(st *state.SessionState, route policy.ApprovedRoute) preflightResult {
	// 1. supervisor 明确要求澄清
	if route.NeedsClarification {
		question := route.ClarificationQuestion
		if question == "" {
			question = "请确认一下您的需求，我再为您详细分析。"
		}
		return preflightResult{ShortCircuit: true, TurnType: "clarification", Text: question}
	}

	profileComplete := st.IsProfileComplete()

	// 2. profile_requirement=full 但没有完整资料
	//    例外：collect_profile / amend_profile — 用户正在提供资料，supervisor 已提取到 route.Slots.Profile
	if route.PolicyHints.ProfileRequirement == "full" && !profileComplete &&
		route.TaskIntent != "collect_profile" &&
		route.TaskIntent != "amend_profile" {
		return preflightResult{
			ShortCircuit: true,
			TurnType:     "ask_missing_profile",
			Text:         "需要通过完整的出生信息进行综合分析。请提供出生年月日时和性别。",
		}
	}

	// 3. 八字主域
	if route.PrimaryDomain == "bazi" {
		hasChart := st.HasBaziResult()
		if !profileComplete && !hasChart &&
			route.TaskIntent != "collect_profile" &&
			route.TaskIntent != "amend_profile" &&
			route.TaskIntent != "direct_bazi" {
			return preflightResult{
				ShortCircuit: true,
				TurnType:     "ask_missing_profile",
				Text:         "请提供您的出生信息（年份、月份、日期、时辰、性别），我来为您排盘分析。",
			}
		}
	}

	// 4. 紫微主域
	if route.PrimaryDomain == "ziwei" {
		hasChart := st.HasZiWeiResult()
		if !profileComplete && !hasChart &&
			route.TaskIntent != "collect_profile" &&
			route.TaskIntent != "amend_profile" {
			return preflightResult{
				ShortCircuit: true,
				TurnType:     "ask_missing_profile",
				Text:         "需要出生信息才能排紫微斗数命盘。请提供出生年月日时和性别。",
			}
		}
	}

	// 5. 其他所有情况放行
	return preflightResult{}
}
