package domain

import "testing"

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
