// This test file belongs to the local observability layer.
// It verifies cheap-gate summary aggregation and protects the related contract from regressions.
// It summarizes evidence for operators; it is not an acceptance source by itself.
package observability

import (
	"strings"
	"testing"
)

func TestSummarizeCheapGateHits_RollsUpBucketsAndPreview(t *testing.T) {
	raw := strings.Join([]string{
		`{"timestamp":"2026-07-12T01:00:00+08:00","session_id":"s1","primary_domain":"bazi","task_intent":"fortune_followup","decision_source":"cheap_followup_reuse","gate_reason":"cheap_followup_reuse","execution_mode":"reuse_followup","reuse_cached_result":true}`,
		`{"timestamp":"2026-07-12T01:01:00+08:00","session_id":"s2","primary_domain":"bazi","task_intent":"interpret_chart","decision_source":"cheap_followup_reuse","gate_reason":"cheap_followup_reuse","execution_mode":"reuse_followup","reuse_cached_result":true}`,
		`{"timestamp":"2026-07-12T01:02:00+08:00","session_id":"s3","primary_domain":"ziwei","task_intent":"fortune_followup","decision_source":"cheap_followup_reuse","gate_reason":"cheap_followup_reuse","execution_mode":"reuse_followup","reuse_cached_result":true}`,
	}, "\n")

	got, err := SummarizeCheapGateHits(strings.NewReader(raw), 2)
	if err != nil {
		t.Fatalf("SummarizeCheapGateHits() error = %v", err)
	}
	if got.TotalHits != 3 {
		t.Fatalf("TotalHits = %d, want 3", got.TotalHits)
	}
	if len(got.ByPrimaryDomain) != 2 || got.ByPrimaryDomain[0].Value != "bazi" || got.ByPrimaryDomain[0].Count != 2 {
		t.Fatalf("ByPrimaryDomain = %+v, want bazi bucket first with count 2", got.ByPrimaryDomain)
	}
	if len(got.ByTaskIntent) != 2 || got.ByTaskIntent[0].Value != "fortune_followup" || got.ByTaskIntent[0].Count != 2 {
		t.Fatalf("ByTaskIntent = %+v, want fortune_followup bucket first with count 2", got.ByTaskIntent)
	}
	if len(got.Preview) != 2 {
		t.Fatalf("Preview len = %d, want 2", len(got.Preview))
	}
}

func TestSummarizeCheapGateHits_SkipsMalformedLines(t *testing.T) {
	raw := strings.Join([]string{
		`{"primary_domain":"bazi","task_intent":"fortune_followup","decision_source":"cheap_followup_reuse","gate_reason":"cheap_followup_reuse","execution_mode":"reuse_followup"}`,
		`not-json`,
		`{"primary_domain":"ziwei","task_intent":"fortune_followup","decision_source":"cheap_followup_reuse","gate_reason":"cheap_followup_reuse","execution_mode":"reuse_followup"}`,
	}, "\n")

	got, err := SummarizeCheapGateHits(strings.NewReader(raw), 5)
	if err != nil {
		t.Fatalf("SummarizeCheapGateHits() error = %v", err)
	}
	if got.TotalHits != 2 {
		t.Fatalf("TotalHits = %d, want 2", got.TotalHits)
	}
}
