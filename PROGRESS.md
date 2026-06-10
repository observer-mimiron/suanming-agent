# 项目进度

> 将此文件全文作为上下文输入给 Claude Code，即可独立继续开发。

---

## 当前状态

**阶段：** v1.1 全部验收通过（26/26 任务）  
**最后更新：** 2026-06-10  
**状态：** 全链路验证通过 — Go 后端 + Vue 前端 + DeepSeek v4 流式解读 + E2E 5 场景

**当前正在进行：** 收敛 v1 实施文档边界，修复 M3/M4 阻塞点，并将 LLM 方案统一为 DeepSeek v4

## 执行方式

使用 **superpowers:subagent-driven-development**。每个子任务派一个独立子 Agent。完成后对照验收标准，通过才进入下一个。

```bash
# 后端
LLM_API_KEY=sk-xxx go run ./cmd/server/

# 前端
cd web && npm run dev
```

## 子任务列表

```
M0: 项目脚手架 (30min)
  M0.1  Go 项目初始化 + gin + sse writer           → docs/v1/implementation/m0.1-go-init.md
  M0.2  Vue 3 + Naive UI + Vite 代理配置            → docs/v1/implementation/m0.2-vue-init.md
  M0.3  验证: go build + npm run build 均通过       → docs/v1/implementation/m0.3-verify.md

M1: 八字引擎 (1h)
  M1.1  工具注册中心 (Registry) + lunar-go 安装      → docs/v1/implementation/m1.1-registry.md
  M1.2  bazi_calc Tool 实现 (四柱/天干/地支/十神/五行/大运) → docs/v1/implementation/m1.2-bazi-calc.md
  M1.3  单元测试 (含十神验证 + 边界年份)             → docs/v1/implementation/m1.3-test.md

M3: Go Orchestrator (2h)
  M3.1  SessionState 模型 + 会话管理                → docs/v1/implementation/m3.1-session.md
  M3.2  Orchestrator 主逻辑 + extractProfileAndQuestion → docs/v1/implementation/m3.2-orchestrator.md
  M3.3  handleAsk 追问逻辑                          → docs/v1/implementation/m3.3-ask.md
  M3.4  handleFullReading 完整排盘流程               → docs/v1/implementation/m3.4-full-reading.md
  M3.5  handleFollowupReading 复用已有命盘           → docs/v1/implementation/m3.5-followup.md
  M3.6  重写 main.go 串联 + curl 集成测试            → docs/v1/implementation/m3.6-main-curl.md

M4: 知识库 MCP + LLM (1h)
  M4.1  MCP KnowledgeClient + knowledge_search Tool  → docs/v1/implementation/m4.1-knowledge-client.md
  M4.2  LLM Client 流式调用 (DeepSeek v4 API)       → docs/v1/implementation/m4.2-llm-client.md
  M4.3  llm_generate 集成 + 失败降级逻辑             → docs/v1/implementation/m4.3-llm-tool.md
  M4.4  注册确定性 Tool + curl 验证                 → docs/v1/implementation/m4.4-register-verify.md

M5: Vue 3 前端 (3h)
  M5.1  类型定义 + SSE composable                   → docs/v1/implementation/m5.1-types-sse.md
  M5.2  ChatPanel + ChatBubble (消息列表+输入框)     → docs/v1/implementation/m5.2-chat-panel.md
  M5.3  TextSegment + ThinkingSegment + ToolCallSegment → docs/v1/implementation/m5.3-segments.md
  M5.4  BaziChartCard 命盘卡片 (四柱+十神+五行+大运) → docs/v1/implementation/m5.4-bazi-card.md
  M5.5  KnowledgeSourceCard 知识引用卡片             → docs/v1/implementation/m5.5-knowledge-card.md
  M5.6  App.vue 暗色主题 + 编译验证                 → docs/v1/implementation/m5.6-app-build.md

M6: 集成联调 (2h)
  M6.1  全流程手动测试 (5 个 E2E 场景)               → docs/v1/implementation/m6.1-e2e.md
  M6.2  错误处理 + 降级完善                          → docs/v1/implementation/m6.2-error-handling.md
  M6.3  Prompt 调优 (解读风格)                       → docs/v1/implementation/m6.3-prompt.md
  M6.4  start.sh 一键启动脚本                        → docs/v1/implementation/m6.4-startup.md
```

## 进度总览

| 子任务 | 状态 |
|--------|------|
| M0.1 Go 项目初始化 | ✅ |
| M0.2 Vue 前端初始化 | ✅ |
| M0.3 验证编译 | ✅ |
| M1.1 Registry + lunar-go | ✅ |
| M1.2 bazi_calc 实现 | ✅ |
| M1.3 单元测试 | ✅ |
| M3.1 SessionState | ✅ |
| M3.2 Orchestrator + extractProfileAndQuestion | ✅ |
| M3.3 handleAsk | ✅ |
| M3.4 handleFullReading | ✅ |
| M3.5 handleFollowupReading | ✅ |
| M3.6 main.go + curl 测试 | ✅ |
| M4.1 KnowledgeClient | ✅ |
| M4.2 LLM Client 流式 | ✅ |
| M4.3 llm_generate 集成 | ✅ |
| M4.4 确定性 Tool 注册 + 验证 | ✅ |
| M5.1 类型 + SSE | ✅ |
| M5.2 ChatPanel + ChatBubble | ✅ |
| M5.3 基础 Segment | ✅ |
| M5.4 BaziChartCard | ✅ |
| M5.5 KnowledgeSourceCard | ✅ |
| M5.6 App.vue + 构建 | ✅ |
| M6.1 E2E 测试 | ✅ |
| M6.2 错误处理 | ✅ |
| M6.3 Prompt 调优 | ✅ |
| M6.4 启动脚本 | ✅ |

## 每个子 Agent 执行流程

1. 从进度总览取下一个 ⬜ 任务，标记为进行中
2. 读对应 `docs/v1/implementation/<任务id>.md`（自包含: 步骤 + 代码 + 验收用例）
3. 实施代码，运行验收用例
4. 全部验收通过后，标记 ✅
5. Commit: `feat: <任务编号> <描述>`

## 关键约束

- Go 单栈，不引入 Python/LangGraph
- 八字引擎用 lunar-go（天干用 GetYearGan 等，非 GetYearShengXiao）
- 十神是 P0 功能，bazi_calc 必须返回
- 参数校验必须做 type assertion + ok 模式，防止 panic
- tools.Get 必须检查 ok，nil tool 调用会 panic
- 知识检索走项目知识库 MCP
- SSE 6 种事件: thinking / tool_call / component / text / error / done
- LLM 使用 DeepSeek v4，默认关闭 thinking 输出，不在产品里展示原始 CoT
- Eino 参考代码在 `eino-agent/eino-examples/`

## 关键决策记录

- 2026-06-10：v1 中 `llm_generate` 不强制注册为 Tool。`bazi_calc / knowledge_search` 保持 Tool 化，流式 LLM 输出由 Orchestrator 直接调用 `internal/llm.Client` 并负责 SSE 推送。
- 2026-06-10：LLM 供应商切换为 DeepSeek v4。默认走 Anthropic 兼容接口，`LLM_BASE_URL=https://api.deepseek.com/anthropic`，`LLM_MODEL=deepseek-v4-pro`，并显式关闭 thinking 输出。
- 2026-06-10：正式 system prompt 从 `prompts/interpret.md` 读取；Prompt 采用“依据边界 + 追问分流 + 引用规则”三段约束，避免模型重复总评、伪造引用或脱离用户问题泛谈。

## 文档索引

| 用途 | 路径 |
|------|------|
| 产品定义 | `docs/product.md` |
| 架构总图 | `docs/architecture.md` |
| 验收标准 | `docs/v1/acceptance-criteria.md` |
| 后端方案 | `docs/v1/tech-backend.md` |
| 前端方案 | `docs/v1/tech-frontend.md` |
| 工程自检 | `docs/v1/checklist-agent-engineering.md` |
| 实施任务 | `docs/v1/implementation/m*.md` |
