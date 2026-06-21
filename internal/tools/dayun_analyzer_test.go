package tools

import (
	"context"
	"testing"
)

func TestDayunAnalyzer_ExecuteAcceptsJSONRoundTripDayun(t *testing.T) {
	tool := &DayunAnalyzer{}
	params := map[string]any{
		"dayun": []interface{}{
			map[string]interface{}{"startAge": float64(10), "endAge": float64(19), "ganZhi": "甲子"},
		},
		"bazi_result": map[string]any{
			"dayGan":            "甲",
			"day_master_wuxing": "木",
			"yongshen": map[string]any{
				"yong_shen": []interface{}{"木"},
				"ji_shen":   []interface{}{"金"},
			},
		},
	}

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T, want map[string]any", result)
	}
	items, ok := got["dayun_analyzed"].([]map[string]any)
	if !ok {
		t.Fatalf("dayun_analyzed type = %T, want []map[string]any", got["dayun_analyzed"])
	}
	if len(items) != 1 {
		t.Fatalf("len(dayun_analyzed) = %d, want 1", len(items))
	}
	if items[0]["ganZhi"] != "甲子" {
		t.Fatalf("ganZhi = %v, want 甲子", items[0]["ganZhi"])
	}
}
