// package adapter 包含 Manager 拥有的八字综合适配。
//
// 本文件负责全程大运 DTO 与合同，并只读取已接受的本命投影；
// 不改写本命或当前应期结果，也不拥有最终答复。
package adapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
)

// runLifetimeDayunSynthesis calls the dedicated all-period model contract.
func (e *Executor) runLifetimeDayunSynthesis(ctx context.Context, view *specialists.SessionView, chart baziCharterState, question string) (baziLifetimeDayunSynthesis, error) {
	payload := map[string]any{
		"input":            map[string]any{"dynamic_facts": buildDynamicFactsView(chart.Input), "fact_capsule": buildBaziFactCapsulePromptView(chart, true)},
		"natal_assessment": chart.StaticSynthesis,
		"runtime_catalog":  baziLifetimeRuntimeCatalogView(chart),
		"evidence_bundle":  buildEvidenceBundleView(chart.EvidenceBundle, true),
		"evidence_quality": chart.EvidenceQuality,
	}
	out, err := runBaziInnerAgentJSON[baziLifetimeDayunSynthesis](ctx, e.builder, baziLifetimeDayunSynthesisConfig(), view, buildBaziCharterPrompt("全程大运综合", question, payload))
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
	periods := bazidomain.DayunPeriods(chart.Input.Dayun)
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
	return bazidomain.LifetimeRuntimeCatalogView(chart)
}
