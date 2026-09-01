package adapter

import "testing"

func TestDecodeStructuredOutputAcceptsBoundedEvidencePlan(t *testing.T) {
	var output baziEvidencePlan
	raw := `{"need_retrieval":true,"allow_reflection":true,"stage":"static","evidence_gaps":["格局解释参考"],"recommended_sources":["子平真诠"],"query_packets":[{"topic":"geju","query":"子平真诠 食神制杀 月令","preferred_sources":["子平真诠"],"source_tier":"A"},{"topic":"tiaohou","query":"穷通宝鉴 丙火 亥月 调候","preferred_sources":["穷通宝鉴"],"source_tier":"A"}]}`
	if err := decodeStructuredOutput(structuredSchemaBaziEvidencePlan, raw, &output); err != nil {
		t.Fatalf("decodeStructuredOutput() error = %v", err)
	}
	if len(output.QueryPackets) != baziEvidenceInitialQueryBudget || !output.AllowReflection {
		t.Fatalf("evidence plan = %#v", output)
	}
}

func TestDecodeStructuredOutputRejectsUnknownDynamicField(t *testing.T) {
	var output baziStructuredDynamicSynthesis
	raw := `{"current_period_ref":"dayun[0]","current_period_realization":"maintain","period_claims":[],"liunian_claim":{"verdict":"x","fact_refs":[],"relation_refs":[],"claim_refs":[],"evidence_topics":[],"confidence":"保守判断"},"dayun_judgments":[]}`
	if err := decodeStructuredOutput(structuredSchemaBaziDynamicSynthesis, raw, &output); err == nil {
		t.Fatal("dynamic schema accepted renderer-only dayun_judgments field")
	}
}

func TestDecodeStructuredOutputAcceptsThreePartPatternEvaluation(t *testing.T) {
	var output baziStructuredStaticSynthesis
	raw := `{"axis_status":"established","pattern_name":"伤官格","pattern_route":"伤官佩印","pattern_evaluation":"成格受限","claims":[{"slot":"main_axis","verdict":"伤官格为主轴","status":"established","fact_refs":[],"evidence_topics":[]},{"slot":"strength","verdict":"日主偏强","status":"established","fact_refs":[],"evidence_topics":[]},{"slot":"tiaohou","verdict":"调候尚待确认","status":"limited","fact_refs":[],"evidence_topics":[]},{"slot":"pattern_usage","verdict":"伤官佩印为取用路线","status":"limited","fact_refs":[],"evidence_topics":[]}]}`
	if err := decodeStructuredOutput(structuredSchemaBaziStaticSynthesis, raw, &output); err != nil {
		t.Fatalf("decodeStructuredOutput() error = %v", err)
	}
	if output.PatternName != "伤官格" || output.PatternRoute != "伤官佩印" || output.PatternEvaluation != "成格受限" {
		t.Fatalf("pattern evaluation = %#v", output)
	}
}
