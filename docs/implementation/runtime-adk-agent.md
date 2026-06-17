# M7: Runtime ADK Agent 改造方案

**状态：** 方案稿 · 待评审
**目标：** 将 `internal/runtime/` 的手工工具编排替换为 Eino ADK ChatModelAgent 自动编排
**工作量预估：** 3-4 天（含测试适配）

---

## 1. 当前问题

`internal/runtime/`（~1600 行含测试）的核心问题是 **Go 代码硬编码了工具调用顺序**：

```go
// internal/runtime/bazi.go handleFullReading
func (e *Executor) handleFullReading(...) {
    data := e.runBaziCalc(ctx, sink, st.Profile)       // 步骤 1
    e.runOptionalYongshen(ctx, sink, st)                // 步骤 2
    e.runOptionalDayun(ctx, st, data)                   // 步骤 3
    emitChartComponent(ctx, sink, "bazi-chart", data)    // 步骤 4
    return e.answerWithKnowledge(ctx, sink, st, "bazi") // 步骤 5+6
}
```

这种写法的缺陷：
- 调用顺序固定，模型无法根据对话上下文调整（如跳过用神直答大运）
- 每个 `run*` 函数重复同一个模板：`tools.Get() → type assertion → Execute() → error check → type assertion → store → emit`
- 8 个 `execute*Route` 函数各自维护一套不同的定序逻辑
- 新增领域（如大六壬）需要再写一套 `run*` 函数

**核心指标：** 6 次工具调用模板重复，~400 行样板代码。

---

## 2. 改造目标

```
当前:                             改造后:
ExecuteRoute() switch (8 路)      ExecuteRoute() → ADK ChatModelAgent
├─ executeCollectProfileRoute()     └─ ReAct loop (模型动态定序)
│  ├─ runBaziCalc()                    ├─ bazi_calc → 结果 → 决定下一步
│  ├─ runOptionalYongshen()           ├─ yongshen →
│  ├─ runOptionalDayun()              ├─ dayun_analyzer →
│  └─ answerWithKnowledge()           ├─ qimen_dunjia →
├─ executeQimenPrimaryRoute()         ├─ ziwei_calc →
│  ├─ runCurrentQimen()               ├─ knowledge_search →
│  └─ answerWithKnowledge()           └─ LLM generate → 最终回答
└─ executeZiweiPrimaryRoute()
   ├─ ziwei_calc()
   └─ answerWithKnowledge()
```

**不变的部分：**
- `state.SessionState` 会话状态模型
- `tools.Registry` 工具注册表（只加适配层不改核心）
- `prompt.go` 中的 prompt 构建逻辑
- `policy.ApprovedRoute` 路由审批契约
- SSE 事件类型体系（thinking/tool_call/component/text/done）

**去除的部分：**
- `candidate.go` — 8 个 `execute*Route` 函数合并/删除
- `bazi.go` — `runBaziCalc`/`runOptionalYongshen`/`runOptionalDayun`/`runCurrentQimen`
- `qimen.go` — `executeQimenPrimaryRoute`/`executeParallelFortuneRoute`
- `ziwei.go` — `executeZiweiPrimaryRoute`
- `answer.go` — `runKnowledgeSearch`/`answerWithKnowledge`

**新增的部分：**
- `adapter.go` — 工具适配层（~100 行）
- `agent.go` — ADK ChatModelAgent 构建（~80 行）
- `bridge.go` — ADK AgentEvent → SSE Event 桥接（~80 行）
- `executor.go` — 裁剪后的入口（~80 行，从 153 行缩减）

---

## 3. 备份策略

```bash
cp -r internal/runtime internal/runtime.bak.$(date +%Y%m%d)
```

备份后删除 `candidate.go`/`bazi.go`/`qimen.go`/`ziwei.go`/`answer.go`，然后重新实现。

测试不改：`answer_test.go`/`prompt_test.go` 保留，新增 `executor_test.go` 覆盖 agent run。

---

## 4. 详细设计

### 4.1 工具适配层（`adapter.go`）

现有工具接口：
```go
// tools.Registry
tool, ok := e.tools.Get("bazi_calc")
result, err := tool.Execute(ctx, params)       // params: map[string]any
data, ok := result.(map[string]any)             // result: map[string]any
```

ADK 需要 `tool.BaseTool`（通过 `utils.InferTool` 创建）：
```go
tool, err := utils.InferTool("bazi_calc", "排八字命盘",
    func(ctx context.Context, input *BaziCalcInput) (*BaziCalcOutput, error) {
        // 内部调用 tools.Registry
    })
```

设计：每个工具写一个适配函数，在函数体内调用 `tools.Registry` 的 `Execute()`。适配函数负责：
- 将 Eino 的 typed input 转为 `map[string]any`
- 调用 `tool.Execute(ctx, params)`
- 将 `map[string]any` 结果转为 typed output（或直接返回 string）
- 通过 `adk.AddSessionValue(ctx, key, value)` 将结果写回会话状态

```go
func newBaziCalcAdapter(reg *tools.Registry) (tool.BaseTool, error) {
    return utils.InferTool("bazi_calc", "根据出生时间推算八字四柱、日主、五行旺衰、大运起运时间等完整命盘信息",
        func(ctx context.Context, input *baziCalcInput) (string, error) {
            params := map[string]any{
                "year":   input.Year,
                "month":  input.Month,
                "day":    input.Day,
                "hour":   input.Hour,
                "gender": input.Gender,
            }
            result, err := reg.Get("bazi_calc").Execute(ctx, params)
            if err != nil {
                return "", err
            }
            data := result.(map[string]any)
            adk.AddSessionValue(ctx, "bazi_result", data)
            jsonBytes, _ := json.Marshal(data)
            return string(jsonBytes), nil
        })
}
```

工具清单（6 个）：

| Eino 工具名 | 注册的 Go 工具 | 输入参数 | 说明 |
|------------|---------------|---------|------|
| `bazi_calc` | `bazi_calc` | year/month/day/hour/gender | 八字排盘 |
| `yongshen` | `yongshen` | day_master/bazi_result | 用神分析（可选） |
| `dayun_analyzer` | `dayun_analyzer` | bazi_result/age | 大运分析（可选） |
| `qimen_dunjia` | `qimen_dunjia` | time/term_id/ju | 奇门遁甲 |
| `ziwei_calc` | `ziwei_calc` | year/month/day/hour/gender | 紫微斗数 |
| `knowledge_search` | `knowledge_search` | query/topK | 知识库检索 |

### 4.2 Agent 构建（`agent.go`）

```go
func NewRuntimeAgent(ctx context.Context, tools []tool.BaseTool, systemPrompt string) (adk.Agent, error) {
    return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:          "fortune_teller",
        Description:   "命理大师，可调用排盘、分析、检索等工具",
        Instruction:   systemPrompt,
        Model:         llmModel,       // Eino ToolCallingChatModel
        ToolsConfig: adk.ToolsConfig{
            ToolsNodeConfig: compose.ToolsNodeConfig{
                Tools: tools,
            },
        },
        MaxIterations: 12,             // 正常 4-6 次，留余量
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

关键配置：
- **MaxIterations=12**：正常解读流程约 4-6 轮（排盘+用神+大运+检索+回答），留一倍余量
- **ModelRetryConfig**：网络层重试，2 次 + 1s backoff
- **无 Summarization middleware**：session 上下文已在系统 prompt 中构建好，对话轮次管理仍由 orchestrator 的 `recordTurnAndMaintainContext` 负责

### 4.3 SSE 桥接（`bridge.go`）

ADK `AgentEvent` → 项目 `runtime.Event` 的映射：

| ADK AgentEvent | 映射为 SSE Event | 说明 |
|---------------|-----------------|------|
| Output (role=assistant) | `text` | 模型的回答文本 |
| Output (role=tool) | `tool_call` | 工具调用事件 |
| Action (Interrupted) | `thinking` | 中间状态更新 |
| Err | `error` | 错误事件 |
| 流结束 | `done` | 对话轮次结束 |

实现方式：在 `Executor.Execute()` 中消费 `AsyncIterator`，同步转发到 `EventSink.Emit()`。

```go
func (e *Executor) runAgent(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute) (string, error) {
    runner := adk.NewRunner(ctx, adk.RunnerConfig{
        Agent:           e.agent,
        EnableStreaming: false,
    })

    iter := runner.Query(ctx, buildUserPrompt(st, route))

    var finalText string
    for {
        event, ok := iter.Next()
        if !ok {
            break
        }
        if event.Err != nil {
            return "", event.Err
        }
        e.bridgeEvent(ctx, sink, event)     // → SSE
        if event.Output != nil && event.Output.MessageOutput != nil {
            msg, _ := event.Output.MessageOutput.GetMessage()
            if msg.Role == schema.Assistant && msg.Content != "" {
                finalText = msg.Content
            }
        }
    }
    return finalText, nil
}
```

### 4.4 会话状态同步

工具调用过程中通过 `adk.AddSessionValue()` 写入的结果，在 agent run 结束后统一同步回 `state.SessionState`：

```go
func (e *Executor) syncSessionState(ctx context.Context, st *state.SessionState) {
    values := adk.GetSessionValues(ctx)
    if v, ok := values["bazi_result"]; ok {
        st.BaziResult = v.(map[string]any)
    }
    if v, ok := values["qimen_result"]; ok {
        st.QimenResult = v.(map[string]any)
    }
    if v, ok := values["ziwei_result"]; ok {
        st.ZiWeiResult = v.(map[string]any)
    }
}
```

### 4.5 Executor 入口

裁剪后的 `executor.go`：

```go
func (e *Executor) Execute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (turnType, assistantText string, err error) {
    updateRoutingSnapshot(st, route)

    // 短路由 1: specialist 直接给最终答复（资料不全需澄清）
    if final, summary := e.dispatchSpecialists(ctx, sink, st, route); final {
        sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": summary}})
        return "ask_missing_profile", summary, nil
    }

    // 短路由 2: supervisor 明确要求澄清
    if route.NeedsClarification {
        return e.executeClarification(ctx, sink, st, route)
    }

    // 主路径: ADK agent run
    return e.executeAgent(ctx, sink, st, route, message)
}
```

**不放进 agent 的两种场景**（继续留在 Go 侧做短路由）：
1. `NeedsClarification=true` — 不启动模型，直接返回澄清问题
2. `specialist` 直接返回 `Final=true` — 如资料不全时直接提示补充（不走模型）

### 4.6 Prompt 构建策略

现有 `Builder.BuildInterpretPrompt()` 将 profile、chart data、knowledge passages、conversation history 全部拼进系统提示词。改造后：

- **系统提示词**（`prompts/interpret.md`）保持为 `ChatModelAgent.Instruction`，只保留回答风格、引用要求、角色定义
- **会话上下文**（profile、已有命盘、知识检索结果）不再注入系统提示词，而是通过 ADK 工具调用的结果消息自然存在于对话历史中
- **session 摘要**（`RunningSummary`、`RecentTurns`）仍注入系统提示词，因为它们是 orchestrator 维护的外部状态

`builder.go` 简化：`BuildInterpretPrompt()` 拆为两部分：
- `BuildAgentInstruction()` → 返回 Instruction（加载 `interpret.md` + 回答指导 + 会话摘要）
- `BuildAgentMessages()` → 构造对话消息（用户问题）——不再包含 chart data

---

## 5. 边界情况处理

| 场景 | 处理方式 |
|------|---------|
| **direct_bazi**（用户直接输入四柱） | 正则检测后走一次 `bazi_calc` 工具 + 写入 session state，然后 agent 正常解读 |
| **profile 收集+排盘同轮** | agent 先调 `bazi_calc`（工具内部访问 `SessionValues` 拿 profile），再调其他工具 |
| **知识检索为空** | `knowledge_search` 返回空结果，模型降级为无引用回答 |
| **工具注册缺失** | `adapter.go` 构建时若 `reg.Get()` 返回 false，跳过该工具不注册到 ADK |
| **模型不可用** | ADK `ModelRetryConfig` 2 次重试后交给 Go 侧 `safeFallback` |
| **ReAct 超限** | `MaxIterations=12` 达到后 agent 返回错误，orchestrator 捕获后降级 |

---

## 6. 文件变更清单

```
internal/runtime/
├── event.go          ··· 不变（15 行）
├── timing.go         ··· 不变（11 行）
├── prompt.go         ··· 微调（拆 Instruction 和 messages）
├── executor.go       ··· 重写（从 153 行缩到 ~80 行）
├── candidate.go      ··· 删除（274 行，被 agent 替代）
├── bazi.go           ··· 删除（253 行，被 agent + adapter 替代）
├── qimen.go          ··· 删除（98 行）
├── ziwei.go          ··· 删除（78 行）
├── answer.go         ··· 删除（145 行）
├── adapter.go        ··· 新增（~100 行，工具适配层）
├── agent.go          ··· 新增（~80 行，ADK agent 构建）
└── bridge.go         ··· 新增（~80 行，SSE 桥接）
```

净减 ~700 Go 行（不包括测试）。

现有测试文件：
- `answer_test.go` — 基本测试 stream 和 answer 管线。改造后测试 agent run 的同步逻辑。
- `prompt_test.go` — 测试 prompt builder。保留，`BuildAgentInstruction()` 替换 `BuildInterpretPrompt()`。

---

## 7. 风险与缓解

| 风险 | 概率 | 缓解 |
|------|------|------|
| ADK ReAct 循环中模型乱调工具 | 低 | 工具描述 + system prompt 约束；ToolsConfig 不注册无关工具 |
| 会话状态同步遗漏 | 中 | `syncSessionState` 做完整字段映射；测试验证每个工具的结果写入 |
| 性能退化（ADK 框架开销） | 低 | ADK ReAct loop 比手工编排多一次模型调用（决定调哪个工具），但整体可控 |
| ADK 版本升级 API 变化 | 低 | 用 `eino` 已 vendor 的版本（`eino-agent/eino/`），不依赖外部升级 |
| `utils.InferTool` 反射限制 | 中 | 若 `map[string]any` 作为 input/output 类型不受支持，改用 string input + string output 返回 JSON |

---

## 8. 验证方案

1. `go test ./internal/runtime/... -v` — 全部通过
2. `go build ./cmd/server/` — 编译通过
3. 手动测试 3 种路由：八字全流程、奇门主链、紫微主链
4. 验证 SSE 事件顺序与改造前一致（thinking → tool_call → text → component → text → done）
5. 验证会话状态持久化正确（命的 BaziResult 在追问轮中可复用）
