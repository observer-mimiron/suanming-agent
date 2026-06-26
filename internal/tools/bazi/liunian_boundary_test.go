package bazi

import (
	"context"
	"testing"
)

func TestFindCurrentDayun_OutOfRange(t *testing.T) {
	// dayun 列表：4-12 甲戌, 13-22 乙亥, 23-32 丙子
	dayunList := []map[string]any{
		{"startAge": 4, "endAge": 12, "ganZhi": "甲戌"},
		{"startAge": 13, "endAge": 22, "ganZhi": "乙亥"},
		{"startAge": 23, "endAge": 32, "ganZhi": "丙子"},
	}

	// 情况 1：currentAge 小于最小 startAge
	if got := findCurrentDayun(dayunList, 1); len(got) != 0 {
		t.Errorf("findCurrentDayun(age=1) = %v, want empty map (age before first dayun)", got)
	}

	// 情况 2：currentAge 大于最大 endAge
	if got := findCurrentDayun(dayunList, 99); len(got) != 0 {
		t.Errorf("findCurrentDayun(age=99) = %v, want empty map (age after last dayun)", got)
	}

	// 情况 3：在范围内应正常返回
	got := findCurrentDayun(dayunList, 15)
	if len(got) == 0 {
		t.Fatal("findCurrentDayun(age=15) returned empty map, want 乙亥")
	}
	if g, ok := got["ganZhi"].(string); !ok || g != "乙亥" {
		t.Errorf("findCurrentDayun(age=15) ganZhi = %v, want 乙亥", g)
	}

	// 情况 4：空列表
	if got := findCurrentDayun([]map[string]any{}, 10); len(got) != 0 {
		t.Errorf("findCurrentDayun(empty list) = %v, want empty map", got)
	}
}

func TestComputeLiunianChonghe_NoRelation(t *testing.T) {
	// pillars: [寅,辰,亥,申]（来自 buildBaziResultForTest 的四柱地支）
	// liunianZhi=丑 与 [寅,辰,亥,申] 无任何冲合刑害关系：
	//   六冲：丑未→未不在 pillars；相害：丑午→午不在 pillars
	//   三合：巳酉丑→巳/酉不在 pillars；三会：亥子丑→需亥+子，子不在
	//   三刑：丑戌未→需戌+未，均不在；自刑：丑不在 {辰,午,酉,亥}
	// dayunZhi 为空（无当前大运）时同样无关系
	allZhi := []string{"寅", "辰", "亥", "申"}
	items := computeLiunianChonghe("丑", allZhi, "")
	if len(items) != 0 {
		t.Errorf("computeLiunianChonghe(丑) 应无冲合关系，但 got %d 条: %v", len(items), items)
	}
}

func TestBaziLiuNian_YearCompare(t *testing.T) {
	// 使用真实命盘，对比 2025 和 2026 两年的流年输出
	baziResult := buildBaziResultForTest(t)
	lt := &BaziLiuNianTool{}

	for _, tc := range []struct {
		year        int
		wantGanZhi  string
		wantStem    string
		wantBranch  string
	}{
		{2025, "乙巳", "乙", "巳"},
		{2026, "丙午", "丙", "午"},
	} {
		r, err := lt.Execute(context.Background(), map[string]any{
			"target_year": float64(tc.year),
			"bazi_result": baziResult,
		})
		if err != nil {
			t.Fatalf("Execute(target_year=%d) err = %v", tc.year, err)
		}
		m := r.(map[string]any)
		if got := m["liunian_ganzhi"]; got != tc.wantGanZhi {
			t.Errorf("%d liunian_ganzhi = %v, want %s", tc.year, got, tc.wantGanZhi)
		}
		if got := m["liunian_stem"]; got != tc.wantStem {
			t.Errorf("%d liunian_stem = %v, want %s", tc.year, got, tc.wantStem)
		}
		if got := m["liunian_branch"]; got != tc.wantBranch {
			t.Errorf("%d liunian_branch = %v, want %s", tc.year, got, tc.wantBranch)
		}
	}
}

func TestComputeLiunianChonghe_MultiRelation(t *testing.T) {
	// 命局 [寅,辰,亥,申]，流年支=申，同时与多柱发生关系：
	//   六冲：寅申冲 → 申冲年柱寅
	//   相害：申亥害 → 申害日柱亥
	// （空 dayunZhi，避免大运带来的额外关系干扰）
	allZhi := []string{"寅", "辰", "亥", "申"}
	items := computeLiunianChonghe("申", allZhi, "")

	if len(items) < 2 {
		t.Fatalf("computeLiunianChonghe(申) 应至少检测到 2 条关系，got %d: %v", len(items), items)
	}

	hasChong := false
	hasHai := false
	for _, item := range items {
		switch item["type"] {
		case "六冲":
			if item["zhi"] == "寅申" {
				hasChong = true
			}
		case "相害":
			if item["zhi"] == "申亥" {
				hasHai = true
			}
		}
		t.Logf("item: %v", item)
	}

	if !hasChong {
		t.Errorf("computeLiunianChonghe(申) 未检测到寅申六冲，items: %v", items)
	}
	if !hasHai {
		t.Errorf("computeLiunianChonghe(申) 未检测到申亥相害，items: %v", items)
	}
}