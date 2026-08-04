// This test file belongs to the BaZi deterministic calculation layer.
// It verifies DaYun quality coverage and protects the related contract from regressions.
// It computes reproducible BaZi facts; it must not generate narrative readings.
package bazi

import (
	"context"
	"testing"
)

func TestDayunAnalyzer_AlwaysDefersQualityToRuleProfile(t *testing.T) {
	raw, err := (&DayunAnalyzer{}).Execute(context.Background(), map[string]any{
		"dayun": []map[string]any{{"startAge": 30, "endAge": 39, "startAt": "2020-10-05 12:00:00", "endAtExclusive": "2030-10-05 12:00:00", "ganZhi": "甲午"}},
		"bazi_result": map[string]any{
			"dayGan": "戊",
			"pillars": []map[string]any{
				{"stem": "辛", "branch": "未"}, {"stem": "丁", "branch": "酉"},
				{"stem": "戊", "branch": "申"}, {"stem": "戊", "branch": "午"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	items := raw.(map[string]any)["dayun_analyzed"].([]map[string]any)
	if len(items) != 1 || items[0]["quality"] != "待profile裁断" {
		t.Fatalf("dayun quality must not be linearly scored: %#v", items)
	}
	if _, ok := items[0]["dayun_chonghe"].([]map[string]string); !ok {
		t.Fatalf("relation facts must remain available: %#v", items[0])
	}
	if items[0]["startAt"] != "2020-10-05 12:00:00" || items[0]["endAtExclusive"] != "2030-10-05 12:00:00" {
		t.Fatalf("annotated dayun must retain date boundaries for current-period recovery: %#v", items[0])
	}
}
