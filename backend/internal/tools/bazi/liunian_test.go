package bazi

import (
	"context"
	"testing"
)

// buildBaziResultForTest 构造测试用 bazi_result（1974-04-28 16:00 男）。
// 复用 bazi_calc + yongshen 工具产出真实数据，避免手工拼凑。
func buildBaziResultForTest(t *testing.T) map[string]any {
	t.Helper()
	ctx := context.Background()

	ct := &CalcTool{}
	cr, err := ct.Execute(ctx, map[string]any{
		"year": float64(1974), "month": float64(4), "day": float64(28),
		"hour": float64(16), "gender": "男",
	})
	if err != nil {
		t.Fatal(err)
	}
	baziResult := cr.(map[string]any)

	yt := &YongShenTool{}
	yr, err := yt.Execute(ctx, map[string]any{
		"year": float64(1974), "month": float64(4), "day": float64(28),
		"hour": float64(16), "gender": "男",
	})
	if err != nil {
		t.Fatal(err)
	}
	baziResult["yongshen"] = yr

	da := &DayunAnalyzer{}
	dr, err := da.Execute(ctx, map[string]any{
		"dayun":       baziResult["dayun"],
		"bazi_result": baziResult,
	})
	if err != nil {
		t.Fatal(err)
	}
	baziResult["dayun_analyzed"] = dr

	return baziResult
}

func TestBaziLiuNian_Structure(t *testing.T) {
	baziResult := buildBaziResultForTest(t)
	lt := &BaziLiuNianTool{}
	r, err := lt.Execute(context.Background(), map[string]any{
		"target_year": float64(2026),
		"bazi_result": baziResult,
	})
	if err != nil {
		t.Fatal(err)
	}
	m := r.(map[string]any)

	for _, key := range []string{"liunian_year", "liunian_ganzhi", "liunian_stem", "liunian_branch", "liunian_shi_shen", "current_dayun", "liunian_chonghe"} {
		if _, ok := m[key]; !ok {
			t.Errorf("liunian 结果缺少字段 %q（got keys: %v）", key, m)
		}
	}
	t.Logf("liunian 结果：%v", m)
}

func TestBaziLiuNian_KnownHits_2026(t *testing.T) {
	// 1974-04-28 16:00 男 日主己(土)
	// target_year=2026 → 流年丙午
	// shiShenTable["己"]["丙"]="正印"（火生土，丙阳己阴异→正印）
	// currentAge = 2026-1974+1 = 53
	baziResult := buildBaziResultForTest(t)
	lt := &BaziLiuNianTool{}
	r, err := lt.Execute(context.Background(), map[string]any{
		"target_year": float64(2026),
		"bazi_result": baziResult,
	})
	if err != nil {
		t.Fatal(err)
	}
	m := r.(map[string]any)

	if got := m["liunian_ganzhi"]; got != "丙午" {
		t.Errorf("liunian_ganzhi = %v, want 丙午", got)
	}
	if got := m["liunian_shi_shen"]; got != "正印" {
		t.Errorf("liunian_shi_shen = %v, want 正印", got)
	}
	cd, ok := m["current_dayun"].(map[string]any)
	if !ok || len(cd) == 0 {
		t.Fatalf("current_dayun 缺失或为空：%v", m["current_dayun"])
	}
	startAge := toInt(cd["startAge"])
	endAge := toInt(cd["endAge"])
	if got := m["current_dayun_selection"]; got != "date_boundary" {
		t.Errorf("current_dayun_selection = %v, want date_boundary", got)
	}
	// 该命盘的下一步大运在 2026-11 才交接，因此默认年中取样仍在上一运；
	// 虚岁 53 不再被用作整年切换条件。
	if startAge != 43 || endAge != 52 {
		t.Errorf("current_dayun 区间 = %v-%v, want 43-52 before the 2026-11 boundary", startAge, endAge)
	}
	t.Logf("2026 流年：%s，十神：%s，当前大运：%v-%v %v",
		m["liunian_ganzhi"], m["liunian_shi_shen"], startAge, endAge, cd["ganZhi"])
}

func TestComputeLiunianChonghe(t *testing.T) {
	// 合成输入：流年午 vs 命局四柱 [寅,辰,亥,申] + 大运支（假设为子）
	// 预期：流年午冲命局子（若大运支=子）→ 子午六冲
	//       流年午与命局寅→寅午半合（不算，computeLiunianChonghe 只算完整三合/六冲/相刑/相害）
	//       流年午自刑（命局无午，不算）
	liunianZhi := "午"
	allZhi := []string{"寅", "辰", "亥", "申"}
	dayunZhi := "子"
	items := computeLiunianChonghe(liunianZhi, allZhi, dayunZhi)

	// 流年午 + 大运子 → 子午六冲
	found := false
	for _, item := range items {
		if item["type"] == "六冲" && item["zhi"] == "子午" {
			found = true
			t.Logf("命中：%v", item)
			break
		}
	}
	if !found {
		t.Errorf("computeLiunianChonghe 未检测到流年午与大运子的子午六冲，got：%v", items)
	}
}
