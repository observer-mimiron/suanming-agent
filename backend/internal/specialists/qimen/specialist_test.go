package qimen

import "testing"

func TestToolNamesContainsCatalog(t *testing.T) {
	cfg := GetConfig()
	found := false
	for _, name := range cfg.ToolNames {
		if name == "knowledge_catalog" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ToolNames should contain knowledge_catalog as first tool")
	}
	if len(cfg.ToolNames) == 0 || cfg.ToolNames[0] != "knowledge_catalog" {
		t.Error("knowledge_catalog should be the first tool in ToolNames")
	}
}

func TestInstructionContainsAgenticRAG(t *testing.T) {
	cfg := GetConfig()
	keywords := []string{
		"目录探索", "证据规划", "质量评估", "条件重搜",
		"knowledge_catalog", "系统限制",
	}
	for _, kw := range keywords {
		if !containsStr(cfg.Instruction, kw) {
			t.Errorf("instruction missing keyword: %q", kw)
		}
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
