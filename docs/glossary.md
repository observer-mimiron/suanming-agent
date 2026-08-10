# 名词对照表 (Glossary)

> 本文档定义 suanming-agent 项目中核心概念的中英文名称、行为定义、代码位置及与其它概念的关系。
> 面向学习用途与团队语义统一，按架构层级组织。
>
> 最后更新：2026-07-07

---

## 一、架构层

### Orchestrator · 编排器

**是什么**：会话生命周期外壳。不执行业务逻辑，只做取锁、加载状态、启动 Trace、驱动全流程、收尾持久化。

**定义位置**：[internal/orchestrator/orchestrator.go](../internal/orchestrator/orchestrator.go)

**关联**：持有 `supervisor`（路由审批）、`runtime.Executor`（路由执行）、`state.Store/Locker`（会话持久化）。调用链路：`Orchestrator.Run()` → `supervisor.Approve()` → `runtime.Execute()` → SSE 推送。

**细分职责**：
- 取会话锁 → 加载 `SessionState` → 启动 `TurnTrace`
- 调 supervisor 做路由审批
- 将 `ApprovedRoute` 交给 `runtime.Executor` 执行
- 维护上下文窗口（`RecentTurns` / `RunningSummary`）
- 推送 SSE trace panel 事件后持久化

---

### Supervisor · RouteAdvisor · 路由顾问 / 监督器

**是什么**：路由决策层。接收用户消息和会话状态，产出 `SupervisorDecision`（结构化路由），经三层降级保护始终可工作。

**定义位置**：[internal/supervisor/](../internal/supervisor/)（RouteAdvisor 实现）
接口在 [internal/orchestrator/orchestrator.go](../internal/orchestrator/orchestrator.go) 中定义为 `RouteAdvisor` 接口。

**关联**：被 `Orchestrator` 调用；产出 `SupervisorDecision` 后经 `Policy Gate` 加工为 `ApprovedRoute`。

**三层降级**：ADK structured route → textDecide (LLM 文本) → fallbackExtract (规则) → safeFallback (硬编码兜底)

**注意**：项目中 "Supervisor" 一词容易混淆。当前权威口径里，路由层统一称 `RouteAdvisor`；执行层不再把 `Supervisor Agent` 视为默认主链组件。

---

### RouteEngine · 路由引擎

**是什么**：ADK 路由引擎。当前固定使用 Eino ADK 实现（`ChatModelAgent` 承载 layer-1 structured route），Go 侧保留 textDecide → fallbackExtract → safeFallback 作为外层降级。

**定义位置**：[internal/supervisor/adk_engine.go](../internal/supervisor/adk_engine.go)

---

### Policy Gate · 策略门控

**是什么**：Go 侧确定性校验层。对 `SupervisorDecision` 做领域白名单过滤、置信度强制澄清、并行硬禁用、资料完整性校验、显式术数 obey，产出 `ApprovedRoute`。

**定义位置**：[internal/policy/gate.go](../internal/policy/gate.go)

**核心规则**：
- 领域白名单：仅允许 bazi / qimen / ziwei
- 并行硬禁用：阶段一 `ParallelAllowed` 恒为 false
- 低置信度强制澄清（阈值 0.6）
- 显式术数 obey：用户说了「用紫微/奇门/八字」→ 强制执行

---

### ApprovedRoute · 已批准路由

**是什么**：经 Policy Gate 批准后的执行路线，是 runtime 的主控输入。含 L0 意图、L1 领域、L2 任务、L3 槽位。

**定义位置**：[internal/policy/gate.go](../internal/policy/gate.go)

**关键字段**：`ConversationIntent` / `PrimaryDomain` / `SecondaryDomains` / `TaskIntent` / `Slots` / `PolicyHints` / `NeedsClarification` / `Confidence` / `ParallelAllowed`

**关联**：输入自 `SupervisorDecision`，输出给 `runtime.Executor`。

---

### Executor · 运行时执行器

**是什么**：负责执行已批准路由的运行时引擎。当前默认主链是 `Manager -> ExecutionPlan -> Prefill -> specialist runner(s) -> final guard`。

**定义位置**：[internal/runtime/executor.go](../internal/runtime/executor.go)

**执行流程**：`Execute()` → `BuildExecutionPlan()` → preflight / short-circuit → prefill → `runExecutionPlan()` → final guard

---

### Preflight · 预检（确定性短路层）

**是什么**：在进入 LLM Agent 之前做确定性硬判断。可能的短路：缺少资料时返回澄清提问，不发 Agent。

**定义位置**：[internal/runtime/preflight.go](../internal/runtime/preflight.go)

**关联**：被 `Executor` 调用，产出可能是短路文本（澄清/缺资料）或放行信号。

---

### Prefill · 确定性排盘链

**是什么**：Go 代码直接执行排盘/用神/大运等工具链，结果注入 `SessionState` 和 Agent SessionValues。LLM Agent 不接触排盘工具。

**定义位置**：[internal/runtime/executor.go](../internal/runtime/executor.go)

**核心原则**：排盘/用神/大运全部由 Go 确定性执行，不挂载到 Specialist Agent 的工具列表。

---

## 二、路由模型层

### SupervisorDecision · 监督器路由决策

**是什么**：LLM Supervisor 的结构化路由原始输出，含四层决策。

**定义位置**：[internal/schemas/supervisor_decision.go](../internal/schemas/supervisor_decision.go)

**四层模型**：

| 层 | 字段 | 含义 |
|---|---|---|
| L0 | `ConversationIntent` | 对话意图：consult / clarify / smalltalk / meta_help / switch_topic |
| L1 | `PrimaryDomain` + `SecondaryDomains` | 主/辅领域：bazi / qimen / ziwei |
| L2 | `TaskIntent` | 任务意图：collect_profile / interpret_chart / fortune_followup / timing_followup / cross_domain_consult |
| L3 | `Slots` + `PolicyHints` | 槽位（出生资料、咨询主题、时间范围）与策略提示（奇门模式、资料要求） |

---

### ConversationIntent · 对话意图 (L0)

**是什么**：本轮对话的宏观目的。取值：`consult`（咨询）、`clarify`（澄清）、`smalltalk`（闲聊）、`meta_help`（关于系统本身的帮助）、`switch_topic`（切换话题）。

**定义位置**：[internal/schemas/supervisor_decision.go](../internal/schemas/supervisor_decision.go)

---

### PrimaryDomain / SecondaryDomains · 主/辅领域 (L1)

**是什么**：本轮最主要的命理领域（bazi / qimen / ziwei）及辅助领域。

**领域选择策略**：
- **bazi**：本命结构 / 长期趋势
- **ziwei**：宫位导向的人生主题（婚姻结构、子女、人生角色分布）
- **qimen**：当前时机 / 行动选择 / 短期运势

---

### TaskIntent · 任务意图 (L2)

**是什么**：本领域内的具体任务。当前八字任务：`collect_profile`、`amend_profile`、`direct_bazi`、`interpret_chart`、`fortune_followup`、`timing_followup`、`cross_domain_consult`。

**定义位置**：[internal/schemas/supervisor_decision.go](../internal/schemas/supervisor_decision.go)

---

### DecisionSlots · 决策槽位 (L3)

**是什么**：从用户消息中提取的结构化槽位值。

**定义位置**：[internal/schemas/supervisor_decision.go](../internal/schemas/supervisor_decision.go)

**字段**：`Profile`（出生资料 map）、`QuestionText`（咨询问题原文）、`TimeScope`（时间范围）、`TargetSubject`（咨询主题，如婚姻/事业）、`Language`（语言）

---

### PolicyHints · 策略提示

**是什么**：通知 Policy Gate 的可选行为标志，不定义在 L0-L3 框架内但参与路由控制。

**定义位置**：[internal/schemas/supervisor_decision.go](../internal/schemas/supervisor_decision.go)

**关键字段**：`NeedsKnowledge`、`NeedsQimen`、`QimenMode`（none / primary / supplement）、`ProfileRequirement`（none / partial / full）、`CanReuseSessionProfile`、`CanReuseCachedResult`

---

### NormalizeApprovedRoute · 路由标准化（确定性纠偏）

**是什么**：Policy 层面的确定性修正逻辑。检测用户是否显式指定术数方法（八字/紫微/奇门），强制纠偏主领域；检测 collect_profile 已满足时自动升级为 amend_profile 或 fortune_followup。

**定义位置**：[internal/supervisor/approved_route.go](../internal/supervisor/approved_route.go)

---

### Contract Gate · 执行契约验收（后置校验）

**是什么**：Agent 运行完成后对输出做 contract 校验。若 PrimaryDomain=qimen 但没有 QimenResult，或 PrimaryDomain=ziwei 但没有 ZiWeiResult → 阻止输出最终结论。

**定义位置**：[internal/runtime/executor.go](../internal/runtime/executor.go) 的 `guardFinalAnswerWithPlan` 调用点

**关联**：在 `agentEventBridge` 后调用，是最终回答的最后一道门。

---

## 三、Agent 层

### Specialist Agent · 领域专家 Agent

**是什么**：各命理领域的专家 LLM Agent。拥有领域知识、检索工具，负责单领域分析流程。当前有三个：`bazi_specialist`、`qimen_specialist`、`ziwei_specialist`。

**定义位置**：
- [internal/specialists/bazi/specialist.go](../internal/specialists/bazi/specialist.go)
- [internal/specialists/qimen/specialist.go](../internal/specialists/qimen/specialist.go)
- [internal/specialists/ziwei/specialist.go](../internal/specialists/ziwei/specialist.go)

**工具限制**：Specialist 只挂载 `knowledge_catalog` 和 `knowledge_search`。排盘/用神/大运由 prefill 确定性执行。

**Instruction 来源**：基础指令从 [prompts/interpret.md](../prompts/interpret.md) 加载，运行时由 `AgentBuilder.BuildSpecialist()` 注入出生资料、命盘数据块、当前日期等上下文。

---

### specialist runner · 领域执行器

**是什么**：runtime 调用领域 Agent 的有界执行接口。由 Go 侧按 `ExecutionPlan.Domains` 决定触发哪些 runner，而不是把调度权交给额外的执行 supervisor。

**定义位置**：[internal/specialists/runner.go](../internal/specialists/runner.go)、[internal/runtime/specialist_runner.go](../internal/runtime/specialist_runner.go)

**当前实现**：`ADKSpecialistRunner`（ADK 专家执行器）直接运行单个 Specialist Agent，并把结果回收成 `specialists.Result`（领域结构化结果）。

---

### AgentBuilder · Agent 构建器

**是什么**：运行时动态构建 Specialist Agent 的工具。负责把基础指令、会话里的命盘/资料上下文，以及知识检索工具装配成可运行的领域 Agent。

**定义位置**：[internal/runtime/agent_route.go](../internal/runtime/agent_route.go)

**核心方法**：`BuildSpecialist()`（构建单个 Specialist Agent）

---

### agentEventBridge · Agent 事件桥接器

**是什么**：消费 ADK 的异步事件流，桥接到 SSE。负责区分领域正文与普通工具结果、解析 XML 标签（analysis/response）、检测排盘结果自动推送命盘卡牌。

**定义位置**：[internal/runtime/bridge.go](../internal/runtime/bridge.go)

**核心逻辑**：
- `isSpecialistTool()`：通过 `_specialist` 后缀区分领域专家节点和普通 Tool
- `specialistDone` 标记：防止领域正文被重复加入
- `parseXMLSections()`：解析 `<analysis>` 和 `<response>` XML 段

---

### Specialist Config · 领域专家配置

**是什么**：领域专家的静态元数据，包含领域名、Agent 名、描述、基础指令、工具列表。由各 specialist 包注册到 Registry。

**定义位置**：[internal/specialists/types.go](../internal/specialists/types.go)

---

### Specialist Registry · 领域专家注册表

**是什么**：按注册顺序保存所有领域专家 Config，供 AgentBuilder 构建 AgentTool 列表。

**定义位置**：[internal/specialists/types.go](../internal/specialists/types.go)

---

### DomainResult · 领域结果

**是什么**：领域专家返回的结构化契约，包含分析摘要、结构化数据、证据引用、后续追问。

**定义位置**：[internal/schemas/domain_result.go](../internal/schemas/domain_result.go)

---

## 四、会话状态层

### SessionState · 会话状态

**是什么**：单个会话的完整持久化状态。Go runtime 持有所有权，LLM 只能建议更新，不能直接写。

**定义位置**：[internal/state/session.go](../internal/state/session.go)

**关键字段**：

| 字段 | 含义 |
|---|---|
| `SessionID` | 会话标识 |
| `Profile` | 出生资料 {year, month, day, hour, gender, ...} |
| `BaziResult` | bazi_calc 排盘结果 |
| `QimenResult` | qimen_dunjia 排盘结果 |
| `ZiWeiResult` | ziwei_calc 排盘结果 |
| `Routing` | 上一轮路由快照 |
| `RecentTurns` | 最近 N 轮对话（MaxRecentTurns=30） |
| `RunningSummary` | 溢出窗口的滚动摘要 |
| `Guidance` | conversation guidance 状态 |
| `Subject` | 当前命盘归属（自己/孩子/父亲等） |

---

### RoutingSnapshot · 路由快照

**是什么**：写入 SessionState 的上一次路由决策摘要，供下一轮路由参考。

**定义位置**：[internal/state/session.go](../internal/state/session.go)

**字段**：`ConversationIntent`、`PrimaryDomain`、`SecondaryDomains`、`TaskIntent`、`QimenMode`、`AwaitingClarification`、`Confidence`、`TimeScope`、`TargetSubject`

---

### DomainStates · 领域状态

**是什么**：聚合各领域独立状态（BaziState / QimenState / ZiWeiState）。每个领域拥有自己的结果缓存和复用规则。

**定义位置**：[internal/state/session.go](../internal/state/session.go)

---

### RecentTurns · 最近对话

**是什么**：会话内最近多轮对话的保留窗口（最大 30 条 Turn）。超出的内容滚动合并到 RunningSummary。

**定义位置**：[internal/state/session.go](../internal/state/session.go)

---

### RunningSummary · 滚动摘要

**是什么**：上下文窗口溢出后的增量摘要。超出 RecentTurns 的历史对话通过摘要合并，失败不丢历史。

**定义位置**：[internal/state/session.go](../internal/state/session.go)

---

### GuidanceState · 对话引导状态

**是什么**：记录当前「引导式对话」的进度（引导到哪一步 + 少量复用信息）。不负责决定如何迁移，只保存状态。

**定义位置**：[internal/state/session.go](../internal/state/session.go)

**关联**：与 [internal/runtime/guidance_gate.go](../internal/runtime/guidance_gate.go) 配合使用，后者通过 hard gate 判断本轮是否允许进入/继续 guidance。

---

### Turn · 对话轮次

**是什么**：一轮对话中的一条消息（用户或助手），包含 role、content、timestamp。

**定义位置**：[internal/state/session.go](../internal/state/session.go)

---

## 五、工具与知识层

### Tool Registry · 工具注册表

**是什么**：统一注册所有命理工具，通过 `adapter.go` 适配为 Eino BaseTool。

**定义位置**：[internal/tools/](../internal/tools/)；适配器在 [internal/runtime/adapter.go](../internal/runtime/adapter.go)

---

### bazi_calc · 八字排盘工具

**是什么**：确定性工具，排四柱命盘（含严格子正换日、太阳时校正、神煞）。由 prefill 阶段执行，不挂到 Specialist Agent。

**关联**：底层引擎是 [lunar-go](https://github.com/nicholasciang/lunar-go) 的项目远程 fork，用于固化严格 `子正换日` 下“日柱与时柱同口径换日”的八字规则。

---

### yongshen · 用神分析工具

**是什么**：确定性工具，分析日主强弱、用神忌神。由 prefill 阶段执行。

---

### dayun_analyzer · 大运分析工具

**是什么**：确定性工具，大运走势分析。由 prefill 阶段执行。

---

### qimen_dunjia · 奇门遁甲排盘工具

**是什么**：确定性工具，排奇门遁甲盘。由 prefill 阶段执行。

---

### ziwei_calc · 紫微斗数排盘工具 / ziwei_liunian · 紫微流年分析

**是什么**：确定性工具，排紫微命盘 / 流年分析。由 prefill 阶段执行。

---

### knowledge_catalog · 知识目录

**是什么**：获取知识库图结构的工具。通过 `/api/wiki/graph` 获取按 slug 前缀过滤后的目录摘要（古籍名称、章节数、前 5 章节标题），供 Agent 规划检索策略。

**挂载于**：Specialist Agent 的工具列表。

---

### knowledge_search · 知识检索

**是什么**：MCP 连接 Yopedia 知识库检索古籍原文。返回 passages 数组（content + source）。Go adapter 层硬控每轮最多 3 次调用。

**挂载于**：Specialist Agent 的工具列表。降级路径：MCP 不可用 → REST API `/api/wiki/search` fallback。

---

### MCP · Model Context Protocol

**是什么**：连接本地 Yopedia 知识库的传输协议。Go 后端通过 MCP HTTP client 调知识库服务（:3100）。

**定义位置**：[internal/mcp/client.go](../internal/mcp/client.go)

---

### RAG · Retrieval-Augmented Generation · 检索增强生成

**是什么**：检索 + 生成的混合范式。本项目中 Specialist Agent 先通过 knowledge_search 检索古籍原文，再将 passages 注入 LLM 回答。

**项目定位**：采用 Agentic RAG（证据规划 + 条件反思），而非纯 query-based 检索。Go 强制检索次数上限，质量不足时才触发反射。

---

### Yopedia · 知识库引擎

**是什么**：项目使用的本地 Wiki 知识库。suanming-agent 的独立实例运行在端口 3100（与 lisense 知识库 3000 端口隔离）。已导入 19 个页面，涵盖古籍原文、八字基础、格局用神等模块。

---

## 六、可观测性层

### TurnTrace · 轮次追踪（原始包络）

**是什么**：一次对话轮次的完整链路追踪记录。本地 `logs/traces/` 的唯一落盘 envelope，也是 RunInspection 的事实来源。

**定义位置**：[internal/tracing/turn_trace.go](../internal/tracing/turn_trace.go)

**组成**：`TraceID` + `SessionID` + `TurnType` + `UserMessage` + `Spans[]TraceSpan`（子 span 列表）

---

### TraceSpan · 追踪子段

**是什么**：追踪中的一个工作单元。有 5 种 Kind：AGENT / CHAIN / TOOL / RETRIEVER / LLM。包含 SpanID、ParentSpanID、Name、Status、DurationMs、InputPreview、OutputPreview、Error、Attributes。

**定义位置**：[internal/tracing/turn_trace.go](../internal/tracing/turn_trace.go)

---

### RunInspection · 单轮运行诊断

**是什么**：TurnTrace 面向聊天页的白名单排障投影。它承载 trace_id、runtime 摘要、确定性诊断、span tree 和 span detail，帮助定位路由、资料、检索、领域 Agent、合同护栏或 SSE 传输问题。

**定义位置**：[internal/tracing/run_inspection.go](../internal/tracing/run_inspection.go)

---

### Eino Callback · Eino 回调钩子

**是什么**：Eino 框架的 callback 机制，自动记录 ChatModel、Tool、Retriever 三类低层事件 span 到 TurnTrace 和 OTel。

**定义位置**：[internal/tracing/eino_callback.go](../internal/tracing/eino_callback.go)

---

### OTel · OpenTelemetry · 标准观测层

**是什么**：OpenTelemetry GenAI Semantic Conventions 兼容的 span/attribute 映射，作为对外标准面。通过可选 OTLP exporter 镜像到外部 backend（默认关闭）。

**定义位置**：[internal/tracing/otel_bridge.go](../internal/tracing/otel_bridge.go) + [internal/tracing/otel_export.go](../internal/tracing/otel_export.go)

**环境变量**：`OTEL_ENABLED`、`OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`、`OTEL_EXPORTER_OTLP_HEADERS` 等。

---

### Langfuse

**是什么**：首个推荐的 AI-native observability backend。支持接收 OTLP，可先当纯观测后端，后续承接 prompt、score、dataset 和在线 eval。

**与项目关系**：不强制使用，不替换本地 TurnTrace。当 `OTEL_ENABLED=true` 时可配置 endpoint 指向 Langfuse。

---

## 七、前端事件层

### SSE · Server-Sent Events · 服务端推送事件

**是什么**：Go 后端 → Vue 前端的实时推送协议。6 种结构化事件，前端按类型渲染为不同 UI 组件。

**定义位置**：[internal/sse/writer.go](../internal/sse/writer.go)

---

### SSE 事件类型

| 类型 | 含义 | 前端渲染 |
|------|------|----------|
| `thinking` | 内部思考过程（supervisor 推理 + analysis 段） | ThinkingSegment |
| `tool_call` | 工具调用事件与结果 | ToolCallSegment |
| `component` | 结构化组件（命盘卡片 / 过程面板 / 执行链路） | 对应 component 渲染器 |
| `text` | 流式回答文本 | 正文内容 |
| `error` | 错误信息 | 错误提示 |
| `done` | 本轮结束 | 停止 loading |

---

### EventSink · 事件推送接口

**是什么**：SSE 事件输出的抽象接口。Orchestrator 和 Runtime 通过它对前端推送事件。

**定义位置**：[internal/orchestrator/orchestrator.go](../internal/orchestrator/orchestrator.go)

---

### XML 标签格式 (analysis / response)

**是什么**：LLM 输出的结构化 XML 标签。`<analysis>` 段为内部推理（走 thinking 事件），`<response>` 段为最终回答（走 text 事件）。

**解析位置**：[internal/runtime/bridge.go](../internal/runtime/bridge.go)

---

### RunInspector · 单轮执行诊断台

**是什么**：前端主视图的排障面板。由 RunInspection + 前端 transport inspection 驱动，展示首要诊断、agent 链路概览、span tree、span detail 和复制诊断 JSON。本地 debug 模式下可按 trace_id 懒加载 Raw Trace，查看完整 TurnTrace。

---

### Raw Trace · 全量追踪 JSON

**是什么**：RunInspector 内的懒加载调试视图。它通过 `GET /api/debug/traces/:trace_id` 读取 `logs/traces/` 中持久化的完整 TurnTrace；默认折叠用户原文、prompt preview、模型完整输出等敏感字段。

---

## 八、框架与基础设施

### Eino · Go LLM 应用框架

**是什么**：CloudWeGo 开源的 Go 语言 LLM 应用框架，提供组件（ChatModel/Tool/Embedding/Retriever）、编排（Graph/Chain/Workflow）和 Agent 开发套件（ADK）。

**在项目中的角色**：当前承担 ChatModel 底座（Phase 1）、route engine（Phase 3）、callback tracing（Phase 5A/5B）。Graph 编排推迟到后续阶段。

---

### ADK · Agent Development Kit

**是什么**：Eino 框架的 Agent 开发工具包。提供 ChatModelAgent（ReAct Agent 实现）、AsyncIterator 等运行时能力。

**在项目中的角色**：RouteAdvisor 用 ChatModelAgent 做结构化路由，specialist runner 用 ADK 直接运行单个 Specialist Agent。

---

### ChatModelAgent

**是什么**：Eino ADK 的 Agent 实现，支持 tool calling 循环。项目中的 RouteAdvisor 路由节点和 Specialist Agent 都会使用它。

---

### BaseTool · Eino 工具接口

**是什么**：Eino 框架的标准工具接口。Go 工具通过 [internal/runtime/adapter.go](../internal/runtime/adapter.go) 适配为 BaseTool。

---

### ToolCallingChatModel · Eino 工具调用模型

**是什么**：支持 tool calling 的 ChatModel 接口。当前主要用于 RouteAdvisor 的结构化路由和 Specialist Agent 的领域执行。

---

### lunar-go

**是什么**：开源八字排盘库。提供公历→农历转换、天干地支、节气排盘等功能。项目八字引擎的核心依赖；当前项目使用远程 fork 固化严格 `子正换日` 规则。

**选择原因**：开源成熟方案，不自研。

---

### Vue 3 + TypeScript · 前端技术栈

**是什么**：前端使用 Vue 3 + TypeScript + SSE，开发端口 5173，生产构建端口 8080（由 Go Gin 托管静态文件）。

---

### Gin · Go HTTP 框架

**是什么**：Go HTTP 框架，承载 `/api/chat` SSE 端点和服务端口 8080。

---

## 九、演进概念（规划中 / 部分实施）

### Graph 编排 · Eino compose.Graph

**是什么**：用 Eino 的 compose.Graph 承载显式 Go 侧控制流。当前已经用于 orchestration 主骨架（preflight / prefill / agent / guard），后续再决定是否继续细化更多节点。

**当前拓扑**：`[Preflight] → [Prefill] → [Agent/ExecutionPlan Dispatch] → [Final Guard]`

---

### Agentic RAG · 智能检索增强生成

**是什么**：在 controlled retrieval（Go 强制预算和来源边界）基础上，增加 evidence planning（模型表达缺失证据）、retrieval quality check（证据质量评估）、conditional reflection（仅在证据弱/冲突时触发多步反思）。

**当前状态**：设计已完成，待实施。

---

### Guidance Entry · 引导式入口

**是什么**：conversation guidance 的状态机。当用户没有明确的咨询目标时，引导式对话帮助用户选择咨询方向。

**定义位置**：`ShouldEnterGuidance()` 在 [internal/runtime/guidance_gate.go](../internal/runtime/guidance_gate.go)；`GuidanceState` 在 [internal/state/session.go](../internal/state/session.go)

---

### HasTimingFocus / ContainsTimingKeyword

**是什么**：语义分离的两个判断：`HasTimingFocus` 是 scope+intent 双条件判断（供 guidance_gate 用），`ContainsTimingKeyword` 是宽松关键词匹配（供 supervisor 用）。两者不可混用。

**定义位置**：[internal/intent](../internal/intent/) 包。

---

### SpecialistDone / isSpecialistTool

**是什么**：agentEventBridge 中的两个适配标记。
- `specialistDone`：标记至少一个 Specialist 已返回，后续 supervisor 的总结文本需丢弃
- `isSpecialistTool()`：通过 `_specialist` 后缀区分领域专家节点和普通 Tool

两者本质上是 ADK 事件桥接层的去重与归类补丁，后续如果事件模型继续收口，仍可继续简化。

---

## 附录：概念关系速查图

\`\`\`mermaid
flowchart TD
    U["用户消息"] --> OR["Orchestrator<br/>生命周期壳"]
    OR --> RA["RouteAdvisor<br/>路由决策<br/>三层降级"]
    RA --> SD["SupervisorDecision<br/>结构化路由"]
    SD --> PG["Policy Gate<br/>确定性校验"]
    PG --> AR["ApprovedRoute<br/>已批准路由"]
    AR --> EX["Executor<br/>运行时执行"]

    EX --> PF1["Preflight<br/>短路检查"]
    PF1 -->|短路| CL["澄清/缺资料"]
    PF1 -->|放行| PF2["Prefill<br/>确定性排盘链"]
    PF2 --> DP["ExecutionPlan Dispatch<br/>manager-owned"]
    DP --> SP["Specialist runner(s)<br/>领域执行 xN"]
    SP --> TK["knowledge_catalog<br/>knowledge_search"]
    SP --> BR["agentEventBridge / specialistEventBridge"]
    BR --> CG["Contract Gate<br/>后置校验"]
    CG --> SSE["SSE → 前端"]

    EX --> TT["TurnTrace<br/>raw envelope"]
    TT --> RI["RunInspection"]
    TT --> OT["OTel 标准层"]
\`\`\`

---

## 附录：术语中英对照速查表

| 中文 | English |
|------|---------|
| 编排器 | Orchestrator |
| 路由顾问 / 监督器 | RouteAdvisor / Supervisor |
| 路由引擎 | RouteEngine |
| 策略门控 | Policy Gate |
| 已批准路由 | ApprovedRoute |
| 监督器决策 | SupervisorDecision |
| 对话意图 | ConversationIntent (L0) |
| 主领域 | PrimaryDomain (L1) |
| 任务意图 | TaskIntent (L2) |
| 决策槽位 | DecisionSlots (L3) |
| 策略提示 | PolicyHints |
| 预检 | Preflight |
| 预填充（确定性排盘） | Prefill |
| 执行器 | Executor |
| 领域专家 Agent | Specialist Agent |
| 领域执行器 | specialist runner |
| Agent 构建器 | AgentBuilder |
| 事件桥接器 | agentEventBridge |
| 会话状态 | SessionState |
| 路由快照 | RoutingSnapshot |
| 领域状态 | DomainStates |
| 最近对话 | RecentTurns |
| 滚动摘要 | RunningSummary |
| 契约门 / 后置校验 | Contract Gate |
| 路由标准化 | NormalizeApprovedRoute |
| 轮次追踪 | TurnTrace |
| 追踪子段 | TraceSpan |
| 单轮运行诊断 | RunInspection |
| 全量追踪 JSON | Raw Trace |
| 服务端推送事件 | SSE |
| 事件推送 | EventSink |
| 单轮执行诊断台 | RunInspector |
| Eino 回调 | Eino Callback |
| 智能检索增强生成 | Agentic RAG |
| 检索增强生成 | RAG |
| 知识目录 | knowledge_catalog |
| 知识检索 | knowledge_search |
| 模型上下文协议 | MCP |
| 引导状态 | GuidanceState |
| 引导式入口 | Guidance Entry |
| 三层降级 | Three-tier Fallback |
| 八字排盘 | bazi_calc |
| 用神分析 | yongshen |
| 大运分析 | dayun_analyzer |
| 奇门遁甲 | qimen_dunjia |
| 紫微斗数 | ziwei_calc |
| 紫微流年 | ziwei_liunian |
