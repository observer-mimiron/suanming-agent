// This file belongs to the LLM adapter layer.
// It owns embedding model construction for this package.
// It wraps model providers; domain prompts and contracts stay outside this package.
package llm

import (
	"context"
	"log"
	"time"

	einoopenai "github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/observer-mimiron/suanming-agent/internal/config"
)

// NewEmbedder 根据配置构造 Eino Embedder。
// 用 eino-ext 的 openai embedding ext 指向 DashScope 的 OpenAI-compatible 端点
// （和知识库 KB 同款方案，支持自定义 base_url）。
// API key 为空时返回 nil, nil——调用方据此走 regex 兜底（router=nil）。
func NewEmbedder(ctx context.Context, cfg *config.Config) (embedding.Embedder, error) {
	if cfg.EmbeddingApiKey == "" {
		log.Println("WARNING: EMBEDDING_API_KEY not set — semantic router will fall back to regex")
		return nil, nil
	}

	embedder, err := einoopenai.NewEmbedder(ctx, &einoopenai.EmbeddingConfig{
		APIKey:  cfg.EmbeddingApiKey,
		BaseURL: cfg.EmbeddingBaseUrl,
		Model:   cfg.EmbeddingModel,
		Timeout: 10 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return embedder, nil
}
