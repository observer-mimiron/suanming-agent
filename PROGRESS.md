# 项目进度

> 将此文件全文作为上下文输入给 Claude Code，即可独立继续开发。

---

## 当前状态

**阶段：** v1.5 Supervisor Phase 1.5 收口 + qimen 独立回答  
**分支：** feat/multi-prompt  
**最后更新：** 2026-06-13  
**状态：** 92 个测试通过，0 失败。Server / frontend 编译通过。工具包正在从扁平文件重构为子包（`tools/bazi/`、`tools/qimen/`、`tools/knowledge/`、`tools/ziwei/`），原有文件仍保留。

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

## 上下文工程后续衔接

当前第一阶段（会话内）已搭好 `RecentTurns` + `RunningSummary` 的数据结构和接入链路。后续 V1.5（跨会话）可在现有基础上扩展：
- 将 `RunningSummary` 作为会话级快照，跨会话加载时可作为初始摘要
- 新增 `UserProfile` / `BaziProfile` 复用现有 `SessionState.Profile` / `BaziResult` 的序列化
- 新增 `ConsultTopic` / `AdviceRecord` 时，`Turn` 结构可直接作为 `QuestionRecord` 的数据源
不改变现有架构和数据流。
