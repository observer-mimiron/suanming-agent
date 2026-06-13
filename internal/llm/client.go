package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// TokenUsage reports token consumption for a single LLM call.
type TokenUsage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
}

// ToolDef describes a tool available to the model for structured output.
type ToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
}

// Chat is the interface all LLM clients must implement.
type Chat interface {
	Stream(ctx context.Context, systemPrompt string, messages []Message, onText func(string)) error
	Generate(ctx context.Context, systemPrompt string, messages []Message) (string, TokenUsage, error)
	GenerateWithTool(ctx context.Context, systemPrompt string, messages []Message, tool ToolDef) (map[string]any, TokenUsage, error)
}

type Client struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewClient(apiKey, baseURL, model string, temperature float64) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}
	if baseURL == "" {
		baseURL = getEnv("LLM_BASE_URL", "https://api.deepseek.com/anthropic")
	}
	if model == "" {
		model = getEnv("LLM_MODEL", "deepseek-v4-pro")
	}
	if apiKey == "" {
		log.Println("WARNING: LLM_API_KEY not set — LLM calls will fail")
	}
	_ = temperature // reserved for future use
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *Client) ChatStream(systemPrompt string, messages []Message, onText func(string)) error {
	if c.apiKey == "" {
		return fmt.Errorf("LLM_API_KEY not set")
	}
	body := map[string]any{
		"model":      c.model,
		"system":     systemPrompt,
		"messages":   messages,
		"stream":     true,
		"max_tokens": 4096,
		"thinking":   map[string]string{"type": "disabled"},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("llm marshal: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("llm request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("llm %d (failed to read body: %v)", resp.StatusCode, err)
		}
		return fmt.Errorf("llm %d: %s", resp.StatusCode, string(b))
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" || line == "data: [DONE]" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Text string `json:"text"`
			} `json:"delta"`
		}
		if json.Unmarshal([]byte(data), &event) != nil {
			continue
		}
		if event.Type == "content_block_delta" && event.Delta.Text != "" {
			onText(event.Delta.Text)
		}
	}
}

// Chat sends a non-streaming request and returns the complete response text
func (c *Client) Chat(systemPrompt string, messages []Message) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("LLM_API_KEY not set")
	}
	body := map[string]any{
		"model":       c.model,
		"system":      systemPrompt,
		"messages":    messages,
		"stream":      false,
		"max_tokens":  1024,
		"thinking":    map[string]string{"type": "disabled"},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("llm marshal: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("llm %d (failed to read body: %v)", resp.StatusCode, err)
		}
		return "", fmt.Errorf("llm %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Content) > 0 {
		return result.Content[0].Text, nil
	}
	return "", nil
}

// Stream adapter for Chat interface.
func (c *Client) Stream(_ context.Context, systemPrompt string, messages []Message, onText func(string)) error {
	return c.ChatStream(systemPrompt, messages, onText)
}

// Generate adapter for Chat interface.
func (c *Client) Generate(_ context.Context, systemPrompt string, messages []Message) (string, TokenUsage, error) {
	text, err := c.Chat(systemPrompt, messages)
	return text, TokenUsage{}, err
}

// GenerateWithTool adapter for Chat interface.
func (c *Client) GenerateWithTool(_ context.Context, systemPrompt string, messages []Message, _ ToolDef) (map[string]any, TokenUsage, error) {
	text, err := c.Chat(systemPrompt, messages)
	if err != nil {
		return nil, TokenUsage{}, err
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, TokenUsage{}, fmt.Errorf("structured output parse: %w", err)
	}
	return result, TokenUsage{}, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
