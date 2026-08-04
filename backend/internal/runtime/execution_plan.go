// Package runtime contains the manager-owned execution flow.
//
// This file owns the ExecutionPlan contract that turns an approved route into
// exact domains, artifact requirements, follow-up policy, and debug snapshot.
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

// ExecutionPlan 是 Manager（运行时对话 owner）从 ApprovedRoute 生成的执行合同。
// ApprovedRoute 只说明“可以做什么”；ExecutionPlan 进一步明确“本轮必须准备哪些资产、
// 调度哪些领域、追问是否可复用”，供 preflight、prefill、specialist 和 trace 共用。
type ExecutionPlan struct {
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
}

// selectDomains 将批准路由压缩成本轮实际调度的领域顺序。
// 紫微本命需要八字资料作底层资产，奇门则只在 gate 明确 primary/supplement 时加入。
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
// 奇门资产按问事 Case 归属，这里集中创建 Case，避免 prefill 和 specialist
// 各自暗自决定 owner，导致同一轮问事写入不同资产命名空间。
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
