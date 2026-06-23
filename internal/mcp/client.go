// Package mcp 实现 Model Context Protocol 客户端，用于连接知识库服务进行语义搜索。
// 同时支持通用 MCP 搜索协议和本地知识库 REST API 两种检索方式，带自动降级。
package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient 创建一个 MCP 客户端，连接到指定 baseURL 的知识库服务。
func NewClient(baseURL string) *Client {
	ensureLogDir()
	return &Client{baseURL: baseURL, client: &http.Client{Timeout: 10 * time.Second}}
}

// Passage 是一条知识库检索结果，包含文本内容和来源标识。
type Passage struct {
	Content string `json:"content"`
	Source  string `json:"source"`
}

// GraphNode 是知识库图谱中的一个页面节点。
type GraphNode struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Tags      []string `json:"tags"`
	LinkCount int      `json:"linkCount"`
	Tenant    string   `json:"tenant"`
}

// GraphEdge 是知识库图谱中的一条链接边。边由页面正文中的 markdown 链接扫描生成。
// Source 和 Target 都是 page slug。
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// GetGraph 获取知识库完整的图结构（节点 + 边）。
//
// 边由 markdown 正文链接生成，不是系统目录树——但目录页（tags 含 "目录"）的正文
// 恰好以链接形式列出了章节，可用作章节统计。
func (c *Client) GetGraph() ([]GraphNode, []GraphEdge, error) {
	resp, err := c.client.Get(c.baseURL + "/api/wiki/graph")
	if err != nil {
		logMCP("GetGraph", err)
		return nil, nil, fmt.Errorf("get graph: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("get graph returned %d", resp.StatusCode)
	}
	var result struct {
		Nodes []GraphNode `json:"nodes"`
		Edges []GraphEdge `json:"edges"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("decode graph: %w", err)
	}
	return result.Nodes, result.Edges, nil
}


// Search 通用 MCP 搜索（向后兼容）
func (c *Client) Search(query string, topK int) ([]Passage, error) {
	body := map[string]any{"query": query, "limit": topK}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("rag marshal: %w", err)
	}
	resp, err := c.client.Post(c.baseURL+"/search", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		logMCP("Search POST /search", err)
		return c.SearchKnowledge(query, topK) // 降级搜索
	}
	defer resp.Body.Close()
	var result struct {
		Passages []Passage `json:"passages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return c.SearchKnowledge(query, topK) // 降级搜索
	}
	if len(result.Passages) == 0 {
		return c.SearchKnowledge(query, topK) // 降级搜索
	}
	return result.Passages, nil
}

// SearchKnowledge 使用知识库 REST API 搜索 (/api/wiki/search?q=xxx)
func (c *Client) SearchKnowledge(query string, topK int) ([]Passage, error) {
	searchURL := fmt.Sprintf("%s/api/wiki/search?q=%s&limit=%d",
		c.baseURL, url.QueryEscape(query), topK)
	resp, err := c.client.Get(searchURL)
	if err != nil {
		logMCP("SearchKnowledge", err)
		return nil, fmt.Errorf("knowledge search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("knowledge search returned %d", resp.StatusCode)
	}

	var result struct {
		Results []struct {
			Slug    string `json:"slug"`
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("knowledge parse: %w", err)
	}

	var passages []Passage
	for _, r := range result.Results {
		snippet := strings.TrimSpace(r.Snippet)
		if snippet == "" {
			continue
		}
		passages = append(passages, Passage{
			Content: snippet,
			Source:  fmt.Sprintf("knowledge://%s (%s)", r.Slug, r.Title),
		})
	}
	if len(passages) == 0 {
		return []Passage{}, nil
	}
	return passages, nil
}

// ——— error logging ———

var mcpLogPath = "logs/mcp_error.log"

func ensureLogDir() {
	os.MkdirAll("logs", 0755)
}

func logMCP(method string, err error) {
	f, openErr := os.OpenFile(mcpLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if openErr != nil {
		log.Printf("mcp: cannot open error log: %v", openErr)
		return
	}
	defer f.Close()
	logger := log.New(f, "", log.LstdFlags)
	// ponytail: global file lock; per-client lock if contention matters.
	logger.Printf("%s: %v", method, err)
	// Also write to stderr so `go run` shows it immediately.
	log.Printf("mcp error [%s]: %v", method, err)
}
