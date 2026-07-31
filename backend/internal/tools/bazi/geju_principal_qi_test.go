package bazi

import (
	"strings"
	"testing"
)

func TestYongShen_UsesPrincipalQiForGuiHaiMonthGeju(t *testing.T) {
	m := execYongShenForTest(t, map[string]any{
		"year": float64(2025), "month": float64(11), "day": float64(10),
		"hour": float64(23), "gender": "男",
	})

	if got := m["day_master"]; got != "癸" {
		t.Fatalf("expected day_master=癸, got %v", got)
	}
	if got := m["geju_candidate"]; got != "月劫格" {
		t.Fatalf("expected geju candidate=月劫格, got %v", got)
	}

	basis, ok := m["geju_basis"].(string)
	if !ok {
		t.Fatalf("expected geju_basis string, got %T", m["geju_basis"])
	}
	if !strings.Contains(basis, "月令亥本气壬") {
		t.Fatalf("expected geju_basis to mention principal qi 壬, got %q", basis)
	}
	if strings.Contains(basis, "中气甲") || strings.Contains(basis, "暗格伤官") {
		t.Fatalf("expected geju_basis to stop using middle-qi hidden shangguan fallback, got %q", basis)
	}
}
