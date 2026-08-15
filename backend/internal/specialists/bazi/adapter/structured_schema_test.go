package adapter

import "testing"

func TestDecodeStructuredOutputRejectsInvalidAnalysisPlan(t *testing.T) {
	var output baziAnalysisPlan
	err := decodeStructuredOutput(structuredSchemaBaziAnalysisPlan, `{"mode":"static_full"}`, &output)
	if err == nil {
		t.Fatal("decodeStructuredOutput() error = nil, want schema validation failure")
	}
}

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
	raw := `{"current_period_ref":"dayun[0]","current_period_realization":"maintain","period_claims":[],"liunian_claim":{"verdict":"x","fact_refs":[],"relation_refs":[],"claim_refs":[],"evidence_topics":[],"confidence":"保守判断","boundary":"x"},"limitations":[],"reasoning_summary":"x","reasoning_steps":["x"],"outcome_domains":["structure"],"dayun_judgments":[]}`
	if err := decodeStructuredOutput(structuredSchemaBaziDynamicSynthesis, raw, &output); err == nil {
		t.Fatal("dynamic schema accepted renderer-only dayun_judgments field")
	}
}
