package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/mcp"
)

// KnowledgeCatalogTool 知识库目录探索工具。通过图遍历统计每部古籍的章节数。
type KnowledgeCatalogTool struct {
	client *mcp.Client
}

// NewKnowledgeCatalogTool 创建一个知识库目录探索工具。
func NewKnowledgeCatalogTool(client *mcp.Client) *KnowledgeCatalogTool {
	return &KnowledgeCatalogTool{client: client}
}

func (t *KnowledgeCatalogTool) Name() string { return "knowledge_catalog" }

func (t *KnowledgeCatalogTool) Description() string {
	return "获取知识库的目录结构。返回每部古籍的名称、章节数和前 5 个章节标题。" +
		"在首次检索前调用此工具了解有哪些资料可用。"
}

func (t *KnowledgeCatalogTool) Execute(_ context.Context, _ map[string]any) (any, error) {
	nodes, edges, err := t.client.GetGraph()
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("catalog unavailable: %v", err)}, nil
	}
	return map[string]any{
		"total_pages": len(nodes),
		"books":       summarizeCatalog(nodes, edges),
	}, nil
}

// summarizeCatalog 通过图遍历生成目录摘要。
//
// 边由 markdown 正文链接生成（不是系统目录树），因此额外加 slug 前缀过滤：
// 目录页只统计以自身 slug 为前缀的目标节点，排除跨书引用。
// 例如：ref-bazi-ziping → ref-bazi-sanming-s045（跨书引用）会被过滤。
func summarizeCatalog(nodes []mcp.GraphNode, edges []mcp.GraphEdge) []map[string]any {
	nodeMap := map[string]mcp.GraphNode{}
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	var books []map[string]any
	for _, n := range nodes {
		if !hasTag(n.Tags, "目录") {
			continue
		}
		var chapterTitles []string
		for _, e := range edges {
			if e.Source != n.ID {
				continue
			}
			// slug 前缀过滤：只保留同书章节，排除跨书引用
			if !strings.HasPrefix(e.Target, n.ID+"-") && !strings.HasPrefix(e.Target, n.ID+"_") {
				continue
			}
			if target, ok := nodeMap[e.Target]; ok {
				chapterTitles = append(chapterTitles, target.Label)
			}
		}
		sample := chapterTitles
		if len(sample) > 5 {
			sample = sample[:5]
		}
		books = append(books, map[string]any{
			"name":          n.Label,
			"slug":          n.ID,
			"chapters":      len(chapterTitles),
			"sample_titles": sample,
			"tags":          n.Tags,
		})
	}
	return books
}

func hasTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}
