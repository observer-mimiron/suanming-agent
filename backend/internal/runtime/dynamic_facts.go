// Package runtime contains the manager-owned execution flow.
//
// This file projects deterministic Prefill capability into a per-turn dynamic
// facts contract. It does not persist dynamic facts or generate narrative claims.
package runtime

import (
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/state"
)

const (
	dynamicFactsStatusReady       = "ready"
	dynamicFactsStatusDegraded    = "degraded"
	dynamicFactsStatusUnavailable = "unavailable"
)

// DynamicFacts describes deterministic facts available for one requested time scope.
// It is a per-turn Prefill result, not a persisted DomainAsset and not specialist output.
type DynamicFacts struct {
	Scope    string         `json:"scope"`
	TargetAt string         `json:"target_at"`
	Status   string         `json:"status"`
	Facts    map[string]any `json:"facts"`
}

// dynamicFactsForPlan derives one capability result for each dynamic requirement.
// A missing implementation is explicit so downstream models cannot fill the gap by guessing.
func dynamicFactsForPlan(st *state.SessionState, plan ExecutionPlan) []DynamicFacts {
	results := make([]DynamicFacts, 0, len(plan.Requirements))
	for _, requirement := range plan.Requirements {
		if requirement.Scope == "" || requirement.Scope == "none" || requirement.Kind == artifactQimenChart {
			continue
		}
		results = append(results, dynamicFactsForRequirement(st, requirement))
	}
	return results
}

// dynamicFactsForRequirement reports ready only when the deterministic payload
// matches the plan's target. Unimplemented liuyue remains unavailable.
func dynamicFactsForRequirement(st *state.SessionState, requirement ArtifactRequirement) DynamicFacts {
	result := DynamicFacts{
		Scope:    requirement.Scope,
		TargetAt: strings.TrimSpace(requirement.TargetAt),
		Status:   dynamicFactsStatusUnavailable,
		Facts:    map[string]any{},
	}
	if st == nil || result.TargetAt == "" {
		return result
	}
	if requirement.Scope == "liuyue" {
		return result
	}

	var raw map[string]any
	switch requirement.Kind {
	case artifactBaziChart:
		raw = dynamicBaziFacts(st, requirement.Scope)
	case artifactZiweiChart:
		raw = dynamicZiweiFacts(st, requirement.Scope)
	}
	if len(raw) == 0 || !dynamicFactsTargetMatches(raw, requirement) {
		return result
	}
	raw["domain"] = strings.TrimSuffix(strings.TrimSuffix(requirement.Kind, "_chart"), "_case")
	result.Status = dynamicFactsStatusReady
	result.Facts = raw
	return result
}

// dynamicBaziFacts selects only the deterministic dynamic payload needed by the plan.
func dynamicBaziFacts(st *state.SessionState, scope string) map[string]any {
	if st == nil || st.BaziResult == nil {
		return nil
	}
	if scope == "dayun" {
		if value, ok := st.BaziResult["dayun_analyzed"].(map[string]any); ok {
			return copyAnyMap(value)
		}
		if value, ok := st.BaziResult["dayun"].(map[string]any); ok {
			return copyAnyMap(value)
		}
		return nil
	}
	value, _ := st.BaziResult["liunian"].(map[string]any)
	return copyAnyMap(value)
}

// dynamicZiweiFacts selects the deterministic Zi Wei dynamic payload for a plan.
func dynamicZiweiFacts(st *state.SessionState, scope string) map[string]any {
	if st == nil || st.ZiWeiResult == nil || scope == "dayun" {
		return nil
	}
	value, _ := st.ZiWeiResult["liunian"].(map[string]any)
	return copyAnyMap(value)
}

// dynamicFactsTargetMatches prevents an arbitrary cached dynamic payload from being
// reported as ready for another target time or year.
func dynamicFactsTargetMatches(facts map[string]any, requirement ArtifactRequirement) bool {
	target, ok := parseTurnTime(requirement.TargetAt)
	if !ok {
		return false
	}
	if requirement.Kind == artifactBaziChart {
		return stringValue(facts["liunian_target_at"]) == target.Format("2006-01-02 15:04:05")
	}
	return intValue(facts["year"]) == target.Year()
}

// dynamicFactsNotice returns a fixed user-visible explanation for unavailable data.
func dynamicFactsNotice(facts []DynamicFacts) string {
	for _, item := range facts {
		if item.Status != dynamicFactsStatusUnavailable && item.Status != dynamicFactsStatusDegraded {
			continue
		}
		label := map[string]string{"dayun": "大运", "liunian": "流年", "liuyue": "流月"}[item.Scope]
		if label == "" {
			label = item.Scope
		}
		return "补充说明：本轮" + label + "确定性资料当前不可用，系统未由模型补算，只按已就绪的结构化事实作保守参考。"
	}
	return ""
}

// appendDynamicFactsNotice appends the capability boundary without changing the reading itself.
func appendDynamicFactsNotice(text string, facts []DynamicFacts) string {
	notice := dynamicFactsNotice(facts)
	if notice == "" || strings.Contains(text, notice) {
		return text
	}
	if strings.TrimSpace(text) == "" {
		return notice
	}
	return strings.TrimSpace(text) + "\n\n" + notice
}

// dynamicFactsFromResult reads the runtime-private projection attached to a specialist result.
func dynamicFactsFromResult(result map[string]any) []DynamicFacts {
	if result == nil {
		return nil
	}
	items, _ := result["dynamic_facts"].([]DynamicFacts)
	return items
}

// attachDynamicFacts preserves existing specialist metadata while adding the Prefill projection.
func attachDynamicFacts(result map[string]any, facts []DynamicFacts) map[string]any {
	if len(facts) == 0 {
		return result
	}
	patch := make(map[string]any, len(result)+1)
	for key, value := range result {
		patch[key] = value
	}
	patch["dynamic_facts"] = append([]DynamicFacts(nil), facts...)
	return patch
}
