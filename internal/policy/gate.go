// Package policy 实现策略校验门控，对 supervisor 的决策结果应用阶段约束和业务规则。
// 负责领域白名单过滤、并行执行开关、低置信度强制澄清、资料完整性校验、奇门降级等。
package policy

import (
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// 阶段一的常量定义。
const (
	confidenceThreshold = 0.6
)

// phase1Allowlist 是阶段一允许执行的领域集合。
var phase1Allowlist = map[string]bool{
	"bazi":  true,
	"qimen": true,
}

// ApprovedRoute 是经策略门控批准的执行路线，包含领域、任务意图、槽位和策略提示。
type ApprovedRoute struct {
	ConversationIntent    string
	PrimaryDomain         string
	SecondaryDomains      []string
	TaskIntent            string
	Confidence            float64
	NeedsClarification    bool
	ClarificationQuestion string
	ParallelAllowed       bool
	Slots                 schemas.DecisionSlots
	PolicyHints           schemas.PolicyHints
}

// Apply 对 supervisor 的决策进行阶段一策略校验并返回批准后的执行路线。
func Apply(decision schemas.SupervisorDecision, st *state.SessionState) ApprovedRoute {
	route := ApprovedRoute{
		ConversationIntent:    decision.ConversationIntent,
		PrimaryDomain:         decision.PrimaryDomain,
		SecondaryDomains:      decision.SecondaryDomains,
		TaskIntent:            decision.TaskIntent,
		Confidence:            decision.Confidence,
		NeedsClarification:    decision.NeedsClarification,
		ClarificationQuestion: decision.ClarificationQuestion,
		ParallelAllowed:       false, // 阶段一强制禁用并行
		Slots:                 decision.Slots,
		PolicyHints:           decision.PolicyHints,
	}

	applyPhase1Allowlist(&route)
	applyParallelHardDisable(&route)
	applyConfidenceClarification(&route, decision.Confidence)
	applyProfileClarification(&route, st)
	applyQimenRouting(&route)

	return route
}

// applyPhase1Allowlist 阶段一领域白名单过滤，不支持的领域降级为 bazi。
func applyPhase1Allowlist(route *ApprovedRoute) {
	if !phase1Allowlist[route.PrimaryDomain] {
		route.PrimaryDomain = "bazi"
		route.TaskIntent = "collect_profile"
	}
	filtered := make([]string, 0, len(route.SecondaryDomains))
	for _, d := range route.SecondaryDomains {
		if phase1Allowlist[d] {
			filtered = append(filtered, d)
		}
	}
	route.SecondaryDomains = filtered
}

// applyParallelHardDisable 阶段一强制禁用并行执行。
func applyParallelHardDisable(route *ApprovedRoute) {
	route.ParallelAllowed = false
}

// applyConfidenceClarification 低置信度强制要求澄清。
func applyConfidenceClarification(route *ApprovedRoute, confidence float64) {
	if confidence < confidenceThreshold && !route.NeedsClarification {
		route.NeedsClarification = true
		if route.ClarificationQuestion == "" {
			route.ClarificationQuestion = "请确认一下您的需求，我再为您详细分析。"
		}
	}
}

// applyProfileClarification 资料不完整时强制澄清或转为收集资料。
func applyProfileClarification(route *ApprovedRoute, st *state.SessionState) {
	profileReady := st.IsProfileComplete() || st.HasBaziResult()
	if !profileReady && route.TaskIntent != "collect_profile" && route.TaskIntent != "amend_profile" && route.TaskIntent != "direct_bazi" {
		if !allowsProfilelessQimenPrimary(*route) && !route.NeedsClarification {
			route.NeedsClarification = true
			route.ClarificationQuestion = "请提供您的出生信息（年份、月份、日期、时辰、性别），我来为您排盘分析。"
		}
	}
}

// applyQimenRouting 奇门主域路由：仅在明确批准时作为主域，否则降级为 bazi。
func applyQimenRouting(route *ApprovedRoute) {
	if route.PrimaryDomain == "qimen" && !wantsQimenPrimary(*route) {
		route.PrimaryDomain = "bazi"
		route.TaskIntent = "collect_profile"
		if !hasDomain(route.SecondaryDomains, "qimen") {
			route.SecondaryDomains = append(route.SecondaryDomains, "qimen")
		}
	}
}


func hasDomain(domains []string, target string) bool {
	for _, d := range domains {
		if d == target {
			return true
		}
	}
	return false
}

func wantsQimenPrimary(route ApprovedRoute) bool {
	if route.PrimaryDomain != "qimen" {
		return false
	}
	if route.PolicyHints.QimenMode == "primary" {
		return true
	}
	switch route.TaskIntent {
	case "timing_followup", "cross_domain_consult":
		return true
	}
	return false
}

func allowsProfilelessQimenPrimary(route ApprovedRoute) bool {
	if !wantsQimenPrimary(route) {
		return false
	}
	return route.PolicyHints.ProfileRequirement != "full"
}
