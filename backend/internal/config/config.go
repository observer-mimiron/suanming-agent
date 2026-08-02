// Package config 提供应用配置的加载与管理。
// 从环境变量读取 LLM、知识库、服务监听地址等运行时配置。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// Config 包含全部由环境变量驱动的应用配置。
type Config struct {
	LLMApiKey         string
	LLMBaseURL        string
	LLMModel          string
	LLMFlashModel     string  // 可选的快速模型，用于轻量任务（分类、提取）
	LLMTemperature    float64 // 0.0-1.0，越低越确定
	KnowledgeURL      string
	StaticDir         string
	ListenAddr        string
	DebugHTTP         bool
	DebugTrace        bool // 为 true 时，将 TurnTrace 持久化到 logs/traces/ 目录
	OTelEnabled       bool
	OTelEndpoint      string
	OTelHeaders       string
	OTelServiceName   string
	OTelInsecure      bool
	ConversationLimit int // 传入 agent 的最近对话消息条数上限，默认 10

	// Embedding 配置——用于 semantic router（意图识别）。沿用 KB 同款 env 变量名。
	EmbeddingApiKey  string
	EmbeddingBaseUrl string
	EmbeddingModel   string
	// RouterMode 控制 semantic router 上线节奏：off | shadow | enforce。
	// off=不初始化 router，走旧 regex；shadow=旁路只 log；enforce=接入决策。
	RouterMode string
}

// Load 从环境变量读取并返回应用配置。
// 优先从项目根目录的 .env 文件加载，若文件不存在则静默跳过。
func Load() *Config {
	loadDotEnv()

	return &Config{
		LLMApiKey:         os.Getenv("LLM_API_KEY"),
		LLMBaseURL:        getEnv("LLM_BASE_URL", "https://api.deepseek.com/anthropic"),
		LLMModel:          getEnv("LLM_MODEL", "deepseek-v4-pro"),
		LLMFlashModel:     os.Getenv("LLM_FLASH_MODEL"),
		LLMTemperature:    getEnvFloat("LLM_TEMPERATURE", 0.3),
		KnowledgeURL:      getEnv("KNOWLEDGE_MCP_URL", "http://localhost:3100"),
		StaticDir:         getEnv("STATIC_DIR", "web/dist"),
		ListenAddr:        getEnv("LISTEN_ADDR", ":8080"),
		DebugHTTP:         os.Getenv("DEBUG_HTTP") == "1",
		DebugTrace:        os.Getenv("DEBUG_TRACE") == "1",
		OTelEnabled:       getEnvBool("OTEL_ENABLED", false) || resolveOTelEndpoint() != "",
		OTelEndpoint:      resolveOTelEndpoint(),
		OTelHeaders:       getEnv("OTEL_EXPORTER_OTLP_HEADERS", ""),
		OTelServiceName:   getEnv("OTEL_SERVICE_NAME", "suanming-agent"),
		OTelInsecure:      getEnvBool("OTEL_EXPORTER_OTLP_INSECURE", false),
		ConversationLimit: getEnvInt("CONVERSATION_LIMIT", 10),

		EmbeddingApiKey:  os.Getenv("EMBEDDING_API_KEY"),
		EmbeddingBaseUrl: getEnv("EMBEDDING_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1"),
		EmbeddingModel:   getEnv("EMBEDDING_MODEL", "text-embedding-v4"),
		RouterMode:       getEnv("ROUTER_MODE", "off"),
	}
}

func loadDotEnv() {
	cwd, err := os.Getwd()
	if err != nil {
		_ = godotenv.Load()
		return
	}
	root, err := findProjectRoot(cwd)
	if err != nil {
		_ = godotenv.Load()
		return
	}
	_ = godotenv.Load(filepath.Join(root, ".env"))
}

func findProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("project root not found from %s", startDir)
		}
		dir = parent
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

// getEnvInt 从环境变量读取整数配置，若无效或不存在则返回 fallback。
func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
		return n
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func resolveOTelEndpoint() string {
	if v := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
}
