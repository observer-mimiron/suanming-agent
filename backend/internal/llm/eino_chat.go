// This file belongs to the LLM adapter layer.
// It owns Eino chat-model adapter behavior for this package.
// It wraps model providers; domain prompts and contracts stay outside this package.
package llm

import (
	"context"
	"io"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// EinoChat 是 Chat 接口的 Eino 实现，包装 einomodel.ToolCallingChatModel 以支持流式和非流式调用。
type EinoChat struct {
	model einomodel.ToolCallingChatModel
}

// NewEinoChat 创建一个包装给定 Eino ToolCallingChatModel 的 Chat 实例。
func NewEinoChat(model einomodel.ToolCallingChatModel) *EinoChat {
	return &EinoChat{model: model}
}

// IsEinoChat 判断一个 Chat 实例是否由 EinoChat 实现（底层为 Eino 框架）。
func IsEinoChat(chat Chat) bool {
	_, ok := chat.(*EinoChat)
	return ok
}

func (c *EinoChat) Stream(ctx context.Context, systemPrompt string, messages []Message, onText func(string)) error {
	stream, err := c.model.Stream(ctx, toEinoMessages(systemPrompt, messages))
	if err != nil {
		return err
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if chunk == nil || chunk.ReasoningContent != "" || chunk.Content == "" {
			continue
		}
		onText(chunk.Content)
	}
}

func (c *EinoChat) Generate(ctx context.Context, systemPrompt string, messages []Message) (string, TokenUsage, error) {
	msg, err := c.model.Generate(ctx, toEinoMessages(systemPrompt, messages))
	if err != nil {
		return "", TokenUsage{}, err
	}
	return msg.Content, toTokenUsage(msg), nil
}

func toEinoMessages(systemPrompt string, messages []Message) []*schema.Message {
	out := make([]*schema.Message, 0, len(messages)+1)
	if systemPrompt != "" {
		out = append(out, schema.SystemMessage(systemPrompt))
	}

	for _, msg := range messages {
		switch msg.Role {
		case "assistant":
			out = append(out, schema.AssistantMessage(msg.Content, nil))
		case "system":
			out = append(out, schema.SystemMessage(msg.Content))
		default:
			out = append(out, schema.UserMessage(msg.Content))
		}
	}

	return out
}

func toTokenUsage(msg *schema.Message) TokenUsage {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return TokenUsage{}
	}
	return TokenUsage{
		Input:  msg.ResponseMeta.Usage.PromptTokens,
		Output: msg.ResponseMeta.Usage.CompletionTokens,
	}
}
