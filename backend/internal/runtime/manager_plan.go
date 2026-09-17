// Package runtime 包含 Manager 所有的执行主链。
//
// 本文件负责路由协调和 ExecutionPlan 构建；
// 不负责跨轮会话投影、领域规则、传输层输出或 specialist 内部推理。
package runtime

import (
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// ReconcileRoute 在 Manager 边界执行依赖会话上下文的确定性路由修正。
func (m *Manager) ReconcileRoute(st *state.SessionState, route policy.ApprovedRoute, message string) policy.ApprovedRoute {
	route = dropImplicitZiweiSupplement(route, message)
	if st == nil {
		return route
	}

	if route.TaskIntent == "collect_profile" && len(st.Profile) > 0 {
		route.TaskIntent = "amend_profile"
		route.PolicyHints.CanReuseSessionProfile = true
		if st.HasBaziResult() {
			route.PolicyHints.CanReuseCachedResult = true
		}
	}

	if route.TaskIntent == "collect_profile" && st.HasBaziResult() && !intent.ContainsBirthInfo(message) {
		route.TaskIntent = "fortune_followup"
		route.PolicyHints.CanReuseCachedResult = true
		route.PolicyHints.CanReuseSessionProfile = true
	}

	if route.TaskIntent == "interpret_chart" && st.HasBaziResult() && !intent.ContainsBirthInfo(message) {
		route.TaskIntent = "fortune_followup"
		route.PolicyHints.CanReuseCachedResult = true
		route.PolicyHints.CanReuseSessionProfile = true
	}

	return route
}

// dropImplicitZiweiSupplement 阻止普通八字建盘被模型无意扩展为跨领域综合。
func dropImplicitZiweiSupplement(route policy.ApprovedRoute, message string) policy.ApprovedRoute {
	if route.PrimaryDomain != "bazi" || intent.MentionsZiweiMethod(message) {
		return route
	}
	if !intent.ContainsBirthInfo(message) && route.TaskIntent != "collect_profile" && route.TaskIntent != "amend_profile" && route.TaskIntent != "direct_bazi" {
		return route
	}
	route.SecondaryDomains = removeExecutionDomain(route.SecondaryDomains, "ziwei")
	route.Gate.AllowedDomains = removeExecutionDomain(route.Gate.AllowedDomains, "ziwei")
	return route
}

// removeExecutionDomain removes one domain from a route list while preserving order.
func removeExecutionDomain(domains []string, target string) []string {
	out := domains[:0]
	for _, domain := range domains {
		if strings.TrimSpace(domain) == target {
			continue
		}
		out = append(out, domain)
	}
	return out
}

// BuildExecutionPlan converts an approved route into a manager-owned execution plan.
func (m *Manager) BuildExecutionPlan(st *state.SessionState, route policy.ApprovedRoute, message string) ExecutionPlan {
	return m.buildExecutionPlanWithContext(st, route, message, contracts.TurnContext{}, true)
}

// BuildExecutionPlanForTurn converts a route with the already-captured turn context.
func (m *Manager) BuildExecutionPlanForTurn(st *state.SessionState, route policy.ApprovedRoute, message string, turn contracts.TurnContext) ExecutionPlan {
	return m.buildExecutionPlanWithContext(st, route, message, turn, true)
}

// buildExecutionPlan converts an approved route into an execution plan for existing package callers.
func (m *Manager) buildExecutionPlan(st *state.SessionState, route policy.ApprovedRoute, message string, resolveFocus bool) ExecutionPlan {
	return m.buildExecutionPlanWithContext(st, route, message, contracts.TurnContext{}, resolveFocus)
}

// buildExecutionPlanWithContext builds one plan without allowing graph nodes to create a second Case.
func (m *Manager) buildExecutionPlanWithContext(st *state.SessionState, route policy.ApprovedRoute, message string, turn contracts.TurnContext, resolveFocus bool) ExecutionPlan {
	if resolveFocus {
		route = resolveArtifactFocus(st, route, message)
	}
	route = m.ReconcileRoute(st, route, message)
	turn = m.ensureTurnContext(st, route, message, turn)
	domains := selectDomains(route)
	requirements := selectArtifactRequirementsForTurn(st, route, turn, domains)
	requiredArtifactKinds := artifactKinds(requirements)
	followupMode, directAnswer := resolveFollowupPolicy(st, route, message)
	if followupMode == followupModeRerunSpecialist {
		if text, ok := maybeReuseFollowupArtifact(m, st, route, domains, message); ok {
			followupMode = followupModeReuseArtifact
			directAnswer = text
		}
	}
	return ExecutionPlan{
		ConsultationKind:     route.ConsultationKind,
		SafetyProfile:        safetyProfileForRoute(route),
		DomainSteps:          domainStepsForRoute(route),
		TurnContext:          turn,
		Route:                route,
		Domains:              domains,
		Requirements:         requirements,
		FollowupMode:         followupMode,
		FollowupDirectAnswer: directAnswer,
		Snapshot: contracts.ExecutionSnapshot{
			ConsultationKind:   route.ConsultationKind,
			SafetyProfile:      safetyProfileForRoute(route),
			DomainSteps:        append([]contracts.DomainStep(nil), domainStepsForRoute(route)...),
			TurnContext:        turn,
			PrimaryDomain:      route.PrimaryDomain,
			SecondaryDomains:   append([]string(nil), route.SecondaryDomains...),
			Domains:            append([]string(nil), domains...),
			TaskIntent:         route.TaskIntent,
			ConversationIntent: route.ConversationIntent,
			RequiredArtifacts:  append([]string(nil), requiredArtifactKinds...),
			FollowupMode:       followupMode,
			NeedsClarification: route.NeedsClarification,
			QimenMode:          route.PolicyHints.QimenMode,
			TargetSubject:      route.Slots.TargetSubject,
			TimeScope:          route.Slots.TimeScope,
			Gate:               route.Gate,
		},
	}
}
