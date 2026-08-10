// This test file belongs to the manager-owned runtime layer.
// It verifies BaZi evidence bundle behavior and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
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
	for _, required := range []string{"透干、藏干层级、通根、时令、承接和反证", "月令本气未透不能单独否定", "不得因“印星根气不足”"} {
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
	for _, required := range []string{"若为未成年人", "成长环境、照护节奏和可观察发展", "完整大运目录、干支、年龄和日期由 runtime 渲染"} {
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

func TestNormalizeBaziEvidencePlan_SpecializesTiaohouQuery(t *testing.T) {
	plan := baziEvidencePlan{
		Stage: "static",
		QueryPackets: []baziQueryPacket{{
			Topic:            "tiaohou",
			Query:            "穷通宝鉴 调候 丁火 酉月",
			PreferredSources: []string{"穷通宝鉴"},
			SourceTier:       "A",
		}},
	}
	input := baziCharterInput{BaziResult: map[string]any{
		"dayGan":  "丁",
		"pillars": []map[string]any{{"name": "年柱", "stem": "甲", "branch": "戌"}, {"name": "月柱", "stem": "辛", "branch": "酉"}},
	}}

	out := normalizeBaziEvidencePlan(plan, input, baziAnalysisPlan{RetrievalStage: "static"})
	var query string
	for _, packet := range out.QueryPackets {
		if packet.Topic == "tiaohou" && packet.SourceTier == "A" {
			query = packet.Query
			break
		}
	}
	if !strings.Contains(query, "丁火") || !strings.Contains(query, "酉月") || !strings.Contains(query, "八月丁火") {
		t.Fatalf("specialized tiaohou query missing chart terms: %q", query)
	}
}

// TestNormalizeBaziEvidencePlan_AddsTierQualificationTopics protects the
// upstream contract required by TierEvidenceComplete. A planner omission must
// become a retrievable gap, not an unconditional runtime withholding.
func TestNormalizeBaziEvidencePlan_AddsTierQualificationTopics(t *testing.T) {
	out := normalizeBaziEvidencePlan(baziEvidencePlan{Stage: "static"}, baziCharterInput{}, baziAnalysisPlan{RetrievalStage: "static"})
	for _, topic := range []string{"qingzhuo", "bingyao", "jiuying", "poge"} {
		if !hasATierEvidenceTopic(out, topic) {
			t.Fatalf("static evidence plan missing required tier topic %q: %+v", topic, out.QueryPackets)
		}
	}
}

// TestExtractAuthorityClassic_MapsKnowledgeSlugToClassic protects the retrieval
// source contract: local wiki slugs must still count as their canonical classics.
func TestExtractAuthorityClassic_MapsKnowledgeSlugToClassic(t *testing.T) {
	source := "knowledge://ref-bazi-qiongtong-s001 (五行总论)"
	classic := extractAuthorityClassic(source)
	if classic != "穷通宝鉴" {
		t.Fatalf("classic = %q, want 穷通宝鉴", classic)
	}

	plan := baziEvidencePlan{QueryPackets: []baziQueryPacket{{Topic: "tiaohou", SourceTier: "A"}}}
	bundle := baziEvidenceBundle{
		Stage: "static",
		CriticalTopicBuckets: map[string][]baziCitation{
			"tiaohou": {{Classic: classic}},
		},
	}
	quality := evaluateEvidenceBundleQuality(plan, bundle)
	if !quality.Enough || !containsString(quality.CoveredTopics, "tiaohou") {
		t.Fatalf("qiongtong slug should satisfy tiaohou authority coverage: %+v", quality)
	}
}

func TestBaziTiaohouCoverage_UsesEvidenceStatus(t *testing.T) {
	if got := baziTiaohouCoverage(baziEvidenceQuality{CoveredTopics: []string{"tiaohou"}}); got != "authority_evidence_covered" {
		t.Fatalf("covered tiaohou = %q", got)
	}
	if got := baziTiaohouCoverage(baziEvidenceQuality{MissingTopics: []string{"tiaohou"}}); got != "missing_authority_evidence" {
		t.Fatalf("missing tiaohou = %q", got)
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
