// Package adapter owns BaZi's structured model contracts and runtime capability bindings.
//
// This file registers only BaZi schemas with the shared structured decoder.
// It does not own the decoder implementation, Manager flow, or SSE transport.
package adapter

import (
	_ "embed"
	"fmt"

	"github.com/observer-mimiron/suanming-agent/internal/structured"
)

//go:embed schemas/bazi-analysis-plan.schema.json
var baziAnalysisPlanSchema []byte

//go:embed schemas/bazi-evidence-plan.schema.json
var baziEvidencePlanSchema []byte

//go:embed schemas/bazi-static-synthesis.schema.json
var baziStaticSynthesisSchema []byte

//go:embed schemas/bazi-dynamic-synthesis.schema.json
var baziDynamicSynthesisSchema []byte

//go:embed schemas/bazi-lifetime-dayun-synthesis.schema.json
var baziLifetimeDayunSynthesisSchema []byte

const (
	structuredSchemaBaziAnalysisPlan           = "bazi.analysis_plan"
	structuredSchemaBaziEvidencePlan           = "bazi.evidence_plan"
	structuredSchemaBaziStaticSynthesis        = "bazi.static_synthesis"
	structuredSchemaBaziDynamicSynthesis       = "bazi.dynamic_synthesis"
	structuredSchemaBaziLifetimeDayunSynthesis = "bazi.lifetime_dayun_synthesis"
)

func init() {
	for _, schema := range []struct {
		name string
		raw  []byte
	}{
		{structuredSchemaBaziAnalysisPlan, baziAnalysisPlanSchema},
		{structuredSchemaBaziEvidencePlan, baziEvidencePlanSchema},
		{structuredSchemaBaziStaticSynthesis, baziStaticSynthesisSchema},
		{structuredSchemaBaziDynamicSynthesis, baziDynamicSynthesisSchema},
		{structuredSchemaBaziLifetimeDayunSynthesis, baziLifetimeDayunSynthesisSchema},
	} {
		if err := structured.RegisterJSON(schema.name, schema.raw); err != nil {
			panic(err)
		}
	}
}

// decodeStructuredOutput validates a raw model result against the BaZi-owned schema.
func decodeStructuredOutput(name, raw string, target any) error {
	if err := structured.Decode(name, raw, target); err != nil {
		return err
	}
	switch value := target.(type) {
	case *baziStructuredDynamicSynthesis:
		for index, claim := range value.PeriodClaims {
			if err := validateStructuredClaimConfidence(name, fmt.Sprintf("period_claims[%d]", index), claim.Confidence); err != nil {
				return err
			}
		}
		return validateStructuredClaimConfidence(name, "liunian_claim", value.LiunianClaim.Confidence)
	case *baziLifetimeDayunSynthesis:
		for index, claim := range value.PeriodClaims {
			if err := validateStructuredClaimConfidence(name, fmt.Sprintf("period_claims[%d]", index), claim.Confidence); err != nil {
				return err
			}
		}
	case *baziAnalysisPlan:
		if value.WriterTemplate == "topic" && value.TopicMode == "" {
			return fmt.Errorf("schema_error[%s]: topic writer requires topic_mode", name)
		}
	}
	return nil
}

func validateStructuredClaimConfidence(name, field, confidence string) error {
	for _, allowed := range []string{"保守判断", "倾向成立", "明确成立"} {
		if confidence == allowed {
			return nil
		}
	}
	return fmt.Errorf("schema_error[%s]: %s.confidence is outside the closed model enum", name, field)
}
