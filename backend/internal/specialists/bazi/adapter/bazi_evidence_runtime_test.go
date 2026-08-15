package adapter

import (
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/mcp"
)

func TestNormalizeBaziEvidencePlanLimitsStaticReferencesWithoutRetry(t *testing.T) {
	plan := normalizeBaziEvidencePlan(baziEvidencePlan{
		NeedRetrieval: true,
		Stage:         "static",
		QueryPackets: []baziQueryPacket{
			{Topic: "bingyao", Query: "滴天髓 病药 制化", SourceTier: "A"},
			{Topic: "geju", Query: "子平真诠 格局 月令 取格", SourceTier: "A"},
			{Topic: "tiaohou", Query: "穷通宝鉴 甲木 酉月 调候", SourceTier: "A"},
			{Topic: "poge", Query: "子平真诠 破格 败格", SourceTier: "A"},
		},
	}, baziCharterInput{}, baziAnalysisPlan{RetrievalStage: "static"})

	if len(plan.QueryPackets) != baziEvidenceInitialQueryBudget {
		t.Fatalf("query count = %d, want %d", len(plan.QueryPackets), baziEvidenceInitialQueryBudget)
	}
	if got := plan.QueryPackets[0].Topic; got != "bingyao" {
		t.Fatalf("query topic = %q, want first configured packet", got)
	}
}

func TestNormalizeBaziEvidencePlanCapsPlannerQueries(t *testing.T) {
	plan := normalizeBaziEvidencePlan(baziEvidencePlan{
		NeedRetrieval: true,
		QueryPackets: []baziQueryPacket{
			{Topic: "geju", Query: "子平真诠 格局 月令 取格"},
			{Topic: "tiaohou", Query: "穷通宝鉴 丙火 亥月 调候"},
			{Topic: "fuyi", Query: "滴天髓 扶抑 病药"},
		},
	}, baziCharterInput{}, baziAnalysisPlan{RetrievalStage: "static"})
	if len(plan.QueryPackets) != baziEvidenceInitialQueryBudget {
		t.Fatalf("planner query count = %d, want %d", len(plan.QueryPackets), baziEvidenceInitialQueryBudget)
	}
	if baziEvidenceQueryBudget != 3 {
		t.Fatalf("total evidence budget = %d, want 3", baziEvidenceQueryBudget)
	}
}

func TestBuildEvidenceSupplementPlanSkipsToolFailures(t *testing.T) {
	plan := baziEvidencePlan{AllowReflection: true, Stage: "static", QueryPackets: []baziQueryPacket{{Topic: "geju", Query: "子平真诠 格局", PreferredSources: []string{"子平真诠"}}}}
	supplement := buildEvidenceSupplementPlan(plan, baziEvidenceBundle{DegradedTopics: []string{"geju"}})
	if supplement.NeedRetrieval {
		t.Fatalf("supplement = %#v, want no transport retry", supplement)
	}
}

func TestBuildEvidenceSupplementPlanUsesOneWidenedQuery(t *testing.T) {
	plan := baziEvidencePlan{AllowReflection: true, Stage: "static", QueryPackets: []baziQueryPacket{{Topic: "geju", Query: "子平真诠 格局", PreferredSources: []string{"子平真诠"}}}}
	supplement := buildEvidenceSupplementPlan(plan, baziEvidenceBundle{})
	if !supplement.NeedRetrieval || len(supplement.QueryPackets) != baziEvidenceSupplementQueryBudget {
		t.Fatalf("supplement = %#v", supplement)
	}
	if got := supplement.QueryPackets[0].Query; got != "子平真诠 geju 原文" {
		t.Fatalf("supplement query = %q", got)
	}
}

func TestCitationsFromKnowledgeResultRejectsBookLandingPages(t *testing.T) {
	result := map[string]any{"passages": []mcp.Passage{
		{Source: "knowledge://ref-bazi-qiongtong (穷通宝鉴)", Content: "穷通宝鉴 > 清代余春台"},
		{Source: "knowledge://ref-bazi-qiongtong-s001 (五行总论)", Content: "丙火生于亥月，火气受制，取用须先察寒暖燥湿与全局通关。"},
	}}
	citations := citationsFromKnowledgeResult(result, baziQueryPacket{PreferredSources: []string{"穷通宝鉴"}})
	if len(citations) != 1 || citations[0].Classic != "穷通宝鉴" {
		t.Fatalf("citations = %#v, want one substantive 穷通宝鉴 chapter", citations)
	}
}

func TestBaziEvidenceNeedsActionAtMostOnce(t *testing.T) {
	if !baziEvidenceNeedsAction(&baziInternalGraphState{}) {
		t.Fatal("first evidence action must be scheduled")
	}
	if baziEvidenceNeedsAction(&baziInternalGraphState{EvidenceAttempts: 1}) {
		t.Fatal("empty or failed evidence must not schedule another retrieval")
	}
}

func TestLimitCitationQuotesKeepsOnlyTwoPromptPassages(t *testing.T) {
	limited := limitCitationQuotes([]baziCitation{{Classic: "穷通宝鉴", Quotes: []string{"甲", "乙", "丙"}}}, 2)
	if len(limited) != 2 || limited[0].Quotes[0] != "甲" || limited[1].Quotes[0] != "乙" {
		t.Fatalf("limited citations = %#v, want first two quotes", limited)
	}
}
