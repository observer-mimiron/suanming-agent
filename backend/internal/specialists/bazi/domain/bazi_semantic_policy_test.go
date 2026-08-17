package domain

import "testing"

func TestValidateBaziTierAssessmentRatedWithoutKnowledgeCoverage(t *testing.T) {
	grounded := TierDimension{State: "usable", FactRefs: []FactRef{"fact_capsule.month_command"}}
	assessment := TierAssessment{
		Status:     "rated",
		Level:      7,
		Confidence: "明确成立",
		Dimensions: TierDimensions{
			MainAxis: grounded, YouQing: grounded, YouLi: grounded, QingZhuo: grounded,
			Disease: TierDimension{State: "light", FactRefs: []FactRef{"fact_capsule.month_command"}},
			Remedy:  grounded, Rescue: grounded, Tiaohou: grounded, HeZhiZhang: grounded,
		},
	}

	if err := validateBaziTierAssessment(BaziFactCapsule{CoreFactsReady: true}, "established", assessment); err != nil {
		t.Fatalf("validateBaziTierAssessment() error = %v, want no knowledge-retrieval gate", err)
	}
}
