package tools

import (
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/mcp"
)

func TestKnowledgeCatalogTool_Name(t *testing.T) {
	tool := &KnowledgeCatalogTool{}
	if tool.Name() != "knowledge_catalog" {
		t.Errorf("expected name 'knowledge_catalog', got %s", tool.Name())
	}
}

func TestSummarizeCatalog_FiltersBySlugPrefix(t *testing.T) {
	nodes := []mcp.GraphNode{
		{ID: "ref-bazi-ziping", Label: "子平真诠评注", Tags: []string{"八字", "古籍", "目录"}},
		{ID: "ref-bazi-ziping-s001", Label: "方重审序", Tags: []string{"八字", "古籍"}},
		{ID: "ref-bazi-sanming-s045", Label: "官煞混杂", Tags: []string{"八字", "古籍"}},
	}
	edges := []mcp.GraphEdge{
		{Source: "ref-bazi-ziping", Target: "ref-bazi-ziping-s001"},
		{Source: "ref-bazi-ziping", Target: "ref-bazi-sanming-s045"},
	}

	books := summarizeCatalog(nodes, edges)
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}
	b := books[0]
	chapters, _ := b["chapters"].(int)
	if chapters != 1 {
		t.Errorf("expected 1 chapter (cross-book edge filtered), got %v", b["chapters"])
	}
	name, _ := b["name"].(string)
	if name != "子平真诠评注" {
		t.Errorf("expected '子平真诠评注', got %v", b["name"])
	}
}
