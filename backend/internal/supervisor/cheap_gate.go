// This file belongs to the route approval layer.
// It owns cheap follow-up gate behavior for this package.
// It approves routes; execution contracts are built later by Manager.
package supervisor

import (
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/schemas"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

var cheapGateCrossDomainMarkers = []string{
	"全面",
	"综合",
	"一起看",
	"都看",
	"两个",
	"多领域",
}

// tryCheapFollowupRoute 在极窄的 follow-up 场景下直接复用上一轮执行合同，
// 避免每次追问都重新跑完整 supervisor LLM 路由。
//
// 设计边界：
//   - 只处理已有可复用结果的普通追问
//   - 一旦出现补资料、显式换术数、时机类问题或多域诉求，立即回退到完整路由链
func (c *Client) tryCheapFollowupRoute(msg string, st *state.SessionState) (policy.ApprovedRoute, bool) {
	trimmed := strings.TrimSpace(msg)
	if st == nil || trimmed == "" {
		return policy.ApprovedRoute{}, false
	}

	snapshot := st.Execution
	if !snapshot.HasSignal() || !snapshot.Gate.Admitted {
		return policy.ApprovedRoute{}, false
	}
	if !policy.ValidConsultationKind(snapshot.ConsultationKind) {
		return policy.ApprovedRoute{}, false
	}
	// 事件和健康轮次不能走复用捷径：事件需要为当前问题重新绑定 Case，
	// 健康需要重新应用 safety profile。只有阶段运势和单域出生盘追问可复用。
	switch snapshot.ConsultationKind {
	case contracts.ConsultationKindPeriodFortune, contracts.ConsultationKindNatalChart:
	default:
		return policy.ApprovedRoute{}, false
	}
	if snapshot.Gate.ExecutionMode != "" && snapshot.Gate.ExecutionMode != "execute" {
		return policy.ApprovedRoute{}, false
	}
	if snapshot.Gate.FollowupPolicy != "" && snapshot.Gate.FollowupPolicy != "allow" {
		return policy.ApprovedRoute{}, false
	}
	if !hasReusableResultForExecution(st, snapshot.PrimaryDomain) {
		return policy.ApprovedRoute{}, false
	}

	switch snapshot.TaskIntent {
	case "collect_profile", "amend_profile", "direct_bazi", "cross_domain_consult", "":
		return policy.ApprovedRoute{}, false
	case "interpret_chart":
		if !cheapGateAllowsInterpretChartReuse(snapshot) {
			return policy.ApprovedRoute{}, false
		}
	}

	if intent.ContainsBirthInfo(trimmed) {
		return policy.ApprovedRoute{}, false
	}
	if intent.MentionsBaziMethod(trimmed) || intent.MentionsZiweiMethod(trimmed) || intent.MentionsQimenMethod(trimmed) {
		return policy.ApprovedRoute{}, false
	}
	if isEventQuestion(trimmed) || mentionsHealthRisk(trimmed) || isNatalChartRequest(trimmed) {
		return policy.ApprovedRoute{}, false
	}
	if intent.HasTimingFocus(trimmed) || intent.ContainsTimingKeyword(trimmed) {
		return policy.ApprovedRoute{}, false
	}
	if mentionsCheapGateCrossDomain(trimmed) {
		return policy.ApprovedRoute{}, false
	}

	secondaryDomains := append([]string(nil), snapshot.SecondaryDomains...)
	route := policy.ApprovedRoute{
		ConsultationKind:   snapshot.ConsultationKind,
		ConversationIntent: firstNonEmptyLocal(snapshot.ConversationIntent, "consult"),
		PrimaryDomain:      snapshot.PrimaryDomain,
		SecondaryDomains:   secondaryDomains,
		TaskIntent:         "fortune_followup",
		Confidence:         1.0,
		Slots: schemas.DecisionSlots{
			QuestionText:  trimmed,
			TimeScope:     snapshot.TimeScope,
			TargetSubject: firstNonEmptyLocal(snapshot.TargetSubject, st.Subject),
		},
		PolicyHints: schemas.PolicyHints{
			NeedsQimen:             snapshot.QimenMode != "" && snapshot.QimenMode != "none",
			QimenMode:              firstNonEmptyLocal(snapshot.QimenMode, "none"),
			ProfileRequirement:     snapshot.Gate.ProfileRequirement,
			CanReuseSessionProfile: snapshot.Gate.ReuseSessionProfile || st.IsProfileComplete(),
			CanReuseCachedResult:   true,
		},
		Gate: snapshot.Gate,
	}
	route.Gate.Admitted = true
	route.Gate.Reason = "cheap_followup_reuse"
	route.Gate.ExecutionMode = "reuse_followup"
	route.Gate.ReuseSessionProfile = route.PolicyHints.CanReuseSessionProfile
	route.Gate.ReuseCachedResult = true
	route.Gate.AllowedDomains = append([]string{route.PrimaryDomain}, route.SecondaryDomains...)
	return route, true
}

func cheapGateAllowsInterpretChartReuse(snapshot contracts.ExecutionSnapshot) bool {
	if strings.TrimSpace(snapshot.PrimaryDomain) == "" {
		return false
	}
	if len(snapshot.SecondaryDomains) > 0 || len(snapshot.Domains) > 1 {
		return false
	}
	if snapshot.NeedsClarification {
		return false
	}
	return snapshot.Gate.ReuseCachedResult
}

func hasReusableResultForExecution(st *state.SessionState, domain string) bool {
	switch domain {
	case "qimen":
		return st.HasQimenResult()
	case "ziwei":
		return st.HasZiWeiResult()
	default:
		return st.HasBaziResult()
	}
}

func mentionsCheapGateCrossDomain(msg string) bool {
	for _, marker := range cheapGateCrossDomainMarkers {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func firstNonEmptyLocal(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
