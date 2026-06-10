package tools

import (
	"testing"
)

func TestBaziCalc_Validation(t *testing.T) {
	tt := &BaziCalcTool{}

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
			_, err := tt.Execute(tc.params)
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
	tt := &BaziCalcTool{}

	result, err := tt.Execute(map[string]any{
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

	// Verify pillars
	pillars, ok := r["pillars"].([]map[string]string)
	if !ok {
		// Could be []any — handle both
		pAny, ok := r["pillars"].([]any)
		if !ok {
			t.Fatal("pillars is not a slice")
		}
		pillars2 := make([]map[string]string, len(pAny))
		for i, p := range pAny {
			m, ok := p.(map[string]any)
			if !ok {
				t.Fatalf("pillar[%d] is not map", i)
			}
			pillars2[i] = map[string]string{
				"name":    m["name"].(string),
				"stem":    m["stem"].(string),
				"branch":  m["branch"].(string),
				"shiShen": m["shiShen"].(string),
			}
		}
		pillars = pillars2
	}
	if len(pillars) != 4 {
		t.Fatalf("expected 4 pillars, got %d", len(pillars))
	}
	for i, p := range pillars {
		if p["name"] == "" {
			t.Errorf("pillar[%d] missing name", i)
		}
		if p["stem"] == "" {
			t.Errorf("pillar[%d] missing stem", i)
		}
		if p["branch"] == "" {
			t.Errorf("pillar[%d] missing branch", i)
		}
		if p["shiShen"] == "" {
			t.Errorf("pillar[%d] missing shiShen", i)
		}
	}

	// Verify wuxing
	wx, ok := r["wuxing"].(map[string]int)
	if !ok {
		wxAny, ok := r["wuxing"].(map[string]any)
		if !ok {
			t.Fatal("wuxing is not a map")
		}
		wx = make(map[string]int)
		for k, v := range wxAny {
			wx[k] = int(v.(float64))
		}
	}
	total := wx["木"] + wx["火"] + wx["土"] + wx["金"] + wx["水"]
	if total != 8 {
		t.Errorf("expected 8 wuxing elements, got %d", total)
	}

	// Verify dayun
	dayun, ok := r["dayun"].([]map[string]any)
	if !ok {
		dyAny, ok := r["dayun"].([]any)
		if !ok {
			t.Fatal("dayun is not a slice")
		}
		dayun = make([]map[string]any, len(dyAny))
		for i, dy := range dyAny {
			dayun[i] = dy.(map[string]any)
		}
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
	tool := &BaziCalcTool{}
	result, err := tool.Execute(map[string]any{
		"year": float64(1990), "month": float64(5),
		"day": float64(20), "hour": float64(8), "gender": "男",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	data := result.(map[string]any)

	pillars := data["pillars"].([]map[string]string)
	if len(pillars) != 4 {
		t.Errorf("pillars=%d, want 4", len(pillars))
	}
	if data["dayGan"].(string) == "" {
		t.Error("dayGan empty")
	}

	for _, p := range pillars {
		if shiShen, ok := p["shiShen"]; !ok || shiShen == "" {
			t.Errorf("pillar %s shiShen empty", p["name"])
		}
	}

	wuxing := data["wuxing"].(map[string]int)
	total := 0
	for _, v := range wuxing {
		total += v
	}
	if total != 8 {
		t.Errorf("wuxing total=%d, want 8", total)
	}

	dayun := data["dayun"].([]map[string]any)
	if len(dayun) == 0 {
		t.Errorf("dayun is empty")
	}

	t.Logf("日主=%s 五行=%v 大运=%d步", data["dayGan"], wuxing, len(dayun))
}

func TestBaziCalcInvalidYear(t *testing.T) {
	tool := &BaziCalcTool{}
	_, err := tool.Execute(map[string]any{
		"year": float64(1800), "month": float64(1),
		"day": float64(1), "hour": float64(0), "gender": "男",
	})
	if err == nil {
		t.Error("expected error for year 1800")
	}
}
