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
