package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	einojsonschema "github.com/eino-contrib/jsonschema"
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

func (c *EinoChat) GenerateWithTool(ctx context.Context, systemPrompt string, messages []Message, tool ToolDef) (map[string]any, TokenUsage, error) {
	ti, err := toToolInfo(tool)
	if err != nil {
		return nil, TokenUsage{}, err
	}

	bound, err := c.model.WithTools([]*schema.ToolInfo{ti})
	if err != nil {
		return nil, TokenUsage{}, err
	}

	msg, err := bound.Generate(
		ctx,
		toEinoMessages(systemPrompt, messages),
		einomodel.WithToolChoice(schema.ToolChoiceForced, tool.Name),
	)
	if err != nil {
		return nil, TokenUsage{}, err
	}

	parser := schema.NewMessageJSONParser[map[string]any](&schema.MessageJSONParseConfig{
		ParseFrom: schema.MessageParseFromToolCall,
	})
	parsed, err := parser.Parse(ctx, msg)
	if err != nil {
		return nil, TokenUsage{}, err
	}

	return parsed, toTokenUsage(msg), nil
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

func toToolInfo(tool ToolDef) (*schema.ToolInfo, error) {
	var js *einojsonschema.Schema
	if tool.InputSchema != nil {
		b, err := json.Marshal(tool.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("marshal tool schema: %w", err)
		}
		js = &einojsonschema.Schema{}
		if err := json.Unmarshal(b, js); err != nil {
			return nil, fmt.Errorf("unmarshal tool schema: %w", err)
		}
	}

	return &schema.ToolInfo{
		Name:        tool.Name,
		Desc:        tool.Description,
		ParamsOneOf: schema.NewParamsOneOfByJSONSchema(js),
	}, nil
}
