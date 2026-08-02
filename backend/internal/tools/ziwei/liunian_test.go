package ziwei

import (
	"context"
	"testing"
)

func TestZiWeiLiuNianTool_Execute(t *testing.T) {
	tool := &ZiWeiLiuNianTool{}
	if tool.Name() != "ziwei_liunian" {
		t.Errorf("expected name 'ziwei_liunian', got %s", tool.Name())
	}
	desc := tool.Description()
	if !stringsContains(desc, "流年") {
		t.Errorf("description should mention 流年")
	}

	params := map[string]any{
		"year":        1990.0,
		"month":       5.0,
		"day":         20.0,
		"hour":        8.0,
		"gender":      "男",
		"target_year": 2026.0,
		"age":         37.0,
	}
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatal("result should be map[string]any")
	}
	if _, ok := m["year_stem"]; !ok {
		t.Error("result should have year_stem")
	}
	if _, ok := m["year_branch"]; !ok {
		t.Error("result should have year_branch")
	}
	if _, ok := m["age_palace"]; !ok {
		t.Error("result should have age_palace")
	}
}

func stringsContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
