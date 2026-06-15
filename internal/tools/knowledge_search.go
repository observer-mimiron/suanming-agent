package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino/schema"
	"github.com/wikiglobal/suanming-agent/internal/mcp"
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

func (t *KnowledgeSearchTool) Name() string        { return "knowledge_search" }
func (t *KnowledgeSearchTool) Description() string { return "检索项目知识库中的命理资料" }
func (t *KnowledgeSearchTool) EinoToolInfo() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: t.Description(),
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "检索问题或关键词",
				Required: true,
			},
			"topK": {
				Type: schema.Number,
				Desc: "返回条数，默认 3",
			},
		}),
	}
}

func (t *KnowledgeSearchTool) Execute(_ context.Context, params map[string]any) (any, error) {
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
