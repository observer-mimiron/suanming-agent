package runtime

import (
	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

const (
	artifactBaziChart           = "bazi_chart"
	artifactZiweiChart          = "ziwei_chart"
	artifactQimenChart          = "qimen_chart"
	followupModeDirect          = "direct"
	followupModeReuseArtifact   = "reuse_artifact"
	followupModeRerunSpecialist = "rerun_specialist"
)

// ExecutionPlan is the manager-owned execution contract derived from an approved route.
type ExecutionPlan struct {
	Route                policy.ApprovedRoute
	Domains              []string
	Requirements         []ArtifactRequirement
	FollowupMode         string
	FollowupDirectAnswer string
	Snapshot             contracts.ExecutionSnapshot
}

// ArtifactRequirement names one exact owner and subject set that prefill must satisfy.
type ArtifactRequirement struct {
	Kind         string
	OwnerRef     state.AssetRef
	SubjectIDs   []string
	CalendarRule string
}

func selectDomains(route policy.ApprovedRoute) []string {
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

func selectArtifactRequirements(st *state.SessionState, domains []string) []ArtifactRequirement {
	if st == nil {
		return nil
	}
	subject := st.ActiveSubject()
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
		if kind == artifactQimenChart {
			item := st.StartCase("qimen", "", false)
			owner = state.AssetRef{Kind: "case", ID: item.ID}
		}
		requirements = append(requirements, ArtifactRequirement{
			Kind:         kind,
			OwnerRef:     owner,
			SubjectIDs:   []string{subject.ID},
			CalendarRule: calendarRuleForArtifact(kind),
		})
	}
	return requirements
}

func calendarRuleForArtifact(kind string) string {
	if kind == artifactBaziChart {
		return "zi_zheng_v1"
	}
	return ""
}

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
