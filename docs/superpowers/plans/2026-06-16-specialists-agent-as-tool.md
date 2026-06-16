# Specialists 转 AgentAsTool 实施方案

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将三个 specialist（bazi/qimen/ziwei）从薄状态机升级为独立 ChatModelAgent + AgentAsTool，父 agent 做路由，子 agent 拥有领域专属工具集

**Architecture:** Orchestrator → Supervisor Agent（ChatModelAgent，只做路由）→ 三个 specialist AgentTool → 领域 ChatModelAgent 执行。每个 specialist 只挂自己领域的工具，父 agent 通过 AgentAsTool 链式调用

**Tech Stack:** Go 1.21+, Eino ADK (ChatModelAgent + AgentAsTool), lunar-go, Vue 3

---

## 设计模式

| 模式 | 载体 | 解决的问题 |
|------|------|------------|
| **AgentBuilder（建造者）** | `runtime/agent.go` | 收敛 `adk.NewChatModelAgent` 的重复配置；统一的 retry、maxIterations |
| **SpecialistConfig（配置对象）** | `runtime/agent.go` | 领域专家的名称、描述、instruction、工具构建器集中声明 |
| **SpecialistRegistry（注册表）** | `specialists/registry.go` | 新增领域 = `Register(config)`，不再写重复工厂函数 |
| **AgentAsTool（ADK 内建）** | `adk.NewAgentTool` | 父 agent 调用子 agent 的标准方式 |

## 改动总览

```
新增: internal/specialists/registry.go           (注册表 + BuildAll)
改: internal/runtime/agent.go                    (AgentBuilder + SpecialistConfig)
改: internal/runtime/adapter.go                  (3 个按域构建函数)
改: internal/specialists/bazi/specialist.go       (Register 注册)
改: internal/specialists/qimen/specialist.go      (Register 注册)
改: internal/specialists/ziwei/specialist.go       (Register 注册)
改: internal/runtime/executor.go                  (initAgents 用 registry 遍历)
删: internal/specialists/types.go                 (DomainHandler 废弃)
改: internal/container/container.go               (wiring 简化)
不改: orchestrator, supervisor.Client(三层防御), bridge, event, prompt, tools/*
```

## 新架构图

```
Orchestrator.Run()
  ├─ supervisor.Approve() → route  (保留，三层防御结构体路由)
  └─ executor.Execute()
       ├─ 短路由: 追问/澄清       (保留不变)
       └─ 主路径: runAgent()
            └─ Supervisor Agent (ChatModelAgent)
                 ├─ bazi_specialist (AgentTool)  → [bazi_calc, yongshen, dayun, knowledge]
                 ├─ qimen_specialist (AgentTool) → [qimen_dunjia, knowledge]
                 └─ ziwei_specialist (AgentTool) → [ziwei_calc, knowledge]
```

**Supervisor Agent**: 只有三个 AgentTool，不做业务工具调用。Instruction 是路由 prompt。`ReturnDirectly` 不设（允许链式调用）。

**Specialist Agent**: 独立 ChatModelAgent，拥有领域工具。`MaxIterations=8`。`ReturnDirectly` 不设（领域内允许 ReAct 多步）。

---

## 任务分解

### Task 1: 拆分 adapter 构建函数

**Files:**
- Modify: `internal/runtime/adapter.go`

- [ ] **Step 1: 新增 3 个按域构建函数**

在 `adapter.go` 末尾新增：

```go
// buildBaziAdapters 创建八字领域专属工具适配器列表。
func buildBaziAdapters(reg *tools.Registry) ([]tool.BaseTool, error) {
    var adapters []tool.BaseTool
    entries := []struct {
        name    string
        builder func() (tool.BaseTool, error)
    }{
        {"bazi_calc", func() (tool.BaseTool, error) { return newBaziCalcAdapter(reg) }},
        {"yongshen", func() (tool.BaseTool, error) { return newYongshenAdapter(reg) }},
        {"dayun_analyzer", func() (tool.BaseTool, error) { return newDayunAdapter(reg) }},
        {"knowledge_search", func() (tool.BaseTool, error) { return newKnowledgeSearchAdapter(reg) }},
    }
    for _, entry := range entries {
        if _, ok := reg.Get(entry.name); !ok {
            continue
        }
        t, err := entry.builder()
        if err != nil {
            return nil, err
        }
        adapters = append(adapters, t)
    }
    return adapters, nil
}

// buildQimenAdapters 创建奇门遁甲领域专属工具适配器列表。
func buildQimenAdapters(reg *tools.Registry) ([]tool.BaseTool, error) {
    entries := []struct {
        name    string
        builder func() (tool.BaseTool, error)
    }{
        {"qimen_dunjia", func() (tool.BaseTool, error) { return newQimenAdapter(reg) }},
        {"knowledge_search", func() (tool.BaseTool, error) { return newKnowledgeSearchAdapter(reg) }},
    }
    return buildAdaptersFromEntries(reg, entries)
}

// buildZiweiAdapters 创建紫微斗数领域专属工具适配器列表。
func buildZiweiAdapters(reg *tools.Registry) ([]tool.BaseTool, error) {
    entries := []struct {
        name    string
        builder func() (tool.BaseTool, error)
    }{
        {"ziwei_calc", func() (tool.BaseTool, error) { return newZiweiAdapter(reg) }},
        {"knowledge_search", func() (tool.BaseTool, error) { return newKnowledgeSearchAdapter(reg) }},
    }
    return buildAdaptersFromEntries(reg, entries)
}

// buildAdaptersFromEntries 通用适配器构建辅助函数。
func buildAdaptersFromEntries(reg *tools.Registry, entries []struct {
    name    string
    builder func() (tool.BaseTool, error)
}) ([]tool.BaseTool, error) {
    var adapters []tool.BaseTool
    for _, entry := range entries {
        if _, ok := reg.Get(entry.name); !ok {
            continue
        }
        t, err := entry.builder()
        if err != nil {
            return nil, err
        }
        adapters = append(adapters, t)
    }
    return adapters, nil
}
```

- [ ] **Step 2: 旧的 `buildAdapters` 保留但标记废弃**

`buildAdapters` 暂时保留（可能被其他测试引用），但不再被 executor 调用。

- [ ] **Step 3: 编译验证**

Run: `go build ./internal/runtime/`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/runtime/adapter.go
git commit -m "feat: add per-domain adapter builders for specialists"
```

---

### Task 2: 新增 AgentBuilder + Specialist 配置类型

**Files:**
- Modify: `internal/runtime/agent.go`

核心思路：收敛 `adk.NewChatModelAgent` 的重复参数到一个 `AgentBuilder`，所有 agent 通过统一的 builder 创建。

- [ ] **Step 1: 定义 SpecialistConfig 和 AgentBuilder**

将 `agent.go` 替换为：

```go
package runtime

import (
    "context"
    "time"

    "github.com/cloudwego/eino/adk"
    einomodel "github.com/cloudwego/eino/components/model"
    einotool "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/compose"
)

const (
    defaultMaxIterations = 12
    specialistMaxIterations = 8
)

// NewSpecialistAgent 创建领域专家 ChatModelAgent。
//
// name/description 用于 AgentAsTool 暴露给父 agent 的工具信息。
// tools 为领域专属工具列表。
// instruction 为系统提示词。
func NewSpecialistAgent(ctx context.Context, name, description, instruction string, model einomodel.ToolCallingChatModel, tools []einotool.BaseTool) (adk.Agent, error) {
    return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:          name,
        Description:   description,
        Instruction:   instruction,
        Model:         model,
        MaxIterations: specialistMaxIterations,
        ToolsConfig: adk.ToolsConfig{
            ToolsNodeConfig: compose.ToolsNodeConfig{
                Tools: tools,
            },
        },
        ModelRetryConfig: &adk.ModelRetryConfig{
            MaxRetries: 2,
            ShouldRetry: func(ctx context.Context, rc *adk.RetryContext) *adk.RetryDecision {
                if rc.Err != nil {
                    return &adk.RetryDecision{Retry: true, Backoff: time.Second}
                }
                return &adk.RetryDecision{Retry: false}
            },
        },
    })
}

// NewSupervisorAgent 创建路由 Supervisor ChatModelAgent。
//
// specialistTools 是从三个 specialist 构建的 AgentTool 列表，
// 父 agent 通过工具调用决定路由到哪个领域。
func NewSupervisorAgent(ctx context.Context, model einomodel.ToolCallingChatModel, specialistTools []einotool.BaseTool, instruction string) (adk.Agent, error) {
    return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:          "supervisor",
        Description:   "命理咨询路由主管，根据用户问题选择合适的领域专家（八字/奇门/紫微）进行分析。",
        Instruction:   instruction,
        Model:         model,
        MaxIterations: defaultMaxIterations,
        ToolsConfig: adk.ToolsConfig{
            ToolsNodeConfig: compose.ToolsNodeConfig{
                Tools: specialistTools,
            },
            // ReturnDirectly 不设 —— 允许链式调用多个 specialist
        },
        ModelRetryConfig: &adk.ModelRetryConfig{
            MaxRetries: 2,
            ShouldRetry: func(ctx context.Context, rc *adk.RetryContext) *adk.RetryDecision {
                if rc.Err != nil {
                    return &adk.RetryDecision{Retry: true, Backoff: time.Second}
                }
                return &adk.RetryDecision{Retry: false}
            },
        },
    })
}
```

旧 `NewRuntimeAgent` 函数删除。

- [ ] **Step 2: 编译验证**

Run: `go build ./internal/runtime/`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/runtime/agent.go
git commit -m "feat: add AgentBuilder, SpecialistConfig, and supervisor instruction"
```

---

### Task 3: 重写 specialists 为 Agent 工厂

**Files:**
- Modify: `internal/specialists/bazi/specialist.go`
- Modify: `internal/specialists/qimen/specialist.go`
- Modify: `internal/specialists/ziwei/specialist.go`
- Delete: `internal/specialists/types.go`

- [ ] **Step 1: 重写 bazi specialist**

`internal/specialists/bazi/specialist.go` 替换为：

```go
// Package bazi 提供八字领域专家 Agent 的构建。
// 专家 Agent 通过 ChatModelAgent + 领域专属工具实现排盘、用神分析、大运解读，
// 通过 AgentAsTool 模式被 Supervisor Agent 调用。
package bazi

import (
    "context"
    "fmt"

    "github.com/cloudwego/eino/adk"
    einomodel "github.com/cloudwego/eino/components/model"
    "github.com/wikiglobal/suanming-agent/internal/runtime"
    "github.com/wikiglobal/suanming-agent/internal/tools"
)

// New 创建八字领域专家 Agent。
//
// 领域工具：bazi_calc（排盘）、yongshen（用神）、dayun_analyzer（大运）、knowledge_search（古籍检索）。
// agent 的名称为 "bazi_specialist"，作为 AgentTool 暴露给 Supervisor Agent。
func New(ctx context.Context, reg *tools.Registry, model einomodel.ToolCallingChatModel) (adk.Agent, error) {
    adapters, err := runtime.BuildBaziAdapters(reg)
    if err != nil {
        return nil, fmt.Errorf("bazi specialist: build adapters: %w", err)
    }

    instruction := `你是八字命理专家（Bazi Specialist）。

## 职责
- 根据用户提供的出生信息排八字命盘（调用 bazi_calc）
- 分析日主强弱、取用神忌神（调用 yongshen）
- 解读大运走势和各步大运的吉凶（调用 dayun_analyzer）
- 检索古籍原文作为论据支撑（调用 knowledge_search）

## 执行流程
1. 如果用户提供了出生信息（年/月/日/时/性别），先调用 bazi_calc 排盘
2. 排盘后可调用 yongshen 分析用神，调用 dayun_analyzer 分析大运
3. 关键论断前调用 knowledge_search 查古籍原文
4. 综合以上信息给出完整解读

## 输出要求
- 面向用户，中文表达，专业但不晦涩
- 引用古籍原文时标注出处（如"《渊海子平》云：...""）
- 命盘数据以系统排盘结果为准，不得自行推算`

    return runtime.NewSpecialistAgent(ctx,
        "bazi_specialist",
        "八字命理专家。根据出生时间排盘、分析用神忌神、解读大运走势。适用于婚恋、事业、财运、健康、性格等八字相关问题。",
        instruction, model, adapters,
    )
}
```

- [ ] **Step 2: 重写 qimen specialist**

`internal/specialists/qimen/specialist.go` 替换为：

```go
// Package qimen 提供奇门遁甲领域专家 Agent 的构建。
// 专家 Agent 通过 ChatModelAgent + 领域专属工具实现排盘和时空分析，
// 通过 AgentAsTool 模式被 Supervisor Agent 调用。
package qimen

import (
    "context"
    "fmt"

    "github.com/cloudwego/eino/adk"
    einomodel "github.com/cloudwego/eino/components/model"
    "github.com/wikiglobal/suanming-agent/internal/runtime"
    "github.com/wikiglobal/suanming-agent/internal/tools"
)

// New 创建奇门遁甲领域专家 Agent。
//
// 领域工具：qimen_dunjia（排盘）、knowledge_search（古籍检索）。
func New(ctx context.Context, reg *tools.Registry, model einomodel.ToolCallingChatModel) (adk.Agent, error) {
    adapters, err := runtime.BuildQimenAdapters(reg)
    if err != nil {
        return nil, fmt.Errorf("qimen specialist: build adapters: %w", err)
    }

    instruction := `你是奇门遁甲专家（QiMen Specialist）。

## 职责
- 根据时间和方位信息排奇门遁甲盘（调用 qimen_dunjia）
- 检索古籍原文作为论据支撑（调用 knowledge_search）

## 执行流程
1. 调用 qimen_dunjia 排盘获取九宫信息
2. 调用 knowledge_search 查相关古籍原文
3. 分析宫、门、星、神组合，给出吉凶判断

## 输出要求
- 面向用户，中文表达
- 重点分析当前时空对问卜事项的有利/不利因素
- 引用古籍原文时标注出处`

    return runtime.NewSpecialistAgent(ctx,
        "qimen_specialist",
        "奇门遁甲专家。分析当前时空的吉凶方位、门星神组合。适用于择时、方位选择、时机判断等问题。",
        instruction, model, adapters,
    )
}
```

- [ ] **Step 3: 重写 ziwei specialist**

`internal/specialists/ziwei/specialist.go` 替换为：

```go
// Package ziwei 提供紫微斗数领域专家 Agent 的构建。
// 专家 Agent 通过 ChatModelAgent + 领域专属工具实现排盘和十二宫解读，
// 通过 AgentAsTool 模式被 Supervisor Agent 调用。
package ziwei

import (
    "context"
    "fmt"

    "github.com/cloudwego/eino/adk"
    einomodel "github.com/cloudwego/eino/components/model"
    "github.com/wikiglobal/suanming-agent/internal/runtime"
    "github.com/wikiglobal/suanming-agent/internal/tools"
)

// New 创建紫微斗数领域专家 Agent。
//
// 领域工具：ziwei_calc（排盘）、knowledge_search（古籍检索）。
func New(ctx context.Context, reg *tools.Registry, model einomodel.ToolCallingChatModel) (adk.Agent, error) {
    adapters, err := runtime.BuildZiweiAdapters(reg)
    if err != nil {
        return nil, fmt.Errorf("ziwei specialist: build adapters: %w", err)
    }

    instruction := `你是紫微斗数专家（ZiWei Specialist）。

## 职责
- 根据用户提供的出生信息排紫微斗数命盘（调用 ziwei_calc）
- 检索古籍原文作为论据支撑（调用 knowledge_search）

## 执行流程
1. 调用 ziwei_calc 排盘，获取十二宫星曜分布和四化飞星
2. 调用 knowledge_search 查相关古籍原文
3. 分析命宫、身宫、三方四正的星曜组合，结合大限流年判断运势

## 输出要求
- 面向用户，中文表达
- 重点关注命宫主星和四化飞星的吉凶应期
- 引用古籍原文时标注出处`

    return runtime.NewSpecialistAgent(ctx,
        "ziwei_specialist",
        "紫微斗数专家。根据出生信息排盘，分析十二宫星曜布局、四化飞星、大限流年。适用于运势、性格、六亲等紫微相关问题。",
        instruction, model, adapters,
    )
}
```

- [ ] **Step 4: 验证编译**

保留 `types.go`（executor.go 在 Task 4 重写前仍引用 `specialists.DomainHandler`）。删除操作移到 Task 6。

Run: `go build ./internal/specialists/...`
Expected: PASS

- [ ] **Step 5: 删除旧测试**

删除 `internal/specialists/bazi/specialist_test.go`、`internal/specialists/qimen/specialist_test.go`。新测试后续补充。

Run: `go build ./internal/specialists/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git rm internal/specialists/bazi/specialist_test.go
git rm internal/specialists/qimen/specialist_test.go
git add internal/specialists/bazi/specialist.go internal/specialists/qimen/specialist.go internal/specialists/ziwei/specialist.go
git commit -m "refactor: specialists become ChatModelAgent factories via AgentAsTool"
```

---

### Task 4: 重写 executor.runAgent

**Files:**
- Modify: `internal/runtime/executor.go`

这是核心改动。`runAgent` 从创建单个 `fortune_teller` agent 变为创建 supervisor agent + 三个 specialist AgentTool。

- [ ] **Step 1: 修改 Executor 结构体和构造函数**

`Executor` 结构体新增：

```go
type Executor struct {
    model              einomodel.ToolCallingChatModel
    llmModel           string
    promptBuilder      *Builder
    reg                *tools.Registry          // 工具注册表
    specialistRegistry *specialists.Registry    // specialist 配置注册表
    builder            *AgentBuilder             // agent 构建器
    historyLimit       int

    supervisorAgent adk.Agent                   // 预构建，多轮复用
}
```

`NewExecutor` 签名和实现：

```go
func NewExecutor(reg *tools.Registry, model einomodel.ToolCallingChatModel, promptMode string) (*Executor, error) {
    // 注册所有领域专家
    sr := specialists.NewRegistry()
    bazi.Register(sr)
    qimen.Register(sr)
    ziwei.Register(sr)

    return &Executor{
        model:              model,
        promptBuilder:      NewBuilder(promptMode),
        reg:                reg,
        specialistRegistry: sr,
        builder:            NewAgentBuilder(model),
    }, nil
}
```

- [ ] **Step 2: 新增 `initAgents` 延迟初始化方法**

```go
// initAgents 通过 SpecialistRegistry + AgentBuilder 构建所有 agent。
//
// 仅在首次 runAgent 时调用一次。遍历 registry 构建 specialist agent，
// 用 AgentAsTool 包装后注入 supervisor agent。
func (e *Executor) initAgents(ctx context.Context) error {
    if e.supervisorAgent != nil {
        return nil
    }

    reg := e.specialistRegistry  // 在 NewExecutor 中初始化
    builder := runtime.NewAgentBuilder(e.model)

    agents, err := reg.BuildAll(ctx, builder, e.reg)
    if err != nil {
        return fmt.Errorf("build specialists: %w", err)
    }

    tools := make([]einotool.BaseTool, len(agents))
    for i, a := range agents {
        tools[i] = adk.NewAgentTool(ctx, a)
    }

    e.supervisorAgent, err = builder.Supervisor(ctx, tools)
    if err != nil {
        return fmt.Errorf("build supervisor: %w", err)
    }
    return nil
}
```

需要在 executor.go 头部新增 import：

需要在 executor.go 头部新增 import：

```go
import (
    "github.com/cloudwego/eino/adk"
    "github.com/wikiglobal/suanming-agent/internal/specialists"
    "github.com/wikiglobal/suanming-agent/internal/specialists/bazi"
    "github.com/wikiglobal/suanming-agent/internal/specialists/qimen"
    "github.com/wikiglobal/suanming-agent/internal/specialists/ziwei"
)
```

- [ ] **Step 3: 重写 `runAgent`**

```go
func (e *Executor) runAgent(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (string, string, error) {
    // 延迟初始化 agent（首次调用时构建）
    if err := e.initAgents(ctx); err != nil {
        return "", "", fmt.Errorf("init agents: %w", err)
    }

    ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
        Name:       "adk_supervisor_agent",
        Kind:       tracing.KindChain,
        Attributes: map[string]any{"model": e.llmModel},
    })

    runner := adk.NewRunner(ctx, adk.RunnerConfig{
        Agent:           e.supervisorAgent,
        EnableStreaming: true,
    })

    // 构建输入消息
    msgs := e.buildConversationMessages(st, message)

    // SessionValues 传递会话上下文
    vals := map[string]any{
        "profile": st.Profile,
        "domain":  route.PrimaryDomain,
    }
    if st.BaziResult != nil {
        vals["bazi_result"] = st.BaziResult
    }
    if st.QimenResult != nil {
        vals["qimen_result"] = st.QimenResult
    }
    if st.ZiWeiResult != nil {
        vals["ziwei_result"] = st.ZiWeiResult
    }

    iter := runner.Run(ctx, msgs, adk.WithSessionValues(vals))

    finalText, err := agentEventBridge(ctx, sink, iter, func(toolName, resultJSON string) {
        e.saveToolResult(st, toolName, resultJSON)
    })
    if err != nil {
        return "agent_error", finalText, err
    }

    return "agent_reading", finalText, nil
}
```

- [ ] **Step 4: 删除 `dispatchSpecialists` 和 `selectPrimarySpecialist`**

这两个函数整体删除。`Execute()` 中短路由逻辑保留但以新方式实现——资料不全追问移到 specialist agent 的 instruction 里处理。

`Execute()` 简化为：

```go
func (e *Executor) Execute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (turnType string, assistantText string, err error) {
    updateRoutingSnapshot(st, route)

    // 短路由: supervisor 要求澄清
    if route.NeedsClarification {
        question := route.ClarificationQuestion
        if question == "" {
            question = "请确认一下您的需求，我再为您详细分析。"
        }
        sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": question}})
        return "clarification", question, nil
    }

    // 主路径: supervisor agent → specialist
    return e.runAgent(ctx, sink, st, route, message)
}
```

- [ ] **Step 5: `SetSpecialists` 标记弃用**

```go
// SetSpecialists 已弃用 —— specialists 现在由 executor 内部构建。
// 保留以兼容 container.go 的调用，但不再使用传入的实例。
func (e *Executor) SetSpecialists(baziSp, qimenSp, ziweiSp specialists.DomainHandler) {
    // no-op: specialists are now built internally
}
```

- [ ] **Step 6: 编译验证**

Run: `go build ./...`
Expected: 编译通过

- [ ] **Step 7: Commit**

```bash
git add internal/runtime/executor.go
git commit -m "refactor: executor uses supervisor agent + AgentAsTool for specialist dispatch"
```

---

### Task 5: 更新 container wiring

**Files:**
- Modify: `internal/container/container.go`

- [ ] **Step 1: 简化 executor 创建**

`container.go` 中，删除 `executor.SetSpecialists(...)` 调用（因为 executor 现在内部构建 specialist）。

```diff
- executor, err := appRuntime.NewExecutor(reg, runtimeModel, cfg.PromptMode)
+ executor, err := appRuntime.NewExecutor(reg, runtimeModel, cfg.PromptMode)
  if err != nil { panic(err) }
  executor.SetLLMModel(cfg.LLMModel)
  executor.SetHistoryLimit(cfg.ConversationLimit)
- executor.SetSpecialists(baziSp.New(), qimenSp.New(), ziweiSp.New())  # 此行删除
```

同时删除 `local "github.com/wikiglobal/suanming-agent/internal/specialists/bazi"` 等不再需要的 import（如果 container 不再直接引用 specialist）。

- [ ] **Step 2: 编译验证**

Run: `go build ./cmd/server/`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/container/container.go
git commit -m "chore: remove SetSpecialists wiring, executor self-builds agents"
```

---

### Task 6: 旧接口清理

**Files:**
- Delete: `internal/specialists/types.go`

- [ ] **Step 1: 删除 DomainHandler 接口**

`internal/specialists/types.go` 删除。

- [ ] **Step 2: 检查 schemas.DomainResult 引用**

```bash
rg "DomainHandler" internal/ --ignore-case
rg "DomainResult" internal/ --ignore-case
```

如无业务引用，标记 `schemas.DomainResult` 废弃（暂不删，前端可能引用）。

- [ ] **Step 3: 全量编译**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: remove deprecated DomainHandler interface and SetSpecialists"
```

---

### Task 7: 集成测试（手动验证）

- [ ] **Step 1: 启动服务**

```bash
LLM_API_KEY=sk-xxx go run ./cmd/server/
```

- [ ] **Step 2: 测试八字路由**

发送消息："我是1990年5月15日早上8点出生的男生，帮我算一下事业运"

Expected: supervisor agent 路由到 `bazi_specialist`，SSE 事件流中能看到 bazi_calc 工具调用和八字解读文本

- [ ] **Step 3: 测试奇门路由**

发送消息："今天适合签约吗？"

Expected: supervisor agent 路由到 `qimen_specialist`，SSE 事件流中能看到 qimen_dunjia 工具调用

- [ ] **Step 4: 测试跨领域路由**

发送消息："帮我看看八字婚姻，顺便看看今天适不适合相亲"

Expected: supervisor agent 先调 `bazi_specialist` 分析婚姻，再调 `qimen_specialist` 择时

- [ ] **Step 5: 测试紫微路由**

发送消息："帮我看一下紫微斗数命盘，1990年5月15日早上8点出生的男生"

Expected: supervisor agent 路由到 `ziwei_specialist`

---

## 关键决策

1. **Agent 复用策略**：supervisor agent + 三个 specialist agent 在首次 `runAgent` 时构建，后续轮次复用（不重建）。这避免了每轮重建 agent 的开销，但意味着 Instruction 不能随会话状态动态变化——需依赖 SessionValues 传递动态上下文

2. **supervisor.Client 保留但不参与 AgentAsTool 链路**：三层防御提取的 `route` 仍然用于 orchestrator 层面的短路由判断（追问/澄清），但不再用于选择 specialist——这个职责交给 supervisor agent 自己

3. **knowledge_search 工具跨 specialist 共享**：每个 specialist 都注册 knowledge_search——它是无状态检索工具，不需要独占

## 回滚方案

如果 AgentAsTool 链路出现问题：

1. 将 `runAgent` 恢复为创建单个 `fortune_teller` agent 挂全部工具（旧逻辑保留为 `runAgentFallback`）
2. 关键降级触发条件：`initAgents` 失败时自动走 fallback
3. container 中 `SetSpecialists` 重新激活

## 风险

| 风险 | 等级 | 缓解 |
|------|------|------|
| AgentAsTool 内部 gob 序列化有边界 bug | 低 | Eino 已发布稳定版，agent_tool.go 代码成熟 |
| 多 agent 链式调用导致 token 消耗翻倍 | 中 | supervisor agent instruction 极简（只做路由），specialist 才做重推理 |
| EmitInternalEvents 事件桥接与现有 bridge.go 冲突 | 中 | 方案中不设 EmitInternalEvents，由 supervisor agent 收集子 agent 输出后统一 emit |
| 测试覆盖不足 | 中 | Task 7 手动验证覆盖核心路径；单元测试后续补充 |- [ ] **Step 1: 创建 registry.go**

`internal/specialists/registry.go`：

```go
// Package specialists 管理所有命理领域专家的注册、配置和构建。
package specialists

import (
    "context"

    "github.com/cloudwego/eino/adk"
    "github.com/wikiglobal/suanming-agent/internal/runtime"
    "github.com/wikiglobal/suanming-agent/internal/tools"
)

// Registry 收纳所有领域专家的配置，提供统一的构建入口。
type Registry struct {
    configs []runtime.SpecialistConfig
}

// NewRegistry 创建注册表。
func NewRegistry() *Registry {
    return &Registry{}
}

// Register 注册一个领域专家配置。
func (r *Registry) Register(cfg runtime.SpecialistConfig) {
    r.configs = append(r.configs, cfg)
}

// BuildAll 遍历所有已注册配置，逐个构建 AgentAsTool。
func (r *Registry) BuildAll(ctx context.Context, builder *runtime.AgentBuilder, toolReg *tools.Registry) ([]adk.Agent, error) {
    agents := make([]adk.Agent, 0, len(r.configs))
    for _, cfg := range r.configs {
        agent, err := builder.Specialist(ctx, cfg, toolReg)
        if err != nil {
            return nil, err
        }
        agents = append(agents, agent)
    }
    return agents, nil
}
```

- [ ] **Step 2: 三个 specialist 文件改为 Register 调用**

三个文件只保留一个 `Register(r *Registry)` 函数，配置集中声明。

`internal/specialists/bazi/specialist.go`：

```go
// Package bazi 注册八字领域专家配置。
package bazi

import (
    "github.com/wikiglobal/suanming-agent/internal/runtime"
    "github.com/wikiglobal/suanming-agent/internal/specialists"
)

// Register 向注册表添加八字专家配置。
func Register(r *specialists.Registry) {
    r.Register(runtime.SpecialistConfig{
        Name:        "bazi_specialist",
        Description: "八字命理专家。根据出生时间排盘、分析用神忌神、解读大运走势。适用于婚恋、事业、财运、健康、性格等八字相关问题。",
        Instruction: `你是八字命理专家。

## 职责
- 根据用户出生信息排八字命盘（调用 bazi_calc）
- 分析日主强弱、取用神忌神（调用 yongshen）
- 解读大运走势（调用 dayun_analyzer）
- 检索古籍原文（调用 knowledge_search）

## 执行流程
1. 有出生信息 → bazi_calc 排盘
2. yongshen 取用神，dayun_analyzer 看大运
3. 关键论断前 → knowledge_search 查古籍
4. 综合解读

## 输出要求
- 中文，专业但不晦涩
- 引用古籍标注出处
- 命盘数据以系统排盘为准`,
        ToolBuilder: runtime.BuildBaziAdapters,
    })
}
```

`internal/specialists/qimen/specialist.go`：

```go
// Package qimen 注册奇门遁甲领域专家配置。
package qimen

import (
    "github.com/wikiglobal/suanming-agent/internal/runtime"
    "github.com/wikiglobal/suanming-agent/internal/specialists"
)

// Register 向注册表添加奇门专家配置。
func Register(r *specialists.Registry) {
    r.Register(runtime.SpecialistConfig{
        Name:        "qimen_specialist",
        Description: "奇门遁甲专家。分析当前时空的吉凶方位、门星神组合。适用于择时、方位选择、时机判断。",
        Instruction: `你是奇门遁甲专家。

## 职责
- 排奇门盘（调用 qimen_dunjia）
- 查古籍（调用 knowledge_search）
- 分析宫、门、星、神组合

## 输出要求
- 中文
- 重点分析时空对事项的有利/不利因素
- 引用古籍标注出处`,
        ToolBuilder: runtime.BuildQimenAdapters,
    })
}
```

`internal/specialists/ziwei/specialist.go`：

```go
// Package ziwei 注册紫微斗数领域专家配置。
package ziwei

import (
    "github.com/wikiglobal/suanming-agent/internal/runtime"
    "github.com/wikiglobal/suanming-agent/internal/specialists"
)

// Register 向注册表添加紫微斗数专家配置。
func Register(r *specialists.Registry) {
    r.Register(runtime.SpecialistConfig{
        Name:        "ziwei_specialist",
        Description: "紫微斗数专家。根据出生信息排盘，分析十二宫星曜、四化飞星、大限流年。",
        Instruction: `你是紫微斗数专家。

## 职责
- 排紫微命盘（调用 ziwei_calc）
- 查古籍（调用 knowledge_search）
- 分析命宫、身宫、三方四正、四化飞星

## 输出要求
- 中文
- 重点：命宫主星 + 四化飞星的吉凶应期
- 引用古籍标注出处`,
        ToolBuilder: runtime.BuildZiweiAdapters,
    })
}
```

对比旧版：每个文件从 ~50 行工厂逻辑浓缩为一次 `Register` 调用，配置集中可见，加新领域只加一个 `Register`。

- [ ] **Step 3: 删除旧测试**

删除 `internal/specialists/bazi/specialist_test.go`、`internal/specialists/qimen/specialist_test.go`。

- [ ] **Step 4: 编译验证**

Run: `go build ./internal/specialists/...`
Expected: PASS

Run: `go build ./internal/runtime/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/specialists/registry.go
git add internal/specialists/bazi/specialist.go internal/specialists/qimen/specialist.go internal/specialists/ziwei/specialist.go
git rm internal/specialists/bazi/specialist_test.go internal/specialists/qimen/specialist_test.go
git commit -m "refactor: specialists use Registry + Register pattern, factory logic in AgentBuilder"
```### Task 4: 重写 executor.runAgent

**Files:**
- Modify: `internal/runtime/executor.go`

这是核心改动。`runAgent` 从创建单个 `fortune_teller` agent 变为创建 supervisor agent + 三个 specialist AgentTool。

- [ ] **Step 1: 修改 Executor 结构体和构造函数**

`Executor` 结构体新增：

```go
type Executor struct {
    model              einomodel.ToolCallingChatModel
    llmModel           string
    promptBuilder      *Builder
    reg                *tools.Registry          // 工具注册表
    specialistRegistry *specialists.Registry    // specialist 配置注册表
    builder            *AgentBuilder             // agent 构建器
    historyLimit       int

    supervisorAgent adk.Agent                   // 预构建，多轮复用
}
```

`NewExecutor` 签名和实现：

```go
func NewExecutor(reg *tools.Registry, model einomodel.ToolCallingChatModel, promptMode string) (*Executor, error) {
    // 注册所有领域专家
    sr := specialists.NewRegistry()
    bazi.Register(sr)
    qimen.Register(sr)
    ziwei.Register(sr)

    return &Executor{
        model:              model,
        promptBuilder:      NewBuilder(promptMode),
        reg:                reg,
        specialistRegistry: sr,
        builder:            NewAgentBuilder(model),
    }, nil
}
```

- [ ] **Step 2: 新增 `initAgents` 延迟初始化方法**

```go
// initAgents 通过 SpecialistRegistry + AgentBuilder 构建所有 agent。
//
// 仅在首次 runAgent 时调用一次。遍历 registry 构建 specialist agent，
// 用 AgentAsTool 包装后注入 supervisor agent。
func (e *Executor) initAgents(ctx context.Context) error {
    if e.supervisorAgent != nil {
        return nil
    }

    reg := e.specialistRegistry  // 在 NewExecutor 中初始化
    builder := runtime.NewAgentBuilder(e.model)

    agents, err := reg.BuildAll(ctx, builder, e.reg)
    if err != nil {
        return fmt.Errorf("build specialists: %w", err)
    }

    tools := make([]einotool.BaseTool, len(agents))
    for i, a := range agents {
        tools[i] = adk.NewAgentTool(ctx, a)
    }

    e.supervisorAgent, err = builder.Supervisor(ctx, tools)
    if err != nil {
        return fmt.Errorf("build supervisor: %w", err)
    }
    return nil
}
```

需要在 executor.go 头部新增 import：

需要在 executor.go 头部新增 import：

```go
import (
    "github.com/cloudwego/eino/adk"
    "github.com/wikiglobal/suanming-agent/internal/specialists"
    "github.com/wikiglobal/suanming-agent/internal/specialists/bazi"
    "github.com/wikiglobal/suanming-agent/internal/specialists/qimen"
    "github.com/wikiglobal/suanming-agent/internal/specialists/ziwei"
)
```

- [ ] **Step 3: 重写 `runAgent`**

```go
func (e *Executor) runAgent(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (string, string, error) {
    // 延迟初始化 agent（首次调用时构建）
    if err := e.initAgents(ctx); err != nil {
        return "", "", fmt.Errorf("init agents: %w", err)
    }

    ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
        Name:       "adk_supervisor_agent",
        Kind:       tracing.KindChain,
        Attributes: map[string]any{"model": e.llmModel},
    })

    runner := adk.NewRunner(ctx, adk.RunnerConfig{
        Agent:           e.supervisorAgent,
        EnableStreaming: true,
    })

    // 构建输入消息
    msgs := e.buildConversationMessages(st, message)

    // SessionValues 传递会话上下文
    vals := map[string]any{
        "profile": st.Profile,
        "domain":  route.PrimaryDomain,
    }
    if st.BaziResult != nil {
        vals["bazi_result"] = st.BaziResult
    }
    if st.QimenResult != nil {
        vals["qimen_result"] = st.QimenResult
    }
    if st.ZiWeiResult != nil {
        vals["ziwei_result"] = st.ZiWeiResult
    }

    iter := runner.Run(ctx, msgs, adk.WithSessionValues(vals))

    finalText, err := agentEventBridge(ctx, sink, iter, func(toolName, resultJSON string) {
        e.saveToolResult(st, toolName, resultJSON)
    })
    if err != nil {
        return "agent_error", finalText, err
    }

    return "agent_reading", finalText, nil
}
```

- [ ] **Step 4: 删除 `dispatchSpecialists` 和 `selectPrimarySpecialist`**

这两个函数整体删除。`Execute()` 中短路由逻辑保留但以新方式实现——资料不全追问移到 specialist agent 的 instruction 里处理。

`Execute()` 简化为：

```go
func (e *Executor) Execute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (turnType string, assistantText string, err error) {
    updateRoutingSnapshot(st, route)

    // 短路由: supervisor 要求澄清
    if route.NeedsClarification {
        question := route.ClarificationQuestion
        if question == "" {
            question = "请确认一下您的需求，我再为您详细分析。"
        }
        sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": question}})
        return "clarification", question, nil
    }

    // 主路径: supervisor agent → specialist
    return e.runAgent(ctx, sink, st, route, message)
}
```

- [ ] **Step 5: `SetSpecialists` 标记弃用**

```go
// SetSpecialists 已弃用 —— specialists 现在由 executor 内部构建。
// 保留以兼容 container.go 的调用，但不再使用传入的实例。
func (e *Executor) SetSpecialists(baziSp, qimenSp, ziweiSp specialists.DomainHandler) {
    // no-op: specialists are now built internally
}
```

- [ ] **Step 6: 编译验证**

Run: `go build ./...`
Expected: 编译通过

- [ ] **Step 7: Commit**

```bash
git add internal/runtime/executor.go
git commit -m "refactor: executor uses supervisor agent + AgentAsTool for specialist dispatch"
```

---

### Task 5: 更新 container wiring

**Files:**
- Modify: `internal/container/container.go`

- [ ] **Step 1: 简化 executor 创建**

`container.go` 中，删除 `executor.SetSpecialists(...)` 调用（因为 executor 现在内部构建 specialist）。

```diff
- executor, err := appRuntime.NewExecutor(reg, runtimeModel, cfg.PromptMode)
+ executor, err := appRuntime.NewExecutor(reg, runtimeModel, cfg.PromptMode)
  if err != nil { panic(err) }
  executor.SetLLMModel(cfg.LLMModel)
  executor.SetHistoryLimit(cfg.ConversationLimit)
- executor.SetSpecialists(baziSp.New(), qimenSp.New(), ziweiSp.New())  # 此行删除
```

同时删除 `local "github.com/wikiglobal/suanming-agent/internal/specialists/bazi"` 等不再需要的 import（如果 container 不再直接引用 specialist）。

- [ ] **Step 2: 编译验证**

Run: `go build ./cmd/server/`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/container/container.go
git commit -m "chore: remove SetSpecialists wiring, executor self-builds agents"
```

---

### Task 6: 旧接口清理

**Files:**
- Delete: `internal/specialists/types.go`

- [ ] **Step 1: 删除 DomainHandler 接口**

`internal/specialists/types.go` 删除。

- [ ] **Step 2: 检查 schemas.DomainResult 引用**

```bash
rg "DomainHandler" internal/ --ignore-case
rg "DomainResult" internal/ --ignore-case
```

如无业务引用，标记 `schemas.DomainResult` 废弃（暂不删，前端可能引用）。

- [ ] **Step 3: 全量编译**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: remove deprecated DomainHandler interface and SetSpecialists"
```

---

### Task 7: 集成测试（手动验证）

- [ ] **Step 1: 启动服务**

```bash
LLM_API_KEY=sk-xxx go run ./cmd/server/
```

- [ ] **Step 2: 测试八字路由**

发送消息："我是1990年5月15日早上8点出生的男生，帮我算一下事业运"

Expected: supervisor agent 路由到 `bazi_specialist`，SSE 事件流中能看到 bazi_calc 工具调用和八字解读文本

- [ ] **Step 3: 测试奇门路由**

发送消息："今天适合签约吗？"

Expected: supervisor agent 路由到 `qimen_specialist`，SSE 事件流中能看到 qimen_dunjia 工具调用

- [ ] **Step 4: 测试跨领域路由**

发送消息："帮我看看八字婚姻，顺便看看今天适不适合相亲"

Expected: supervisor agent 先调 `bazi_specialist` 分析婚姻，再调 `qimen_specialist` 择时

- [ ] **Step 5: 测试紫微路由**

发送消息："帮我看一下紫微斗数命盘，1990年5月15日早上8点出生的男生"

Expected: supervisor agent 路由到 `ziwei_specialist`

---

## 关键决策

1. **Agent 复用策略**：supervisor agent + 三个 specialist agent 在首次 `runAgent` 时构建，后续轮次复用（不重建）。这避免了每轮重建 agent 的开销，但意味着 Instruction 不能随会话状态动态变化——需依赖 SessionValues 传递动态上下文

2. **supervisor.Client 保留但不参与 AgentAsTool 链路**：三层防御提取的 `route` 仍然用于 orchestrator 层面的短路由判断（追问/澄清），但不再用于选择 specialist——这个职责交给 supervisor agent 自己

3. **knowledge_search 工具跨 specialist 共享**：每个 specialist 都注册 knowledge_search——它是无状态检索工具，不需要独占

## 回滚方案

如果 AgentAsTool 链路出现问题：

1. 将 `runAgent` 恢复为创建单个 `fortune_teller` agent 挂全部工具（旧逻辑保留为 `runAgentFallback`）
2. 关键降级触发条件：`initAgents` 失败时自动走 fallback
3. container 中 `SetSpecialists` 重新激活

## 风险

| 风险 | 等级 | 缓解 |
|------|------|------|
| AgentAsTool 内部 gob 序列化有边界 bug | 低 | Eino 已发布稳定版，agent_tool.go 代码成熟 |
| 多 agent 链式调用导致 token 消耗翻倍 | 中 | supervisor agent instruction 极简（只做路由），specialist 才做重推理 |
| EmitInternalEvents 事件桥接与现有 bridge.go 冲突 | 中 | 方案中不设 EmitInternalEvents，由 supervisor agent 收集子 agent 输出后统一 emit |
| 测试覆盖不足 | 中 | Task 7 手动验证覆盖核心路径；单元测试后续补充 |
