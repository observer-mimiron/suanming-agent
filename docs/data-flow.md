# 数据链路：用户消息 → AI 回答

一条用户消息从此处进入，最终以 SSE 事件流返回前端。本文档描述完整的调用链路、每个节点的职责、关键文件和数据结构。

## 总览

```
用户消息
  │
  ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Orchestrator.Run()                            │
│  ① 取会话锁 → ② 加载状态 → ③ 启动 Trace                        │
│  ④ 路由决策 → ⑤ Preflight → ⑥ Agent 执行 → ⑦ SSE 汇总         │
└─────────────────────────────────────────────────────────────────┘
  │
  ▼
SSE 事件流 ──► 前端 useSSE.ts 解析 ──► AssistantTurn.vue 渲染
```

---

## 第一阶段：入口

```
用户消息 → POST /api/chat
  → handler/chat.go → Orchestrator.Run(ctx, sink, sessionID, message)
```

**关键文件：**

| 文件 | 职责 |
|------|------|
| [internal/handler/chat.go](internal/handler/chat.go) | HTTP handler，解析 `session_id`、创建 SSE Writer |
| [internal/orchestrator/orchestrator.go](internal/orchestrator/orchestrator.go) | 会话主控：取锁、加载状态、驱动全流程 |

**会话状态** `SessionState` 定义在 [internal/state/session.go](internal/state/session.go)，包含：
- `Profile` — 出生资料（year/month/day/hour/gender/birthplace）
- `BaziResult` / `QimenResult` / `ZiWeiResult` — 已排盘的领域结果
- `Routing` — 上一轮的路由快照
- `RecentTurns` — 对话历史
- `RunningSummary` — 溢出窗口的滚动摘要
- `Guidance` — conversation guidance 状态

---

## 第二阶段：路由决策

```
Orchestrator.Run()
  →
  supervisor.Approve(ctx, message, sessionState)
    → RouteAdvisor (Go ADK RouteEngine)
      → ADK structured 路由
      → textDecide (文本降级)
      → safeFallback (兜底)
```

**三层降级链路：**

```
ADK structured output (L0 意图 → L1 领域 → L2 任务 → L3 槽位)
  │ 失败
  ▼
textDecide (LLM 文本路由)
  │ 失败
  ▼
safeFallback (硬编码默认路由)
```

**产出：** `ApprovedRoute` 结构体，包含：

| 字段 | 含义 | 示例 |
|------|------|------|
| `ConversationIntent` | 对话意图 | `consult` |
| `PrimaryDomain` | 主领域 | `bazi` / `qimen` / `ziwei` |
| `SecondaryDomains` | 辅领域 | `["ziwei"]` |
| `TaskIntent` | 任务意图 | `interpret_chart` / `collect_profile` |
| `Slots.Profile` | 提取的出生资料 | `{year:2010, month:10...}` |
| `Slots.TargetSubject` | 咨询主题 | `婚姻` / `事业` |
| `PolicyHints.QimenMode` | 奇门模式 | `none` / `primary` / `supplement` |
| `Confidence` | 置信度 | `0.95` |

**关键文件：**

| 文件 | 职责 |
|------|------|
| [internal/supervisor/](internal/supervisor/) | RouteAdvisor 实现 + 路由 prompt |
| [prompts/supervisor/](prompts/supervisor/) | 路由 prompt 模板 |
| [internal/policy/](internal/policy/) | `ApprovedRoute` 类型定义 |

---

## 第三阶段：Preflight（确定性预检）

```
executor.Execute(ctx, sink, st, route, message)
  →
  preflight(st, route, message)
```

**职责：** 在进入 LLM Agent 之前做确定性硬判断，可能的短路返回：

| 场景 | 行为 |
|------|------|
| 缺少出生资料且需要排盘 | 返回澄清提问 → **短路，不发 Agent** |
| 已排过盘，用户只是追问 | 放行，复用已有命盘 |
| guidance state 中的 forced route | 替换当前 route |

**关键文件：**

| 文件 | 职责 |
|------|------|
| [internal/runtime/preflight.go](internal/runtime/preflight.go) | Preflight 逻辑 |
| [internal/runtime/guidance_gate.go](internal/runtime/guidance_gate.go) | Guidance 状态机 |

---

## 第四阶段：Prefill（确定性排盘）

```
executor.prefill(ctx, sink, st, route, vals)  // 在 runAgentRoute 中调用
  →
  按领域执行确定性工具链:
    bazi → bazi_calc → yongshen → dayun_analyzer → knowledge_search
    qimen → qimen_dunjia
    ziwei → ziwei_calc → ziwei_liunian
```

**核心原则：** 所有排盘、用神、大运计算由 Go 确定性执行，**LLM Agent 不接触排盘工具**。排盘结果通过 `vals` 注入 Agent 的 SessionValues，LLM 只做解读。Specialist 的工具列表仅包含 `knowledge_catalog` 和 `knowledge_search`。

**关键文件：**

| 文件 | 职责 |
|------|------|
| [internal/runtime/executor.go](internal/runtime/executor.go) | `prefill` / `prefillBazi` / `prefillQimen` / `prefillZiWei` |

---

## 第五阶段：Agent 路由与执行

```
executor.runAgentRoute(ctx, sink, st, route, message)
  │
  ├─ ① prefill(ctx, sink, st, route, vals)
  │     确定性排盘链，结果注入 vals 和 SessionState
  │
  ├─ ② AgentBuilder.BuildSupervisor(route, st, allowed_specialists)
  │     每轮动态构建 supervisor agent
  │     → 内部为每个 allowed specialist 调用 BuildSpecialist + NewAgentTool
  │     instruction = buildSupervisorInstruction(route, allowed)
  │     「你是命理咨询执行主管…本轮批准的主领域是 bazi…」
  │
  └─ ③ AgentBuilder.BuildSpecialist(cfg, sessionState)
        每个 specialist 注入:
        · 基础 instruction（从 prompts/interpret.md 加载）
        · 运行时上下文（profile、命盘数据块、当前日期）
        · 工具列表: knowledge_catalog + knowledge_search
```

### Supervisor Agent

**角色：** 纯调度，不回答。**禁止做命理分析。**

**instruction 来源：** [agent_route.go](internal/runtime/agent_route.go) `buildSupervisorInstruction()`

**可见的 AgentTool：** 根据 `ApprovedRoute` 动态过滤：
- 始终包含主域 specialist
- 辅域仅当 `SecondaryDomains` 或 `QimenMode` 明确指定时可见
- 婚姻/感情类问题自动包含 ziwei（分析夫妻宫）

### Specialist Agent（以 bazi 为例）

**角色：** 八字命理专家，负责完整的分析流程。

**instruction 来源：** 基础指令从 `prompts/interpret.md` 加载（[internal/specialists/bazi/specialist.go](internal/specialists/bazi/specialist.go) 注册时读取），运行时上下文由 `AgentBuilder.BuildSpecialist()` 注入。

**instruction 组装过程：**

```
基础 instruction (prompts/interpret.md):
  「你是八字命理专家…」
  + 可调用工具: knowledge_catalog / knowledge_search
  + 知识检索流程 (目录探索→证据规划→受控检索→质量评估→引用回答)
  （排盘/用神/大运由 prefill 确定性执行，不在 specialist 工具列表中）

运行时上下文注入 (AgentBuilder.BuildSpecialist):
  + buildProfileSection(st)     — 出生资料
  + buildBaziDataBlock(st)      — 命盘数据摘要 (四柱十神/五行/用神/大运/神煞)
  + buildQimenDataBlock(st)     — 奇门盘数据摘要（若已排盘）
  + buildZiWeiDataBlock(st)     — 紫微命盘数据摘要（若已排盘）
  + 当前日期、时区

对话历史 (buildConversationMessages):
  + RecentTurns (最近 N 轮对话)
  + 当前用户消息
```

**工具限制：** 知识检索最多 3 次 `knowledge_search` 调用，由 adapter 闭包计数器硬控。

**关键文件：**

| 文件 | 职责 |
|------|------|
| [internal/runtime/agent_route.go](internal/runtime/agent_route.go) | `AgentBuilder` / `BuildSupervisor` / `BuildSpecialist` / `buildBaziDataBlock` / `buildQimenDataBlock` / `buildZiWeiDataBlock` |
| [internal/specialists/bazi/specialist.go](internal/specialists/bazi/specialist.go) | 八字 specialist 注册 + instruction 加载 |
| [internal/specialists/qimen/specialist.go](internal/specialists/qimen/specialist.go) | 奇门 specialist 注册 |
| [internal/specialists/ziwei/specialist.go](internal/specialists/ziwei/specialist.go) | 紫微 specialist 注册 |
| [internal/specialists/types.go](internal/specialists/types.go) | `Config` 和 `Registry` 类型定义 |

---

## 第六阶段：Agent 事件桥接

```
runner.Run(ctx, messages)
  → adk.AsyncIterator[*adk.AgentEvent]
  → agentEventBridge(ctx, sink, iter)
```

**agentEventBridge** 消费 ADK 的异步事件流，桥接到 SSE：

| Agent 事件 | SSE 事件 | 说明 |
|------------|----------|------|
| Assistant 消息 (含 ToolCalls) | `thinking` | supervisor 内部推理 |
| Specialist AgentAsTool 响应 | `text`（主文本） | 名称以 `_specialist` 结尾的响应不走 tool_call |
| 普通 Tool 调用完成 | `tool_call` | 展示工具名和结果 |
| Tool 结果中含排盘 JSON | `component: bazi-chart` 等 | 渲染命盘卡片 |
| `<analysis>` XML 段 | `thinking` | parseXMLSections 解析后推送 |
| `<response>` XML 段 | `text` | parseXMLSections 解析后推送 |

**最终回答格式：** LLM 输出含 XML 标签：

```xml
<analysis>
（内部推理过程，不展示给用户）
</analysis>
<response>
（最终回答，展示给用户）
</response>
```

**关键文件：**

| 文件 | 职责 |
|------|------|
| [internal/runtime/bridge.go](internal/runtime/bridge.go) | `agentEventBridge` / `parseXMLSections` |
| [internal/runtime/final_guard.go](internal/runtime/final_guard.go) | 回答后置校验 |
| [internal/sse/writer.go](internal/sse/writer.go) | SSE 事件推送 |

---

## 第七阶段：SSE 汇总与推送

```
orchestrator.emitTracePanels(ctx, sink, turnType)
  →
  → SSE: component → route-decision     (路由快照)
  → SSE: component → process-panel      (产品过程面板)
  → SSE: component → debug-trace        (调试追踪)
  → SSE: component → execution-tree     (统一执行链路树)
  → SSE: done
```

**6 种 SSE 事件类型：**

| 事件 | 含义 | 前端渲染 |
|------|------|----------|
| `thinking` | 内部思考过程 | ThinkingSegment |
| `tool_call` | 工具调用 | ToolCallSegment |
| `component` | 结构化组件 | 命盘卡片 / 过程面板 / 执行链路 |
| `text` | 文本回答 | 正文内容 |
| `error` | 错误信息 | 错误提示 |
| `done` | 本轮结束 | 停止 loading |

**关键文件：**

| 文件 | 职责 |
|------|------|
| [internal/tracing/execution_tree.go](internal/tracing/execution_tree.go) | `BuildExecutionTree` — 按语义阶段分组 span |
| [internal/tracing/process_digest.go](internal/tracing/process_digest.go) | `BuildProcessDigest` — 产品过程摘要 |
| [internal/tracing/debug_digest.go](internal/tracing/debug_digest.go) | `BuildDebugDigest` — 调试追踪摘要 |
| [internal/tracing/eino_callback.go](internal/tracing/eino_callback.go) | Eino 回调钩子 — LLM/Tool/Retriever span 自动记录 |

---

## 第八阶段：前端渲染

```
SSE 事件流
  → useSSE.ts (XHR 流式解析)
  → ChatMessage.segments[]
  → buildAssistantTurnViewModel()
  → AssistantTurn.vue
```

**ViewModel 构建流程** ([assistantTurn.ts](web/src/utils/assistantTurn.ts))：

```
segments 遍历:
  text          → answerBlocks        (Markdown 正文)
  thinking      → debugEvents         (思考面板)
  tool_call     → debugEvents         (工具调用记录)
  component:
    bazi-chart   → resultBlocks       (八字命盘卡片)
    qimen-chart  → resultBlocks       (奇门盘卡片)
    ziwei-chart  → resultBlocks       (紫微盘卡片)
    process-panel → process            (过程面板)
    debug-trace   → debugTrace         (调试追踪)
    execution-tree → debugTrace.root  (执行链路树)
    knowledge-sources → evidence      (古籍出处卡片)
  error         → errors              (错误提示)
```

**前端组件树：**

```
ChatPanel.vue
  └─ AssistantTurn.vue
       ├─ ResultBlock + BaziChartCard    (命盘卡片)
       ├─ ThinkingSegment                (思考过程)
       ├─ markdown 正文                  (AI 回答)
       ├─ TracePanel                     (过程面板)
       ├─ DebugTracePanel                (执行链路)
       │    └─ ExecutionNodeItem × N    (链路节点)
       └─ KnowledgeSourceCard            (古籍出处)
```

---

## 数据流中的关键数据结构

### TurnTrace → 执行链路树的映射

后端收集的所有 span 通过 `BuildExecutionTree()` 按语义阶段分组：

```
Span 名称                          → 语义阶段
supervisor_model / output          → 路由决策
preflight / policy_gate            → 执行前校验
prefill                            → 预填充复用
bazi_calc / yongshen / dayun       → 命盘计算
knowledge_catalog / knowledge_search → 知识检索
adk_supervisor_agent / *_specialist → 专家分析
sse_emit_batch                     → SSE 输出
```

### Span 属性（通过 Eino 回调自动记录）

| Span 类型 | 记录的属性 | 来源 |
|-----------|-----------|------|
| LLM | `model`, `output_tokens`, `gen_ai.usage.*` | eino_callback.go ChatModel handler |
| Tool | `args` (JSON), `response` (JSON) | eino_callback.go Tool handler |
| Retriever | `query`, `top_k`, `filter`, `hits`, `degrade_reason` | eino_callback.go Retriever handler |

---

## 会话持久化

`orchestrator.RecordTurnAndMaintainContext()` 在每轮结束后：

1. 记录用户消息和助手回复文本
2. 裁剪 `RecentTurns` 窗口（溢出时用 flash 模型生成滚动摘要）
3. 持久化到 `data/sessions/{sessionID}.json`

**注意：** 仅持久化文本，不存储 SSE 事件链。这意味着刷新页面后，历史消息不含 process panel、执行链路等结构化内容。

---

## 领域扩展指南

新增领域（如测字/风水）需要的步骤：

1. `internal/specialists/{domain}/specialist.go` — 定义 `Config` + `Register()`，instruction 从 prompt 文件加载
2. `internal/container/container.go` — 在 `BuildContainer` 中注册新 specialist
3. `internal/runtime/agent_route.go` — 在 `allowedSpecialists` 中注册领域路由规则
4. `internal/runtime/executor.go` — 添加 `prefill{Domain}` 确定性排盘逻辑
5. `internal/tracing/execution_tree.go` — 在 `phaseGroups` 中添加 span 名称到语义阶段的映射
6. `prompts/supervisor/unified_router.md` — 更新路由 prompt 的领域定义
7. 前端：添加对应的 ChartCard 组件并在 `assistantTurn.ts` 中注册
