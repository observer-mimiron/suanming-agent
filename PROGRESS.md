# 项目进度

> 将此文件全文作为上下文输入给 Claude Code，即可独立继续开发。

---

## 当前状态

**阶段：** v1.5 Supervisor Phase 1.5 收口 + Eino Phase 1-3 完成中 + Phase 5A 启动  
**分支：** codex/eino-phase1-2  
**最后更新：** 2026-06-16  
**状态：** `llm.Chat` 已收口为 Eino backend，原生 HTTP LLM 路径已删除；supervisor 运行面也已进一步收口为固定 ADK route engine，`SUPERVISOR_ENGINE` 配置、classic `structuredDecide` 路径、`GenerateWithTool` 接口以及 registry 的 Eino `InvokableTool` 兼容导出都已删除。现有 supervisor / orchestrator / SSE 协议保持不变，Go 侧仍保留 `textDecide + fallbackExtract + safeFallback` 三层降级骨架。
最新收口：ADK route engine 遇到 `output` tool 校验失败时，不再直接把 ToolNode 错误冒泡结束；外层会抽取校验反馈并按同一用户消息做一次本地重试，补齐 classic structured route 的自纠正能力。
最新收口：Phase 5A 先以最小粒度接入 Eino callback tracing，已覆盖 `streamInterpretation` 的 Eino ChatModel 主回答链，以及 supervisor ADK 路由和 text fallback 中的底层 Eino ChatModel 调用；现有 `TurnTrace` / `TracePanel` / 文件落盘模型继续保留为稳定展示契约。Phase 5B 的第一刀已把 `knowledge_search` 的 retriever 事件源切到 Eino callback，generic tool callback 迁移仍待后续推进。
最新收口：`bridgeDecision` 和重复的 `specialists.ApprovedRoute` 已删除；`ApprovedRoute` 现在被 runtime 直接消费，`specialists` 改为复用统一的 `policy.ApprovedRoute` 契约。
最新收口：已引入 `internal/runtime/` 作为“已批准路由执行层”，把候选会话流转、领域执行、知识检索、prompt 构建与流式回答从 `internal/orchestrator/` 主包移出；`orchestrator` 现收缩为 turn lifecycle 外壳。
最新收口：已修复普通追问误带奇门的问题。`fortune_followup` 不再默认触发 `qimen_dunjia`，且 `NeedsQimen` 只由当前批准路由决定，不再把上一轮择时状态粘到后续八字概念追问。
最新收口：奇门触发从“强绑 task_intent”调整为“task_intent + capability hints”混合模式。新增 `policy_hints.qimen_mode=none|supplement|primary` 与 `profile_requirement=none|full`：允许“今天运气/本日运道”这类问题在无八字资料时直接走奇门主链；只有明确要求结合个人命盘时，才因资料不足追问出生信息。

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
- 用户气泡文本颜色加深与 Naive UI 样式防覆盖，统一两状态下输入框物理尺寸（52px 高度与 16px 圆角）
- 引入大屏页面整体水平居中布局与自适应留白（大屏 32px padding 与 margin: 0 auto）
- 奇门遁甲盘升级现代塔罗几何星阵版式，剔除杂乱的五行彩色图标，实现精美虚线页眉、天人并排对称、天干克应胶囊图章，展现极简神秘美学
- 修复八字命盘组件内部与外层 ResultBlock 卡片标题重复渲染两个“八字命盘”的问题
- 升级前端自适应布局为响应式流式布局，放宽主体与输入框限制至 920px 黄金尺寸，配合排盘卡牌 100% 宽度，支持在大屏下自适应极致横向铺展
- 引入输入框 focus 淡金色呼吸发光动态边缘，引入奇门九宫卡牌 hover 向上物理浮起与金色外发光微动效以增强占卜仪式感
- 修复八字命盘神煞渲染缺失，在四柱卡片底端通过精致虚线分隔呈现各柱神煞（如天乙贵人、桃花、华盖等），并设计吉/凶/平的专属主题调色小徽章
- 支持对“思考链”与“知识来源依据资料”卡片进行点击折叠/展开交互，并将默认初始状态设为折叠收起，优化垂直阅读动线
- 解决遗留的 TypeScript 编译及事件处理报错，补齐 TraceDigest / TraceStep 数据结构，实现前端一键生产构建（npm run build）零报错通过
- 重构八字大运为“命运时间轴 (Fate Timeline)”，使用黄金横轴轨道、高亮呼吸星节点与大字 Serif 干支，大幅改善原本平平无奇的药丸标签格式
- 为排盘卡牌（八字、奇门）编写 3D Parallax Tilt 视差偏转与 hover 掠光扫金（Shimmer Sweep）反光效果，升级四角为精细的 L 金属直角包边。针对奇门遁甲盘，在前端 computed 属性中根据后天八卦物理方位对九宫格进行对齐重排，若中宫（中五宫）缺失则自动补齐带有自旋太极图标和虚线边框的 Dummy 卡牌（‘中宫定盘’）凑满 3x3 矩阵；并去除效果差的水印，改在虚线页眉最左侧设计具有各五行专属轻柔底色与圆角边框的“属性小徽章”，大幅提升整体精致度与年轻塔罗风格感
- 移除传统跳跃小圆点，自研纯 CSS+SVG 的“星体轨道运转加载仪 (Celestial Loader)”，以行星异步圆周公转展现等待过程，极大提升占卜预测的交互仪式感



### Supervisor 架构（Phase 1 + 1.5）
- `SupervisorDecision` 结构化路由 + `DecisionSlots` + `PolicyHints`
- `DomainHandler` 接口 + `Bazi` / `Qimen` / `Ziwei` specialist 骨架
- LLM Supervisor Client（flash 模型、JSON 解析、安全降级）
- Policy Gate（白名单、并行硬禁用、低置信度强制澄清）
- `ApprovedRoute` 主控 runtime 分发，已由 `internal/runtime` 直接消费
- `bridgeDecision` 已删除，不再保留 action 兼容桥
- Legacy classify 回退路径保留，与 route-driven 路径互不干扰
- qimen 主域 runtime lane：`executeQimenPrimaryRoute()` 使奇门成为 mainline
- qimen 独立回答：`prompts/qimen.md` + 奇门知识检索 + 无八字直接回答

### Eino 迁移（Phase 1 + 2）
- `internal/llm/eino_chat.go`：classic Eino `ToolCallingChatModel` 适配到现有 `llm.Chat`
- `internal/llm/factory.go`：Eino-only 工厂，supervisor flash route 使用独立 `DisableThinking` client 避免 DeepSeek `tool_choice` + thinking 冲突
- 现有命理工具仍由 Go runtime 显式调度，不交给模型自治执行
- `internal/orchestrator/orchestrator.go`：当消息里已带完整出生时间但 supervisor 误判为 followup/interpret 时，Go 侧会强制回到 `collect_profile`

### Eino 迁移（Phase 3 最小落地）
- `internal/supervisor/client.go`：保留 `RouteEngine` 注入点，但运行时固定接入 ADK layer-1 structured route
- `internal/supervisor/adk_engine.go`：ADK `ChatModelAgent` + `output` tool 承载结构化路由；tool 负责 `parseAndValidate`，当 ToolNode 因校验失败返回错误时，route engine 外层会提取反馈并自动重试一次，Go 外层 fallback 仍不变
- `internal/supervisor/decision_contract.go`：ADK route engine 复用统一 `output` tool 名称、描述、校验反馈文案与输出 contract
- `internal/container/container.go`：固定构建 ADK route engine，不再保留 classic / auto 切换
- `internal/llm/factory.go`：新增 `NewToolCallingModel`，供 supervisor ADK engine 复用 classic Eino `ToolCallingChatModel`

### Eino 可观测性（Phase 5A 最小落地）
- `internal/tracing/eino_callback.go`：新增 `EinoTraceCallbackHandler`，通过 Eino `callbacks.Handler` 把 ChatModel 调用映射回现有 `TurnTrace` span
- `internal/orchestrator/orchestrator.go`：`streamInterpretation` 在 Eino backend 下改由 callback 产出 `llm_generate` span，避免与原手工 LLM span 重复
- `internal/supervisor/client.go` / `internal/supervisor/adk_engine.go`：classic structured route、text fallback、ADK route run 都会在 Eino backend 下产出 `supervisor_model` LLM span
- `internal/container/container.go`：启动时安装一次全局 callback handler
- `internal/runtime/answer.go`：`knowledge_search` 不再手工创建 retriever span，改为复用 Eino retriever callback 生命周期回填 `TurnTrace`

### 可观测性
- `TurnTrace` 统一模型 + 文件持久化（`logs/traces/`）
- 前端 trace panel 通过 SSE 推送 digest

## 关键文件

| 文件 | 用途 |
|------|------|
| `internal/orchestrator/orchestrator.go` | 会话生命周期外壳（Run / supervisor approve / runtime execute / trace / save） |
| `internal/orchestrator/event.go` | runtime 事件类型别名，供 handler/SSE 适配层复用 |
| `internal/orchestrator/summarize.go` | 滚动摘要生成 |
| `internal/runtime/executor.go` | 已批准路由执行入口（路由快照、specialist 分发、进入执行链） |
| `internal/runtime/candidate.go` | 候选会话流转与 route handler 分发 |
| `internal/runtime/bazi.go` | 八字主链执行、排盘复用、工具触发 |
| `internal/runtime/qimen.go` | 奇门主链与奇门辅助 followup 执行 |
| `internal/runtime/ziwei.go` | 紫微主链执行 |
| `internal/runtime/answer.go` | 回答管线（知识检索、流式解读、统一错误回传） |
| `internal/runtime/prompt.go` | Prompt 渲染、主领域回答指导与知识检索 query 构建 |
| `internal/runtime/timing.go` | 奇门时间处理 |
| `internal/supervisor/client.go` | LLM Supervisor client（Decide / parseDecision / safeFallback） |
| `internal/policy/gate.go` | Policy Gate（白名单、置信度、并行禁用） |
| `internal/schemas/supervisor_decision.go` | SupervisorDecision / DecisionSlots / PolicyHints |
| `internal/schemas/domain_result.go` | DomainResult 返回契约 |
| `internal/specialists/types.go` | DomainHandler 接口 + EventSink（复用 `policy.ApprovedRoute`） |
| `internal/specialists/bazi/specialist.go` | Bazi specialist |
| `internal/specialists/qimen/specialist.go` | Qimen specialist |
| `internal/specialists/ziwei/specialist.go` | Ziwei specialist（骨架） |
| `internal/state/session.go` | 会话状态（含 RecentTurns/RunningSummary/DomainStates） |
| `internal/state/store.go` | 会话持久化存储 |
| `internal/llm/eino_chat.go` | Eino classic chat model 适配层（保留 `llm.Chat` 接口） |
| `internal/llm/factory.go` | Eino 工厂与 DeepSeek base URL 归一化 |
| `internal/tools/registry.go` | 工具注册表与 Go runtime tool 视图 |
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
- 2026-06-13：Eino 迁移先落前两步：`llm.Chat` 底座切 classic Eino，工具注册表补 Eino 兼容视图；保留当前 supervisor / orchestrator 主控与 deterministic tool dispatch。
- 2026-06-16：原生 HTTP LLM client 与 `LLM_BACKEND` 双后端分支已删除，运行时正式收口为 Eino-only；这是后续继续清理 supervisor / tool 兼容层之前的阶段性状态。
- 2026-06-16：supervisor 的 `classic|adk` 切换、`SUPERVISOR_ENGINE` 配置、`llm.Chat.GenerateWithTool` 接口和 registry 的 Eino tool 兼容导出已删除；当前后端仅保留 Eino runtime + ADK route engine 一条主线。
- 2026-06-16：执行层从单一 fortune_teller Agent 迁移为 Supervisor Agent + AgentAsTool + Specialist Agent 架构。父 Agent 被 ApprovedRoute 约束，线程安全。确定性 preflight 替代 DomainHandler 薄状态机。specialists 包只承载 Config/Registry，DomainHandler 已删除。
- 2026-06-13：supervisor 的 structured routing 不与主回答 client 共用同一个 DeepSeek thinking 配置；独立 flash/no-thinking client 更稳，也更接近“路由模型”和“回答模型”职责分离。
- 2026-06-13：当 LLM 路由把“已含出生时间的首轮消息”误判成 followup/interpret 时，由 Go 侧按消息内容做一次确定性纠偏，优先保证首轮排盘主链可达。
- 2026-06-13：Phase 3 不直接让 `ChatModelAgent` 吞掉整套 supervisor 逻辑，而是只替换 layer-1 structured route 执行器；`textDecide / fallbackExtract / safeFallback` 仍保留在 Go 侧，避免把多层降级语义误简化成单一 retry。
- 2026-06-13：Eino `ChatModelAgent` 的 `ReturnDirectly` tool 一旦在 ToolNode 校验报错，会直接以 `NodeRunError` 结束当前 run；因此 supervisor ADK 路由的“结构化自纠正”放在 route engine 外层做一次本地重试，而不是假设 ADK 内层会继续 ReAct。
- 2026-06-13：Phase 5A 不直接重写整套 tracing，也不先碰 tool / graph callback；只先把 Eino ChatModel 主回答链接进 callback，并在 Eino backend 下关闭对应手工 `llm_generate` span，避免双记。
- 2026-06-13：Supervisor 的 callback tracing 只记录底层模型调用为 `supervisor_model` 子 span，不替代现有 `supervisor_decision` 业务链路 span；前者看模型耗时/重试，后者看整段路由决策语义。
- 2026-06-13：当前阶段适合作为“原生实现 vs Eino 渐进接入”的学习断点，先不继续推进更深的 Graph / agent 编排改造；优先通过并存代码路径理解 Go 主控边界与 Eino 基础设施边界。
- 2026-06-15：出生资料提取的唯一主责任重新收回 supervisor；orchestrator 不再维护第二套正则版资料提取器，只保留 cheap deterministic signal helpers 做误路由纠偏和直接八字识别。
- 2026-06-15：`route_handlers.go` 只保留 runtime 分发职责；奇门/紫微主链拆到独立文件，后续新增术数优先按领域文件扩展，而不是继续往单文件堆 handler。
- 2026-06-15：`orchestrator` 的职责继续收缩为 turn runtime：加载/保存 session、消费已批准路由、驱动 candidate 流转、调度领域执行、输出 SSE 与 trace。route approval 细节不再留在本层。
- 2026-06-15：`supervisor` 不再只返回原始 `SupervisorDecision` 给 orchestrator；它现在负责产出最终 `ApprovedRoute`，包括策略门、确定性纠偏和资料补抽，目录职责边界比之前更清楚。
- 2026-06-15：`bridgeDecision` 已彻底删除；runtime 直接读取 `policy.ApprovedRoute` 与原始用户消息，不再通过 action 兼容桥回退旧模型。
- 2026-06-15：新增 `internal/runtime/` 作为已批准路由执行层；`internal/orchestrator/` 收缩为生命周期壳层，`internal/specialists/` 改为只承载 bounded domain policy/preflight，而不再维护第二份 route 契约。
- 2026-06-15：普通八字追问与择时追问已在 runtime 明确分流：`fortune_followup` 不再自动视为奇门补充，且 `NeedsQimen` 不再跨轮粘滞，避免“上一轮问奇门，下一轮问印绶/财星仍弹奇门盘”的会话污染。
- 2026-06-15：前端 UI 细节重构，穿透设置用户气泡字色以解决全局样式覆盖；统一输入框 52px 高度与 16px 圆角；升级前端为 920px 宽的响应式流式布局并自动居中；重构奇门遁甲盘至现代塔罗几何星阵版式，在右上角保留 14px 元素微标，实现天人星门对称与天干胶囊，引入 hover 物理悬浮微光；并将“思考链”与“知识来源依据资料”卡片设为点击折叠且默认收起以优化版面。
- 2026-06-15：彻底修复浏览器 304 强缓存导致奇门九宫格补齐及属性徽章无法更新的 bug。通过在 `vite.config.ts` 中配置打包输出文件名挂载构建时间戳后缀（如 `index-[hash]-[timestamp].js`），从而完美打破浏览器强缓存，强制加载带有 3x3 矩阵中宫定盘太极图的最新前端代码。

## 上下文工程后续衔接

当前第一阶段（会话内）已搭好 `RecentTurns` + `RunningSummary` 的数据结构和接入链路。后续 V1.5（跨会话）可在现有基础上扩展：
- 将 `RunningSummary` 作为会话级快照，跨会话加载时可作为初始摘要
- 新增 `UserProfile` / `BaziProfile` 复用现有 `SessionState.Profile` / `BaziResult` 的序列化
- 新增 `ConsultTopic` / `AdviceRecord` 时，`Turn` 结构可直接作为 `QuestionRecord` 的数据源
不改变现有架构和数据流。
