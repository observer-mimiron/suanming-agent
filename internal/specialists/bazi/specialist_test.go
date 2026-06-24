package bazi

import (
	"os"
	"testing"
)

func TestBaziToolNamesContainsCatalog(t *testing.T) {
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

// TestBaziInstructionContainsAgenticRAG 验证 prompts/interpret.md 包含 agentic RAG
// 检索流程的关键步骤和工具声明。loadInstruction() 在运行时从项目根目录读取文件，
// 测试从包目录运行，因此直接读文件验证内容。
func TestBaziInstructionContainsAgenticRAG(t *testing.T) {
	data, err := os.ReadFile("../../../prompts/interpret.md")
	if err != nil {
		t.Skipf("prompts/interpret.md not found: %v", err)
	}
	instruction := string(data)
	keywords := []string{
		"目录探索", "证据规划", "质量评估", "条件重搜",
		"knowledge_catalog", "系统限制",
	}
	for _, kw := range keywords {
		if !containsStr(instruction, kw) {
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
