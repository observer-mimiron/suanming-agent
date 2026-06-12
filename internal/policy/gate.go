package policy

import (
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// Phase 1 constants.
const (
	confidenceThreshold = 0.6
)

// phase1Allowlist is the set of domains permitted in phase 1.
var phase1Allowlist = map[string]bool{
	"bazi":  true,
	"qimen": true,
}

// ApprovedRoute is the policy-gate-approved execution route.
type ApprovedRoute struct {
	ConversationIntent    string
	PrimaryDomain         string
	SecondaryDomains      []string
	TaskIntent            string
	NeedsClarification    bool
	ClarificationQuestion string
	ParallelAllowed       bool
	Slots                 schemas.DecisionSlots
	PolicyHints           schemas.PolicyHints
}

// Apply validates a supervisor decision against phase-1 policy rules.
func Apply(decision schemas.SupervisorDecision, st *state.SessionState) ApprovedRoute {
	route := ApprovedRoute{
		ConversationIntent:    decision.ConversationIntent,
		PrimaryDomain:         decision.PrimaryDomain,
		SecondaryDomains:      decision.SecondaryDomains,
		TaskIntent:            decision.TaskIntent,
		NeedsClarification:    decision.NeedsClarification,
		ClarificationQuestion: decision.ClarificationQuestion,
		ParallelAllowed:       false, // hard-disabled in phase 1
		Slots:                 decision.Slots,
		PolicyHints:           decision.PolicyHints,
	}

	// 1. Phase-1 domain allowlist: drop unsupported domains.
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

	// 2. Hard-disable parallel execution.
	route.ParallelAllowed = false

	// 3. Low confidence forces clarification.
	if decision.Confidence < confidenceThreshold && !route.NeedsClarification {
		route.NeedsClarification = true
		if route.ClarificationQuestion == "" {
			route.ClarificationQuestion = "请确认一下您的需求，我再为您详细分析。"
		}
	}

	// 4. Incomplete profile forces clarification or collect_profile.
	profileReady := st.IsProfileComplete() || st.HasBaziResult()
	if !profileReady && route.TaskIntent != "collect_profile" && route.TaskIntent != "amend_profile" && route.TaskIntent != "direct_bazi" {
		if !route.NeedsClarification {
			route.NeedsClarification = true
			route.ClarificationQuestion = "请提供您的出生信息（年份、月份、日期、时辰、性别），我来为您排盘分析。"
		}
	}

	// 5. Qimen can serve as primary only for timing-oriented tasks.
	if route.PrimaryDomain == "qimen" && route.TaskIntent != "timing_followup" && route.TaskIntent != "cross_domain_consult" {
		// Downgrade: qimen primary without timing intent → bazi primary, qimen secondary.
		route.PrimaryDomain = "bazi"
		route.TaskIntent = "collect_profile"
		if !hasDomain(route.SecondaryDomains, "qimen") {
			route.SecondaryDomains = append(route.SecondaryDomains, "qimen")
		}
	}

	return route
}

func hasDomain(domains []string, target string) bool {
	for _, d := range domains {
		if d == target {
			return true
		}
	}
	return false
}
