package bazi

import (
	"context"
	"testing"
)

func TestBaziCalc_Validation(t *testing.T) {
	tt := &CalcTool{}

	tests := []struct {
		name   string
		params map[string]any
		errMsg string
	}{
		{"missing year", map[string]any{"month": float64(1), "day": float64(1), "hour": float64(12), "gender": "男"}, "year out of range"},
		{"year low", map[string]any{"year": float64(1899), "month": float64(1), "day": float64(1), "hour": float64(12), "gender": "男"}, "year out of range"},
		{"year high", map[string]any{"year": float64(2101), "month": float64(1), "day": float64(1), "hour": float64(12), "gender": "男"}, "year out of range"},
		{"year not number", map[string]any{"year": "abc", "month": float64(1), "day": float64(1), "hour": float64(12), "gender": "男"}, "year out of range"},
		{"month out of range", map[string]any{"year": float64(2000), "month": float64(13), "day": float64(1), "hour": float64(12), "gender": "男"}, "month out of range"},
		{"month zero", map[string]any{"year": float64(2000), "month": float64(0), "day": float64(1), "hour": float64(12), "gender": "男"}, "month out of range"},
		{"day out of range", map[string]any{"year": float64(2000), "month": float64(1), "day": float64(32), "hour": float64(12), "gender": "男"}, "day out of range"},
		{"day zero", map[string]any{"year": float64(2000), "month": float64(1), "day": float64(0), "hour": float64(12), "gender": "男"}, "day out of range"},
		{"hour out of range", map[string]any{"year": float64(2000), "month": float64(1), "day": float64(1), "hour": float64(24), "gender": "男"}, "hour out of range"},
		{"hour negative", map[string]any{"year": float64(2000), "month": float64(1), "day": float64(1), "hour": float64(-1), "gender": "男"}, "hour out of range"},
		{"invalid gender", map[string]any{"year": float64(2000), "month": float64(1), "day": float64(1), "hour": float64(12), "gender": "其他"}, "gender must be 男/女"},
		{"gender not string", map[string]any{"year": float64(2000), "month": float64(1), "day": float64(1), "hour": float64(12), "gender": float64(123)}, "gender must be 男/女"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tt.Execute(context.Background(), tc.params)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tc.errMsg {
				t.Errorf("expected %q, got %q", tc.errMsg, err.Error())
			}
		})
	}
}

func TestBaziCalc_ValidInput(t *testing.T) {
	tt := &CalcTool{}

	result, err := tt.Execute(context.Background(), map[string]any{
		"year":   float64(1990),
		"month":  float64(5),
		"day":    float64(15),
		"hour":   float64(14),
		"gender": "男",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r, ok := result.(map[string]any)
	if !ok {
		t.Fatal("result is not a map")
	}

	// Check required fields
	if r["pillars"] == nil {
		t.Error("missing pillars")
	}
	if r["dayGan"] == nil {
		t.Error("missing dayGan")
	}
	if r["wuxing"] == nil {
		t.Error("missing wuxing")
	}
	if r["dayun"] == nil {
		t.Error("missing dayun")
	}
	if r["gender"] != "男" {
		t.Errorf("expected gender 男, got %v", r["gender"])
	}
	if r["birthday"] != "1990-05-15 14:00" {
		t.Errorf("unexpected birthday: %v", r["birthday"])
	}

	// Verify pillars (bazi_calc returns []map[string]any)
	pillars, ok := r["pillars"].([]map[string]any)
	if !ok {
		t.Fatal("pillars is not []map[string]any")
	}
	if len(pillars) != 4 {
		t.Fatalf("expected 4 pillars, got %d", len(pillars))
	}
	expectedKeys := []string{"name", "stem", "branch", "shiShen"}
	for i, p := range pillars {
		for _, k := range expectedKeys {
			v, ok := p[k].(string)
			if !ok || v == "" {
				t.Errorf("pillar[%d] missing or empty %s", i, k)
			}
		}
	}

	// Verify wuxing (bazi_calc returns map[string]int)
	wx, ok := r["wuxing"].(map[string]int)
	if !ok {
		t.Fatal("wuxing is not map[string]int")
	}
	total := wx["木"] + wx["火"] + wx["土"] + wx["金"] + wx["水"]
	if total != 8 {
		t.Errorf("expected 8 wuxing elements, got %d", total)
	}

	// Verify dayun (bazi_calc returns []map[string]any)
	dayun, ok := r["dayun"].([]map[string]any)
	if !ok {
		t.Fatal("dayun is not []map[string]any")
	}
	if len(dayun) == 0 {
		t.Error("dayun is empty")
	}
	for i, dy := range dayun {
		if _, ok := dy["startAge"]; !ok {
			t.Errorf("dayun[%d] missing startAge", i)
		}
		if _, ok := dy["endAge"]; !ok {
			t.Errorf("dayun[%d] missing endAge", i)
		}
		if _, ok := dy["ganZhi"]; !ok {
			t.Errorf("dayun[%d] missing ganZhi", i)
		}
	}
}

func TestBaziCalcKnownCase(t *testing.T) {
	tool := &CalcTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"year": float64(1990), "month": float64(5),
		"day": float64(20), "hour": float64(8), "gender": "男",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	data := result.(map[string]any)

	pillars, ok := data["pillars"].([]map[string]any)
	if !ok {
		t.Fatal("pillars is not []map[string]any")
	}
	if len(pillars) != 4 {
		t.Errorf("pillars=%d, want 4", len(pillars))
	}
	if data["dayGan"].(string) == "" {
		t.Error("dayGan empty")
	}

	for _, p := range pillars {
		shiShen, ok := p["shiShen"].(string)
		if !ok || shiShen == "" {
			name, _ := p["name"].(string)
			t.Errorf("pillar %s shiShen empty", name)
		}
	}

	wuxing, ok := data["wuxing"].(map[string]int)
	if !ok {
		t.Fatal("wuxing is not map[string]int")
	}
	total := 0
	for _, v := range wuxing {
		total += v
	}
	if total != 8 {
		t.Errorf("wuxing total=%d, want 8", total)
	}

	dayun, ok := data["dayun"].([]map[string]any)
	if !ok {
		t.Fatal("dayun is not []map[string]any")
	}
	if len(dayun) == 0 {
		t.Errorf("dayun is empty")
	}

	t.Logf("日主=%s 五行=%v 大运=%d步", data["dayGan"], wuxing, len(dayun))
}

func TestBaziCalc_ShenshaStructure(t *testing.T) {
	tool := &CalcTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"year": float64(1990), "month": float64(5),
		"day": float64(20), "hour": float64(8), "gender": "男",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	data := result.(map[string]any)

	pillars, ok := data["pillars"].([]map[string]any)
	if !ok {
		t.Fatal("pillars missing")
	}
	for i, p := range pillars {
		shensha, ok := p["shensha"].([]ShenshaItem)
		if !ok {
			t.Errorf("pillar[%d] shensha missing or wrong type", i)
			continue
		}
		// Must be array (empty or not), never nil
		if shensha == nil {
			t.Errorf("pillar[%d] shensha is nil, want empty slice", i)
		}
	}

	summary, ok := data["shensha_summary"].(map[string]any)
	if !ok {
		t.Fatal("shensha_summary missing")
	}
	if _, ok := summary["all"]; !ok {
		t.Error("shensha_summary missing 'all'")
	}
	if _, ok := summary["by_pillar"]; !ok {
		t.Error("shensha_summary missing 'by_pillar'")
	}
}

func TestBaziCalc_ShenshaKnownHits(t *testing.T) {
	tool := &CalcTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"year": float64(1990), "month": float64(5),
		"day": float64(20), "hour": float64(8), "gender": "男",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	data := result.(map[string]any)

	// 1990-05-20 08:00 男 → 年庚午 月辛巳 日乙酉 时庚辰
	// 日干乙: 文昌贵人→午(年柱)
	// 日支酉(巳酉丑): 桃花→午(年柱), 将星→酉(日柱)
	// 年支午(巳午未): 寡宿→辰(时柱)

	summary := data["shensha_summary"].(map[string]any)
	byPillar := summary["by_pillar"].(map[string][]ShenshaItem)

	findSS := func(pillar, name string) bool {
		for _, s := range byPillar[pillar] {
			if s.Name == name {
				return true
			}
		}
		return false
	}

	if !findSS("年柱", "文昌贵人") {
		t.Error("年柱 should have 文昌贵人")
	}
	if !findSS("年柱", "桃花") {
		t.Error("年柱 should have 桃花")
	}
	if !findSS("日柱", "将星") {
		t.Error("日柱 should have 将星")
	}
	if !findSS("时柱", "寡宿") {
		t.Error("时柱 should have 寡宿")
	}

	t.Logf("all shensha: %v", summary["all"])
}

func TestBaziCalcInvalidYear(t *testing.T) {
	tool := &CalcTool{}
	_, err := tool.Execute(context.Background(), map[string]any{
		"year": float64(1800), "month": float64(1),
		"day": float64(1), "hour": float64(0), "gender": "男",
	})
	if err == nil {
		t.Error("expected error for year 1800")
	}
}
