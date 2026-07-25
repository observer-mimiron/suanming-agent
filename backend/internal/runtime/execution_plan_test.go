package runtime

import (
	"reflect"
	"testing"

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

func TestSelectDomains_AddsBaziForZiweiPrimary(t *testing.T) {
	route := policy.ApprovedRoute{
		PrimaryDomain: "ziwei",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "none",
		},
	}

	got := selectDomains(route)
	want := []string{"ziwei", "bazi"}
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
