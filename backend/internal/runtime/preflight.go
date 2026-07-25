package runtime

import (
	"github.com/observer-mimiron/suanming-agent/internal/guidance"
	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/schemas"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// preflightResult 描述 preflight 检查的结果。
type preflightResult struct {
	ShortCircuit bool                  // 是否短路
	TurnType     string                // 转向类型标识
	Text         string                // 用户可见的提示文本
	GuidanceNext *state.GuidanceState  // 由 executor 应用的下一个 guidance 状态
	ForcedRoute  *policy.ApprovedRoute // guided_fallback 接受后强制路由
}

// preflight 在执行 agent 前做确定性硬判断，确保模型只能在批准边界内行动。
//
// 流程：
//  1. sniff + 已有 guidance → 计算 guidance 下一步（不写 session）
//  2. collect_profile 意图 → 根据资料完整度决定是否短路
//  3. NeedsClarification → 短路返回澄清文本
//  4. profile_requirement=full 但没有完整资料 → 短路
//  5. bazi 主域无资料且无命盘 → 短路
//  6. ziwei 主域无资料且无命盘 → 短路
func preflight(st *state.SessionState, route policy.ApprovedRoute, message string, router intent.Router) preflightResult {
	return preflightWithPlan(st, ExecutionPlan{Route: route}, message, router)
}

// preflightWithPlan 是生产主链使用的 preflight 入口。
// 这里明确消费 manager 下发的 ExecutionPlan，避免 preflight 自己再做一套 follow-up 策略判断。
func preflightWithPlan(st *state.SessionState, plan ExecutionPlan, message string, router intent.Router) preflightResult {
	route := plan.Route
	// preflight 必须保持纯函数，不直接写 session；但首轮 collect_profile 时，
	// route.Slots.Profile 已经承载了本轮刚提取出的资料。这里克隆一份 workingState
	// 并临时合并 slots，避免在 executor 真正持久化前把“已识别的输入”误判成空资料。
	workingState := st
	if st == nil {
		workingState = state.NewSession("")
	} else if len(route.Slots.Profile) > 0 {
		workingState = st.Clone()
		workingState.MergeProfile(route.Slots.Profile)
	}

	// 1. sniff + guidance 判定（code-owned 路径）
	if shouldHandleGuidance(st, route, message, router) {
		var next *state.GuidanceState
		var currentGuidance *state.GuidanceState
		if st != nil {
			currentGuidance = st.Guidance
		}
		if currentGuidance != nil {
			// 已有 guidance → 推进
			next = policy.ReduceGuidance(policy.GuidanceReducerInput{
				Current: currentGuidance,
				Message: message,
				Profile: workingState.Profile,
			})
		} else {
			// 新入场 → 初始化
			signal := guidance.Sniff(message)
			if signal.ShouldOfferConsult() {
				next = &state.GuidanceState{DirectiveKind: "offer_consult"}
			} else if signal.ShouldChooseTopic() {
				next = &state.GuidanceState{DirectiveKind: "choose_topic"}
				if signal.Topic != "" {
					next.ChosenTopic = signal.Topic
				}
			}
		}

		// guided_fallback 被接受 → 强制路由到 qimen primary profileless 链
		if next == nil && currentGuidance != nil && currentGuidance.DirectiveKind == "guided_fallback" {
			return preflightResult{
				ShortCircuit: false,
				TurnType:     "fortune_followup",
				Text:         "好的，我来用奇门遁甲帮您综合看一下当前局势。",
				GuidanceNext: nil,
				ForcedRoute: &policy.ApprovedRoute{
					PrimaryDomain: "qimen",
					TaskIntent:    "fortune_followup",
					PolicyHints: schemas.PolicyHints{
						QimenMode:          "primary",
						ProfileRequirement: "none",
					},
				},
			}
		}

		if next != nil {
			return preflightResult{
				ShortCircuit: true,
				TurnType:     "clarification",
				Text: guidance.RenderGuidance(next, guidance.Context{
					ClarificationQuestion: route.ClarificationQuestion,
				}),
				GuidanceNext: next,
			}
		}
		// next == nil && not guided_fallback acceptance → guidance complete, fall through
	}

	// 2. collect_profile 时，根据资料完整度决定是否短路。
	if route.TaskIntent == "collect_profile" {
		if workingState.IsProfileComplete() {
			return preflightResult{}
		}
		if _, hasGender := workingState.Profile["gender"]; !hasGender {
			_, hasHour := workingState.Profile["hour"]
			_, hasDay := workingState.Profile["day"]
			if hasHour || hasDay {
				return preflightResult{
					ShortCircuit: true,
					TurnType:     "clarification",
					Text:         guidance.Render(guidance.Request{Boundary: guidance.BoundaryCollectGenderFromBirthTime}),
				}
			}
		}
		missingFields := workingState.MissingFields()
		if len(missingFields) == 1 && missingFields[0] == "birthplace" {
			return preflightResult{
				ShortCircuit: true,
				TurnType:     "clarification",
				Text:         guidance.Render(guidance.Request{Boundary: guidance.BoundaryCollectBirthplaceFromProfile}),
			}
		}
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

	// 3. supervisor 明确要求澄清
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

	profileComplete := workingState.IsProfileComplete()

	// 4. profile_requirement=full 但没有完整资料
	if route.PolicyHints.ProfileRequirement == "full" && !profileComplete &&
		route.TaskIntent != "collect_profile" &&
		route.TaskIntent != "amend_profile" {
		return preflightResult{
			ShortCircuit: true,
			TurnType:     "ask_missing_profile",
			Text:         guidance.Render(guidance.Request{Boundary: guidance.BoundaryAskFullProfile}),
		}
	}

	// 5. 八字主域
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

	// 6. 紫微主域
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

	// 7. manager-owned follow-up：preflight 只消费 plan，不再自判追问策略。
	if (plan.FollowupMode == followupModeDirect || plan.FollowupMode == followupModeReuseArtifact) && plan.FollowupDirectAnswer != "" {
		return preflightResult{
			ShortCircuit: true,
			TurnType:     "fortune_followup",
			Text:         plan.FollowupDirectAnswer,
		}
	}

	// 8. 放行
	return preflightResult{}
}

// shouldHandleGuidance 判断本轮是否仍应由 guidance 接管。
// 当 supervisor 已稳定判成 follow-up，且当前主域已有可复用命盘结果时，
// 继续沿用旧 GuidanceState 会把正式追问误短路成 choose_topic / fallback 文案。
// 这里显式放行 execution path，避免“陈旧 guidance 劫持已明确的追问”。
func shouldHandleGuidance(st *state.SessionState, route policy.ApprovedRoute, message string, router intent.Router) bool {
	if st == nil {
		return ShouldEnterGuidance(router, message, route, st)
	}
	if st.Guidance == nil {
		if shouldBypassNewGuidance(st, route) {
			return false
		}
		return ShouldEnterGuidance(router, message, route, st)
	}
	if shouldBypassStaleGuidance(st, route, message) {
		return false
	}
	return true
}

func shouldBypassNewGuidance(st *state.SessionState, route policy.ApprovedRoute) bool {
	if st == nil {
		return false
	}
	if route.NeedsClarification {
		return false
	}
	if route.TaskIntent != "fortune_followup" {
		return false
	}
	// 已有可复用结果的正式追问，应直接进入执行链，避免被 broad-intent sniff 重新拉回 choose_topic。
	return hasReusableResultForDomain(st, route.PrimaryDomain)
}

func shouldBypassStaleGuidance(st *state.SessionState, route policy.ApprovedRoute, message string) bool {
	if st == nil || st.Guidance == nil {
		return false
	}
	if route.NeedsClarification {
		return false
	}
	if st.Guidance.DirectiveKind == "guided_fallback" && guidance.Sniff(message).GuidanceAcceptance {
		return false
	}
	if route.TaskIntent != "fortune_followup" {
		return false
	}
	return hasReusableResultForDomain(st, route.PrimaryDomain)
}

func hasReusableResultForDomain(st *state.SessionState, domain string) bool {
	switch domain {
	case "qimen":
		return st.HasQimenResult()
	case "ziwei":
		return st.HasZiWeiResult()
	default:
		return st.HasBaziResult()
	}
}
