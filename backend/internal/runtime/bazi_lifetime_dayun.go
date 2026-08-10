// Package runtime contains the manager-owned BaZi synthesis adapters.
//
// This file owns the isolated all-life Dayun DTO and its contract. It reads the
// accepted natal projection but never changes natal or current-period results.
package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// runLifetimeDayunSynthesis calls the dedicated all-period model contract.
func (e *Executor) runLifetimeDayunSynthesis(ctx context.Context, st *state.SessionState, chart baziCharterState, question string) (baziLifetimeDayunSynthesis, error) {
	payload := map[string]any{
		"input":            map[string]any{"dynamic_facts": buildDynamicFactsView(chart.Input), "fact_capsule": buildBaziFactCapsulePromptView(chart, true)},
		"natal_assessment": chart.StaticSynthesis,
		"runtime_catalog":  baziLifetimeRuntimeCatalogView(chart),
		"evidence_bundle":  buildEvidenceBundleView(chart.EvidenceBundle, true),
		"evidence_quality": chart.EvidenceQuality,
	}
	out, err := runBaziInnerAgentJSON[baziLifetimeDayunSynthesis](ctx, e.builder, baziLifetimeDayunSynthesisConfig(), st, buildBaziCharterPrompt("全程大运综合", question, payload))
	if err != nil {
		return baziLifetimeDayunSynthesis{}, err
	}
	if err := validateLifetimeDayunOutput(chart, out); err != nil {
		return baziLifetimeDayunSynthesis{}, err
	}
	return out, nil
}

// validateLifetimeDayunOutput requires exact coverage of the deterministic Dayun directory.
func validateLifetimeDayunOutput(chart baziCharterState, out baziLifetimeDayunSynthesis) error {
	periods := dayunPeriods(chart.Input.Dayun)
	if len(out.PeriodClaims) != len(periods) {
		return fmt.Errorf("lifetime period coverage: got %d, want %d", len(out.PeriodClaims), len(periods))
	}
	seen := map[string]bool{}
	for _, claim := range out.PeriodClaims {
		if seen[claim.PeriodRef] {
			return fmt.Errorf("lifetime period coverage: duplicate %s", claim.PeriodRef)
		}
		seen[claim.PeriodRef] = true
		periodIndex, ok := dynamicPeriodIndex(claim.PeriodRef, periods)
		if !ok {
			return fmt.Errorf("lifetime period coverage: unknown %s", claim.PeriodRef)
		}
		if len(claim.FactRefs) == 0 {
			return fmt.Errorf("lifetime period %s has no fact reference", claim.PeriodRef)
		}
		prefix := fmt.Sprintf("dayun[%d].", periodIndex)
		hasPeriodFact := false
		for _, ref := range claim.FactRefs {
			if strings.HasPrefix(string(ref), prefix) {
				hasPeriodFact = true
				break
			}
		}
		if !hasPeriodFact {
			return fmt.Errorf("lifetime period %s has no self-period fact reference", claim.PeriodRef)
		}
	}
	return nil
}

// validateLifetimeDayunSynthesis rechecks the accepted state before rendering.
func validateLifetimeDayunSynthesis(chart baziCharterState) error {
	return validateLifetimeDayunOutput(chart, chart.LifetimeSynthesis)
}

// baziLifetimeRuntimeCatalogView exposes all Dayun IDs, unlike the current-only dynamic catalog.
func baziLifetimeRuntimeCatalogView(chart baziCharterState) map[string]any {
	view := baziRuntimeCatalogViewFor(chart, buildBaziRuntimeCatalog(chart))
	view["period_refs"] = baziDynamicPeriodRefs(chart)
	return view
}
