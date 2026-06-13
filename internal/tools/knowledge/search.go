package knowledge

import (
	"context"
	"fmt"

	"github.com/wikiglobal/suanming-agent/internal/mcp"
)

type SearchTool struct {
	client *mcp.Client
}

// NewSearchTool creates a SearchTool with an injected MCP client.
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
