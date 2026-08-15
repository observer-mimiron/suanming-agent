package domain

import (
	"strings"
	"testing"
)

func TestTierAssessmentJudgmentUsesStatusSpecificWording(t *testing.T) {
	tests := []struct {
		name       string
		assessment TierAssessment
		want       string
	}{
		{name: "rated", assessment: TierAssessment{Status: "rated", Level: 6}, want: "格局评价已定"},
		{name: "provisional", assessment: TierAssessment{Status: "provisional", Level: 6}, want: "格局判断暂定"},
		{name: "withheld", assessment: TierAssessment{Status: "withheld"}, want: "格局暂不立评（仅作结构观察）"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TierAssessmentJudgment(tt.assessment); got != tt.want {
				t.Fatalf("TierAssessmentJudgment() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTierAssessmentBasisExplainsProvisionalDimensions(t *testing.T) {
	basis := TierAssessmentBasis(TierAssessment{
		Status: "provisional",
		Dimensions: TierDimensions{
			QingZhuo: TierDimension{State: "missing"},
			Disease:  TierDimension{State: "unresolved"},
			Rescue:   TierDimension{State: "limited"},
		},
	})
	for _, want := range []string{"清浊证据缺位", "病药关系未明", "救应条件受限"} {
		if !strings.Contains(basis, want) {
			t.Fatalf("TierAssessmentBasis() = %q, missing %q", basis, want)
		}
	}
}

func TestTierAssessmentBasisPrioritizesMissingBeforeMixed(t *testing.T) {
	basis := TierAssessmentBasis(TierAssessment{
		Status: "provisional",
		Dimensions: TierDimensions{
			YouQing:  TierDimension{State: "mixed"},
			YouLi:    TierDimension{State: "limited"},
			QingZhuo: TierDimension{State: "missing"},
			Disease:  TierDimension{State: "unresolved"},
			Rescue:   TierDimension{State: "limited"},
		},
	})
	for _, want := range []string{"清浊证据缺位", "病药关系未明", "有力条件受限", "救应条件受限"} {
		if !strings.Contains(basis, want) {
			t.Fatalf("TierAssessmentBasis() = %q, missing %q", basis, want)
		}
	}
	if strings.Contains(basis, "有情条件并见") {
		t.Fatalf("TierAssessmentBasis() prioritized a mixed state: %q", basis)
	}
}
