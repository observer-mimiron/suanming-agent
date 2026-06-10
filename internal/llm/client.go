package llm

import (
	"bufio"
	"bytes"
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

type Client struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewClient() *Client {
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		log.Println("WARNING: LLM_API_KEY not set — LLM calls will fail")
	}
	return &Client{
		baseURL: getEnv("LLM_BASE_URL", "https://api.deepseek.com/anthropic"),
		apiKey:  apiKey,
		model:   getEnv("LLM_MODEL", "deepseek-v4-pro"),
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

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
