package llm

import "context"

// Message 表示一条 LLM 对话消息，包含角色和内容。
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// TokenUsage 报告一次 LLM 调用的 Token 消耗（输入和输出）。
type TokenUsage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
}

// Chat 是所有 LLM 客户端必须实现的统一接口，支持流式和非流式生成。
type Chat interface {
	Stream(ctx context.Context, systemPrompt string, messages []Message, onText func(string)) error
	Generate(ctx context.Context, systemPrompt string, messages []Message) (string, TokenUsage, error)
}
