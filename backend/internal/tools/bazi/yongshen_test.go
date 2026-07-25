package bazi

import (
	"context"
	"strings"
	"testing"
)

func execYongShenForTest(t *testing.T, params map[string]any) map[string]any {
	t.Helper()

	yt := &YongShenTool{}
	r, err := yt.Execute(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	return r.(map[string]any)
}

func TestYongShen(t *testing.T) {
	m := execYongShenForTest(t, map[string]any{
		"year": float64(1974), "month": float64(4), "day": float64(28),
		"hour": float64(16), "gender": "男",
	})
	t.Logf("日主: %s(%s) 季节: %s 强弱: %s", m["day_master"], m["day_master_wuxing"], m["season"], m["strength"])
	t.Logf("用神: %v  喜神: %v  忌神: %v", m["yong_shen"], m["xi_shen"], m["ji_shen"])
	t.Logf("调候: %s", m["tiao_hou"])
	t.Logf("支撑分: 月令=%v 根=%v 同元=%v 生扶=%v 总分=%v", m["month_score"], m["root_count"], m["same_element"], m["generate_count"], m["total_support"])
}

func TestYongShen_GejuQingZhuoReason_QingCase(t *testing.T) {
	m := execYongShenForTest(t, map[string]any{
		"year": float64(1988), "month": float64(8), "day": float64(8),
		"hour": float64(8), "gender": "男",
	})

	if got := m["geju_qing_zhuo"]; got != "清" {
		t.Fatalf("expected geju_qing_zhuo=清, got %v", got)
	}

	reason, ok := m["geju_qing_zhuo_reason"].(map[string]any)
	if !ok {
		t.Fatalf("expected geju_qing_zhuo_reason map, got %T", m["geju_qing_zhuo_reason"])
	}
	if got := reason["label"]; got != "清" {
		t.Fatalf("expected reason label=清, got %v", got)
	}
	if _, ok := reason["summary"].(string); !ok {
		t.Fatalf("expected reason summary string, got %T", reason["summary"])
	}
	if signals, ok := reason["signals"].(map[string]any); !ok {
		t.Fatalf("expected reason signals map, got %T", reason["signals"])
	} else if got := signals["month_order_revealed"]; got != true {
		t.Fatalf("expected month_order_revealed=true, got %v", got)
	}
}

func TestYongShen_GejuQingZhuoReason_ZhuoZhongYouQingCase(t *testing.T) {
	m := execYongShenForTest(t, map[string]any{
		"year": float64(1965), "month": float64(11), "day": float64(24),
		"hour": float64(0), "gender": "男",
	})

	if got := m["geju"]; got != "建禄格" {
		t.Fatalf("expected geju=建禄格, got %v", got)
	}
	if got := m["geju_qing_zhuo"]; got != "浊中有清" {
		t.Fatalf("expected geju_qing_zhuo=浊中有清, got %v", got)
	}

	reason, ok := m["geju_qing_zhuo_reason"].(map[string]any)
	if !ok {
		t.Fatalf("expected geju_qing_zhuo_reason map, got %T", m["geju_qing_zhuo_reason"])
	}
	if got := reason["label"]; got != "浊中有清" {
		t.Fatalf("expected reason label=浊中有清, got %v", got)
	}
	if signals, ok := reason["signals"].(map[string]any); !ok {
		t.Fatalf("expected reason signals map, got %T", reason["signals"])
	} else {
		if got := signals["month_order_revealed"]; got != false {
			t.Fatalf("expected month_order_revealed=false, got %v", got)
		}
		if got := signals["has_relief"]; got != true {
			t.Fatalf("expected has_relief=true, got %v", got)
		}
	}
}

func TestYongShen_GejuQingZhuoReason_BianCase(t *testing.T) {
	m := execYongShenForTest(t, map[string]any{
		"year": float64(1966), "month": float64(6), "day": float64(12),
		"hour": float64(12), "gender": "男",
	})

	if got := m["geju_qing_zhuo"]; got != "变" {
		t.Fatalf("expected geju_qing_zhuo=变, got %v", got)
	}

	reason, ok := m["geju_qing_zhuo_reason"].(map[string]any)
	if !ok {
		t.Fatalf("expected geju_qing_zhuo_reason map, got %T", m["geju_qing_zhuo_reason"])
	}
	if got := reason["label"]; got != "变" {
		t.Fatalf("expected reason label=变, got %v", got)
	}
	if got := reason["summary"]; got == "" {
		t.Fatalf("expected non-empty summary, got %v", got)
	}
}

func TestYongShen_StrongJiaWoodPrefersHuoTuJin(t *testing.T) {
	m := execYongShenForTest(t, map[string]any{
		"year": float64(2025), "month": float64(11), "day": float64(10),
		"hour": float64(23), "gender": "男",
	})

	if got := m["day_master"]; got != "癸" {
		t.Fatalf("expected day_master=癸, got %v", got)
	}
	if got := m["strength"]; got == "" {
		t.Fatalf("expected non-empty strength, got %v", got)
	}
}

func TestYongShen_GejuCombinationDistinguishesHiddenStemStrength(t *testing.T) {
	m := execYongShenForTest(t, map[string]any{
		"year": float64(2025), "month": float64(11), "day": float64(11),
		"hour": float64(0), "gender": "男",
	})

	combo, ok := m["geju_combination"].(string)
	if !ok {
		t.Fatalf("expected geju_combination string, got %T", m["geju_combination"])
	}

	for _, want := range []string{"日支申中本气庚", "年支巳中余气庚"} {
		if !strings.Contains(combo, want) {
			t.Fatalf("expected geju_combination to contain %q, got %s", want, combo)
		}
	}

	if !strings.Contains(combo, "七杀在日支申中本气庚、年支巳中余气庚") {
		t.Fatalf("expected geju_combination to order hidden-stem evidence by strength, got %s", combo)
	}

	if strings.Contains(combo, "七杀在年支、日支") {
		t.Fatalf("expected geju_combination to avoid flattening hidden-stem strength, got %s", combo)
	}
}
