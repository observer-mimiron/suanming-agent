package tools

import (
	"fmt"
	"os"

	"github.com/wikiglobal/suanming-agent/internal/mcp"
)

type KnowledgeSearchTool struct {
	client *mcp.Client
}

func NewKnowledgeSearchTool() *KnowledgeSearchTool {
	url := os.Getenv("KNOWLEDGE_MCP_URL")
	if url == "" {
		url = "http://localhost:3001"
	}
	return &KnowledgeSearchTool{client: mcp.NewClient(url)}
}

func (t *KnowledgeSearchTool) Name() string        { return "knowledge_search" }
func (t *KnowledgeSearchTool) Description() string { return "检索项目知识库中的命理资料" }

func (t *KnowledgeSearchTool) Execute(params map[string]any) (any, error) {
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
