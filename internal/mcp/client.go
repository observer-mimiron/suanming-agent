package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	client  *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, client: &http.Client{Timeout: 5 * time.Second}}
}

type Passage struct {
	Content string `json:"content"`
	Source  string `json:"source"`
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
		return c.SearchYopedia(query, topK) // fallback
	}
	defer resp.Body.Close()
	var result struct {
		Passages []Passage `json:"passages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return c.SearchYopedia(query, topK) // fallback
	}
	if len(result.Passages) == 0 {
		return c.SearchYopedia(query, topK) // fallback
	}
	return result.Passages, nil
}

// SearchYopedia 使用 yopedia REST API 搜索 (/api/wiki/search?q=xxx)
func (c *Client) SearchYopedia(query string, topK int) ([]Passage, error) {
	searchURL := fmt.Sprintf("%s/api/wiki/search?q=%s&limit=%d",
		c.baseURL, url.QueryEscape(query), topK)
	resp, err := c.client.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("yopedia search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("yopedia search returned %d", resp.StatusCode)
	}

	var result struct {
		Results []struct {
			Slug    string `json:"slug"`
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("yopedia parse: %w", err)
	}

	var passages []Passage
	for _, r := range result.Results {
		snippet := strings.TrimSpace(r.Snippet)
		if snippet == "" {
			continue
		}
		passages = append(passages, Passage{
			Content: snippet,
			Source:  fmt.Sprintf("yopedia://%s (%s)", r.Slug, r.Title),
		})
	}
	if len(passages) == 0 {
		return []Passage{}, nil
	}
	return passages, nil
}
