package config

import (
	"os"
	"testing"
)

func TestLoad_EmbeddingAndRouterMode(t *testing.T) {
	t.Setenv("EMBEDDING_API_KEY", "sk-test-embed-key")
	t.Setenv("EMBEDDING_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1")
	t.Setenv("EMBEDDING_MODEL", "text-embedding-v4")
	t.Setenv("ROUTER_MODE", "shadow")

	cfg := Load()

	if cfg.EmbeddingApiKey != "sk-test-embed-key" {
		t.Fatalf("EmbeddingApiKey = %q, want %q", cfg.EmbeddingApiKey, "sk-test-embed-key")
	}
	if cfg.EmbeddingBaseUrl != "https://dashscope.aliyuncs.com/compatible-mode/v1" {
		t.Fatalf("EmbeddingBaseUrl = %q", cfg.EmbeddingBaseUrl)
	}
	if cfg.EmbeddingModel != "text-embedding-v4" {
		t.Fatalf("EmbeddingModel = %q", cfg.EmbeddingModel)
	}
	if cfg.RouterMode != "shadow" {
		t.Fatalf("RouterMode = %q, want shadow", cfg.RouterMode)
	}
}

func TestLoad_RouterModeDefault(t *testing.T) {
	os.Unsetenv("ROUTER_MODE")
	cfg := Load()
	if cfg.RouterMode != "off" {
		t.Fatalf("default RouterMode = %q, want off", cfg.RouterMode)
	}
}