# 项目进度

> 将此文件全文作为上下文输入给 Claude Code，即可独立继续开发。

---

## 当前状态

**阶段：** v1.4 Supervisor Phase 1 已实现  
**最后更新：** 2026-06-12  
**状态：** Supervisor phase 1 已合入：`SupervisorDecision` + Policy Gate + Bazi/Qimen Specialist 边界 + Orchestrator 集成，40 个后端测试通过，server 编译通过

## 已完成功能

### v1.3 可探测 Agent（阶段 0-2）
- 结构化 debug 输出：`DEBUG_HTTP=1` 时写入 JSON Lines（`logs/debug/*.jsonl`）
- `TurnTrace` 统一模型（`internal/tracing/`）：回合级 trace，含 child spans
- Span 覆盖：classify/ask/bazi_calc/yongshen/dayun_analyzer/knowledge_search/qimen_dunjia/llm_generate/fallback/degrade
- 文件持久化：`DEBUG_TRACE=1` 时落盘到 `logs/traces/{date}/{trace_id}.json`
- 前端 trace panel：通过 `component` SSE 事件推送 trace digest，可展开查看步骤时间线
- 5 个 span kind：AGENT / CHAIN / TOOL / RETRIEVER / LLM
- 配置项 `DEBUG_TRACE` 默认关闭，保守默认值

### 工作台式聊天界面改造（2026-06-11）
- **AssistantTurn 分区渲染：** 新增 `AssistantTurn.vue`，将 assistant 线性 segments 聚合为四层结构（结构化结果 → 主回答 → 过程摘要 → 依据资料）
- **纯聚合层：** `web/src/utils/assistantTurn.ts` — 8 个单元测试覆盖分组、合并、边界条件
- **ChatBubble 职责分离：** user 保留气泡，assistant 委托给 AssistantTurn
- **TracePanel 产品化：** 从调试面板升级为"处理过程"卡，收起态摘要 + 展开态时间线
- **KnowledgeSourceCard 分组折叠：** 按来源典籍分组，默认全部折叠，每份资料支持独立展开
- **ResultBlock 统一外壳：** 为 BaziChartCard / QimenChart 提供统一的卡片视觉语言
- **全局主题切换：** 从紫色系切换到玉色/墨色/金石感深色主题，jade accent (#5a9e8f)
- **页面工作台化：** ChatPanel 加宽至 900px，顶部栏精简，输入区更像工作输入台
- **前端单测能力：** 添加 vitest，`npm run test:unit` 可运行纯工具函数测试

### 上下文工程第一阶段（2026-06-11）
- **会话内最近多轮对话保留：** `SessionState` 新增 `RecentTurns []Turn`，每次对话记录 user+assistant 消息
- **滚动摘要（RunningSummary）：** 当 `RecentTurns` 超过 8 条（约 4 轮问答）时，旧 turn 被 trim 并压缩进摘要
- **摘要合并：** `summarizeTurns()` 使用 flash 模型，在已有摘要基础上补充新内容，不从头重写
- **摘要降级安全：** 摘要模型调用失败不中断主回答链路，保留旧摘要继续
- **Prompt 接入：** `buildInterpretPrompt()` 中注入「历史摘要」和「最近对话」两段上下文，置于当前问题之前
- **会话持久化兼容：** `RecentTurns`/`RunningSummary` 通过 `json:"...,omitempty"` 标签保证旧 session 文件平滑加载
- **无外部依赖引入：** 不引入数据库、向量检索、LangGraph 或新服务

### 上下文工程第一阶段修复（2026-06-12）
- **摘要失败不丢历史：** `recordTurnAndMaintainContext()` 在摘要生成失败时回滚 overflow turn，不再静默丢失旧对话
- **失败新命盘不污染旧上下文：** `new_profile` / `bazi_input` 失败时，不再把本轮 user/assistant turn 写进既有 session 的 `RecentTurns` / `RunningSummary`
- **过滤文案不写回记忆：** 被免责声明过滤掉的流式 chunk 不再进入 assistant 持久化文本，避免后续 prompt 反复回灌
- **回归测试补齐：** 新增 3 个 orchestrator 回归测试覆盖上述失败路径

### Supervisor 架构设计（2026-06-12）
- **统一入口架构定稿：** 确认后续产品维持单对话框入口，采用 `LLM Supervisor + Go Runtime + bounded specialists` 形态
- **控制边界定稿：** 明确采用 `Model for understanding, Code for control`，语义判断交给 LLM，状态/权限/策略/执行/聚合继续由 Go 持有
- **分页设计文档：** 新增 `docs/architecture/supervisor/01-07` 七页设计，覆盖概览、路由模型、状态、specialists、policy gate、trace、rollout
- **主线边界保留：** phase 1 仍以命理主线为核心，`bazi` 为第一主域，`qimen` 为第一辅助域，`emotion/career` 等非命理域先预留结构
- **实施计划已写好：** 新增 `docs/superpowers/plans/2026-06-12-supervisor-phase1-implementation.md`，按 phase 1 拆成可交给弱上下文 AI 执行的分步任务、文件边界、测试命令和 stop rules

### Supervisor Phase 1 实现（2026-06-12）
- **`SupervisorDecision` 结构化路由：** `internal/schemas/supervisor_decision.go` — `SupervisorDecision`、`DecisionSlots`、`PolicyHints`、`Normalize()`
- **`DomainResult` & Specialist 契约：** `internal/schemas/domain_result.go` + `internal/specialists/types.go` — `DomainHandler` interface
- **Supervisor Prompt 资产：** `prompts/supervisor/conversation_router.md`、`domain_router.md`、`task_router.md`
- **LLM Supervisor Client：** `internal/supervisor/client.go` — `Decide()` 方法，加载 prompt 资产、调用 flash 模型、解析 JSON、安全降级
- **Policy Gate：** `internal/policy/gate.go` — `ApprovedRoute`、`Apply()`，phase-1 白名单（bazi/qimen）、并行硬禁用、低置信度强制澄清
- **Bazi Specialist：** `internal/specialists/bazi/specialist.go` — 薄封装层，处理资料完备性检查和路由分发
- **Qimen Specialist：** `internal/specialists/qimen/specialist.go` — 薄封装层，仅响应 timing 路由，结果始终 supplemental
- **Orchestrator 集成：** `internal/orchestrator/orchestrator.go` — `SetSupervisor()` 方法，supervisor 可用时使用新路由，否则回退 legacy classify
- **路由级 Trace Span：** `supervisor_decision`、`policy_gate`
- **Session 扩展：** `RoutingSnapshot`、`BaziState`、`QimenState`、`DomainStates`
- **Container 注入：** `internal/container/container.go` — supervisor client 创建和注入
- **测试覆盖：** 40 个测试覆盖 7 个包（supervisor/policy/state/bazi/qimen/orchestrator/tracing），server 编译通过
- **约束遵守：** 无并行 fan-out、无非命理域启用、前端/核心工具未改动

## 关键文件

| 文件 | 用途 |
|------|------|
| `internal/tools/shensha.go` | 神煞查表 |
| `internal/tools/qimen.go` | 奇门遁甲 Tool |
| `internal/tools/bazi_calc.go` | 八字排盘（含晚子时+太阳时校正+神煞集成） |
| `internal/tools/yongshen.go` | 用神分析（含晚子时+分钟精度太阳时） |
| `internal/orchestrator/extract.go` | LLM 意图分类 + 信息提取 |
| `internal/orchestrator/orchestrator.go` | 主编排（bazi_input/免责过滤/条件知识检索/大运分析） |
| `internal/orchestrator/timing.go` | 奇门时间处理 |
| `internal/orchestrator/prompt.go` | 知识检索 query 构建 + prompt 渲染 |
| `internal/tracing/turn_trace.go` | TurnTrace/TraceSpan 数据模型 + BuildDigest |
| `internal/tracing/real_tracer.go` | 真实 Tracer 实现 + span 收集 |
| `internal/tracing/file_collector.go` | Trace 文件持久化（logs/traces/） |
| `internal/tracing/context.go` | Trace 上下文存取 |
| `web/src/components/ChatPanel.vue` | 工作台式聊天页面骨架 |
| `web/src/components/ChatBubble.vue` | 用户气泡 / assistant 委托 AssistantTurn |
| `web/src/components/AssistantTurn.vue` | assistant 四层分区渲染（新增） |
| `web/src/components/ResultBlock.vue` | 统一结构化结果卡外壳（新增） |
| `web/src/components/TracePanel.vue` | 产品化过程摘要卡 |
| `web/src/components/KnowledgeSourceCard.vue` | 按来源分组、可折叠的知识引用 |
| `web/src/components/BaziChartCard.vue` | 八字命盘（去外壳，收进 ResultBlock） |
| `web/src/components/QimenChart.vue` | 奇门九宫格（去外壳，收进 ResultBlock） |
| `web/src/utils/assistantTurn.ts` | assistant 消息 segment → 分区 view model 聚合逻辑（新增） |
| `web/src/utils/assistantTurn.test.ts` | 聚合逻辑 8 个单元测试（新增） |
| `web/src/style.css` | 全局玉色/墨色主题变量 |
| `web/src/App.vue` | Naive UI 主题覆写 |
| `web/vitest.config.ts` | vitest 配置（新增） |
| `prompts/interpret.md` | soft 模式 system prompt（完整 SOP） |
| `prompts/direct.md` | direct 模式 system prompt（简洁版） |
| `internal/orchestrator/summarize.go` | 滚动摘要生成（flash 模型，合并已有摘要） |
| `internal/state/session.go` | 会话状态（含 RecentTurns/RunningSummary/NeedsQimen/NeedsKnowledge） |
| `internal/state/store.go` | 会话持久化存储 |
| `internal/llm/client.go` | LLM 客户端（含 temperature） |
| `docs/superpowers/specs/2026-06-12-supervisor-architecture-design.md` | Supervisor 架构总设计索引 |
| `docs/architecture/supervisor/01-overview.md` ~ `07-rollout-plan.md` | Supervisor 架构分页设计文档 |
| `docs/superpowers/plans/2026-06-12-supervisor-phase1-implementation.md` | Supervisor phase 1 可执行实施计划 |
| `internal/schemas/supervisor_decision.go` | SupervisorDecision / DecisionSlots / PolicyHints schema |
| `internal/schemas/domain_result.go` | DomainResult 返回契约 |
| `internal/specialists/types.go` | DomainHandler 接口 + EventSink + ApprovedRoute |
| `internal/specialists/bazi/specialist.go` | Bazi specialist 薄封装 |
| `internal/specialists/qimen/specialist.go` | Qimen specialist 薄封装（仅 timing，始终 supplemental） |
| `internal/supervisor/client.go` | LLM Supervisor client（Decide / parseDecision / safeFallback） |
| `internal/policy/gate.go` | Policy Gate（ApprovedRoute / Apply / phase-1 白名单） |
| `prompts/supervisor/conversation_router.md` | 对话意图路由 prompt |
| `prompts/supervisor/domain_router.md` | 领域路由 prompt |
| `prompts/supervisor/task_router.md` | 任务路由 prompt |

## 环境变量

`.env` 文件：
```
LLM_API_KEY=sk-xxx
LLM_BASE_URL=https://api.deepseek.com/anthropic
LLM_MODEL=deepseek-v4-pro
LLM_FLASH_MODEL=deepseek-v4-flash
LLM_TEMPERATURE=0.3
KNOWLEDGE_MCP_URL=http://localhost:3100
DEBUG_HTTP=1
```

## 前端验证命令

```bash
cd web && npm run test:unit    # 单元测试（vitest）
cd web && npx vue-tsc --noEmit # TypeScript 类型检查
cd web && npm run build        # 生产构建
```

## 待做

- [ ] 上下文工程第二阶段：跨会话用户档案 / 主题线程 / 建议记录（参考 `docs/learning/long-term-consulting-evolution.md` V1.5 → V2）
- [ ] 可观测性：结构化 trace 日志（计划 `docs/learning/13-observability-and-trace-ui-plan.md`）
- [x] Supervisor phase 1：引入 `SupervisorDecision` / policy gate / `bazi` 与 `qimen` specialist 骨架（2026-06-12 已完成）
- [ ] 测试集回归（晚子时修复后重跑）
- [ ] Makefile `dev-restart-backend` 修复
- [ ] 前端 E2E 测试（当前无 E2E 覆盖）
- [ ] 移动端响应式适配验证（当前未在 375px 宽度做系统验证）

## 关键决策记录

- 2026-06-12：后续统一入口多专业域扩展采用 Supervisor 架构，但最终控制权仍归 Go runtime。LLM 负责语义理解和路由建议，Go 负责状态、策略、执行、聚合与 SSE。
- 2026-06-12：phase 1 不做自由 swarm 或平级多 agent 协作，采用 `single supervisor + bounded specialists + optional parallel fan-out`。
- 2026-06-12：Supervisor phase 1 实现完成。通过 `SetSupervisor()` 注入 orchestrator，supervisor 不可用时保留 legacy classify 回退路径。新增 40 个测试覆盖 7 个包。Server 编译通过。Phase 2 待启动。

### 上下文工程后续衔接说明
当前第一阶段（会话内）已搭好 `RecentTurns` + `RunningSummary` 的数据结构和接入链路。
后续 V1.5（跨会话）可在现有基础上扩展：
- 将 `RunningSummary` 作为会话级快照，跨会话加载时可作为初始摘要
- 新增 `UserProfile` / `BaziProfile` 复用现有 `SessionState.Profile` / `BaziResult` 的序列化
- 新增 `ConsultTopic` / `AdviceRecord` 时，`Turn` 结构可直接作为 `QuestionRecord` 的数据源
不改变现有架构和数据流。
