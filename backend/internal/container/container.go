// Package container 实现依赖注入容器，按依赖顺序组装所有顶层组件（配置、LLM 客户端、工具注册表、会话存储、编排器等）。
package container

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/gin-gonic/gin"
	"github.com/observer-mimiron/suanming-agent/internal/config"
	"github.com/observer-mimiron/suanming-agent/internal/handler"
	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/llm"
	"github.com/observer-mimiron/suanming-agent/internal/mcp"
	"github.com/observer-mimiron/suanming-agent/internal/observability"
	"github.com/observer-mimiron/suanming-agent/internal/orchestrator"
	appRuntime "github.com/observer-mimiron/suanming-agent/internal/runtime"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/specialists/bazi"
	baziAdapter "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/adapter"
	qimenAdapter "github.com/observer-mimiron/suanming-agent/internal/specialists/qimen/adapter"
	ziweiAdapter "github.com/observer-mimiron/suanming-agent/internal/specialists/ziwei/adapter"
	ziweiTools "github.com/observer-mimiron/suanming-agent/internal/specialists/ziwei/adapter"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/supervisor"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
	baziCalc "github.com/observer-mimiron/suanming-agent/internal/tools/bazi"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// Container 持有所有顶层组件（配置、路由、处理器）。
type Container struct {
	Config                  *config.Config
	Router                  *gin.Engine
	Handler                 *handler.ChatHandler
	Tools                   *tools.Registry
	DebugDir                string
	TraceDir                string
	TracePersistenceEnabled bool
	OTelEnabled             bool
	OTelEndpoint            string
	Shutdown                func(context.Context) error
}

// TraceStartupLines returns human-readable tracing status lines for startup logs.
// We log both local TurnTrace persistence and optional OTel mirroring so operators
// can immediately tell why logs/traces is empty without reading code or env files.
func (c *Container) TraceStartupLines() []string {
	if c == nil {
		return nil
	}

	local := "[tracing] local TurnTrace persistence: disabled (set DEBUG_TRACE=1 to write JSON files)"
	if c.TracePersistenceEnabled {
		local = "[tracing] local TurnTrace persistence: enabled"
	}
	if c.TraceDir != "" {
		local += " -> " + c.TraceDir
	}

	otel := "[tracing] OTel export mirror: disabled"
	if c.OTelEnabled {
		otel = "[tracing] OTel export mirror: enabled"
		if c.OTelEndpoint != "" {
			otel += " -> " + c.OTelEndpoint
		}
	}

	return []string{local, otel}
}

// resolveProjectPath 将相对项目根目录的路径解析为稳定的绝对路径，
// 避免从 backend/ 等子目录启动时把 data/logs 写进错误位置。
func resolveProjectPath(relative string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Clean(relative)
	}
	resolved, err := resolveProjectPathFrom(cwd, relative)
	if err != nil {
		return filepath.Clean(relative)
	}
	return resolved
}

func resolveProjectPathFrom(startDir, relative string) (string, error) {
	root, err := findProjectRoot(startDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(relative)), nil
}

func findProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		if isProjectRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("project root not found from %s", startDir)
		}
		dir = parent
	}
}

func isProjectRoot(dir string) bool {
	markers := []string{"go.work", ".git", "AGENTS.md"}
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}

// BuildContainer 按依赖顺序组装全部组件并返回根容器。
func BuildContainer() *Container {
	cfg := config.Load()
	tracing.InstallEinoCallbackTracing()
	sessionDir := resolveProjectPath("data/sessions")
	traceDir := resolveProjectPath("logs/traces")
	debugDir := resolveProjectPath("logs/debug")
	cheapGateReportPath := resolveProjectPath("logs/reports/cheap-gate/hits.jsonl")

	// LLM 客户端（主模型，用于解读）— 低温度参数以保证一致性
	llmTemp := cfg.LLMTemperature
	if llmTemp <= 0 {
		llmTemp = 0.3
	}

	// LLM 快速客户端（用于分类/提取）— 确定性输出，不启用思考。
	flashModel := cfg.LLMFlashModel
	if flashModel == "" {
		flashModel = cfg.LLMModel
	}
	flashClient := mustNewChatClient(cfg, llm.FactoryConfig{
		APIKey:          cfg.LLMApiKey,
		BaseURL:         cfg.LLMBaseURL,
		Model:           flashModel,
		Temperature:     0.0,
		DisableThinking: true,
	})

	// MCP 客户端，用于知识检索
	mcpClient := mcp.NewClient(cfg.KnowledgeURL)

	// 工具注册表
	reg := tools.NewRegistry()
	registerTool(reg, &baziCalc.CalcTool{})
	registerTool(reg, &baziCalc.YongShenTool{})
	registerTool(reg, &baziCalc.DayunAnalyzer{})
	registerTool(reg, &baziCalc.BaziLiuNianTool{})
	registerTool(reg, &qimenAdapter.Tool{})
	registerTool(reg, &ziweiTools.ZiWeiCalcTool{})
	registerTool(reg, &ziweiTools.ZiWeiLiuNianTool{})
	registerTool(reg, tools.NewKnowledgeSearchTool(mcpClient))
	registerTool(reg, tools.NewKnowledgeCatalogTool(mcpClient))

	// 会话存储 + 锁
	store := state.NewPersistentStore(sessionDir)
	locker := state.NewMemoryLocker()

	// 编排器 — 始终使用真实跟踪器用于前端 trace-panel；通过 DEBUG_TRACE 控制文件持久化
	var collector *tracing.FileCollector
	if cfg.DebugTrace {
		collector = tracing.NewFileCollector(traceDir)
	}
	var shutdown func(context.Context) error
	tracer := tracing.NewRealTracer(collector)
	if cfg.OTelEnabled {
		bridge, closeFn, err := tracing.NewOTelBridge(context.Background(), tracing.OTelConfig{
			Enabled:     cfg.OTelEnabled,
			Endpoint:    cfg.OTelEndpoint,
			Headers:     cfg.OTelHeaders,
			ServiceName: cfg.OTelServiceName,
			Insecure:    cfg.OTelInsecure,
		})
		if err != nil {
			panic(err)
		}
		tracer = tracing.NewRealTracerWithOTel(collector, bridge)
		shutdown = closeFn
	}

	// 运行时执行器 — 使用 ADK ChatModelAgent 动态调度工具。
	runtimeModel := mustNewToolCallingModel(cfg, cfg.LLMModel, llmTemp)
	// JSON Mode 节点只承担合同化输出，固定低温度以减少同一输入下的采样漂移；普通模型仍使用配置温度。
	runtimeJSONModel := mustNewToolCallingJSONModel(cfg, cfg.LLMModel, 0.0)
	flashRuntimeModel := mustNewToolCallingModel(cfg, flashModel, 0.0)
	flashRuntimeJSONModel := mustNewToolCallingJSONModel(cfg, flashModel, 0.0)
	// 摘要模型 — 复用 flash 配置，用于 specialist summarization 中间件压缩长对话历史。
	summarizerModel := mustNewToolCallingModel(cfg, flashModel, 0.0)

	// Semantic Router 构造（仅 enforce/shadow 模式）
	// off 模式或 embedder 构造失败时 router=nil，走旧 regex 兜底
	var router intent.Router
	if cfg.RouterMode == "enforce" || cfg.RouterMode == "shadow" {
		embedder, err := llm.NewEmbedder(context.Background(), cfg)
		if err != nil {
			log.Printf("[container] embedder init failed: %v — router disabled, falling back to regex", err)
		} else if embedder != nil {
			sr, err := intent.NewSemanticRouter(context.Background(), embedder, intent.Utterances, 0.75)
			if err != nil {
				log.Printf("[container] semantic router init failed: %v — falling back to regex", err)
			} else {
				router = sr
				log.Printf("[container] semantic router initialized in %s mode", cfg.RouterMode)
			}
		}
	}
	// 注册所有领域专家（composition root 负责，runtime 只消费 Registry 接口）。
	// 默认执行路径是 direct ADK specialist runner。
	sr := specialists.NewRegistry()
	executor, err := appRuntime.NewExecutor(reg, sr, runtimeModel, flashClient, summarizerModel, appRuntime.ExecutorConfig{
		LLMModel:     cfg.LLMModel,
		HistoryLimit: cfg.ConversationLimit,
		Router:       router,
		Builder: appRuntime.AgentBuilderConfig{
			ModelCreator:     runtimeJSONModel,
			FastModel:        flashRuntimeModel,
			FastModelCreator: flashRuntimeJSONModel,
		},
	})
	if err != nil {
		panic(err)
	}
	runtimeServices := executor.SpecialistServices()
	sr.Register(bazi.GetConfig(), &bazi.Runner{
		Primary: &baziAdapter.Runner{Port: baziAdapter.RuntimePort{
			Builder:  runtimeServices.Builder,
			Registry: runtimeServices.Registry,
			Sink: func(ctx context.Context, event baziAdapter.Event) error {
				return runtimeServices.Emit(ctx, event.Type, event.Data)
			},
		}},
		Support: &appRuntime.ADKSpecialistRunner{
			Domain:   "bazi",
			Config:   bazi.GetConfig(),
			Executor: executor,
		},
	})
	sr.Register(qimenAdapter.GetConfig(), &appRuntime.ADKSpecialistRunner{
		Domain:   "qimen",
		Config:   qimenAdapter.GetConfig(),
		Executor: executor,
	})
	sr.Register(ziweiAdapter.GetConfig(), &appRuntime.ADKSpecialistRunner{
		Domain:   "ziwei",
		Config:   ziweiAdapter.GetConfig(),
		Executor: executor,
	})

	// Orchestrator — 会话生命周期管理，注入已构建的执行器。
	orch := orchestrator.New(executor, flashClient, store, locker, tracer)

	// Supervisor 客户端固定使用 ADK route engine；外层 text fallback 仍由 Go supervisor 保留。
	routeEngine := mustNewSupervisorRouteEngine(cfg, flashModel)
	cheapGateReporter := observability.NewCheapGateReporter(cheapGateReportPath)
	supervisorClient := supervisor.NewClient(
		flashClient,
		supervisor.WithRouteEngine(routeEngine),
		supervisor.WithSemanticRouter(router),
		supervisor.WithRouterMode(cfg.RouterMode),
		supervisor.WithCheapGateReporter(cheapGateReporter),
	)
	orch.SetSupervisor(supervisorClient)

	// 处理器
	chatHandler := handler.NewChatHandler(orch, cfg.DebugHTTP, debugDir)
	sessionHandler := handler.NewSessionHandler(store, debugDir)
	debugTraceHandler := handler.NewDebugTraceHandler(traceDir)

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
	r.GET("/api/session/:sessionID", sessionHandler.HandleGetSession)
	if cfg.DebugHTTP {
		r.GET("/api/debug/traces/:traceID", debugTraceHandler.HandleGetTrace)
	}

	r.POST("/api/chat", chatHandler.HandleChat)

	return &Container{
		Config:                  cfg,
		Router:                  r,
		Handler:                 chatHandler,
		Tools:                   reg,
		DebugDir:                debugDir,
		TraceDir:                traceDir,
		TracePersistenceEnabled: cfg.DebugTrace,
		OTelEnabled:             cfg.OTelEnabled,
		OTelEndpoint:            cfg.OTelEndpoint,
		Shutdown:                shutdown,
	}
}

func registerTool(reg *tools.Registry, tool tools.Tool) {
	reg.RegisterWithContract(tool, tools.DefaultContractFor(tool.Name()))
}

func mustNewChatClient(_ *config.Config, factoryCfg llm.FactoryConfig) llm.Chat {
	client, err := llm.NewChatClient(context.Background(), factoryCfg)
	if err != nil {
		panic(err)
	}
	return client
}

func mustNewToolCallingModel(cfg *config.Config, model string, temperature float64) einomodel.ToolCallingChatModel {
	toolModel, err := llm.NewToolCallingModel(context.Background(), llm.FactoryConfig{
		APIKey:          cfg.LLMApiKey,
		BaseURL:         cfg.LLMBaseURL,
		Model:           model,
		Temperature:     temperature,
		DisableThinking: true,
	})
	if err != nil {
		panic(err)
	}
	return toolModel
}

func mustNewToolCallingJSONModel(cfg *config.Config, model string, temperature float64) einomodel.ToolCallingChatModel {
	toolModel, err := llm.NewToolCallingModelWithJSON(context.Background(), llm.FactoryConfig{
		APIKey:          cfg.LLMApiKey,
		BaseURL:         cfg.LLMBaseURL,
		Model:           model,
		Temperature:     temperature,
		DisableThinking: true,
	})
	if err != nil {
		panic(err)
	}
	return toolModel
}

func mustNewSupervisorRouteEngine(cfg *config.Config, flashModel string) supervisor.RouteEngine {
	model, err := llm.NewToolCallingModel(context.Background(), llm.FactoryConfig{
		APIKey:          cfg.LLMApiKey,
		BaseURL:         cfg.LLMBaseURL,
		Model:           flashModel,
		Temperature:     0.0,
		DisableThinking: true,
	})
	if err != nil {
		panic(err)
	}

	engine, err := supervisor.NewADKRouteEngine(context.Background(), model)
	if err != nil {
		panic(err)
	}
	return engine
}
