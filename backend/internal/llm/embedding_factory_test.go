package llm

import (
	"context"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/config"
)

func TestNewEmbedder_ReturnsNonNilWhenApiKeySet(t *testing.T) {
	cfg := &config.Config{
		EmbeddingApiKey:  "sk-test",
		EmbeddingBaseUrl: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		EmbeddingModel:   "text-embedding-v4",
	}
	embedder, err := NewEmbedder(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewEmbedder err: %v", err)
	}
	if embedder == nil {
		t.Fatal("embedder is nil")
	}
}

func TestNewEmbedder_ReturnsNilWhenApiKeyEmpty(t *testing.T) {
	cfg := &config.Config{
		EmbeddingApiKey: "",
	}
	embedder, err := NewEmbedder(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected nil err when api key empty, got %v", err)
	}
	if embedder != nil {
		t.Fatal("expected nil embedder when api key empty")
	}
}
