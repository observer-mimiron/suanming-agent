package runtime

import (
	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
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
	RequiredArtifacts    []string
	FollowupMode         string
	FollowupDirectAnswer string
	Snapshot             contracts.ExecutionSnapshot
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

func selectRequiredArtifacts(domains []string) []string {
	if len(domains) == 0 {
		return nil
	}
	artifacts := make([]string, 0, len(domains))
	for _, domain := range domains {
		switch domain {
		case "qimen":
			artifacts = append(artifacts, artifactQimenChart)
		case "ziwei":
			artifacts = append(artifacts, artifactZiweiChart)
		case "bazi":
			artifacts = append(artifacts, artifactBaziChart)
		}
	}
	return dedupeStrings(artifacts)
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
