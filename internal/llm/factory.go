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

var newEinoToolCallingChatModel = func(ctx context.Context, cfg FactoryConfig) (einomodel.ToolCallingChatModel, error) {
	apiKey := cfg.APIKey
	if apiKey == "" {
		log.Println("WARNING: LLM_API_KEY not set — Eino chat model will initialize with a placeholder key and requests will fail until a real key is provided")
		apiKey = "missing-api-key"
	}
	var thinkingConfig *deepseekmodel.ThinkingConfig
	if cfg.DisableThinking {
		thinkingConfig = &deepseekmodel.ThinkingConfig{Type: "disabled"}
	}
	return deepseekmodel.NewChatModel(ctx, &deepseekmodel.ChatModelConfig{
		APIKey:         apiKey,
		BaseURL:        normalizeEinoBaseURL(cfg.BaseURL),
		Model:          cfg.Model,
		Temperature:    float32(cfg.Temperature),
		ThinkingConfig: thinkingConfig,
	})
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

func normalizeEinoBaseURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "https://api.deepseek.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/anthropic")
	return baseURL
}
