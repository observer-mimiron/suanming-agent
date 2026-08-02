package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/observer-mimiron/suanming-agent/internal/mcp"
)

// KnowledgeSearchTool 知识库检索工具。通过 MCP 客户端连接项目知识库，检索命理资料注入到 LLM 解读中。
type KnowledgeSearchTool struct {
	client *mcp.Client
}

func NewKnowledgeSearchTool(client *mcp.Client) *KnowledgeSearchTool {
	return &KnowledgeSearchTool{client: client}
}

// NewKnowledgeSearchToolFromEnv 从环境变量创建知识库检索工具。优先读取 KNOWLEDGE_MCP_URL 环境变量，默认连接 http://localhost:3100。
func NewKnowledgeSearchToolFromEnv() *KnowledgeSearchTool {
	url := os.Getenv("KNOWLEDGE_MCP_URL")
	if url == "" {
		url = "http://localhost:3100"
	}
	return &KnowledgeSearchTool{client: mcp.NewClient(url)}
}

func (t *KnowledgeSearchTool) Name() string { return "knowledge_search" }
func (t *KnowledgeSearchTool) Description() string {
	return "在命理古籍知识库中检索原文。query 使用核心术语，优先用典籍名+章节名限定范围。" +
		"返回 passages 数组（content=原文片段, source=来源页面标题）。" +
		"评估质量：content 是否聚焦、是否可引用、source 来自哪部典籍。"
}

func (t *KnowledgeSearchTool) Label() string { return "知识检索" }

func (t *KnowledgeSearchTool) Execute(_ context.Context, params map[string]any) (any, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required")
	}
	topK := 3
	if v, ok := params["top_k"].(float64); ok {
		topK = int(v)
	}
	passages, err := t.client.Search(query, topK)
	if err != nil {
		return map[string]any{"passages": []mcp.Passage{}, "fallback": true}, nil
	}
	return map[string]any{"passages": passages}, nil
}
