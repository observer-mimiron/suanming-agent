# 项目进度

> 将此文件全文作为上下文输入给 Claude Code，即可独立继续开发。

---

## 当前状态

**阶段：** v1.5 Supervisor Phase 1.5 收口 + Eino Phase 1-3 完成中 + Phase 5A 启动  
**分支：** codex/eino-phase1-2  
**最后更新：** 2026-06-13  
**状态：** 已完成 Eino 前两步接入：`llm.Chat` 现支持 classic Eino backend，工具注册表已可导出 Eino `InvokableTool` 视图，现有 supervisor / orchestrator / SSE 协议保持不变。已补上一个真实环境阻塞修复：supervisor 的 structured route 改用独立 `flash + DisableThinking` client，避免 DeepSeek 在 `tool_choice` 下返回 400；同时 orchestrator 增加“出生信息消息被误路由时”的确定性纠偏与出生地提取兜底。Phase 3 已开始落最小版本：`internal/supervisor` 新增可插拔 `RouteEngine`，Eino/ADK 仅承载 layer-1 structured route，Go 侧 `textDecide + fallbackExtract + safeFallback` 三层降级骨架继续保留。新增 `SUPERVISOR_ENGINE=auto|classic|adk`，默认 `auto`：`LLM_BACKEND=eino` 时自动挂 ADK route engine，其余场景保留 classic 路径。
最新收口：ADK route engine 遇到 `output` tool 校验失败时，不再直接把 ToolNode 错误冒泡结束；外层会抽取校验反馈并按同一用户消息做一次本地重试，补齐 classic structured route 的自纠正能力。
最新收口：Phase 5A 先以最小粒度接入 Eino callback tracing，已覆盖 `streamInterpretation` 的 Eino ChatModel 主回答链，以及 supervisor classic / ADK 路由中的底层 Eino ChatModel 调用；现有 `TurnTrace` / `TracePanel` / 文件落盘模型不变。

## 已完成功能

### 上下文工程（会话内）
- `RecentTurns` + `RunningSummary`：会话内最近多轮对话保留，超过 8 条自动滚动摘要
- 摘要合并（增量）、降级安全、失败不丢历史
- 失败新命盘不污染旧上下文、过滤文案不写回记忆

### 工作台式聊天界面
- `AssistantTurn.vue` 四层分区渲染（结构化结果 → 主回答 → 过程摘要 → 依据资料）
- `TracePanel` 产品化为"处理过程"卡
- `KnowledgeSourceCard` 按典籍分组折叠
- `ResultBlock` / `BaziChartCard` / `QimenChart` 统一卡片视觉
- 玉色/墨色/金石感深色主题
- vitest 单元测试能力

### Supervisor 架构（Phase 1 + 1.5）
- `SupervisorDecision` 结构化路由 + `DecisionSlots` + `PolicyHints`
- `DomainHandler` 接口 + `Bazi` / `Qimen` / `Ziwei` specialist 骨架
- LLM Supervisor Client（flash 模型、JSON 解析、安全降级）
- Policy Gate（白名单、并行硬禁用、低置信度强制澄清）
- `ApprovedRoute` 主控 runtime 分发，`executeRoute()` + 7 个 route handler
- `bridgeDecision` 缩减为纯 slot 提取，不再返回 `action`
- Legacy classify 回退路径保留，与 route-driven 路径互不干扰
- qimen 主域 runtime lane：`executeQimenPrimaryRoute()` 使奇门成为 mainline
- qimen 独立回答：`prompts/qimen.md` + 奇门知识检索 + 无八字直接回答

### Eino 迁移（Phase 1 + 2）
- `internal/llm/eino_chat.go`：classic Eino `ToolCallingChatModel` 适配到现有 `llm.Chat`
- `internal/llm/factory.go`：`LLM_BACKEND=eino|native` 双后端工厂，supervisor flash route 使用独立 `DisableThinking` client 避免 DeepSeek `tool_choice` + thinking 冲突
- `internal/tools/eino_adapter.go`：legacy tool → Eino `InvokableTool` 包装层
- `internal/tools/registry.go`：保留 `Get/List`，新增 `EinoTools()` 供后续 ADK / Graph / callback 接入
- 现有命理工具仍由 Go runtime 显式调度，不交给模型自治执行
- `internal/orchestrator/orchestrator.go`：当消息里已带完整出生时间但 supervisor 误判为 followup/interpret 时，Go 侧会强制回到 `collect_profile` 并补抽取 `birthplace`

### Eino 迁移（Phase 3 最小落地）
- `internal/supervisor/client.go`：引入 `RouteEngine` 注入点，layer-1 structured route 可由 classic 或 ADK 承载
- `internal/supervisor/adk_engine.go`：ADK `ChatModelAgent` + `output` tool 承载结构化路由；tool 负责 `parseAndValidate`，当 ToolNode 因校验失败返回错误时，route engine 外层会提取反馈并自动重试一次，Go 外层 fallback 仍不变
- `internal/supervisor/decision_contract.go`：classic / ADK 共用 `output` tool 名称、描述、校验反馈文案与输出 contract
- `internal/container/container.go`：按 `SUPERVISOR_ENGINE` 选择 classic / auto / adk；`auto` 在 `eino` 后端下自动启用 ADK route engine
- `internal/llm/factory.go`：新增 `NewToolCallingModel`，供 supervisor ADK engine 复用 classic Eino `ToolCallingChatModel`

### Eino 可观测性（Phase 5A 最小落地）
- `internal/tracing/eino_callback.go`：新增 `EinoTraceCallbackHandler`，通过 Eino `callbacks.Handler` 把 ChatModel 调用映射回现有 `TurnTrace` span
- `internal/orchestrator/orchestrator.go`：`streamInterpretation` 在 Eino backend 下改由 callback 产出 `llm_generate` span，避免与原手工 LLM span 重复
- `internal/supervisor/client.go` / `internal/supervisor/adk_engine.go`：classic structured route、text fallback、ADK route run 都会在 Eino backend 下产出 `supervisor_model` LLM span
- `internal/container/container.go`：`LLM_BACKEND=eino` 时启动时安装一次全局 callback handler

### 可观测性
- `TurnTrace` 统一模型 + 文件持久化（`logs/traces/`）
- 前端 trace panel 通过 SSE 推送 digest

## 关键文件

| 文件 | 用途 |
|------|------|
| `internal/orchestrator/orchestrator.go` | 主编排（Run / handler / 免责过滤 / 知识检索 / 大运分析） |
| `internal/orchestrator/route_handlers.go` | Route-driven dispatch（`executeRoute` + 7 个 handler） |
| `internal/orchestrator/extract.go` | LLM 意图分类 + `bridgeDecision`（slot 提取，不再返回 action） |
| `internal/orchestrator/prompt.go` | 知识检索 query 构建 + prompt 渲染 |
| `internal/orchestrator/summarize.go` | 滚动摘要生成 |
| `internal/orchestrator/timing.go` | 奇门时间处理 |
| `internal/supervisor/client.go` | LLM Supervisor client（Decide / parseDecision / safeFallback） |
| `internal/policy/gate.go` | Policy Gate（白名单、置信度、并行禁用） |
| `internal/schemas/supervisor_decision.go` | SupervisorDecision / DecisionSlots / PolicyHints |
| `internal/schemas/domain_result.go` | DomainResult 返回契约 |
| `internal/specialists/types.go` | DomainHandler 接口 + EventSink + ApprovedRoute |
| `internal/specialists/bazi/specialist.go` | Bazi specialist |
| `internal/specialists/qimen/specialist.go` | Qimen specialist |
| `internal/specialists/ziwei/specialist.go` | Ziwei specialist（骨架） |
| `internal/state/session.go` | 会话状态（含 RecentTurns/RunningSummary/DomainStates） |
| `internal/state/store.go` | 会话持久化存储 |
| `internal/llm/client.go` | LLM 客户端（含 temperature / flash 模型） |
| `internal/llm/eino_chat.go` | Eino classic chat model 适配层（保留 `llm.Chat` 接口） |
| `internal/llm/factory.go` | `LLM_BACKEND` 双后端工厂与 DeepSeek base URL 归一化 |
| `internal/tools/eino_adapter.go` | legacy tool 的 Eino `InvokableTool` 包装层 |
| `internal/tools/bazi/calc.go` | 八字排盘（晚子时+太阳时校正+神煞集成） |
| `internal/tools/bazi/shensha.go` | 神煞查表 |
| `internal/tools/bazi/yongshen.go` | 用神分析 |
| `internal/tools/bazi/dayun.go` | 大运分析 |
| `internal/tools/qimen/qimen.go` | 奇门遁甲 Tool |
| `internal/tools/ziwei/tool.go` | 紫微斗数 Tool（命盘+流年） |
| `internal/tools/ziwei/chart.go` | 紫微命盘构建 |
| `internal/tools/knowledge/search.go` | 知识库检索 |
| `internal/config/config.go` | 统一配置管理 |
| `internal/handler/chat.go` | SSE 聊天 HTTP handler |
| `internal/tracing/tracing.go` | TurnTrace 数据模型 + BuildDigest |
| `internal/tracing/real_tracer.go` | Tracer 实现 + span 收集 |
| `internal/tracing/file_collector.go` | Trace 文件持久化 |
| `internal/tracing/context.go` | Trace 上下文存取 |
| `internal/tracing/middleware.go` | Trace HTTP 中间件 |
| `prompts/interpret.md` | soft 模式 system prompt（完整 SOP） |
| `prompts/direct.md` | direct 模式 system prompt（简洁版） |
| `prompts/qimen.md` | 奇门遁甲专用 system prompt（无八字时使用） |
| `prompts/forensic.md` | 流年专项判断 system prompt |
| `prompts/snippets/` | 按领域拆分的 prompt 片段（career/health/marriage 等 7 个） |
| `web/src/components/AssistantTurn.vue` | assistant 四层分区渲染 |
| `web/src/components/TracePanel.vue` | 产品化过程摘要卡 |
| `web/src/components/KnowledgeSourceCard.vue` | 知识引用分组折叠卡 |
| `web/src/utils/assistantTurn.ts` | segment → 分区 view model 聚合逻辑 |
| `docs/architecture/supervisor/01-overview.md` ~ `08-phase-1.5-route-driven.md` | Supervisor 架构设计文档 |

## 待做

- [ ] 清理 legacy classify switch 与 route handler 之间的逻辑重复（Phase 2 移除 legacy path）
- [ ] 奇门知识检索接入 `runKnowledgeSearch` 主链（`buildQimenKnowledgeQuery` 已实现，仅测试覆盖）
- [ ] 上下文工程第二阶段：跨会话用户档案 / 主题线程 / 建议记录
- [ ] 测试集回归（晚子时修复后重跑）
- [ ] Makefile `dev-restart-backend` 修复
- [ ] 前端 E2E 测试（当前无 E2E 覆盖）
- [ ] 移动端响应式适配验证

## 环境变量

```
LLM_API_KEY=sk-xxx
LLM_BACKEND=eino
SUPERVISOR_ENGINE=auto
LLM_BASE_URL=https://api.deepseek.com/anthropic
LLM_MODEL=deepseek-v4-pro
LLM_FLASH_MODEL=deepseek-v4-flash
LLM_TEMPERATURE=0.3
KNOWLEDGE_MCP_URL=http://localhost:3100
DEBUG_HTTP=1
```

## 前端验证

```bash
cd web && npm run test:unit    # 单元测试（vitest）
cd web && npx vue-tsc --noEmit # TypeScript 类型检查
cd web && npm run build        # 生产构建
```

## 关键决策

- 2026-06-12：后续统一入口多专业域扩展采用 `LLM Supervisor + Go Runtime + bounded specialists` 形态。LLM 负责语义理解和路由建议，Go 负责状态、策略、执行、聚合与 SSE。
- 2026-06-12：phase 1 不做自由 swarm 或平级多 agent 协作，采用 `single supervisor + bounded specialists + optional parallel fan-out`。
- 2026-06-12：`ApprovedRoute` 成为 runtime 主控输入，`bridgeDecision` 缩减为纯 slot 提取。Legacy classify 回退路径保留但不参与新路由。
- 2026-06-12：qimen 主域 runtime lane 闭环打通 + 独立回答能力补全。无八字无 profile 时直接基于奇门盘生成回答，不再追问出生信息。
- 2026-06-12：`streamInterpretation` / `buildInterpretPrompt` / `selectPrompt` / `runKnowledgeSearch` 签名扩展 `qimenPrimary` / `qimenData` 参数，不改架构、不改 specialist 接口。
- 2026-06-13：Eino 迁移先落前两步：`llm.Chat` 底座切 classic Eino，工具注册表补 Eino 兼容视图；保留当前 supervisor / orchestrator 主控与 deterministic tool dispatch。`LLM_BACKEND=native` 作为显式回退路径保留。
- 2026-06-13：supervisor 的 structured routing 不与主回答 client 共用同一个 DeepSeek thinking 配置；独立 flash/no-thinking client 更稳，也更接近“路由模型”和“回答模型”职责分离。
- 2026-06-13：当 LLM 路由把“已含出生时间的首轮消息”误判成 followup/interpret 时，由 Go 侧按消息内容做一次确定性纠偏，优先保证首轮排盘主链可达。
- 2026-06-13：Phase 3 不直接让 `ChatModelAgent` 吞掉整套 supervisor 逻辑，而是只替换 layer-1 structured route 执行器；`textDecide / fallbackExtract / safeFallback` 仍保留在 Go 侧，避免把多层降级语义误简化成单一 retry。
- 2026-06-13：Eino `ChatModelAgent` 的 `ReturnDirectly` tool 一旦在 ToolNode 校验报错，会直接以 `NodeRunError` 结束当前 run；因此 supervisor ADK 路由的“结构化自纠正”放在 route engine 外层做一次本地重试，而不是假设 ADK 内层会继续 ReAct。
- 2026-06-13：Phase 5A 不直接重写整套 tracing，也不先碰 tool / graph callback；只先把 Eino ChatModel 主回答链接进 callback，并在 Eino backend 下关闭对应手工 `llm_generate` span，避免双记。
- 2026-06-13：Supervisor 的 callback tracing 只记录底层模型调用为 `supervisor_model` 子 span，不替代现有 `supervisor_decision` 业务链路 span；前者看模型耗时/重试，后者看整段路由决策语义。
- 2026-06-13：当前阶段适合作为“原生实现 vs Eino 渐进接入”的学习断点，先不继续推进更深的 Graph / agent 编排改造；优先通过并存代码路径理解 Go 主控边界与 Eino 基础设施边界。

## 上下文工程后续衔接

当前第一阶段（会话内）已搭好 `RecentTurns` + `RunningSummary` 的数据结构和接入链路。后续 V1.5（跨会话）可在现有基础上扩展：
- 将 `RunningSummary` 作为会话级快照，跨会话加载时可作为初始摘要
- 新增 `UserProfile` / `BaziProfile` 复用现有 `SessionState.Profile` / `BaziResult` 的序列化
- 新增 `ConsultTopic` / `AdviceRecord` 时，`Turn` 结构可直接作为 `QuestionRecord` 的数据源
不改变现有架构和数据流。
