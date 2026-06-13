package config

import (
	"fmt"
	"os"
)

// Config holds all environment-driven configuration.
type Config struct {
	LLMApiKey     string
	LLMBaseURL    string
	LLMModel      string
	LLMFlashModel   string  // optional flash/fast model for lightweight tasks (classify, extract)
	LLMTemperature  float64 // 0.0-1.0, lower=more consistent
	KnowledgeURL    string
	StaticDir     string
	ListenAddr    string
	DebugHTTP     bool
	DebugTrace    bool   // When true, persist TurnTrace to logs/traces/
	PromptMode    string // "soft" (default) or "direct" — direct mode is for benchmark testing
}

// Load reads configuration from environment variables.
func Load() *Config {
	mode := os.Getenv("PROMPT_MODE")
	if mode == "" {
		mode = "soft"
	}
	return &Config{
		LLMApiKey:     os.Getenv("LLM_API_KEY"),
		LLMBaseURL:    getEnv("LLM_BASE_URL", "https://api.deepseek.com/anthropic"),
		LLMModel:      getEnv("LLM_MODEL", "deepseek-v4-pro"),
		LLMFlashModel:  os.Getenv("LLM_FLASH_MODEL"),
		LLMTemperature: getEnvFloat("LLM_TEMPERATURE", 0.3),
		KnowledgeURL:   getEnv("KNOWLEDGE_MCP_URL", "http://localhost:3100"),
		StaticDir:     getEnv("STATIC_DIR", "web/dist"),
		ListenAddr:    getEnv("LISTEN_ADDR", ":8080"),
		DebugHTTP:     os.Getenv("DEBUG_HTTP") == "1",
		DebugTrace:    os.Getenv("DEBUG_TRACE") == "1",
		PromptMode:    mode,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var f float64
	if _, err := fmt.Sscanf(v, "%f", &f); err == nil && f >= 0 && f <= 1 {
		return f
	}
	return fallback
}
