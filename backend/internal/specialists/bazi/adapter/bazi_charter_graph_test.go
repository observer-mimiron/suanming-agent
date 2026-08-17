package adapter

import (
	"context"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

func TestAnnotateBaziSynthesisSourcesUsesTierFactGrounds(t *testing.T) {
	grounded := baziTierDimension{State: "usable", FactRefs: []baziFactRef{"fact_capsule.month_command"}}
	state := baziCharterState{StaticSynthesis: baziStaticSynthesis{
		Source: "model",
		TierAssessment: baziTierAssessment{
			Status: "rated", Level: 7, Confidence: "明确成立",
			Dimensions: baziTierDimensions{
				MainAxis: grounded, YouQing: grounded, YouLi: grounded, QingZhuo: grounded,
				Disease: baziTierDimension{State: "light", FactRefs: []baziFactRef{"fact_capsule.month_command"}},
				Remedy:  grounded, Rescue: grounded, Tiaohou: grounded, HeZhiZhang: grounded,
			},
		},
	}}
	ctx, root := tracing.NewRealTracer(nil).StartTrace(context.Background(), "bazi-tier-trace")
	defer root.End()

	annotateBaziSynthesisSources(ctx, state)
	attrs := tracing.TraceFromContext(ctx).Attributes
	if attrs["bazi.tier.evidence_complete"] != true || attrs["bazi.tier.evidence_missing"] != "" {
		t.Fatalf("tier trace = %#v, want fact-grounded completion without knowledge retrieval", attrs)
	}
}
