package domain

import "testing"

func TestBuildReferenceCatalogViewSortsIDsAndKeepsHintsOutOfContract(t *testing.T) {
	view := BuildReferenceCatalogView(ReferenceCatalogInput{
		FactIDs:         []string{"dayun[1].gan_zhi", "chart.day_master"},
		ClaimCategories: map[string]string{"rule.b": "strength", "rule.a": ""},
		RelationTexts:   map[string]string{"relation.natal.1": "", "relation.natal.0": "巳亥冲"},
	})

	facts := view["fact_refs"].([]ReferenceCatalogEntry)
	if len(facts) != 2 || facts[0].ID != "chart.day_master" || facts[1].ID != "dayun[1].gan_zhi" {
		t.Fatalf("fact refs = %#v", facts)
	}
	relations := view["relation_refs"].([]ReferenceCatalogEntry)
	if relations[0].Hint != "已计算关系：巳亥冲" || relations[1].Hint != "输入 core_chart：原局已计算关系" {
		t.Fatalf("relation refs = %#v", relations)
	}
}
