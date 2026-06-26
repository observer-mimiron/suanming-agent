package bazi

import (
	"context"
	"testing"
)

func TestYongShen_ChongheStructure(t *testing.T) {
	yt := &YongShenTool{}
	r, err := yt.Execute(context.Background(), map[string]any{
		"year": float64(1974), "month": float64(4), "day": float64(28),
		"hour": float64(16), "gender": "男",
	})
	if err != nil {
		t.Fatal(err)
	}
	m := r.(map[string]any)

	ch, ok := m["chonghe"].([]map[string]string)
	if !ok {
		t.Fatalf("chonghe 字段缺失或类型错误：got %T", m["chonghe"])
	}
	if ch == nil {
		t.Fatal("chonghe 为 nil，应为空 slice")
	}
	for i, item := range ch {
		for _, key := range []string{"type", "zhi", "pillars", "description"} {
			if _, ok := item[key]; !ok {
				t.Errorf("chonghe[%d] 缺少 key %q", i, key)
			}
		}
	}
	t.Logf("chonghe 共 %d 条：%v", len(ch), ch)
}

func TestYongShen_ChongheKnownHits_1974(t *testing.T) {
	// 1974-04-28 16:00 男 四柱：甲寅 / 戊辰 / 己亥 / 壬申
	// 地支：寅(年) 辰(月) 亥(日) 申(时)
	// 预期：寅申冲(年时) + 申亥害(日时)
	yt := &YongShenTool{}
	r, err := yt.Execute(context.Background(), map[string]any{
		"year": float64(1974), "month": float64(4), "day": float64(28),
		"hour": float64(16), "gender": "男",
	})
	if err != nil {
		t.Fatal(err)
	}
	m := r.(map[string]any)
	ch, _ := m["chonghe"].([]map[string]string)

	expected := []map[string]string{
		{"type": "六冲", "zhi": "寅申", "pillars": "年时", "description": "年时寅申冲"},
		{"type": "相害", "zhi": "申亥", "pillars": "日时", "description": "日时申亥害"},
	}
	if len(ch) != len(expected) {
		t.Fatalf("chonghe 数量不符：expected %d, got %d (%v)", len(expected), len(ch), ch)
	}
	for _, exp := range expected {
		found := false
		for _, got := range ch {
			if got["type"] == exp["type"] && got["zhi"] == exp["zhi"] {
				if got["pillars"] != exp["pillars"] || got["description"] != exp["description"] {
					t.Errorf("chonghe %s%s 内容不符：expected %v, got %v",
						exp["type"], exp["zhi"], exp, got)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("未找到预期 chonghe：%v", exp)
		}
	}
}

func TestYongShen_ShiShenPower_1974(t *testing.T) {
	// 1974-04-28 16:00 男 日主己(土)
	// 预期 weighted：劫财=2.5, 正官=2.0, 正财=2.0, 正印=0.5, 七杀=0.5, 偏财=0.5, 伤官=0.5
	yt := &YongShenTool{}
	r, err := yt.Execute(context.Background(), map[string]any{
		"year": float64(1974), "month": float64(4), "day": float64(28),
		"hour": float64(16), "gender": "男",
	})
	if err != nil {
		t.Fatal(err)
	}
	m := r.(map[string]any)

	ssp, ok := m["shi_shen_power"].(map[string]map[string]float64)
	if !ok {
		t.Fatalf("shi_shen_power 字段缺失或类型错误：got %T", m["shi_shen_power"])
	}
	if len(ssp) < 5 {
		t.Fatalf("shi_shen_power 条目过少：got %d", len(ssp))
	}
	for _, key := range []string{"gan_count", "zhi_count", "total", "weighted"} {
		for god, item := range ssp {
			if _, ok := item[key]; !ok {
				t.Errorf("shi_shen_power[%s] 缺少 key %q", god, key)
			}
		}
	}
	cases := map[string]float64{
		"劫财": 2.5, "正官": 2.0, "正财": 2.0,
		"正印": 0.5, "七杀": 0.5, "偏财": 0.5, "伤官": 0.5,
	}
	for god, want := range cases {
		item, ok := ssp[god]
		if !ok {
			t.Errorf("shi_shen_power 缺少十神 %q", god)
			continue
		}
		if got := item["weighted"]; got != want {
			t.Errorf("shi_shen_power[%s].weighted = %v, want %v", god, got, want)
		}
	}
	t.Logf("shi_shen_power：%v", ssp)
}
