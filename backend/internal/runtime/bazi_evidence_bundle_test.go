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

// TestBaziStaticSynthesisPrompt_RequiresVisibleHiddenRouteComparison prevents
// hidden combinations from being promoted solely because they have a name.
func TestBaziStaticSynthesisPrompt_RequiresVisibleHiddenRouteComparison(t *testing.T) {
	prompt := prompts.BaziStaticSynthesisInstruction
	for _, required := range []string{"透干可见性", "藏干层级", "承接闭环", "纯藏支组合若要压过透干路线", "本气未透”只能改变显用方式", "某十神为忌`属于扶抑或病药层"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("static synthesis prompt missing route-comparison contract %q", required)
		}
	}
	for _, forbidden := range []string{"故只能按暗格取", "因此主轴起点不够纯"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("static synthesis prompt still teaches non-visibility auto-downgrade: %q", forbidden)
		}
	}
}

// TestBaziDynamicSynthesisPrompt_AgeBoundaryCoversAllDynamicFields prevents
// adult-domain leakage through future dayun entries for a minor subject.
func TestBaziDynamicSynthesisPrompt_AgeBoundaryCoversAllDynamicFields(t *testing.T) {
	prompt := prompts.BaziDynamicSynthesisInstruction
	for _, required := range []string{"全部字段", "完整 `dayun_path` / `dayun_judgments`", "未来成年大运"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("dynamic synthesis prompt missing all-field age boundary %q", required)
		}
	}
}

func TestBaziEvidenceBundleQuality_RejectsAuxiliaryOnlyEvidence(t *testing.T) {
	plan := baziEvidencePlan{QueryPackets: []baziQueryPacket{{Topic: "geju", SourceTier: "A"}}}
	bundle := baziEvidenceBundle{
		Stage: "static",
		Citations: []baziCitation{
			{Classic: "神煞"},
		},
	}
	result := evaluateEvidenceBundleQuality(plan, bundle)
	if result.Enough {
		t.Fatalf("auxiliary-only evidence must not pass static quality gate")
	}
}

// TestBaziEvidenceBundleQuality_RequiresCriticalTopicCoverage locks the root
// contract: one authority citation cannot cover unrelated planned topics.
func TestBaziEvidenceBundleQuality_RequiresCriticalTopicCoverage(t *testing.T) {
	plan := baziEvidencePlan{QueryPackets: []baziQueryPacket{
		{Topic: "geju", SourceTier: "A"},
		{Topic: "tiaohou", SourceTier: "A"},
		{Topic: "geju", SourceTier: "B"},
	}}
	bundle := baziEvidenceBundle{
		Stage: "static",
		CriticalTopicBuckets: map[string][]baziCitation{
			"geju": {{Classic: "子平真诠"}},
		},
	}

	result := evaluateEvidenceBundleQuality(plan, bundle)
	if result.Enough {
		t.Fatal("missing tiaohou evidence must keep quality insufficient")
	}
	if !containsString(result.CoveredTopics, "geju") || !containsString(result.MissingTopics, "tiaohou") {
		t.Fatalf("unexpected topic coverage: %+v", result)
	}
}

// TestBaziEvidenceBundleQuality_BTierCannotCoverATierTopic ensures examples do
// not substitute for critical methodology evidence under the same topic name.
func TestBaziEvidenceBundleQuality_BTierCannotCoverATierTopic(t *testing.T) {
	plan := baziEvidencePlan{QueryPackets: []baziQueryPacket{{Topic: "geju", SourceTier: "A"}}}
	bundle := baziEvidenceBundle{
		Stage: "static",
		TopicBuckets: map[string][]baziCitation{
			"geju": {{Classic: "子平真诠"}},
		},
	}

	if result := evaluateEvidenceBundleQuality(plan, bundle); result.Enough {
		t.Fatal("B-tier topic bucket must not satisfy A-tier coverage")
	}
}

// TestBuildEvidenceRetryPlan_RetriesOnlyMissingTopics prevents reflection from
// repeating the same broad query plan after a partial retrieval failure.
func TestBuildEvidenceRetryPlan_RetriesOnlyMissingTopics(t *testing.T) {
	plan := baziEvidencePlan{
		NeedRetrieval: true,
		Stage:         "static",
		QueryPackets: []baziQueryPacket{
			{Topic: "geju", Query: "broad geju query", PreferredSources: []string{"子平真诠"}, SourceTier: "A"},
			{Topic: "tiaohou", Query: "broad tiaohou query", PreferredSources: []string{"穷通宝鉴"}, SourceTier: "A"},
		},
	}

	retry := buildEvidenceRetryPlan(plan, baziEvidenceQuality{MissingTopics: []string{"tiaohou"}})
	if len(retry.QueryPackets) != 1 || retry.QueryPackets[0].Topic != "tiaohou" {
		t.Fatalf("unexpected retry packets: %+v", retry.QueryPackets)
	}
	if retry.QueryPackets[0].Query == plan.QueryPackets[1].Query {
		t.Fatal("retry query must be simplified instead of repeated unchanged")
	}
}

// TestBuildEvidenceRetryPlan_RetriesCriticalTopicsOnHighConflict keeps the
// conflict reflection path useful without repeating the original broad plan.
func TestBuildEvidenceRetryPlan_RetriesCriticalTopicsOnHighConflict(t *testing.T) {
	plan := baziEvidencePlan{
		NeedRetrieval: true,
		Stage:         "static",
		QueryPackets: []baziQueryPacket{{
			Topic: "geju", Query: "original broad query", PreferredSources: []string{"子平真诠"}, SourceTier: "A",
		}},
	}

	retry := buildEvidenceRetryPlan(plan, baziEvidenceQuality{Enough: true, ConflictScore: "high"})
	if len(retry.QueryPackets) != 1 || retry.QueryPackets[0].Query == plan.QueryPackets[0].Query {
		t.Fatalf("high-conflict retry must simplify critical queries: %+v", retry.QueryPackets)
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
