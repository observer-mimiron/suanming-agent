// 本测试文件属于紫微 adapter 层。
// 本文件保护 specialist 配置、工具白名单和提示词合同；不测试排盘算法或运行时执行。
package adapter

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

func TestZiweiInstructionAnalysisGuide(t *testing.T) {
	cfg := GetConfig()
	keywords := []string{
		"命宫", "身宫", "三方四正", "四化飞星", "大限", "流年",
		"庙旺利得", "星曜组合",
	}
	for _, kw := range keywords {
		if !containsStr(cfg.Instruction, kw) {
			t.Errorf("instruction missing keyword: %q", kw)
		}
	}
}

func TestZiweiToolNamesOnlyAllowKnowledgeTools(t *testing.T) {
	cfg := GetConfig()
	want := []string{"knowledge_catalog", "knowledge_search"}
	if len(cfg.ToolNames) != len(want) {
		t.Fatalf("ToolNames = %v, want exactly %v", cfg.ToolNames, want)
	}
	for i, name := range want {
		if cfg.ToolNames[i] != name {
			t.Fatalf("ToolNames[%d] = %q, want %q", i, cfg.ToolNames[i], name)
		}
	}
}
