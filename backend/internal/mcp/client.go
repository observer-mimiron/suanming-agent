// Package mcp 实现 Model Context Protocol 客户端，用于连接知识库服务进行语义搜索。
// 同时支持通用 MCP 搜索协议和本地知识库 REST API 两种检索方式，带自动降级。
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
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

const (
	// SearchFailureTimeout 表示知识库请求超过客户端等待上限。
	SearchFailureTimeout = "timeout"
	// SearchFailureHTTP 表示知识库返回了非成功 HTTP 状态。
	SearchFailureHTTP = "http_error"
	// SearchFailureParse 表示知识库成功响应无法按检索合同解析。
	SearchFailureParse = "parse_error"
	// SearchFailureService 表示未被进一步识别的知识库服务故障。
	SearchFailureService = "service_error"
)

type searchFailure struct {
	kind string
	err  error
}

// Error 返回底层知识库故障文本。
func (e *searchFailure) Error() string { return e.err.Error() }

// Unwrap 暴露底层错误，保留标准库的超时和网络错误判断能力。
func (e *searchFailure) Unwrap() error { return e.err }

// SearchFailureKind 将知识库检索错误归类为可安全记录的失败原因。
func SearchFailureKind(err error) string {
	var classified *searchFailure
	if errors.As(err, &classified) {
		return classified.kind
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return SearchFailureTimeout
	}
	var networkErr net.Error
	if errors.As(err, &networkErr) && networkErr.Timeout() {
		return SearchFailureTimeout
	}
	return SearchFailureService
}

// NewClient 创建一个 MCP 客户端，连接到指定 baseURL 的知识库服务。
func NewClient(baseURL string) *Client {
	ensureLogDir()
	// 个别古籍 hybrid 检索已接近十秒；留出余量避免把慢响应误作空结果。
	return &Client{baseURL: baseURL, client: &http.Client{Timeout: 15 * time.Second}}
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

// Search 调用知识库 REST API 搜索。保留方法名向后兼容调用方。
//
// 之前先试 POST /search 通用 MCP 协议，但 wiki 服务未实现该路由（每次 404
// 返回 HTML，json.Decode 失败后降级到 SearchKnowledge），纯浪费一轮 HTTP。
// 直接走 REST API。
func (c *Client) Search(query string, topK int) ([]Passage, error) {
	return c.SearchKnowledge(query, topK)
}

// SearchKnowledge 使用知识库 hybrid retrieval API 搜索 (/api/wiki/retrieve?q=xxx)
func (c *Client) SearchKnowledge(query string, topK int) ([]Passage, error) {
	searchURL := fmt.Sprintf("%s/api/wiki/retrieve?q=%s&limit=%d",
		c.baseURL, url.QueryEscape(query), topK)
	resp, err := c.client.Get(searchURL)
	if err != nil {
		logMCP("SearchKnowledge", err)
		return nil, fmt.Errorf("knowledge search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, &searchFailure{kind: SearchFailureHTTP, err: fmt.Errorf("knowledge search returned %d", resp.StatusCode)}
	}

	var result struct {
		Results []struct {
			Slug    string `json:"slug"`
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, &searchFailure{kind: SearchFailureParse, err: fmt.Errorf("knowledge parse: %w", err)}
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
