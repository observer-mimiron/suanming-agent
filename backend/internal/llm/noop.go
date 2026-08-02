package llm

import "context"

// NoopClient 是 Chat 接口的 No-op 实现，专用于测试。
// 可通过 GenerateFn 注入自定义行为；未设置时返回结构化默认值。
type NoopClient struct {
	GenerateFn func(ctx context.Context, systemPrompt string, messages []Message) (string, TokenUsage, error)
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
