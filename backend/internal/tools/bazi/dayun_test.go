// This test file belongs to the BaZi deterministic calculation layer.
// It verifies DaYun calculation and protects the related contract from regressions.
// It computes reproducible BaZi facts; it must not generate narrative readings.
package bazi

import (
	"context"
	"testing"
)

// TestComputeDayunChonghe 验证大运支与命局四支的冲合刑害检测。
// 命局 [寅,辰,亥,申] + 大运支申：
//   - 寅申六冲（年柱寅）
//   - 申亥相害（日柱亥）
func TestComputeDayunChonghe(t *testing.T) {
	allZhi := []string{"寅", "辰", "亥", "申"}
	items := computeDayunChonghe("申", allZhi)

	hasChong := false
	hasHai := false
	for _, item := range items {
		t.Logf("item: %v", item)
		if item["type"] == "六冲" && item["zhi"] == "寅申" {
			hasChong = true
		}
		if item["type"] == "相害" && item["zhi"] == "申亥" {
			hasHai = true
		}
	}
	if !hasChong {
		t.Errorf("computeDayunChonghe(申) 未检测到与年柱寅的寅申六冲，items: %v", items)
	}
	if !hasHai {
		t.Errorf("computeDayunChonghe(申) 未检测到与日柱亥的申亥相害，items: %v", items)
	}
}

// TestComputeDayunChonghe_NoRelation 验证大运支与命局无关系时返回空。
func TestComputeDayunChonghe_NoRelation(t *testing.T) {
	// 命局 [寅,辰,亥,申] + 大运支丑：
	//   六冲：丑未→未不在命局；相害：丑午→午不在命局
	//   三合：巳酉丑→巳/酉不在；三会：亥子丑→需亥+子，子不在
	//   三刑：丑戌未→需戌+未，均不在；自刑：丑不在 {辰,午,酉,亥}
	allZhi := []string{"寅", "辰", "亥", "申"}
	items := computeDayunChonghe("丑", allZhi)
	if len(items) != 0 {
		t.Errorf("computeDayunChonghe(丑) 应无冲合关系，但 got %d 条: %v", len(items), items)
	}
}

// TestDayunAnalyzer_ChongheField 验证 DayunAnalyzer 输出含 dayun_chonghe 字段。
func TestDayunAnalyzer_ChongheField(t *testing.T) {
	baziResult := buildBaziResultForTest(t)
	// 1974-04-28 16:00 男 四柱地支=[寅,辰,亥,申]

	da := &DayunAnalyzer{}
	r, err := da.Execute(context.Background(), map[string]any{
		"dayun":       baziResult["dayun"],
		"bazi_result": baziResult,
	})
	if err != nil {
		t.Fatal(err)
	}

	m, ok := r.(map[string]any)
	if !ok {
		t.Fatalf("Execute 返回类型错误: %T", r)
	}
	annotated, ok := m["dayun_analyzed"].([]map[string]any)
	if !ok {
		t.Fatalf("dayun_analyzed 类型错误: %T", m["dayun_analyzed"])
	}
	if len(annotated) == 0 {
		t.Fatal("dayun_analyzed 为空")
	}

	// 每步大运必须有 dayun_chonghe 字段（[]map[string]string）
	hasNonEmpty := false
	for i, dy := range annotated {
		ch, exists := dy["dayun_chonghe"]
		if !exists {
			t.Errorf("annotated[%d] (%v %v) 缺少 dayun_chonghe 字段（keys: %v）",
				i, dy["startAge"], dy["ganZhi"], dy)
			continue
		}
		chList, ok := ch.([]map[string]string)
		if !ok {
			t.Errorf("annotated[%d] dayun_chonghe 类型错误: %T", i, ch)
			continue
		}
		t.Logf("大运 %v-%v %v: dayun_chonghe = %v",
			dy["startAge"], dy["endAge"], dy["ganZhi"], chList)
		if len(chList) > 0 {
			hasNonEmpty = true
		}
	}

	// 8 步大运 + 命局 [寅,辰,亥,申]，至少应有一运与命局有冲合
	if !hasNonEmpty {
		t.Errorf("所有大运 dayun_chonghe 均为空，需检查命局四柱地支是否正确")
	}
}

func TestDayunAnalyzer_StrictZiZhengLuckRetainsCalendarAndReasonContract(t *testing.T) {
	ctx := context.Background()

	ct := &CalcTool{}
	cr, err := ct.Execute(ctx, map[string]any{
		"year": float64(2025), "month": float64(11), "day": float64(10),
		"hour": float64(23), "gender": "男",
	})
	if err != nil {
		t.Fatal(err)
	}
	baziResult := cr.(map[string]any)

	yt := &YongShenTool{}
	yr, err := yt.Execute(ctx, map[string]any{
		"year": float64(2025), "month": float64(11), "day": float64(10),
		"hour": float64(23), "gender": "男",
	})
	if err != nil {
		t.Fatal(err)
	}
	baziResult["yongshen"] = yr

	da := &DayunAnalyzer{}
	r, err := da.Execute(ctx, map[string]any{
		"dayun":       baziResult["dayun"],
		"bazi_result": baziResult,
	})
	if err != nil {
		t.Fatal(err)
	}

	m, ok := r.(map[string]any)
	if !ok {
		t.Fatalf("Execute 返回类型错误: %T", r)
	}
	annotated, ok := m["dayun_analyzed"].([]map[string]any)
	if !ok {
		t.Fatalf("dayun_analyzed 类型错误: %T", m["dayun_analyzed"])
	}

	findByGanZhi := func(gz string) map[string]any {
		for _, item := range annotated {
			if item["ganZhi"] == gz {
				return item
			}
		}
		return nil
	}

	if item := findByGanZhi("辛巳"); item == nil {
		t.Fatal("expected 辛巳 luck item")
	} else if reason, ok := item["quality_reason"].(map[string]any); !ok {
		t.Fatalf("expected quality_reason map, got %T", item["quality_reason"])
	} else if signals, ok := reason["signals"].(map[string]any); !ok {
		t.Fatalf("expected quality_reason.signals map, got %T", reason["signals"])
	} else if _, ok := signals["base_layer"].(string); !ok {
		t.Fatalf("expected quality_reason.signals.base_layer string, got %T", signals["base_layer"])
	} else if _, ok := item["quality"].(string); !ok {
		t.Fatalf("expected a profile-qualified quality field, got %T", item["quality"])
	}
}

func TestDayunAnalyzer_DoesNotLabelQualityWithoutBalanceVerdict(t *testing.T) {
	dayun := []map[string]any{{"startAge": 30, "endAge": 39, "ganZhi": "甲午"}}
	result := map[string]any{
		"dayGan": "戊",
		"pillars": []map[string]any{
			{"stem": "辛", "branch": "未"}, {"stem": "丁", "branch": "酉"},
			{"stem": "戊", "branch": "申"}, {"stem": "戊", "branch": "午"},
		},
		"yongshen": map[string]any{
			"day_master_wuxing": "土",
			"balance_status":    "待选定流派裁断",
		},
	}
	raw, err := (&DayunAnalyzer{}).Execute(context.Background(), map[string]any{"dayun": dayun, "bazi_result": result})
	if err != nil {
		t.Fatal(err)
	}
	items := raw.(map[string]any)["dayun_analyzed"].([]map[string]any)
	if len(items) != 1 || items[0]["quality"] != "待profile裁断" || items[0]["quality_base"] != "待profile裁断" {
		t.Fatalf("expected no automatic quality label, got %#v", items)
	}
	if _, ok := items[0]["dayun_chonghe"].([]map[string]string); !ok {
		t.Fatalf("relation facts must remain available: %#v", items[0])
	}
}

func TestDayunAnalyzer_ExposesDeterministicBranchTenGodFacts(t *testing.T) {
	raw, err := (&DayunAnalyzer{}).Execute(context.Background(), map[string]any{
		"dayun": []map[string]any{{"startAge": 20, "endAge": 29, "ganZhi": "甲未"}},
		"bazi_result": map[string]any{
			"dayGan":  "甲",
			"pillars": []map[string]any{{"stem": "甲", "branch": "子"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result := raw.(map[string]any)["dayun_analyzed"].([]map[string]any)[0]
	if result["branch"] != "未" {
		t.Fatalf("branch = %v, want 未", result["branch"])
	}
	if got := result["branchHiddenStems"].([]string); len(got) != 3 || got[0] != "己" {
		t.Fatalf("branchHiddenStems = %v, want 未藏干 in canonical order", got)
	}
	if got := result["branchMainTenGod"]; got != "正财" {
		t.Fatalf("branchMainTenGod = %v, want 甲日主见己为正财", got)
	}
}
