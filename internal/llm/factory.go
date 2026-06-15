// Package llm 暂与 client.go 共享包注释，本文件提供工厂函数和配置类型。

package llm

import (
	"context"
	"fmt"
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
	Backend         string
	Temperature     float64
	DisableThinking bool
}

var newNativeChatClient = func(cfg FactoryConfig) Chat {
	return NewClient(cfg.APIKey, cfg.BaseURL, cfg.Model, cfg.Temperature)
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

// NewToolCallingModel 根据后端类型创建 ToolCallingChatModel。
// 当前仅支持 "eino" 后端；"native" 后端不支持 ToolCallingChatModel 接口。
func NewToolCallingModel(ctx context.Context, cfg FactoryConfig) (einomodel.ToolCallingChatModel, error) {
	backend := strings.TrimSpace(cfg.Backend)
	if backend == "" {
		backend = "eino"
	}

	switch backend {
	case "eino":
		return newEinoToolCallingChatModel(ctx, cfg)
	case "native":
		return nil, fmt.Errorf("backend %q does not expose ToolCallingChatModel", backend)
	default:
		return nil, fmt.Errorf("unsupported LLM_BACKEND %q", backend)
	}
}

// NewChatClient 根据后端类型创建 Chat 接口实现。
// "native" 返回原生 HTTP 客户端；"eino" 返回 EinoChat 包装。
func NewChatClient(ctx context.Context, cfg FactoryConfig) (Chat, error) {
	backend := strings.TrimSpace(cfg.Backend)
	if backend == "" {
		backend = "eino"
	}

	switch backend {
	case "native":
		return newNativeChatClient(cfg), nil
	case "eino":
		model, err := newEinoToolCallingChatModel(ctx, cfg)
		if err != nil {
			return nil, err
		}
		return NewEinoChat(model), nil
	default:
		return nil, fmt.Errorf("unsupported LLM_BACKEND %q", backend)
	}
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
