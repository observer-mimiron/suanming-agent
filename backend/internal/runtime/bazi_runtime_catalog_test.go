package runtime

import (
	"strings"
	"testing"
)

func TestBaziRuntimeCatalogViewUsesReadableHintsButValidatesOnlyIDs(t *testing.T) {
	state := baziCharterState{Input: baziCharterInput{
		BaziResult: map[string]any{"pillars": map[string]any{"day": "丙戌"}},
		Yongshen: map[string]any{
			"chonghe": []any{map[string]any{"description": "巳亥冲"}},
		},
	}}

	view := baziRuntimeCatalogView(state)
	relations, ok := view["relation_refs"].([]baziCatalogEntry)
	if !ok {
		t.Fatalf("relation_refs type = %T; want []baziCatalogEntry", view["relation_refs"])
	}
	if len(relations) != 1 || relations[0].ID != "relation.natal.0" || relations[0].Hint != "已计算关系：巳亥冲" {
		t.Fatalf("relation catalog = %#v", relations)
	}

	err := validateBaziReferenceCatalog(state, []baziAssertion{{RelationRefs: []baziRelationRef{"relation.natal.0"}}})
	if err != nil {
		t.Fatalf("declared catalog ID rejected: %v", err)
	}
	err = validateBaziReferenceCatalog(state, []baziAssertion{{RelationRefs: []baziRelationRef{"relation.fire_bureau"}}})
	if err == nil {
		t.Fatal("undeclared relation ID must remain rejected")
	}
}

func TestBaziDynamicRuntimeCatalogExcludesStaticReferences(t *testing.T) {
	state := baziCharterState{Input: baziCharterInput{
		Dayun: map[string]any{"dayun_analyzed": []map[string]any{{
			"ganZhi": "甲子", "branch": "子", "branchHiddenStems": []string{"癸"},
			"branchTenGods": []string{"正印"}, "branchMainTenGod": "正印",
			"dayun_chonghe": []string{"子午冲"},
		}}},
		Liunian: map[string]any{"liunian_ganzhi": "丙午", "liunian_chonghe": []string{"午午自刑"}},
	}}
	view := baziDynamicRuntimeCatalogView(state)
	periodRefs, ok := view["period_refs"].([]string)
	if !ok || len(periodRefs) != 1 || periodRefs[0] != "dayun[0]" {
		t.Fatalf("dynamic period catalog = %#v", view["period_refs"])
	}
	facts := view["fact_refs"].([]baziCatalogEntry)
	for _, entry := range facts {
		if !strings.HasPrefix(entry.ID, "dayun[") && !strings.HasPrefix(entry.ID, "liunian.") {
			t.Fatalf("dynamic catalog leaked static fact %q", entry.ID)
		}
	}
	if err := validateDynamicBaziReferenceCatalog(state, []baziAssertion{{FactRefs: []baziFactRef{"chart.day_master"}}}); err == nil {
		t.Fatal("dynamic validator accepted a static fact reference")
	}
	for _, ref := range []baziFactRef{
		"dayun[0].branch", "dayun[0].branch_hidden_stems",
		"dayun[0].branch_ten_gods", "dayun[0].branch_main_ten_god",
	} {
		if err := validateDynamicBaziReferenceCatalog(state, []baziAssertion{{FactRefs: []baziFactRef{ref}}}); err != nil {
			t.Fatalf("dynamic validator rejected deterministic field %q: %v", ref, err)
		}
	}
}

func TestBaziStaticRuntimeCatalogExcludesDynamicReferences(t *testing.T) {
	state := baziCharterState{Input: baziCharterInput{
		Dayun:   map[string]any{"dayun_analyzed": []map[string]any{{"ganZhi": "甲子"}}},
		Liunian: map[string]any{"liunian_ganzhi": "丙午"},
		Yongshen: map[string]any{
			"chonghe": []any{map[string]any{"description": "巳亥冲"}},
		},
	}}
	view := baziStaticRuntimeCatalogView(state)
	for _, entry := range view["fact_refs"].([]baziCatalogEntry) {
		if strings.HasPrefix(entry.ID, "dayun[") || strings.HasPrefix(entry.ID, "liunian.") {
			t.Fatalf("static catalog leaked dynamic fact %q", entry.ID)
		}
	}
	if err := validateStaticBaziReferenceCatalog(state, []baziAssertion{{FactRefs: []baziFactRef{"dayun[0].gan_zhi"}}}); err == nil {
		t.Fatal("static validator accepted a dynamic fact reference")
	}
	if _, ok := view["relation_refs"]; ok {
		t.Fatal("static catalog must not expose original-chart relation selectors")
	}
	if err := validateStaticBaziReferenceCatalog(state, []baziAssertion{{RelationRefs: []baziRelationRef{"relation.natal.0"}}}); err == nil {
		t.Fatal("static validator accepted a relation selector")
	}
}

func TestBaziRuntimeCatalogIncludesFactCapsuleReferences(t *testing.T) {
	state := baziCharterState{}
	for _, ref := range []baziFactRef{
		"fact_capsule.month_command",
		"fact_capsule.support_score",
		"fact_capsule.fire_effectiveness_known",
		"fact_capsule.tier_evidence_missing",
	} {
		if err := validateBaziReferenceCatalog(state, []baziAssertion{{FactRefs: []baziFactRef{ref}}}); err != nil {
			t.Fatalf("fact capsule reference %q rejected: %v", ref, err)
		}
	}
	entries := baziRuntimeCatalogView(state)["fact_refs"].([]baziCatalogEntry)
	for _, entry := range entries {
		if entry.ID == "fact_capsule.support_score" && entry.Hint == "输入 fact_capsule：已计算的裁断前提" {
			return
		}
	}
	t.Fatal("fact capsule reference must expose a matching catalog hint")
}
