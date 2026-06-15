// Package container 实现依赖注入容器，按依赖顺序组装所有顶层组件（配置、LLM 客户端、工具注册表、会话存储、编排器等）。
package container

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wikiglobal/suanming-agent/internal/config"
	"github.com/wikiglobal/suanming-agent/internal/handler"
	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/mcp"
	"github.com/wikiglobal/suanming-agent/internal/orchestrator"
	"github.com/wikiglobal/suanming-agent/internal/specialists/bazi"
	qimenSp "github.com/wikiglobal/suanming-agent/internal/specialists/qimen"
	"github.com/wikiglobal/suanming-agent/internal/specialists/ziwei"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/supervisor"
	"github.com/wikiglobal/suanming-agent/internal/tools"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// Container 持有所有顶层组件（配置、路由、处理器）。
type Container struct {
	Config  *config.Config
	Router  *gin.Engine
	Handler *handler.ChatHandler
}

// BuildContainer 按依赖顺序组装全部组件并返回根容器。
func BuildContainer() *Container {
	cfg := config.Load()
	if strings.EqualFold(cfg.LLMBackend, "eino") {
		tracing.InstallEinoCallbackTracing()
	}

	// LLM 客户端（主模型，用于解读）— 低温度参数以保证一致性
	llmTemp := cfg.LLMTemperature
	if llmTemp <= 0 {
		llmTemp = 0.3
	}
	llmClient := mustNewChatClient(cfg, llm.FactoryConfig{
		APIKey:      cfg.LLMApiKey,
		BaseURL:     cfg.LLMBaseURL,
		Model:       cfg.LLMModel,
		Backend:     cfg.LLMBackend,
		Temperature: llmTemp,
	})

	// LLM 快速客户端（用于分类/提取）— 确定性输出，不启用思考。
	flashModel := cfg.LLMFlashModel
	if flashModel == "" {
		flashModel = cfg.LLMModel
	}
	flashClient := mustNewChatClient(cfg, llm.FactoryConfig{
		APIKey:          cfg.LLMApiKey,
		BaseURL:         cfg.LLMBaseURL,
		Model:           flashModel,
		Backend:         cfg.LLMBackend,
		Temperature:     0.0,
		DisableThinking: true,
	})

	// MCP 客户端，用于知识检索
	mcpClient := mcp.NewClient(cfg.KnowledgeURL)

	// 工具注册表
	reg := tools.NewRegistry()
	reg.Register(&tools.BaziCalcTool{})
	reg.Register(&tools.YongShenTool{})
	reg.Register(&tools.DayunAnalyzer{})
	reg.Register(&tools.QimenTool{})
	reg.Register(tools.NewKnowledgeSearchTool(mcpClient))

	// 会话存储 + 锁
	store := state.NewPersistentStore("data/sessions")
	locker := state.NewMemoryLocker()

	// 编排器 — 始终使用真实跟踪器用于前端 trace-panel；通过 DEBUG_TRACE 控制文件持久化
	var collector *tracing.FileCollector
	if cfg.DebugTrace {
		collector = tracing.NewFileCollector("logs/traces")
	}
	tracer := tracing.NewRealTracer(collector)
	orch := orchestrator.New(reg, llmClient, flashClient, store, locker, tracer, cfg.PromptMode)
	orch.SetLLMModel(cfg.LLMModel)

	// Supervisor 客户端 — 使用快速模型进行路由决策。
	supervisorOpts := []supervisor.ClientOption{}
	if routeEngine, err := buildSupervisorRouteEngine(cfg, flashModel); err != nil {
		if strings.EqualFold(cfg.SupervisorEngine, "adk") {
			panic(err)
		}
		log.Printf("[container] supervisor ADK engine unavailable, falling back to classic structuredDecide: %v", err)
	} else if routeEngine != nil {
		supervisorOpts = append(supervisorOpts, supervisor.WithRouteEngine(routeEngine))
	}
	supervisorClient := supervisor.NewClient(flashClient, supervisorOpts...)
	orch.SetSupervisor(supervisorClient)

	// 领域专家 — 注入到编排器中用于阶段一分发。
	orch.SetSpecialists(bazi.New(), qimenSp.New(), ziwei.New())

	// 处理器
	debugDir := "logs/debug"
	chatHandler := handler.NewChatHandler(orch, cfg.DebugHTTP, debugDir)

	// 路由
	r := gin.Default()
	r.Use(tracing.Middleware(tracer))
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST,GET,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.Static("/assets", cfg.StaticDir+"/assets")
	r.StaticFile("/favicon.svg", cfg.StaticDir+"/favicon.svg")
	r.GET("/", func(c *gin.Context) { c.File(cfg.StaticDir + "/index.html") })
	r.GET("/api/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	r.POST("/api/chat", chatHandler.HandleChat)

	return &Container{
		Config:  cfg,
		Router:  r,
		Handler: chatHandler,
	}
}

func mustNewChatClient(_ *config.Config, factoryCfg llm.FactoryConfig) llm.Chat {
	client, err := llm.NewChatClient(context.Background(), factoryCfg)
	if err != nil {
		panic(err)
	}
	return client
}

func buildSupervisorRouteEngine(cfg *config.Config, flashModel string) (supervisor.RouteEngine, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.SupervisorEngine))
	if mode == "" {
		mode = "auto"
	}

	switch mode {
	case "classic":
		return nil, nil
	case "auto":
		if !strings.EqualFold(cfg.LLMBackend, "eino") {
			return nil, nil
		}
	case "adk":
		if !strings.EqualFold(cfg.LLMBackend, "eino") {
			return nil, fmt.Errorf("SUPERVISOR_ENGINE=adk requires LLM_BACKEND=eino, got %q", cfg.LLMBackend)
		}
	default:
		return nil, fmt.Errorf("unsupported SUPERVISOR_ENGINE %q", cfg.SupervisorEngine)
	}

	model, err := llm.NewToolCallingModel(context.Background(), llm.FactoryConfig{
		APIKey:          cfg.LLMApiKey,
		BaseURL:         cfg.LLMBaseURL,
		Model:           flashModel,
		Backend:         cfg.LLMBackend,
		Temperature:     0.0,
		DisableThinking: true,
	})
	if err != nil {
		return nil, err
	}

	return supervisor.NewADKRouteEngine(context.Background(), model)
}
