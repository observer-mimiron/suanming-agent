// This file belongs to the LLM adapter layer.
// It owns chat-model factory construction for this package.
// It wraps model providers; domain prompts and contracts stay outside this package.
package llm

import (
	"context"
	"log"
	"strings"

	deepseekmodel "github.com/cloudwego/eino-ext/components/model/deepseek"
	einomodel "github.com/cloudwego/eino/components/model"
)

// FactoryConfig 是 LLM 客户端工厂的配置参数，包含 API 密钥、地址、模型、后端类型、温度及思考模式开关。
type FactoryConfig struct {
	APIKey          string
	BaseURL         string
	Model           string
	Temperature     float64
	DisableThinking bool
}

// buildChatModelConfig 从 FactoryConfig 构建 deepseek ChatModelConfig，消除 NewToolCallingModel 与 NewToolCallingModelWithJSON 之间的重复。
func buildChatModelConfig(cfg FactoryConfig) *deepseekmodel.ChatModelConfig {
	apiKey := cfg.APIKey
	if apiKey == "" {
		log.Println("WARNING: LLM_API_KEY not set — Eino chat model will initialize with a placeholder key and requests will fail until a real key is provided")
		apiKey = "missing-api-key"
	}
	var thinkingConfig *deepseekmodel.ThinkingConfig
	if cfg.DisableThinking {
		thinkingConfig = &deepseekmodel.ThinkingConfig{Type: "disabled"}
	}
	return &deepseekmodel.ChatModelConfig{
		APIKey:         apiKey,
		BaseURL:        normalizeEinoBaseURL(cfg.BaseURL),
		Model:          cfg.Model,
		Temperature:    float32(cfg.Temperature),
		ThinkingConfig: thinkingConfig,
	}
}

var newEinoToolCallingChatModel = func(ctx context.Context, cfg FactoryConfig) (einomodel.ToolCallingChatModel, error) {
	return deepseekmodel.NewChatModel(ctx, buildChatModelConfig(cfg))
}

// NewToolCallingModel 创建 Eino ToolCallingChatModel，用于 supervisor ADK route engine 等需要 tool calling 的场景。
func NewToolCallingModel(ctx context.Context, cfg FactoryConfig) (einomodel.ToolCallingChatModel, error) {
	return newEinoToolCallingChatModel(ctx, cfg)
}

// NewChatClient 创建基于 Eino ChatModel 的 Chat 接口实现。
func NewChatClient(ctx context.Context, cfg FactoryConfig) (Chat, error) {
	model, err := newEinoToolCallingChatModel(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return NewEinoChat(model), nil
}

// NewToolCallingModelWithJSON 创建带 JSON Mode 的 Eino ToolCallingChatModel。
// 用于 inner agent 需要结构化 JSON 输出的场景，设置 response_format: json_object。
func NewToolCallingModelWithJSON(ctx context.Context, cfg FactoryConfig) (einomodel.ToolCallingChatModel, error) {
	base := buildChatModelConfig(cfg)
	base.ResponseFormatType = deepseekmodel.ResponseFormatTypeJSONObject
	return deepseekmodel.NewChatModel(ctx, base)
}

func normalizeEinoBaseURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "https://api.deepseek.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/anthropic")
	return baseURL
}
