package domain

import "testing"

func TestEvaluateEvidenceBundleQualityDoesNotRequireRetrieval(t *testing.T) {
	quality := EvaluateEvidenceBundleQuality(EvidencePlan{NeedRetrieval: true}, EvidenceBundle{})
	if !quality.Enough || len(quality.RequiredTopics) != 0 || len(quality.MissingTopics) != 0 {
		t.Fatalf("quality = %#v, empty optional evidence must not block synthesis", quality)
	}
}
