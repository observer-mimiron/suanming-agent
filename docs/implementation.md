# 实施状态

> 进度详情见 [PROGRESS.md](../PROGRESS.md)，本文为状态摘要和模块依赖关系。

## 当前阶段

**v1.5 Supervisor Phase 1.5 收口 + Eino Phase 1-5B 进行中**

最后更新：2026-06-19

## 已完成

### 上下文工程（会话内）
- RecentTurns + RunningSummary：会话内最近多轮对话保留，超过 8 条自动滚动摘要
- 摘要合并（增量）、降级安全、失败不丢历史

### Supervisor 架构
- `SupervisorDecision` 结构化路由 + `DecisionSlots` + `PolicyHints`
- `DomainHandler` 接口 → 已演进为 AgentAsTool Specialist
- LLM Supervisor Client（flash 模型、JSON 解析、安全降级）
- Policy Gate（白名单、并行硬禁用、低置信度强制澄清）
- `ApprovedRoute` 主控 runtime 分发
- `bridgeDecision` 已删除
- routing prompt 已改为“术数能力画像 + 判题步骤”，不再依赖逐条 case 词表
- 显式术数方法（八字 / 紫微 / 奇门）由 `normalizeApprovedRoute` 做 deterministic obey

### 多 Agent 执行（AgentAsTool）
- Supervisor Agent + AgentAsTool + Specialist Agent 架构
- `internal/runtime/` 作为已批准路由执行层
- `internal/orchestrator/` 收缩为生命周期壳层
- Bazi / Qimen / Ziwei 三个领域 Specialist
- agentEventBridge 桥接 ADK 事件到 SSE
- post-run contract gate：`qimen` / `ziwei` 主链必须真拿到对应命盘结果才允许输出最终结论
- `prefill` 已收缩为八字可复用链，不再承担紫微 correctness

### Eino 迁移
- `llm.Chat` 底座 Eino-only
- ADK 固定 route engine，`classic|adk` 开关已删除
- Eino callback tracing 覆盖 ChatModel + supervisor + knowledge_search retriever

### 可观测性
- `TurnTrace` 统一模型 + 文件持久化
- 前端 TracePanel 通过 SSE 推送 digest
- KnowledgeSourceCard 按典籍分组折叠

### 前端
- 工作台式聊天界面（四层分区渲染）
- 八字命盘卡牌、奇门遁甲盘、命运时间轴
- 玉色/墨色/金石感深色主题
- 星体轨道运转加载仪 (Celestial Loader)
- 响应式流式布局（920px 黄金尺寸）
- vitest 单元测试 + vue-tsc 类型检查 + 零报错生产构建

## 待做

- [ ] 清理 legacy classify switch 与 route handler 逻辑重复
- [ ] Agentic RAG 实施（证据规划 + 条件反思）：设计已完成，待写实施计划
- [ ] 奇门知识检索接入 `runKnowledgeSearch` 主链
- [ ] 上下文工程第二阶段：跨会话用户档案 / 主题线程 / 建议记录
- [ ] 测试集回归（晚子时修复后重跑）
- [ ] 前端 E2E 测试
- [ ] 移动端响应式适配验证

## 模块依赖

```
orchestrator (生命周期壳)
  ├── supervisor (路由审批)
  │     ├── LLM route model
  │     ├── ADK engine (Eino ChatModelAgent)
  │     └── fallback (textDecide -> fallbackExtract -> safeFallback)
  ├── policy (策略门)
  └── runtime (路由执行)
        ├── executor (AgentAsTool 调度)
        │     ├── agent_route (Supervisor Agent 构建)
        │     ├── adapter (Tool -> Eino BaseTool)
        │     ├── bridge (ADK 事件 -> SSE 中间事件)
        │     └── final_guard (最终文本契约验收)
        ├── specialist configs
        │     ├── bazi (bazi_calc / yongshen / dayun / knowledge_search)
        │     ├── qimen (qimen_dunjia / knowledge_search)
        │     └── ziwei (ziwei_calc / knowledge_search)
        ├── tools (Registry: bazi / qimen / ziwei / knowledge)
        │     └── mcp (知识库 HTTP client)
        ├── state (会话持久化)
        └── tracing (TurnTrace / callback / file collector)
```

## 关键文件

| 文件 | 用途 |
|------|------|
| `internal/orchestrator/orchestrator.go` | 会话生命周期外壳 |
| `internal/supervisor/client.go` | LLM Supervisor client |
| `internal/supervisor/adk_engine.go` | ADK route engine |
| `internal/policy/gate.go` | Policy Gate |
| `internal/runtime/executor.go` | AgentAsTool 执行入口 |
| `internal/runtime/agent_route.go` | Supervisor Agent 构建 |
| `internal/runtime/adapter.go` | Tool -> Eino 适配 |
| `internal/runtime/bridge.go` | ADK 事件 -> SSE 桥接 |
| `internal/specialists/bazi/specialist.go` | 八字 specialist config |
| `internal/specialists/qimen/specialist.go` | 奇门 specialist config |
| `internal/specialists/ziwei/specialist.go` | 紫微 specialist config |
| `internal/runtime/adapter.go` | 工具 Eino adapter（含 knowledge_search 限次逻辑） |
| `internal/mcp/client.go` | MCP 知识库 HTTP client |
| `internal/tracing/` | Trace 模型 + callback |
| `prompts/interpret.md` | LLM 解读 prompt 模板 |
| `prompts/supervisor/` | 路由 prompt 模板 |
