package domain

import "testing"

func TestValidateDynamicAgainstProfileScopeAllowsConcreteDomainWording(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{
			BaziResult: map[string]any{"birthday": "1990-01-01"},
			Liunian:    map[string]any{"liunian_year": 2026},
		},
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:     "投资、疾病和官非等领域只作趋势观察，不作确定保证。",
			ConsistencyFlags: []string{"仅作结构观察"},
		},
	}
	if err := validateDynamicAgainstProfileScope(state); err != nil {
		t.Fatalf("concrete domain wording should be allowed for adults: %v", err)
	}
}

func TestValidateDynamicAssertionsAllowsConcreteDomainWording(t *testing.T) {
	assertions := []baziAssertion{{
		ID:      "dynamic.dayun.0",
		Kind:    baziAssertionDayunPeriod,
		Verdict: "投资、疾病和官非等领域只作趋势观察。",
	}}
	if err := validateBaziAssertions(baziCharterState{}, assertions); err != nil {
		t.Fatalf("dynamic trend wording should be allowed: %v", err)
	}
}
