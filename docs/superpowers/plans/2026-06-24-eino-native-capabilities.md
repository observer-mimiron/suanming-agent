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

### 为何不选 cancelFn

考虑过的替代方案：`adk.WithCancel()`（`adk/cancel.go:217`）每轮创建 `cancelFn`，新请求到达时调用上轮的 `cancelFn(WithAgentCancelMode(CancelAfterChatModel))`。**驳回原因：** 这需要自研 lifecycle——`SessionState` 新增 `CancelFn` 字段（当前不存在，见 `internal/state/session.go:64`）、并发保护、清理时机。这正是本方案要消灭的同类自研胶水，上了 TurnLoop 后这段代码就要被拆掉，属于"先建后拆"的反模式。除非 TurnLoop 迁移中途阻塞且急需止血，否则不走此路。

**前端对齐:** 两条路径都会在 SSE 流上产生 `CancelError` 事件，协议要先和前端对齐。

- [ ] A2.1 拆解 TurnLoop 迁移子任务（orchestrator + HTTP handler + session store + tracing）
- [ ] A2.2 SSE CancelError 事件格式与前端对齐
- [ ] A2.3 集成测试：快速连续消息

---

## B1: Summarization 中间件

**当前状态:** `buildConversationMessages`（executor.go:433）手动截断 `RecentTurns` 到 `historyLimit` 条。15+ 轮后即使截断，单轮内容仍可能很长，无压缩。

**接入方式:**

1. `BuildSpecialist` 的 `ChatModelAgentConfig` **新增 `Handlers` 字段**（当前没有，见 agent_route.go:58-96）。
2. 注入 summarization middleware。需要 `model.BaseModel[*schema.Message]` 实例做摘要——复用 `AgentBuilder.flashChat`，或从新增的 `LLM_SUMMARIZE_MODEL` 环境变量构造独立客户端（二选一，落地时定）。
3. 中间件顺序（ADK 规定，见 eino-agent skill）：`PatchToolCalls → Reduction → Summarization`。若 B2 同时落地，Reduction 必须在 Summarization 之前。

**与现有代码的联动:**

B1 落地后，`buildConversationMessages` + `SetHistoryLimit` 的职责被 middleware 接管一部分：

- **保留** `buildConversationMessages` 拼装 RecentTurns 的骨架——middleware 在 agent 内部压缩，输入侧仍需提供历史消息。
- **重定义** `historyLimit` 语义——从"硬截断条数"变成"喂给 summarizer 的窗口上限"。
- **不删** middleware 不替代输入拼装，只替代"超长时的压缩策略"。

**验证:** 构造 15 轮对话历史，验证压缩后 message 数量 + 人工命理术语质量。摘要扭曲"用神"/"十神"/"日主"等术语是主要风险。

- [x] B1.1 `BuildSpecialist` 新增 `Handlers` 字段
- [x] B1.2 summarization model 实例来源确定 — 复用 flash 配置新建 `summarizerModel`（ToolCallingChatModel），经 container → executor → AgentBuilder 注入
- [x] B1.3 注入 summarization middleware（顺序：Reduction 在前）
- [ ] B1.4 `historyLimit` 语义重新定义 — **暂缓**：中间件在 agent ReAct 循环内部压缩，与输入侧 `historyLimit` 正交；语义重定义属调优决策，观察中间件线上行为后再做
- [x] B1.5 单测通过（构造 + nil-model 分支）；**人工术语验证待做**（需真实模型调用）

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
