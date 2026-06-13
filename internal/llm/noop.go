package llm

import "context"

// NoopClient is a no-op Chat implementation for testing.
type NoopClient struct {
	GenerateFn         func(ctx context.Context, systemPrompt string, messages []Message) (string, TokenUsage, error)
	GenerateWithToolFn func(ctx context.Context, systemPrompt string, messages []Message, tool ToolDef) (map[string]any, TokenUsage, error)
}

func (n *NoopClient) Stream(_ context.Context, _ string, _ []Message, _ func(string)) error {
	return nil
}

func (n *NoopClient) Generate(ctx context.Context, systemPrompt string, messages []Message) (string, TokenUsage, error) {
	if n.GenerateFn != nil {
		return n.GenerateFn(ctx, systemPrompt, messages)
	}
	return `{"action":"followup"}`, TokenUsage{}, nil
}

func (n *NoopClient) GenerateWithTool(ctx context.Context, systemPrompt string, messages []Message, tool ToolDef) (map[string]any, TokenUsage, error) {
	if n.GenerateWithToolFn != nil {
		return n.GenerateWithToolFn(ctx, systemPrompt, messages, tool)
	}
	return map[string]any{"conversation_intent": "consult", "primary_domain": "bazi", "task_intent": "collect_profile", "confidence": 0.5}, TokenUsage{}, nil
}
