package runtime

import (
	"github.com/wikiglobal/suanming-agent/internal/guidance"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// preflightResult 描述 preflight 检查的结果。
type preflightResult struct {
	ShortCircuit bool   // 是否短路
	TurnType     string // 转向类型标识
	Text         string // 用户可见的提示文本
}

// preflight 在执行 agent 前做确定性硬判断，确保模型只能在批准边界内行动。
func preflight(st *state.SessionState, route *policy.ApprovedRoute, message string) preflightResult {
	// 0. supervisor 显式返回 guidance directive，经 sniff 核验后短路到统一 renderer。
	// 必须在 collect_profile 检查之前，否则 guided entry 的 directive 会被拦截。
	if route.Directive != nil {
		sig := guidance.Sniff(message)
		// sniff 不认可时视为模型误判，忽略 directive 继续走后续步骤。
		// 但如果 session 已有引导状态（多轮引导中），保留 directive 以维持状态机推进。
		trustDirective := (route.Directive.Kind == "offer_consult" && sig.ShouldOfferConsult()) ||
			(route.Directive.Kind == "choose_topic" && sig.ShouldChooseTopic()) ||
			st.Guidance != nil
		if !trustDirective {
			route.Directive = nil // 清掉被拒绝的 directive，避免 downstream 被污染
		} else {
			return preflightResult{
				ShortCircuit: true,
				TurnType:     "clarification",
				Text: guidance.Render(guidance.Request{
					Directive: route.Directive,
					Context: guidance.Context{
						ClarificationQuestion: route.ClarificationQuestion,
					},
				}),
			}
		}
	}

	// 1. collect_profile 时，根据资料完整度决定是否短路。
	// supervisor 决定 collect_profile 时可能还没看到 MergeProfile 后的新资料，
	// 所以这里用会话最新状态做最终判断。
	if route.TaskIntent == "collect_profile" {
		if st.IsProfileComplete() {
			// 资料已齐全 → 直接放行进入解读，忽略 supervisor 可能残留的 NeedsClarification
			return preflightResult{}
		}
		if _, hasGender := st.Profile["gender"]; !hasGender {
			_, hasHour := st.Profile["hour"]
			_, hasDay := st.Profile["day"]
			if hasHour || hasDay {
				// 有出生时间但缺性别 → 短路追问性别
				return preflightResult{
					ShortCircuit: true,
					TurnType:     "clarification",
					Text:         guidance.Render(guidance.Request{Boundary: guidance.BoundaryCollectGenderFromBirthTime}),
				}
			}
		}
		// 资料完全空时，先检查是否适合走引导式入口
		sig := guidance.Sniff(message)
		if sig.ShouldOfferConsult() {
			return preflightResult{
				ShortCircuit: true,
				TurnType:     "clarification",
				Text: guidance.Render(guidance.Request{
					Directive: &schemas.ConversationDirective{Kind: "offer_consult"},
				}),
			}
		}
		if sig.ShouldChooseTopic() {
			return preflightResult{
				ShortCircuit: true,
				TurnType:     "clarification",
				Text: guidance.Render(guidance.Request{
					Directive: &schemas.ConversationDirective{Kind: "choose_topic", OptionSet: "top_topics"},
				}),
			}
		}
		// 直接追问出生信息
		boundary := guidance.BoundaryAskFullProfile
		switch route.PrimaryDomain {
		case "bazi":
			boundary = guidance.BoundaryAskBaziProfile
		case "ziwei":
			boundary = guidance.BoundaryAskZiweiProfile
		}
		return preflightResult{
			ShortCircuit: true,
			TurnType:     "clarification",
			Text:         guidance.Render(guidance.Request{Boundary: boundary}),
		}
	}

	// 2. supervisor 明确要求澄清
	if route.NeedsClarification {
		boundary := guidance.BoundaryClarificationFallback
		if route.ClarificationQuestion == "" {
			switch {
			case route.TaskIntent == "collect_profile" && route.PrimaryDomain == "bazi":
				boundary = guidance.BoundaryAskBaziProfile
			case route.TaskIntent == "collect_profile" && route.PrimaryDomain == "ziwei":
				boundary = guidance.BoundaryAskZiweiProfile
			}
		}
		return preflightResult{
			ShortCircuit: true,
			TurnType:     "clarification",
			Text: guidance.Render(guidance.Request{
				Boundary: boundary,
				Context: guidance.Context{
					ClarificationQuestion: route.ClarificationQuestion,
				},
			}),
		}
	}

	profileComplete := st.IsProfileComplete()

	// 3. profile_requirement=full 但没有完整资料
	//    例外：collect_profile / amend_profile — 用户正在提供资料，supervisor 已提取到 route.Slots.Profile
	if route.PolicyHints.ProfileRequirement == "full" && !profileComplete &&
		route.TaskIntent != "collect_profile" &&
		route.TaskIntent != "amend_profile" {
		return preflightResult{
			ShortCircuit: true,
			TurnType:     "ask_missing_profile",
			Text:         guidance.Render(guidance.Request{Boundary: guidance.BoundaryAskFullProfile}),
		}
	}

	// 4. 八字主域
	if route.PrimaryDomain == "bazi" {
		hasChart := st.HasBaziResult()
		if !profileComplete && !hasChart &&
			route.TaskIntent != "collect_profile" &&
			route.TaskIntent != "amend_profile" &&
			route.TaskIntent != "direct_bazi" {
			return preflightResult{
				ShortCircuit: true,
				TurnType:     "ask_missing_profile",
				Text:         guidance.Render(guidance.Request{Boundary: guidance.BoundaryAskBaziProfile}),
			}
		}
	}

	// 5. 紫微主域
	if route.PrimaryDomain == "ziwei" {
		hasChart := st.HasZiWeiResult()
		if !profileComplete && !hasChart &&
			route.TaskIntent != "collect_profile" &&
			route.TaskIntent != "amend_profile" {
			return preflightResult{
				ShortCircuit: true,
				TurnType:     "ask_missing_profile",
				Text:         guidance.Render(guidance.Request{Boundary: guidance.BoundaryAskZiweiProfile}),
			}
		}
	}

	// 6. 其他所有情况放行
	return preflightResult{}
}
