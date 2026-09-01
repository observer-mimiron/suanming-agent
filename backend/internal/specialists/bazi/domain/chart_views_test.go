package domain

import "testing"

func TestBuildCoreChartViewDropsImplementationPlaceholder(t *testing.T) {
	view := BuildCoreChartView(ChartViewInput{
		BaziResult: map[string]any{
			"dayGan":  "甲",
			"pillars": []map[string]any{{"branch": "子"}, {"branch": "亥"}},
		},
		Yongshen: map[string]any{
			"tiao_hou": "qiongtong_tiaohou_v1 规则表实现",
			"strength": "中和附近",
		},
	})
	if view["day_master"] != "甲" || view["month_pillar"].(map[string]any)["branch"] != "亥" {
		t.Fatalf("chart view = %#v", view)
	}
	if _, ok := view["tiao_hou"]; ok {
		t.Fatalf("placeholder leaked: %#v", view)
	}
}

func TestBuildDynamicFactsViewKeepsOnlyBoundedFields(t *testing.T) {
	view := BuildDynamicFactsView(ChartViewInput{
		Dayun:   map[string]any{"dayun_analyzed": []any{"甲午"}, "private": "drop"},
		Liunian: map[string]any{"liunian_year": 2026, "liunian_ganzhi": "丙午", "private": "drop"},
	})
	if _, ok := view["dayun"].(map[string]any)["private"]; ok {
		t.Fatalf("private dayun field leaked: %#v", view)
	}
	if _, ok := view["liunian"].(map[string]any)["private"]; ok {
		t.Fatalf("private liunian field leaked: %#v", view)
	}
}

func TestBuildCoreChartViewProjectsBoundedPatternCandidates(t *testing.T) {
	view := BuildCoreChartView(ChartViewInput{Yongshen: map[string]any{
		"geju_candidate":   "伤官格",
		"geju_combination": "伤官生财；伤官佩印",
	}})
	candidates, ok := view["pattern_candidates"].([]map[string]any)
	if !ok || len(candidates) != 3 {
		t.Fatalf("pattern candidates = %#v", view["pattern_candidates"])
	}
	if candidates[0]["name"] != "伤官格" || candidates[0]["origin"] != "month_command" {
		t.Fatalf("month candidate = %#v", candidates[0])
	}
}
