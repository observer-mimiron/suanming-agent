# Eino 原生能力增补方案

> **关联文档:** [11-graph-dispatch.md](/Users/wikiglobal/workSapce/suanming-agent/docs/architecture/supervisor/11-graph-dispatch.md) — dispatch Graph 化已驳回，排盘 Graph 化保留
>
> **修订:** 2026-06-24 — 砍掉原 A1 Model Failover（备选模型不实现）；A2 改为以 TurnLoop 为目标架构；补齐 B1/B2 落地细节；新增"框架原生能力盘点"章节

**Goal:** 把项目里的自研胶水下放给 Eino ADK 原生能力。优先消灭：快速切话题时的僵尸 agent、长对话 token 膨胀、工具结果过长。

**Tech Stack:** Eino ADK（本地 fork `./eino-agent/eino`，go.mod 标 v0.9.6，版本可升级；`replace` 指令是否仍需另行重评）；现有 Runner / ChatModelAgent / SSE pipeline。

**原则:** 凡 ADK 已有原生能力的，不再自研。下文每项都标注"下放了什么 / 保留了什么"。

---

## 能力清单

| 能力 | Eino API | 下放什么 | 优先级 |
|------|----------|---------|--------|
| Cancel + Preemption | `adk.TurnLoop` + `WithPreempt` | 终止上一轮、多轮批处理、idle 超时、优雅关闭 | **高** |
| Summarization 中间件 | `adk/middlewares/summarization` | 长对话压缩 | **中** |
| ToolReduction 中间件 | `adk/middlewares/reduction` | 工具结果截断 | **中** |
| Dynamic Interrupt + Checkpoint | `compose.StatefulInterrupt` + `WithCheckPointStore` | 排盘后追问确认再继续 | **低** |

砍掉原 A1 Model Failover：备选模型不实现，维持现有 `ModelRetryConfig`（同模型重试 2 次）。

---

## A2: Cancel + Preemption（TurnLoop 为目标架构）

**当前状态:** `orchestrator.Run` 是请求级（HTTP 每次请求 → `locker.Lock(sessionID)` → Execute → Save → Unlock）。`executor.go:180` 的 `runner.Run` 无 cancel 通路。快速切话题时上一轮 agent 仍在跑，产生僵尸响应。

**决策:采用 TurnLoop（框架原生）。** cancelFn 方案见下文"为何不选 cancelFn"。

### TurnLoop 接入

ADK 的 `TurnLoop`（`eino-agent/eino/adk/turn_loop.go`）就是为"交互式应用 + preemption + idle 管理 + push 输入"设计的。迁过去后，以下自研逻辑全部下放：

| 自研现状 | 下放给 TurnLoop |
|---------|----------------|
| `o.locker.Lock(sessionID)` 串行化每轮 | TurnLoop 单线程事件处理，天然串行 |
| 新请求到达要 cancel 上一轮 | `loop.Push(msg, adk.WithPreempt(adk.AfterChatModel))` |
| 无 idle 管理（session 永驻内存） | `UntilIdleFor(d)` 自动停止 |
| 无优雅关闭 | `loop.Stop(adk.WithGraceful())` |
| 多轮批处理（用户连发两条消息） | `GenInput` 拿到 `[]items` 自行合并 |

**迁移形态:** 每个 session 一个长生命周期 TurnLoop goroutine。HTTP handler `Push(msg)` 后通过 `OnAgentEvents` 回调把事件桥到 SSE。`Orchestrator.Run` 从"执行一轮"改为"投递一条消息并等本轮事件流结束"。

**成本:** orchestrator + HTTP handler + session store 都要改。session 生命周期从"按请求加载/保存"变成"按 TurnLoop 实例驻留"。session 持久化策略要重设（idle 超时落盘、进程重启恢复）。

**风险:** SSE 长连接要和 `OnAgentEvents` 回调对齐；TurnLoop 是 push 模型，现有请求级 tracing/span 结构要适配。

### 前置决策：RunningSummary 归属（2026-06-25）

TurnLoop 迁移前必须先定 `RunningSummary` 的归属，否则 session 生命周期改造时会卡住。

**决策：RunningSummary 留 session store（外层持久化），TurnLoop 不接管。**

理由：
1. **TurnLoop 是内存态**——进程重启后 TurnLoop 实例消失，`RunningSummary` 必须靠外层 store 恢复。
2. **RunningSummary 是跨 run 的持久化状态**——本质属于 session 而非 agent run。TurnLoop 管运行时驻留，外层 store 管跨进程持久化，职责分离。
3. **`buildConversationMessages` 的注入逻辑保留**——TurnLoop 迁移后，由 TurnLoop 的 input 拼装环节调用 `buildConversationMessages`（它读 `st.RunningSummary` 注入 SystemMessage）。函数签名不变，调用时机从"每次 request"变成"TurnLoop 投递消息时"。

**TurnLoop 迁移时 session store 的职责：**
- 持久化 `RecentTurns` + `RunningSummary` + `BaziResult` + `Profile`（现有，不变）
- TurnLoop 实例管理：sessionID → TurnLoop 实例的映射
- idle 超时落盘：TurnLoop `UntilIdleFor(d)` 触发后，保存 session state 并销毁 TurnLoop 实例
- 进程重启恢复：启动时从磁盘加载活跃 sessions（可选，或 lazy 加载——首次 Push 时再创建 TurnLoop）

### 为何不选 cancelFn

考虑过的替代方案：`adk.WithCancel()`（`adk/cancel.go:217`）每轮创建 `cancelFn`，新请求到达时调用上轮的 `cancelFn(WithAgentCancelMode(CancelAfterChatModel))`。**驳回原因：** 这需要自研 lifecycle——`SessionState` 新增 `CancelFn` 字段（当前不存在，见 `internal/state/session.go:64`）、并发保护、清理时机。这正是本方案要消灭的同类自研胶水，上了 TurnLoop 后这段代码就要被拆掉，属于"先建后拆"的反模式。除非 TurnLoop 迁移中途阻塞且急需止血，否则不走此路。

**前端对齐:** 两条路径都会在 SSE 流上产生 `CancelError` 事件，协议要先和前端对齐。

### 共存架构决策（2026-06-26）：TurnLoop 包裹 graph，不取代

**决策：graph 和 TurnLoop 共存。TurnLoop 包裹 graph，graph 逻辑保留。**

早期分析曾考虑"TurnLoop 取代 graph"，经调研推翻——两者不冲突，是不同层面：

| 层面 | 职责 | 生命周期 |
|------|------|----------|
| TurnLoop | turn 间交互（Push/Preempt/Idle/Stop） | per-session 长生命周期 |
| graph | turn 内编排（preflight→prefill→agent→guard） | per-turn（每次 PrepareAgent 创建） |

**共存架构：**
```
TurnLoop (session 级, 长生命周期)
  ├→ PrepareAgent: 构建 GraphAgent（包装 graph）
  ├→ agent run: GraphAgent.Run → graph.Stream → 事件流
  │     ↓ ctx cancel 传导（CancelImmediate）
  │   graph (turn 级, per-turn 新实例)
  │     ├→ preflight (快, 不需 cancel 安全点)
  │     ├→ prefill (快, 不需 cancel 安全点)
  │     ├→ agent node: supervisor ReAct + agentEventBridge → sink  ← cancel 在这生效
  │     └→ guard
  └→ OnAgentEvents: 收 final output → recordTurn
```

**三个关键设计：**

1. **GraphAgent adapter**（新建 `internal/runtime/graph_agent.go`）
   - 把 `compose.Runnable[string, string]` 包装成 `adk.TypedAgent[*schema.Message]`
   - `Run(ctx, input)` → 取最后 user msg → `graph.Stream(ctx, msg)` → 把 `StreamReader[string]` 转成 `AsyncIterator[*AgentEvent]`
   - 每个 string chunk 包装成 `AgentEvent{Output: &AgentOutput{...}}`
   - ctx cancel 时停止 graph.Stream 读取

2. **cancel 用 `CancelImmediate`，不用 `CancelAfterChatModel`**
   - 原因：[cancel.go:53-58](../../../../eino-agent/eino/adk/cancel.go:53) 注释——`CancelAfterChatModel` 只对 root agent 的 ChatModel 安全点生效，GraphAgent 不是 ChatModelAgent（它调 graph.Stream），不生效
   - `CancelImmediate`（[cancel.go:46-48](../../../../eino-agent/eino/adk/cancel.go:46)）通过 ctx cancellation 传导到 graph 内部 supervisor，可靠
   - 代价：不等安全点，可能浪费当前 LLM 调用的 token——但用户切话题场景可接受

3. **agentEventBridge 留 graph 内部，OnAgentEvents 只管收尾**
   - graph 的 agentNode 继续跑 `agentEventBridge`（桥接 supervisor 中间事件到 sink）——不动
   - `OnAgentEvents` 收到 GraphAgent 的 final AgentEvent 后，只做 `recordTurnAndMaintainContext`
   - guard 留 graph 的 guardNode（不动），OnAgentEvents 不重复 guard

**纠正早期错误判断：**
- ✗ "graph 和 TurnLoop 职责重叠" → 错。graph 是 turn 内，TurnLoop 是 turn 间
- ✗ "graph.Invoke 阻塞不对齐" → 错。用 `graph.Stream`（流式）而非 Invoke
- ✗ "per-request vs per-session 不对齐" → 错。TurnLoop 每次 PrepareAgent 创建新 graph（per-turn）

**架构风险：无。**
- graph 和 TurnLoop 不同层面，不破坏 graph 内部逻辑
- GraphAgent adapter 是标准 wrapper 模式
- cancel 用 ctx 传导（Go 原生机制），可靠
- agentEventBridge 不动，向后兼容

- [ ] A2.1 拆解 TurnLoop 迁移子任务（orchestrator + HTTP handler + session store + tracing）
  - [x] A2.1a TurnLoopSessionManager 骨架（已完成，2026-06-26）
    - **文件:** `internal/orchestrator/turnloop_manager.go`（已建）、`turnloop_manager_test.go`（6 单测全过）
    - **完成内容:** TurnLoopSessionManager 管实例映射 + sink 映射 + 生命周期（GetOrCreate/Push/GetSink/ClearSink/StopAll）
    - **当前状态:** cfgFactory 用 mock（echoAgent），未接真 graph。真集成在 A2.1b
    - **纠正:** 早期方案放 state 包，已纠正为 orchestrator 包（sink 是 orchestrator 类型，state 不该感知）
  - [ ] A2.1b GraphAgent adapter + 真集成（核心）
    - **文件:**
      - 新建 `internal/runtime/graph_agent.go` — GraphAgent adapter
      - 改 `internal/orchestrator/turnloop_manager.go` — cfgFactory 接真 graph
      - 改 `internal/orchestrator/orchestrator.go` — Run → Push
      - 改 `internal/handler/chat.go` — HandleChat 调 Push
    - **改动要点:**
      - **GraphAgent adapter:** `Run(ctx, input)` → 取最后 user msg → `graph.Stream(ctx, msg)` → 把 `StreamReader[string]` 转成 `AsyncIterator[*AgentEvent]`；ctx cancel 时停止读取
      - **cfgFactory 真版本:**
        - `GenInput`: 消费 items（string messages），返回 `AgentInput{Messages: buildConversationMessages(st, msg)}`
        - `PrepareAgent`: 调 `executor.buildOrchestrationGraph()` 构建 graph，包装成 GraphAgent 返回
        - `OnAgentEvents`: 收 final output → `recordTurnAndMaintainContext`（不桥接事件，agentEventBridge 留 graph 内部）
      - **Push 改用 `WithPreempt(Immediate)`**（不是 AfterChatModel——GraphAgent 不是 ChatModelAgent，AfterChatModel 不生效）
      - **agentEventBridge 不动**——留 graph 的 agentNode 内部，继续桥接 supervisor 事件到 sink
      - **guard 不动**——留 graph 的 guardNode
      - `locker.Lock(sessionID)` 移除——TurnLoop 单线程串行化替代
    - **验证:**
      - 单测：GraphAgent adapter 的 Stream → AsyncIterator 转换
      - 集成测试：连续 Push 两条消息，第一条被 preempt（CancelImmediate 通过 ctx 传导到 graph 内部 supervisor）
    - **风险:** GraphAgent adapter 的流式转换要正确处理 ctx cancel，避免 goroutine 泄漏
  - [ ] A2.1c tracing/span 适配（请求级 → TurnLoop 事件级）
    - **文件:** `internal/tracing/`、`internal/orchestrator/orchestrator.go`
    - **改动要点:**
      - 现有 `tracer.StartTrace(ctx, "chat.turn")` 时机从 Run 入口移到 `OnAgentEvents` 首 event
      - `OnAgentEvents` 里每 event 创建/补 span
      - CancelError 事件单独打 span（标注 preempt 终止的 turn）
    - **验证:** trace 完整性——一个 turn 内所有 LLM/tool span 都挂在同一 trace 下；preempt 场景 trace 正确标注中断
  - **顺序:** A2.1a 已完成 → A2.1b 核心实施（GraphAgent adapter + 真集成）→ A2.1c 收尾。A2.1b 是最大风险点，建议分两步：(1) GraphAgent adapter 单测；(2) orchestrator+handler 集成。
- [ ] A2.2 SSE CancelError 事件格式与前端对齐
- [ ] A2.3 集成测试：快速连续消息

---

## B1: Summarization 中间件

> **状态变更（2026-06-25）：已移除 specialist summarization middleware。** 原因见下方"零值 bug 记录"。B1.1-B1.3/B1.5 的"已完成"标记撤销——代码里 `buildSpecialistHandlers` 已不再挂载 summarization。如需恢复，必须先修零值 bug 并加测试证明首轮不触发。

**零值 bug 记录（2026-06-25）：**

原配置（`agent_route.go` buildSummarizationMiddleware）只设 `Trigger.ContextMessages=20`，未设 `Trigger.ContextTokens`。Go 零值导致 `ContextTokens=0`。eino 的 `getTriggerContextTokens`（`eino-agent/eino/adk/middlewares/summarization/summarization.go:373`）在 `Trigger != nil` 时直接返回字段值（0），没做"0 表示未设→用默认 160000"的区分。结果 `shouldSummarize` 走到 `tokens > 0` 分支恒成立——**每次 specialist 调用都触发 summarizer（deepseek-v4-flash）改写输入**，污染注入的权威命盘数据（日主癸水→乙木、四柱壬申辛亥癸丑丁巳→壬子辛亥、跳过命盘总览直接流年、"好的我们继续"幻觉）。

修复：`buildSpecialistHandlers` 移除 summarization 挂载（`agent_route.go:124-130`）。会话历史压缩由 orchestrator `recordTurnAndMaintainContext` + `RunningSummary` 负责（外层 session 级，不走 ADK middleware）。

**如需恢复 summarization 的前提条件：**
1. 必须显式设 `Trigger.ContextTokens` 为高值（如 160000），不能依赖零值
2. 必须加单测：构造 1-2 条消息的输入，断言 `shouldSummarize` 返回 false
3. 必须人工验证：summarizer 不改写命盘术语（日主/四柱/用神/十神）

**原接入方式（已撤销，保留供恢复参考）：**

1. `BuildSpecialist` 的 `ChatModelAgentConfig` **新增 `Handlers` 字段**（当前没有，见 agent_route.go:58-96）。
2. 注入 summarization middleware。需要 `model.BaseModel[*schema.Message]` 实例做摘要——复用 `AgentBuilder.flashChat`，或从新增的 `LLM_SUMMARIZE_MODEL` 环境变量构造独立客户端（二选一，落地时定）。
3. 中间件顺序（ADK 规定，见 eino-agent skill）：`PatchToolCalls → Reduction → Summarization`。若 B2 同时落地，Reduction 必须在 Summarization 之前。

**与现有代码的联动（更新）：**

- **保留** `buildConversationMessages` 拼装 RecentTurns 的骨架——外层 session 级历史管理依赖它。
- **新增** `buildConversationMessages` 注入 `RunningSummary`（非空时作为 SystemMessage 放消息列表开头，`executor.go:496-525`）——补上外层压缩结果的注入缺口。
- **不恢复** `historyLimit` 语义重定义（B1.4）——外层只用消息数（MaxRecentTurns=30），不碰 token。

**验证:** 构造 15 轮对话历史，验证 `RunningSummary` 被注入 + 命理术语不被扭曲。摘要扭曲"用神"/"十神"/"日主"等术语是主要风险。

- [~] B1.1 `BuildSpecialist` 新增 `Handlers` 字段 — **撤销**（Handlers 仍存在但只挂 reduction）
- [~] B1.2 summarization model 实例来源确定 — **撤销**（summarizerModel 字段仍在 AgentBuilder/Executor，变 dead code，待清理）
- [~] B1.3 注入 summarization middleware — **撤销**（已移除挂载）
- [ ] B1.4 `historyLimit` 语义重新定义 — **暂缓**（外层不用 token，无重定义需求）
- [~] B1.5 单测通过 — **撤销**（中间件已移除，单测不适用）
- [x] B1.6（新增）`buildConversationMessages` 注入 `RunningSummary` — 已完成（`executor.go:496-525`）
- [ ] B1.7（新增）dead code 清理：`buildSummarizationMiddleware` 函数 + `summarizerModel` 字段 + container.go 初始化 — 待做（不阻塞，按需清理）

---

## B2: ToolReduction 中间件

**当前状态:** `knowledge_search` 返回古籍原文可达数百字，无截断。`agentEventBridge` 把全文推给 SSE 和 LLM。

**接入方式:**

1. `BuildSpecialist` 的 `Handlers` 注入 ToolReduction middleware（与 B1 同一个 `Handlers` 切片）。
2. 策略先用 Truncate（2000 字符），观察古籍关键信息保留度。
3. 中间件顺序：在 Summarization 之前。

**与 `agentEventBridge` 的关系:** ToolReduction 在工具结果进入 LLM 之前截断。`bridge.go:302` 的 `emitChartFromToolResult` 仍发完整 chart payload——截断只影响 LLM 上下文，不影响前端组件。这一点要在单测里确认。

- [x] B2.1 注入 ToolReduction middleware
- [x] B2.2 单测截断验证（超 2000 rune → 截断 + 不落盘）
- [ ] B2.3 人工古籍保留度 — **待做**（需真实 knowledge_search 结果）
- [x] B2.4 确认 chart payload 不被截断 — 排盘工具（bazi_calc/qimen_dunjia/ziwei_calc）走 executor `prefill` 确定性预执行，不经过 agent 工具调用链，reduction 中间件不触及；单测验证非文本 part（image）原样保留

---

## C1: Dynamic Interrupt + Checkpoint（排盘 Graph 化后评估）

**场景:** 排盘 Graph 执行到某阶段后暂停，等用户确认信息后再继续。

**前置条件:** CheckpointStore 实现 + 排盘 Graph 上线。API 已验证（本地 fork + human-in-the-loop 参考）：工具内 `compose.StatefulInterrupt(ctx, info, state)` 暂停 → `runner.ResumeWithParams(ctx, checkpointID, &adk.ResumeParams{Targets: ...})` 恢复 → Runner 配 `WithCheckPointStore`。

---

## 框架原生能力盘点

审核中顺带评估的其他自研代码下放机会：

| 自研代码 | 位置 | 能否下放 | 说明 |
|---------|------|---------|------|
| `buildConversationMessages` 历史截断 | executor.go:433 | **部分** | B1 落地后硬截断语义弱化，但输入拼装骨架保留 |
| `SetHistoryLimit` | executor.go:49 | **部分** | 语义从"截断"变"窗口上限"，函数保留 |
| `noFStringGenModelInput` | agent_route.go:47 | **保留** | 绕开 Eino FString 模板把 `{key}` 当变量的 workaround，instruction 含字面花括号（如 JSON 数据块）时必需。非业务逻辑，不删 |
| `agentEventBridge` | bridge.go:26 | **不下放** | 295 行里大部分是命理业务规则：`<analysis>/<response>` XML 拆分、specialist 去重、supervisor 后置文本丢弃、chart 派发、forwarded streaming 归类。ADK 的 `AgentEvent` 是通用协议，不知道这些契约。仅 `<analysis>/<response>` 拆分可考虑用 ADK 结构化输出或 middleware 替代，属独立改造，不在本方案范围 |
| `prefill*` 排盘预执行 | executor.go:198-326 | **独立方案** | 11-graph-dispatch.md 已规划 Graph 化，不在本方案 |
| `ModelRetryConfig` | agent_route.go:522 | **已用框架** | 维持现状，failover 不做 |

---

## 不变项

| 边界 | 保持不变 |
|------|---------|
| RouteEngine / ApprovedRoute | 不在本方案范围 |
| dispatch 路由 (AgentAsTool) | 不动 |
| specialist 内部 (ReAct L4) | 不动 |
| `<analysis>/<response>` 契约 | 不动（未来或用结构化输出替代，独立改造） |
| `noFStringGenModelInput` | 保留 |
| `agentEventBridge` 主体 | 保留（业务规则） |
| SSE 协议 / 前端 | 除 A2 CancelError 外不变 |

---

## 风险汇总

| 风险 | 等级 | 缓解 |
|------|------|------|
| A2 TurnLoop 迁移改动面大 | **高** | 分阶段推进：先 session store + TurnLoop 骨架，再 HTTP handler 接入，最后 tracing 适配 |
| A2 CancelError 前端不兼容 | **中** | 先对齐事件格式 |
| B1 摘要扭曲命理术语 | **中** | 人工验证 + 可关闭 |
| B2 截断丢失古籍关键信息 | **低** | 先 Truncate 观察 |
| B1+B2 中间件顺序错配 | **中** | 按 ADK 规定：PatchToolCalls → Reduction → Summarization |
| eino 本地 fork 滞后上游 | **低** | 版本可升级；重评 `replace` 指令是否仍需 |
