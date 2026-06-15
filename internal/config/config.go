// Package config 提供应用配置的加载与管理。
// 从环境变量读取 LLM、知识库、服务监听地址等运行时配置。
package config

import (
	"fmt"
	"os"
)

// Config 包含全部由环境变量驱动的应用配置。
type Config struct {
	LLMApiKey        string
	LLMBaseURL       string
	LLMModel         string
	LLMBackend       string
	SupervisorEngine string
	LLMFlashModel    string  // 可选的快速模型，用于轻量任务（分类、提取）
	LLMTemperature   float64 // 0.0-1.0，越低越确定
	KnowledgeURL     string
	StaticDir        string
	ListenAddr       string
	DebugHTTP        bool
	DebugTrace       bool   // 为 true 时，将 TurnTrace 持久化到 logs/traces/ 目录
	PromptMode       string // "soft"（默认）或 "direct" — 直接模式用于基准测试
}

// Load 从环境变量读取并返回应用配置。
func Load() *Config {
	mode := os.Getenv("PROMPT_MODE")
	if mode == "" {
		mode = "soft"
	}
	return &Config{
		LLMApiKey:        os.Getenv("LLM_API_KEY"),
		LLMBaseURL:       getEnv("LLM_BASE_URL", "https://api.deepseek.com/anthropic"),
		LLMModel:         getEnv("LLM_MODEL", "deepseek-v4-pro"),
		LLMBackend:       getEnv("LLM_BACKEND", "eino"),
		SupervisorEngine: getEnv("SUPERVISOR_ENGINE", "auto"),
		LLMFlashModel:    os.Getenv("LLM_FLASH_MODEL"),
		LLMTemperature:   getEnvFloat("LLM_TEMPERATURE", 0.3),
		KnowledgeURL:     getEnv("KNOWLEDGE_MCP_URL", "http://localhost:3100"),
		StaticDir:        getEnv("STATIC_DIR", "web/dist"),
		ListenAddr:       getEnv("LISTEN_ADDR", ":8080"),
		DebugHTTP:        os.Getenv("DEBUG_HTTP") == "1",
		DebugTrace:       os.Getenv("DEBUG_TRACE") == "1",
		PromptMode:       mode,
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
