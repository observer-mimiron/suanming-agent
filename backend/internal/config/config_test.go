// This test file belongs to the configuration layer.
// It verifies configuration loading behavior and protects the related contract from regressions.
// It parses environment input; runtime code should consume typed config, not raw env.
package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testProjectRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// config 包的“项目根”定义为 Go 模块根，也就是 backend/。
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestLoad_EmbeddingAndRouterMode(t *testing.T) {
	t.Setenv("EMBEDDING_API_KEY", "sk-test-embed-key")
	t.Setenv("EMBEDDING_BASE_URL", "https://custom.example.com/v1")
	t.Setenv("EMBEDDING_MODEL", "text-embedding-v3")
	t.Setenv("ROUTER_MODE", "shadow")

	cfg := Load()

	if cfg.EmbeddingApiKey != "sk-test-embed-key" {
		t.Fatalf("EmbeddingApiKey = %q, want %q", cfg.EmbeddingApiKey, "sk-test-embed-key")
	}
	if cfg.EmbeddingBaseUrl != "https://custom.example.com/v1" {
		t.Fatalf("EmbeddingBaseUrl = %q, want custom non-default value", cfg.EmbeddingBaseUrl)
	}
	if cfg.EmbeddingModel != "text-embedding-v3" {
		t.Fatalf("EmbeddingModel = %q, want custom non-default value", cfg.EmbeddingModel)
	}
	if cfg.RouterMode != "shadow" {
		t.Fatalf("RouterMode = %q, want shadow", cfg.RouterMode)
	}
}

func TestLoad_RouterModeDefault(t *testing.T) {
	// 置为空字符串可以阻止 dotenv 覆盖，同时仍然验证 getEnv 的 fallback。
	t.Setenv("ROUTER_MODE", "")
	cfg := Load()
	if cfg.RouterMode != "off" {
		t.Fatalf("default RouterMode = %q, want off", cfg.RouterMode)
	}
}

func TestLoad_OTelEndpointFromExporterEnv(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:3001/api/public/otel")
	t.Setenv("OTEL_ENABLED", "1")

	cfg := Load()

	if cfg.OTelEndpoint != "http://localhost:3001/api/public/otel" {
		t.Fatalf("OTelEndpoint = %q, want local Langfuse OTLP endpoint", cfg.OTelEndpoint)
	}
	if !cfg.OTelEnabled {
		t.Fatal("expected OTelEnabled to be true when OTEL endpoint is configured")
	}
}

func TestFindProjectRoot_FromNestedDir(t *testing.T) {
	root := testProjectRoot(t)
	got, err := findProjectRoot(filepath.Join(root, "internal", "config"))
	if err != nil {
		t.Fatalf("findProjectRoot returned error: %v", err)
	}
	if got != root {
		t.Fatalf("findProjectRoot = %q, want %q", got, root)
	}
}
