package container

import (
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

// Container holds all top-level components.
type Container struct {
	Config  *config.Config
	Router  *gin.Engine
	Handler *handler.ChatHandler
}

// BuildContainer assembles all components in dependency order.
func BuildContainer() *Container {
	cfg := config.Load()

	// LLM client (main, for interpretation) — low temperature for consistency
	llmTemp := cfg.LLMTemperature
	if llmTemp <= 0 {
		llmTemp = 0.3
	}
	llmClient := llm.NewClient(cfg.LLMApiKey, cfg.LLMBaseURL, cfg.LLMModel, llmTemp)

	// LLM flash client (for classification/extraction) — deterministic
	var flashClient llm.Chat = llmClient
	if cfg.LLMFlashModel != "" {
		flashClient = llm.NewClient(cfg.LLMApiKey, cfg.LLMBaseURL, cfg.LLMFlashModel, 0.0)
	}

	// MCP client for knowledge retrieval
	mcpClient := mcp.NewClient(cfg.KnowledgeURL)

	// Tool registry
	reg := tools.NewRegistry()
	reg.Register(&tools.BaziCalcTool{})
	reg.Register(&tools.YongShenTool{})
	reg.Register(&tools.DayunAnalyzer{})
	reg.Register(&tools.QimenTool{})
	reg.Register(tools.NewKnowledgeSearchTool(mcpClient))

	// Session store + locker
	store := state.NewPersistentStore("data/sessions")
	locker := state.NewMemoryLocker()

	// Orchestrator — always use real tracer for frontend trace-panel; file persistence via DEBUG_TRACE
	var collector *tracing.FileCollector
	if cfg.DebugTrace {
		collector = tracing.NewFileCollector("logs/traces")
	}
	tracer := tracing.NewRealTracer(collector)
	orch := orchestrator.New(reg, llmClient, flashClient, store, locker, tracer, cfg.PromptMode)
	orch.SetLLMModel(cfg.LLMModel)

	// Supervisor client — uses flash model for routing decisions.
	supervisorClient := supervisor.NewClient(flashClient)
	orch.SetSupervisor(supervisorClient)

	// Domain specialists — wired into orchestrator for phase-1 dispatch.
	orch.SetSpecialists(bazi.New(), qimenSp.New(), ziwei.New())

	// Handler
	debugDir := "logs/debug"
	chatHandler := handler.NewChatHandler(orch, cfg.DebugHTTP, debugDir)

	// Router
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
