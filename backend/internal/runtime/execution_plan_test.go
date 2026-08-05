// This test file belongs to the manager-owned runtime layer.
// It verifies ExecutionPlan behavior and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"reflect"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/schemas"
)

func TestSelectDomains_UsesPrimaryOnlyByDefault(t *testing.T) {
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "none",
		},
	}

	got := selectDomains(route)
	want := []string{"bazi"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selectDomains() = %v, want %v", got, want)
	}
}

func TestRouteContractDerivation_UsesFrozenRolesAndSafety(t *testing.T) {
	cases := []struct {
		name    string
		route   policy.ApprovedRoute
		profile contracts.SafetyProfile
		steps   []contracts.DomainStep
		domains []string
	}{
		{
			name:    "period",
			route:   policy.ApprovedRoute{ConsultationKind: contracts.ConsultationKindPeriodFortune},
			profile: contracts.SafetyProfileNone,
			steps:   []contracts.DomainStep{{Domain: "bazi", Role: "primary"}, {Domain: "ziwei", Role: "support"}},
			domains: []string{"bazi", "ziwei"},
		},
		{
			name:    "event",
			route:   policy.ApprovedRoute{ConsultationKind: contracts.ConsultationKindEventQuestion},
			profile: contracts.SafetyProfileNone,
			steps:   []contracts.DomainStep{{Domain: "qimen", Role: "primary"}},
			domains: []string{"qimen"},
		},
		{
			name:    "health",
			route:   policy.ApprovedRoute{ConsultationKind: contracts.ConsultationKindHealthRisk},
			profile: contracts.SafetyProfileHealthObservation,
			steps:   []contracts.DomainStep{{Domain: "bazi", Role: "primary"}, {Domain: "ziwei", Role: "support"}},
			domains: []string{"bazi", "ziwei"},
		},
		{
			name:    "natal ziwei",
			route:   policy.ApprovedRoute{ConsultationKind: contracts.ConsultationKindNatalChart, PrimaryDomain: "ziwei"},
			profile: contracts.SafetyProfileNone,
			steps:   []contracts.DomainStep{{Domain: "ziwei", Role: "primary"}},
			domains: []string{"ziwei"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := safetyProfileForRoute(tc.route); got != tc.profile {
				t.Fatalf("safety profile = %q, want %q", got, tc.profile)
			}
			if got := domainStepsForRoute(tc.route); !reflect.DeepEqual(got, tc.steps) {
				t.Fatalf("domain steps = %+v, want %+v", got, tc.steps)
			}
			if got := selectDomains(tc.route); !reflect.DeepEqual(got, tc.domains) {
				t.Fatalf("domains = %v, want %v", got, tc.domains)
			}
		})
	}
}

func TestSelectDomains_ExplicitZiweiNatalUsesZiweiOnly(t *testing.T) {
	route := policy.ApprovedRoute{
		ConsultationKind: contracts.ConsultationKindNatalChart,
		PrimaryDomain:    "ziwei",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "none",
		},
	}

	got := selectDomains(route)
	want := []string{"ziwei"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selectDomains() = %v, want %v", got, want)
	}
}

func TestSelectDomains_AddsQimenForSupplementMode(t *testing.T) {
	route := policy.ApprovedRoute{
		PrimaryDomain:    "bazi",
		SecondaryDomains: []string{"ziwei"},
		PolicyHints: schemas.PolicyHints{
			QimenMode: "supplement",
		},
	}

	got := selectDomains(route)
	want := []string{"bazi", "qimen", "ziwei"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selectDomains() = %v, want %v", got, want)
	}
}
