// This test file belongs to the BaZi deterministic calculation layer.
// It verifies YongShen calculation and protects the related contract from regressions.
// It computes reproducible BaZi facts; it must not generate narrative readings.
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

func TestYongShen_GejuQingZhuoReason_IsDeferredToProfile(t *testing.T) {
	m := execYongShenForTest(t, map[string]any{
		"year": float64(1988), "month": float64(8), "day": float64(8),
		"hour": float64(8), "gender": "男",
	})

	if got := m["geju_qing_zhuo"]; got != "待profile裁断" {
		t.Fatalf("expected deferred qing-zhuo, got %v", got)
	}

	reason, ok := m["geju_qing_zhuo_reason"].(map[string]any)
	if !ok {
		t.Fatalf("expected geju_qing_zhuo_reason map, got %T", m["geju_qing_zhuo_reason"])
	}
	if got := reason["label"]; got != "待profile裁断" {
		t.Fatalf("expected deferred reason label, got %v", got)
	}
	if _, ok := reason["summary"].(string); !ok {
		t.Fatalf("expected reason summary string, got %T", reason["summary"])
	}
}

func TestYongShen_GejuQingZhuoReason_ZhuoZhongYouQingCase(t *testing.T) {
	m := execYongShenForTest(t, map[string]any{
		"year": float64(1965), "month": float64(11), "day": float64(24),
		"hour": float64(0), "gender": "男",
	})

	if got := m["geju_candidate"]; got != "建禄格" {
		t.Fatalf("expected geju candidate=建禄格, got %v", got)
	}
	if got := m["geju_qing_zhuo"]; got != "待profile裁断" {
		t.Fatalf("expected deferred qing-zhuo, got %v", got)
	}

	reason, ok := m["geju_qing_zhuo_reason"].(map[string]any)
	if !ok {
		t.Fatalf("expected geju_qing_zhuo_reason map, got %T", m["geju_qing_zhuo_reason"])
	}
	if got := reason["label"]; got != "待profile裁断" {
		t.Fatalf("expected deferred reason label, got %v", got)
	}
}

func TestYongShen_GejuQingZhuoReason_DoesNotInferBianGeFromBoundedStrength(t *testing.T) {
	m := execYongShenForTest(t, map[string]any{
		"year": float64(1966), "month": float64(6), "day": float64(12),
		"hour": float64(12), "gender": "男",
	})

	if got := m["geju_qing_zhuo"]; got == "变" {
		t.Fatalf("bounded strength must not auto-promote a chart to bian ge, got %v", got)
	}

	reason, ok := m["geju_qing_zhuo_reason"].(map[string]any)
	if !ok {
		t.Fatalf("expected geju_qing_zhuo_reason map, got %T", m["geju_qing_zhuo_reason"])
	}
	if got := reason["label"]; got == "变" {
		t.Fatalf("bounded strength must not emit bian ge reason, got %v", got)
	}
	if got := reason["summary"]; got == "" {
		t.Fatalf("expected non-empty summary, got %v", got)
	}
}

func TestYongShen_GuiWaterHaiMonthCountsMonthCommandAndVisibleCompanion(t *testing.T) {
	m := execYongShenForTest(t, map[string]any{
		"year": float64(2025), "month": float64(11), "day": float64(10),
		"hour": float64(23), "minute": float64(30), "gender": "男",
	})

	if got := m["day_master"]; got != "癸" {
		t.Fatalf("expected day_master=癸, got %v", got)
	}
	if got := m["strength"]; got != "偏强" {
		t.Fatalf("expected strength=偏强, got %v", got)
	}
	evidence, ok := m["strength_evidence"].(map[string]any)
	if !ok {
		t.Fatalf("expected strength evidence, got %T", m["strength_evidence"])
	}
	if support, pressure := evidence["support_score"].(int), evidence["pressure_score"].(int); support <= pressure {
		t.Fatalf("expected support to exceed pressure, got support=%d pressure=%d", support, pressure)
	}
	signals, ok := evidence["support_signals"].([]string)
	if !ok || !strings.Contains(strings.Join(signals, "；"), "同类透干") {
		t.Fatalf("expected visible companion signal, got %#v", evidence["support_signals"])
	}
	if got := m["geju_candidate"]; got != "月劫格" {
		t.Fatalf("expected month-jie candidate, got %v", got)
	}
}

func TestYongShen_UsesSameMinuteAsBaziCalc(t *testing.T) {
	params := map[string]any{
		"year": float64(2025), "month": float64(11), "day": float64(10),
		"hour": float64(23), "minute": float64(30), "gender": "男",
	}
	chartAny, err := (&CalcTool{}).Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("bazi calc: %v", err)
	}
	chart := chartAny.(map[string]any)
	reading := execYongShenForTest(t, params)
	if got, want := reading["day_master"], chart["dayGan"]; got != want {
		t.Fatalf("yongshen day master = %v, bazi calc dayGan = %v", got, want)
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

	if !strings.Contains(combo, "七杀在日支申中本气庚") || !strings.Contains(combo, "七杀在年支巳中余气庚") {
		t.Fatalf("expected geju_combination to preserve hidden-stem locations, got %s", combo)
	}

	if strings.Contains(combo, "[主]") || strings.Contains(combo, "[次]") || strings.Contains(combo, "成立") {
		t.Fatalf("combination candidates must not carry verdict priority or success labels, got %s", combo)
	}
}

func TestYongShen_1991WuShenKeepsHiddenOfficerAndCandidateBoundaries(t *testing.T) {
	m := execYongShenForTest(t, map[string]any{
		"year": float64(1991), "month": float64(10), "day": float64(5),
		"hour": float64(12), "minute": float64(34), "gender": "男",
	})

	if got := m["strength_method"]; got != "balance_evidence_v1" {
		t.Fatalf("expected balance evidence method, got %v", got)
	}
	evidence, ok := m["strength_evidence"].(map[string]any)
	if !ok || evidence["pressure_score"] == nil {
		t.Fatalf("expected bidirectional strength evidence, got %#v", m["strength_evidence"])
	}
	if got := m["strength"]; got == "身旺极" || got == "身弱极" {
		t.Fatalf("expected bounded strength label, got %v", got)
	}
	if basis, _ := m["geju_basis"].(string); strings.Contains(basis, "命盘无官星") {
		t.Fatalf("hidden officer must not be described as absent: %s", basis)
	}
	if geju, _ := m["geju"].(string); strings.Contains(geju, "伤尽") {
		t.Fatalf("tool must not decide 伤尽 before the profile: %s", geju)
	}
	visibility, ok := m["official_visibility"].(map[string]any)
	if !ok {
		t.Fatalf("expected official visibility facts, got %T", m["official_visibility"])
	}
	hidden, ok := visibility["hidden"].([]map[string]string)
	if !ok {
		t.Fatalf("expected hidden officer list, got %T", visibility["hidden"])
	}
	foundYiInWei := false
	for _, item := range hidden {
		if item["pillar"] == "年支" && item["branch"] == "未" && item["stem"] == "乙" && item["tier"] == "余气" {
			foundYiInWei = true
		}
	}
	if !foundYiInWei {
		t.Fatalf("expected 乙 in 未 as hidden officer fact, got %#v", hidden)
	}
	combo, _ := m["geju_combination"].(string)
	for _, forbidden := range []string{"食神生财成立", "富格", "财官相生成立", "贵格"} {
		if strings.Contains(combo, forbidden) {
			t.Fatalf("combination must remain evidence, found %q in %s", forbidden, combo)
		}
	}
}
