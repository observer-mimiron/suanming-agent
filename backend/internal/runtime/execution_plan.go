// Package runtime contains the manager-owned execution flow.
//
// This file owns the ExecutionPlan contract that turns an approved route into
// exact domains, artifact requirements, follow-up policy, and debug snapshot.
package runtime

import (
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

const (
	artifactBaziChart           = "bazi_chart"
	artifactZiweiChart          = "ziwei_chart"
	artifactQimenChart          = state.AssetKindQimenCaseChart
	followupModeDirect          = "direct"
	followupModeReuseArtifact   = "reuse_artifact"
	followupModeRerunSpecialist = "rerun_specialist"
)

// ExecutionPlan 是 Manager（运行时对话 owner）从 ApprovedRoute 生成的执行合同。
// ApprovedRoute 只说明“可以做什么”；ExecutionPlan 进一步明确“本轮必须准备哪些资产、
// 调度哪些领域、追问是否可复用”，供 preflight、prefill、specialist 和 trace 共用。
type ExecutionPlan struct {
	ConsultationKind     contracts.ConsultationKind
	SafetyProfile        contracts.SafetyProfile
	DomainSteps          []contracts.DomainStep
	TurnContext          contracts.TurnContext
	Route                policy.ApprovedRoute
	Domains              []string
	Requirements         []ArtifactRequirement
	FollowupMode         string
	FollowupDirectAnswer string
	Snapshot             contracts.ExecutionSnapshot
}

// ArtifactRequirement 描述 prefill 必须满足的单个资产合同。
// OwnerRef 与 SubjectIDs 共同防止跨对象、跨资料版本或跨问事复用旧盘；
// CalendarRule 用来阻止旧历法口径的八字资产被误当作当前资产。
type ArtifactRequirement struct {
	Kind         string
	OwnerRef     state.AssetRef
	SubjectIDs   []string
	CalendarRule string
	Scope        string
	TargetAt     string
	Purpose      string
	InputRefs    []state.AssetRef
}

// safetyProfileForRoute derives the deterministic output safety profile without side effects.
func safetyProfileForRoute(route policy.ApprovedRoute) contracts.SafetyProfile {
	if route.ConsultationKind == contracts.ConsultationKindHealthRisk {
		return contracts.SafetyProfileHealthObservation
	}
	return contracts.SafetyProfileNone
}

// domainStepsForRoute derives primary/support roles while keeping legacy Domains separate.
func domainStepsForRoute(route policy.ApprovedRoute) []contracts.DomainStep {
	switch route.ConsultationKind {
	case contracts.ConsultationKindPeriodFortune, contracts.ConsultationKindHealthRisk:
		return []contracts.DomainStep{{Domain: "bazi", Role: "primary"}, {Domain: "ziwei", Role: "support"}}
	case contracts.ConsultationKindEventQuestion:
		return []contracts.DomainStep{{Domain: "qimen", Role: "primary"}}
	case contracts.ConsultationKindNatalChart:
		domain := route.PrimaryDomain
		if domain != "ziwei" {
			domain = "bazi"
		}
		return []contracts.DomainStep{{Domain: domain, Role: "primary"}}
	default:
		domains := selectDomains(route)
		if len(domains) == 0 {
			return nil
		}
		primary := route.PrimaryDomain
		if primary == "" {
			primary = "bazi"
		}
		steps := make([]contracts.DomainStep, 0, len(domains))
		for _, domain := range domains {
			role := "support"
			if domain == primary {
				role = "primary"
			}
			steps = append(steps, contracts.DomainStep{Domain: domain, Role: role})
		}
		return steps
	}
}

// selectDomains 将批准路由压缩成本轮实际调度的领域顺序。
// 紫微本命需要八字资料作底层资产，奇门则只在 gate 明确 primary/supplement 时加入。
func selectDomains(route policy.ApprovedRoute) []string {
	switch route.ConsultationKind {
	case contracts.ConsultationKindPeriodFortune, contracts.ConsultationKindHealthRisk:
		return []string{"bazi", "ziwei"}
	case contracts.ConsultationKindEventQuestion:
		return []string{"qimen"}
	case contracts.ConsultationKindNatalChart:
		if route.PrimaryDomain == "ziwei" {
			return []string{"ziwei"}
		}
		return []string{"bazi"}
	}

	domains := make([]string, 0, 1+len(route.SecondaryDomains))
	primary := route.PrimaryDomain
	if primary == "" {
		primary = "bazi"
	}
	domains = append(domains, primary)

	if primary == "ziwei" {
		domains = append(domains, "bazi")
	}

	if route.PolicyHints.QimenMode == "primary" || route.PolicyHints.QimenMode == "supplement" {
		domains = append(domains, "qimen")
	}

	domains = append(domains, route.SecondaryDomains...)
	return dedupeDomains(domains)
}

// dedupeDomains 保留领域首次出现顺序，并丢弃空值和重复值。
func dedupeDomains(domains []string) []string {
	if len(domains) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(domains))
	out := make([]string, 0, len(domains))
	for _, domain := range domains {
		if domain == "" || seen[domain] {
			continue
		}
		seen[domain] = true
		out = append(out, domain)
	}
	return out
}

// artifactKinds 生成面向 ExecutionSnapshot 的观测投影，不作为实际校验源。
func artifactKinds(requirements []ArtifactRequirement) []string {
	if len(requirements) == 0 {
		return nil
	}
	kinds := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		kinds = append(kinds, requirement.Kind)
	}
	return dedupeStrings(kinds)
}

// selectArtifactRequirements 根据领域列表生成 prefill 必须满足的精确资产。
// 这是无副作用的兼容入口；新执行路径应传入完整 route 和 TurnContext。
func selectArtifactRequirements(st *state.SessionState, domains []string) []ArtifactRequirement {
	return selectArtifactRequirementsForTurn(st, policy.ApprovedRoute{
		PrimaryDomain:    firstNonEmpty(domains...),
		SecondaryDomains: append([]string(nil), domains[1:]...),
	}, contracts.TurnContext{}, domains)
}

// selectArtifactRequirementsForTurn creates requirements from already-resolved
// focus and turn context. It never creates Cases or mutates SessionState.
func selectArtifactRequirementsForTurn(st *state.SessionState, route policy.ApprovedRoute, turn contracts.TurnContext, domains []string) []ArtifactRequirement {
	if st == nil {
		return nil
	}
	subjectID := subjectIDForRequirement(st)
	profileID := st.ActiveFocus.ProfileRevisionID
	requirements := make([]ArtifactRequirement, 0, len(domains))
	for _, domain := range domains {
		kind := ""
		switch domain {
		case "bazi":
			kind = artifactBaziChart
		case "qimen":
			kind = artifactQimenChart
		case "ziwei":
			kind = artifactZiweiChart
		}
		if kind == "" {
			continue
		}
		owner := state.AssetRef{Kind: state.AssetKindProfileRevision, ID: profileID}
		inputRefs := []state.AssetRef(nil)
		scope := scopeForRequirement(route, domain)
		targetAt := strings.TrimSpace(turn.TargetAt)
		purpose := string(route.ConsultationKind)
		if purpose == "" {
			purpose = firstNonEmpty(route.TaskIntent, "natal_chart")
		}
		if kind == artifactQimenChart {
			caseID := strings.TrimSpace(turn.CaseID)
			owner = state.AssetRef{Kind: "case", ID: caseID}
			scope = "instant"
			targetAt = strings.TrimSpace(turn.QuestionTime)
			purpose = "event_question"
			if caseID != "" {
				inputRefs = []state.AssetRef{{Kind: "case", ID: caseID}}
			}
		} else if profileID != "" {
			inputRefs = []state.AssetRef{{Kind: state.AssetKindProfileRevision, ID: profileID}}
		}
		subjectIDs := []string(nil)
		if subjectID != "" {
			subjectIDs = []string{subjectID}
		}
		requirements = append(requirements, ArtifactRequirement{
			Kind:         kind,
			OwnerRef:     owner,
			SubjectIDs:   subjectIDs,
			CalendarRule: calendarRuleForArtifact(kind),
			Scope:        scope,
			TargetAt:     targetAt,
			Purpose:      purpose,
			InputRefs:    inputRefs,
		})
	}
	return requirements
}

// subjectIDForRequirement reads the resolved subject pointer without invoking
// ActiveSubject, because requirement construction must not mutate session state.
func subjectIDForRequirement(st *state.SessionState) string {
	if st == nil {
		return ""
	}
	if len(st.ActiveFocus.SubjectIDs) > 0 {
		return strings.TrimSpace(st.ActiveFocus.SubjectIDs[0])
	}
	for _, subject := range st.Subjects {
		if strings.TrimSpace(subject.Display) == "自己" {
			return subject.ID
		}
	}
	if len(st.Subjects) > 0 {
		return st.Subjects[0].ID
	}
	return ""
}

// scopeForRequirement maps consultation time semantics to deterministic facts.
func scopeForRequirement(route policy.ApprovedRoute, domain string) string {
	if route.ConsultationKind == contracts.ConsultationKindNatalChart || domain == "qimen" {
		return "none"
	}
	if route.ConsultationKind != contracts.ConsultationKindPeriodFortune &&
		route.ConsultationKind != contracts.ConsultationKindHealthRisk {
		return "none"
	}
	timeScope := strings.ToLower(strings.TrimSpace(route.Slots.TimeScope))
	if strings.Contains(timeScope, "月") || strings.Contains(timeScope, "month") {
		return "liuyue"
	}
	return "liunian"
}

// calendarRuleForArtifact 返回资产合同需要固定的历法版本。
// 目前只有八字盘需要强校验历法口径，其它领域暂不带 calendar rule。
func calendarRuleForArtifact(kind string) string {
	if kind == artifactBaziChart {
		return currentBaziCalendarRule()
	}
	return ""
}

// dedupeStrings 保留字符串首次出现顺序，并丢弃空值和重复值。
func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
