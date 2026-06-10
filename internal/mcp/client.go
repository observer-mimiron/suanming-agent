package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

func (c *Client) Search(query string, topK int) ([]Passage, error) {
	body := map[string]any{"query": query, "limit": topK}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("rag marshal: %w", err)
	}
	resp, err := c.client.Post(c.baseURL+"/search", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("rag search: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		Passages []Passage `json:"passages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Passages, nil
}
