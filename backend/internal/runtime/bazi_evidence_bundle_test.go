package runtime

import (
	"strings"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/prompts"
)

func TestBaziAuthoritySourceTier_StaticStageUsesPrimaryClassics(t *testing.T) {
	static := stageAuthoritySources("static")
	if len(static.Primary) == 0 {
		t.Fatalf("expected static stage primary sources")
	}
	if static.Primary[0] != "子平真诠" {
		t.Fatalf("expected 子平真诠 as first static primary source")
	}
	if containsString(static.Primary, "神煞") {
		t.Fatalf("static primary sources must not include low-weight materials")
	}
}

func TestBaziEvidencePlannerPrompt_IsEmbedded(t *testing.T) {
	if strings.TrimSpace(prompts.BaziEvidencePlannerInstruction) == "" {
		t.Fatalf("evidence planner prompt should not be empty")
	}
}

func TestBaziEvidenceBundleQuality_RejectsAuxiliaryOnlyEvidence(t *testing.T) {
	bundle := baziEvidenceBundle{
		Stage: "static",
		Citations: []baziCitation{
			{Classic: "神煞"},
		},
	}
	result := evaluateEvidenceBundleQuality(bundle)
	if result.Enough {
		t.Fatalf("auxiliary-only evidence must not pass static quality gate")
	}
}

func TestBaziEvidenceQuality_OnlyWeakEvidenceTriggersReflection(t *testing.T) {
	if shouldReflectOnEvidence(baziEvidenceQuality{Enough: true, FocusScore: "high", ConflictScore: "low"}) {
		t.Fatalf("strong evidence must not trigger reflection")
	}
	if !shouldReflectOnEvidence(baziEvidenceQuality{Enough: false, FocusScore: "low", ConflictScore: "unknown"}) {
		t.Fatalf("weak evidence should trigger reflection")
	}
}
