// Package knowledge 提供命理知识库检索功能。通过 MCP 协议连接外部知识库服务，
// 将古籍原文、命理规则和解读资料注入到 LLM 的决策上下文中。
package knowledge

import (
	"context"
	"fmt"

	"github.com/wikiglobal/suanming-agent/internal/mcp"
)

// SearchTool 知识库检索工具。根据查询关键词从项目知识库中检索相关命理资料段落。
type SearchTool struct {
	client *mcp.Client
}

// NewSearchTool 创建知识库检索工具，注入外部 MCP 客户端用于连接知识库服务。
func NewSearchTool(client *mcp.Client) *SearchTool {
	return &SearchTool{client: client}
}

func (t *SearchTool) Name() string        { return "knowledge_search" }
func (t *SearchTool) Description() string { return "检索项目知识库中的命理资料" }

func (t *SearchTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required")
	}
	topK := 3
	if v, ok := params["topK"].(float64); ok {
		topK = int(v)
	}
	passages, err := t.client.Search(query, topK)
	if err != nil {
		return map[string]any{"passages": []mcp.Passage{}, "fallback": true}, nil
	}
	return map[string]any{"passages": passages}, nil
}
