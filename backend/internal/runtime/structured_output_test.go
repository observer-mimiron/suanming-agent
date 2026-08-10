// This test file protects the manager-owned structured-output boundary.
package runtime

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestStructuredOutputPromptContractUsesRegisteredSchema(t *testing.T) {
	contract, err := structuredOutputPromptContract(structuredSchemaBaziAnalysisPlan)
	if err != nil {
		t.Fatalf("prompt contract: %v", err)
	}
	if !strings.Contains(contract, structuredSchemaBaziAnalysisPlan) || !strings.Contains(contract, "additionalProperties") {
		t.Fatalf("prompt contract must contain registered schema, got %q", contract)
	}
	if !strings.Contains(contract, "http://json-schema.org/draft-07/schema#") {
		t.Fatalf("prompt contract must contain Draft-07 schema metadata")
	}
	if _, err := structuredSchemaHash(structuredSchemaBaziAnalysisPlan); err != nil {
		t.Fatalf("schema hash: %v", err)
	}
}

func TestEachBaziSchemaFileIsInjectedIntoItsMatchingPrompt(t *testing.T) {
	cases := []struct {
		name string
		raw  []byte
	}{
		{structuredSchemaBaziAnalysisPlan, baziAnalysisPlanSchema},
		{structuredSchemaBaziEvidencePlan, baziEvidencePlanSchema},
		{structuredSchemaBaziStaticSynthesis, baziStaticSynthesisSchema},
		{structuredSchemaBaziDynamicSynthesis, baziDynamicSynthesisSchema},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			contract, err := structuredOutputPromptContract(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(contract, string(tc.raw)) {
				t.Fatal("prompt did not receive the exact registered schema file")
			}
			if _, err := structuredSchemaHash(tc.name); err != nil {
				t.Fatalf("schema hash: %v", err)
			}
		})
	}
}

func TestBaziSchemaFieldsMatchRawModelDTOs(t *testing.T) {
	cases := []struct {
		name string
		raw  []byte
		typ  reflect.Type
	}{
		{structuredSchemaBaziAnalysisPlan, baziAnalysisPlanSchema, reflect.TypeOf(baziAnalysisPlan{})},
		{structuredSchemaBaziEvidencePlan, baziEvidencePlanSchema, reflect.TypeOf(baziEvidencePlan{})},
		{structuredSchemaBaziStaticSynthesis, baziStaticSynthesisSchema, reflect.TypeOf(baziStructuredStaticSynthesis{})},
		{structuredSchemaBaziDynamicSynthesis, baziDynamicSynthesisSchema, reflect.TypeOf(baziStructuredDynamicSynthesis{})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertSchemaFieldsMatchDTO(t, tc.name, tc.raw, tc.typ)
		})
	}

	assertSchemaFieldsMatchDTO(t, "static.claim", schemaDefinition(t, baziStaticSynthesisSchema, "claim"), reflect.TypeOf(baziStructuredStaticClaim{}))
	assertSchemaFieldsMatchDTO(t, "static.tier_dimension", schemaDefinition(t, baziStaticSynthesisSchema, "tier_dimension"), reflect.TypeOf(baziTierDimension{}))
	assertSchemaFieldsMatchDTO(t, "static.disease_dimension", schemaDefinition(t, baziStaticSynthesisSchema, "disease_dimension"), reflect.TypeOf(baziTierDimension{}))
	assertSchemaFieldsMatchDTO(t, "static.tier_dimensions", schemaDefinition(t, baziStaticSynthesisSchema, "tier_dimensions"), reflect.TypeOf(baziTierDimensions{}))
	assertSchemaFieldsMatchDTO(t, "static.tier_assessment", schemaDefinition(t, baziStaticSynthesisSchema, "tier_assessment"), reflect.TypeOf(baziTierAssessment{}))
	assertSchemaFieldsMatchDTO(t, "dynamic.claim", schemaDefinition(t, baziDynamicSynthesisSchema, "claim"), reflect.TypeOf(baziStructuredClaim{}))
	assertSchemaFieldsMatchDTO(t, "dynamic.period_claim", schemaDefinition(t, baziDynamicSynthesisSchema, "period_claim"), reflect.TypeOf(baziStructuredPeriodClaim{}))
	assertSchemaFieldsMatchDTO(t, "evidence.query_packet", schemaArrayItem(t, baziEvidencePlanSchema, "query_packets"), reflect.TypeOf(baziQueryPacket{}))
}

func TestBaziRawModelDTOsHaveNoRuntimeOnlyJSONFields(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(baziAnalysisPlan{}),
		reflect.TypeOf(baziEvidencePlan{}),
		reflect.TypeOf(baziStructuredStaticSynthesis{}),
		reflect.TypeOf(baziStructuredDynamicSynthesis{}),
		reflect.TypeOf(baziStructuredClaim{}),
		reflect.TypeOf(baziStructuredStaticClaim{}),
		reflect.TypeOf(baziStructuredPeriodClaim{}),
	} {
		fields := dtoJSONFields(typ)
		for _, forbidden := range []string{
			"kind", "id", "subject", "source", "recovery_reason", "field_audit", "contract_audit",
			"dayun_judgments", "dayun_path", "gan_zhi", "age", "start_at", "end_at_exclusive",
		} {
			if _, ok := fields[forbidden]; ok {
				t.Fatalf("%v exposes runtime or renderer field %q", typ, forbidden)
			}
		}
	}
}

func TestBaziConfidenceSchemaEnumMatchesGoVocabulary(t *testing.T) {
	for _, raw := range [][]byte{baziStaticSynthesisSchema, baziDynamicSynthesisSchema} {
		var document struct {
			Definitions map[string]json.RawMessage `json:"definitions"`
		}
		if err := json.Unmarshal(raw, &document); err != nil {
			t.Fatal(err)
		}
		for _, definitionName := range []string{"claim", "period_claim"} {
			definition, ok := document.Definitions[definitionName]
			if !ok {
				continue
			}
			var object struct {
				Properties map[string]struct {
					Enum []string `json:"enum"`
				} `json:"properties"`
			}
			if err := json.Unmarshal(definition, &object); err != nil {
				t.Fatal(err)
			}
			if _, ok := object.Properties["confidence"]; !ok {
				continue
			}
			if got := object.Properties["confidence"].Enum; !reflect.DeepEqual(got, baziStructuredConfidenceValues) {
				t.Fatalf("confidence enum = %#v; want %#v", got, baziStructuredConfidenceValues)
			}
		}
	}
}

func TestDecodeStructuredOutputRejectsInvalidJSONContracts(t *testing.T) {
	cases := []struct{ name, raw string }{
		{name: "missing required", raw: `{"mode":"static_full"}`},
		{name: "invalid enum", raw: `{"mode":"unknown","retrieval_stage":"static","need_dynamic":true,"focus_topics":[],"writer_template":"full","stage_summary":"x"}`},
		{name: "wrong type", raw: `{"mode":1,"retrieval_stage":"static","need_dynamic":true,"focus_topics":[],"writer_template":"full","stage_summary":"x"}`},
		{name: "unknown field", raw: `{"mode":"static_full","retrieval_stage":"static","need_dynamic":true,"focus_topics":[],"writer_template":"full","stage_summary":"x","unknown":true}`},
		{name: "trailing JSON", raw: `{"mode":"static_full","retrieval_stage":"static","need_dynamic":true,"focus_topics":[],"writer_template":"full","stage_summary":"x"} {}`},
		{name: "fence", raw: "```json\n{}\n```"},
		{name: "empty", raw: "  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out baziAnalysisPlan
			if err := decodeStructuredOutput(structuredSchemaBaziAnalysisPlan, tc.raw, &out); err == nil {
				t.Fatal("expected schema error")
			}
		})
	}
}

func TestStaticAndDynamicModelSchemasRejectRendererAndRuntimeFields(t *testing.T) {
	staticRaw := `{"claims":[],"limitations":[],"reasoning_summary":"x","reasoning_steps":["x"],"advice_boundary":"x","source":"model"}`
	var staticOutput map[string]any
	if err := decodeStructuredOutput(structuredSchemaBaziStaticSynthesis, staticRaw, &staticOutput); err == nil {
		t.Fatal("static model schema accepted runtime source field")
	}
	dynamicRaw := `{"period_claims":[],"liunian_claim":{"id":"l","kind":"liunian","subject":"liunian","verdict":"x","fact_refs":[],"relation_refs":[],"claim_refs":[],"evidence_topics":[],"confidence":"保守判断","boundary":"x"},"limitations":[],"reasoning_summary":"x","reasoning_steps":["x"],"outcome_domains":["structure"],"dayun_judgments":[]}`
	var dynamicOutput baziStructuredDynamicSynthesis
	if err := decodeStructuredOutput(structuredSchemaBaziDynamicSynthesis, dynamicRaw, &dynamicOutput); err == nil {
		t.Fatal("dynamic model schema accepted renderer dayun_judgments field")
	}
	minimalDynamic := `{"current_period_ref":"dayun[0]","current_period_realization":"maintain","period_claims":[{"period_ref":"dayun[0]","verdict":"重点运","fact_refs":[],"relation_refs":[],"claim_refs":[],"evidence_topics":[],"confidence":"倾向成立","boundary":"结构边界"}],"liunian_claim":{"verdict":"x","fact_refs":[],"relation_refs":[],"claim_refs":[],"evidence_topics":[],"confidence":"保守判断","boundary":"x"},"limitations":[],"reasoning_summary":"x","reasoning_steps":["x"],"outcome_domains":["structure"]}`
	if err := decodeStructuredOutput(structuredSchemaBaziDynamicSynthesis, minimalDynamic, &dynamicOutput); err != nil {
		t.Fatalf("minimal dynamic DTO should pass: %v", err)
	}
	selectedDynamic := minimalDynamic
	if err := decodeStructuredOutput(structuredSchemaBaziDynamicSynthesis, selectedDynamic, &dynamicOutput); err != nil {
		t.Fatalf("selected dynamic period DTO should pass: %v", err)
	}
	for _, legacy := range []string{`"id":"l"`, `"kind":"liunian"`, `"subject":"liunian"`} {
		if err := decodeStructuredOutput(structuredSchemaBaziDynamicSynthesis, strings.Replace(minimalDynamic, `"verdict":"x"`, `"verdict":"x",`+legacy, 1), &dynamicOutput); err == nil {
			t.Fatalf("dynamic model schema accepted removed positional field %s", legacy)
		}
	}
}

func TestStaticSchemaAllowsOmittedEmptyReferenceArrays(t *testing.T) {
	claim := `{"verdict":"主轴候选仍需比较","status":"candidate","fact_refs":["chart.pillars"],"evidence_topics":["geju"]}`
	dimension := `{"state":"mixed","evidence_topics":["geju"]}`
	disease := `{"state":"moderate","evidence_topics":["bingyao"]}`
	dimensions := `{"main_axis":` + dimension + `,"youqing":` + dimension + `,"youli":` + dimension + `,"qingzhuo":` + dimension + `,"disease":` + disease + `,"remedy":` + dimension + `,"rescue":` + dimension + `,"tiaohou":` + dimension + `,"hezhizhang":` + dimension + `}`
	raw := `{"claims":[` + claim + `,` + claim + `,` + claim + `,` + claim + `],"axis_status":"established","tier_assessment":{"status":"provisional","level":5,"confidence":"保守判断","dimensions":` + dimensions + `},"natal_risk_status":"withheld"}`
	var output baziStructuredStaticSynthesis
	if err := decodeStructuredOutput(structuredSchemaBaziStaticSynthesis, raw, &output); err != nil {
		t.Fatalf("compact static DTO should pass: %v", err)
	}
	if output.Claims[0].ClaimRefs != nil || output.TierAssessment.Dimensions.MainAxis.FactRefs != nil {
		t.Fatal("omitted empty reference arrays must remain omitted after decoding")
	}
	if err := decodeStructuredOutput(structuredSchemaBaziStaticSynthesis, strings.Replace(raw, `"status":"candidate"`, `"status":"candidate","relation_refs":["relation.natal.0"]`, 1), &output); err == nil {
		t.Fatal("static DTO must reject relation refs")
	}
}

func assertSchemaFieldsMatchDTO(t *testing.T, label string, raw []byte, typ reflect.Type) {
	t.Helper()
	var object struct {
		Properties           map[string]json.RawMessage `json:"properties"`
		Required             []string                   `json:"required"`
		AdditionalProperties bool                       `json:"additionalProperties"`
	}
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatalf("%s schema: %v", label, err)
	}
	if object.AdditionalProperties {
		t.Fatalf("%s schema allows additional properties", label)
	}
	dtoFields := dtoJSONFields(typ)
	if len(object.Properties) != len(dtoFields) {
		t.Fatalf("%s fields = %v; DTO fields = %v", label, object.Properties, dtoFields)
	}
	for field := range object.Properties {
		if _, ok := dtoFields[field]; !ok {
			t.Fatalf("%s schema field %q missing from DTO", label, field)
		}
	}
	for field := range dtoFields {
		if _, ok := object.Properties[field]; !ok {
			t.Fatalf("%s DTO field %q missing from schema", label, field)
		}
	}
	for _, field := range object.Required {
		if _, ok := dtoFields[field]; !ok {
			t.Fatalf("%s required field %q missing from DTO", label, field)
		}
	}
}

func dtoJSONFields(typ reflect.Type) map[string]struct{} {
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	fields := make(map[string]struct{})
	for index := 0; index < typ.NumField(); index++ {
		name := strings.Split(typ.Field(index).Tag.Get("json"), ",")[0]
		if name == "" {
			name = typ.Field(index).Name
		}
		if name != "-" {
			fields[name] = struct{}{}
		}
	}
	return fields
}

func schemaDefinition(t *testing.T, raw []byte, name string) json.RawMessage {
	t.Helper()
	var document struct {
		Definitions map[string]json.RawMessage `json:"definitions"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	definition, ok := document.Definitions[name]
	if !ok {
		t.Fatalf("schema definition %q missing", name)
	}
	return definition
}

func schemaArrayItem(t *testing.T, raw []byte, property string) json.RawMessage {
	t.Helper()
	var document struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	var field struct {
		Items json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(document.Properties[property], &field); err != nil {
		t.Fatal(err)
	}
	return field.Items
}
