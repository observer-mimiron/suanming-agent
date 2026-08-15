package application

import (
	"testing"

	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
)

func TestBuildDynamicFactsViewKeepsOnlyModelFields(t *testing.T) {
	view := BuildDynamicFactsView(bazidomain.CharterInput{
		Dayun: map[string]any{
			"periods": []any{"甲子"}, "current_dayun": "甲子", "private": "drop",
		},
		Liunian: map[string]any{
			"liunian_year": 2026, "liunian_ganzhi": "丙午", "private": "drop",
		},
	})
	dayun, _ := view["dayun"].(map[string]any)
	liunian, _ := view["liunian"].(map[string]any)
	if dayun["private"] != nil || liunian["private"] != nil {
		t.Fatalf("dynamic view leaked non-model fields: %#v", view)
	}
	if dayun["current_dayun"] != "甲子" || liunian["liunian_year"] != 2026 {
		t.Fatalf("dynamic view lost calculated facts: %#v", view)
	}
}

func TestNormalizeByAliasCanonicalizesPlannerMode(t *testing.T) {
	if got := NormalizeByAlias("普通分析", map[string]string{"普通分析": "analysis"}); got != "analysis" {
		t.Fatalf("NormalizeByAlias() = %q, want analysis", got)
	}
}

func TestCanonicalTierTextDoesNotWithholdForMissingOptionalReferences(t *testing.T) {
	judgment, basis, withheld := canonicalTierText(
		baziCharterState{EvidenceQuality: baziEvidenceQuality{MissingTopics: []string{"qingzhuo"}}},
		baziCanonicalUnit{Verdict: "格局评价已定", Boundary: "命盘结构已验收。", EvidenceTopics: []string{"qingzhuo"}},
		baziTierAssessment{},
	)
	if withheld || judgment != "格局评价已定" || basis != "命盘结构已验收。" {
		t.Fatalf("tier text = (%q, %q, %t), want accepted model tier without retrieval cap", judgment, basis, withheld)
	}
}
