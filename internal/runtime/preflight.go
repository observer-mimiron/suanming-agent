package runtime

import (
	"github.com/wikiglobal/suanming-agent/internal/guidance"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// preflightResult 描述 preflight 检查的结果。
type preflightResult struct {
	ShortCircuit bool                   // 是否短路
	TurnType     string                 // 转向类型标识
	Text         string                 // 用户可见的提示文本
	GuidanceNext *state.GuidanceState   // 由 executor 应用的下一个 guidance 状态
	ForcedRoute  *policy.ApprovedRoute  // guided_fallback 接受后强制路由
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
func preflight(st *state.SessionState, route policy.ApprovedRoute, message string) preflightResult {
	// 0. 旧 directive 兼容（Task 4 删除后移除）
	if route.Directive != nil {
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

	// 1. sniff + guidance 判定（code-owned 路径）
	if st.Guidance != nil || ShouldEnterGuidance(message, route, st) {
		var next *state.GuidanceState
		if st.Guidance != nil {
			// 已有 guidance → 推进
			next = policy.ReduceGuidance(policy.GuidanceReducerInput{
				Current: st.Guidance,
				Message: message,
				Profile: st.Profile,
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
		if next == nil && st.Guidance != nil && st.Guidance.DirectiveKind == "guided_fallback" {
			return preflightResult{
				ShortCircuit: true,
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
		if st.IsProfileComplete() {
			return preflightResult{}
		}
		if _, hasGender := st.Profile["gender"]; !hasGender {
			_, hasHour := st.Profile["hour"]
			_, hasDay := st.Profile["day"]
			if hasHour || hasDay {
				return preflightResult{
					ShortCircuit: true,
					TurnType:     "clarification",
					Text:         guidance.Render(guidance.Request{Boundary: guidance.BoundaryCollectGenderFromBirthTime}),
				}
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

	profileComplete := st.IsProfileComplete()

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

	// 7. 放行
	return preflightResult{}
}
