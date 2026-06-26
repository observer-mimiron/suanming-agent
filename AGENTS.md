# AGENTS.md

本文件是项目的**唯一 AI 编码助手指导文件**。Codex、Claude Code 及其他 AI 编码工具均以此文件为准。

## 进度维护规则

**PROGRESS.md 是本项目的上下文恢复文件。** 其作用是：在新对话中，将 PROGRESS.md 的内容作为上下文输入给任何 AI 编码助手，即可独立继续开发。

**必须更新 PROGRESS.md 的时机：**
- 完成一个验收用例节点
- 做出会影响后续开发的架构变更
- 新增或修改环境变量/启动方式
- 遇到并解决了一个阻塞性问题（记录原因和解决方案）

**更新内容：** 更新进度总览表的状态、完成日期；如果正在进行的任务变了，更新「正在进行的任务」；如果做出了新决策，追加到「关键决策记录」。

**会话开始时：** 先读取 PROGRESS.md 了解当前进度，再读对应模块的实施文档。

## 项目概述

命理大师 — AI 八字算命 Agent 聊天应用。学习项目，覆盖 Multi-Agent、RAG、Plan-Execute、SSE 等企业 Agent 开发技巧。

## 架构核心

**全 Go 原生，两层 "supervisor" 各司其职：**

1. **RouteAdvisor**（`internal/supervisor/`，Go ADK RouteEngine）— 路由决策。三层防御（ADK structured → textDecide → safeFallback），产出 `SupervisorDecision`（L0 对话意图 → L1 领域 → L2 任务 → L3 槽位）。
2. **Supervisor Agent**（Go ADK ChatModelAgent，每轮动态构建）— 执行分发。将本轮允许的 AgentTool specialists 挂载后分发给领域专家执行。

```
Vue 3 → SSE → Gin (:8080)
                │
         Eino ADK / lunar-go / MCP→RAG (:3100)
```

**架构单一事实来源：** `docs/architecture.md`（入口）和 `docs/architecture/supervisor/`（专题）。任何架构决策变更必须先更新该文档。

## 三服务启动

可使用 `make dev` 一键启动全部三个服务（推荐），或分别启动：
make knowledge-start      # 仅知识库
make dev-backend          # 仅后端
make dev-frontend         # 仅前端

```bash
# 知识库 (Next.js :3100)
cd knowledge && npx next dev -p 3100

# 执行层 (Go :8080)
LLM_API_KEY=sk-xxx go run ./cmd/server/

# 前端 (Vue :5173)
cd web && npm run dev
```

**端口：** 知识库 :3100，后端默认 :8080（可通过 `LISTEN_ADDR` 环境变量修改），前端 :5173。
环境变量：`LLM_API_KEY`（必填）、`LLM_BASE_URL`、`LLM_MODEL`、`KNOWLEDGE_MCP_URL`。

## 知识库

独立知识库实例运行在端口 3100（数据目录 `knowledge/wiki/`），与 lisense 知识库（端口 3000）完全隔离。已导入 19 个页面，涵盖古籍原文、八字基础、格局用神等模块，含权威分级和交叉引用。

| 命令 | 用途 |
|------|------|
| `make knowledge-start` | 启动知识库 |
| `make knowledge-stop` | 停止知识库 |
| `make knowledge-status` | 查看状态 |
| `make knowledge-import` | 重新导入资料 |

Go 后端的 `knowledge_search` 工具自动连接 `RAG_MCP_URL`（默认 :3100），检索知识库返回命理资料注入到 LLM 解读中。

### 知识库图结构与工具

- `knowledge_catalog`：通过 `/api/wiki/graph` 获取知识库图结构，按 slug 前缀过滤跨书引用后生成目录摘要（古籍名称、章节数、前 5 个章节标题），供 Agent 规划检索策略。
- `knowledge_search`：检索古籍原文，返回 passages 数组（content + source）。Go adapter 层硬控每轮最多 3 次调用。
- `/api/wiki/graph` 的 edges 由 markdown 正文链接扫描生成（`/\[([^\]]*)\]\(([^)]+)\.md\)/g`），不是系统目录树。目录页的章节关系依赖目录页正文中的链接结构。

### 与 Yopedia 的区别

项目专属知识库（:3100）独立于 yopedia 通用知识库（:3000）。前者专为命理咨询场景导入古籍原文、八字基础、格局用神等模块；后者存储通用工作知识和百科页面。

## 开发命令

```bash
# Go
go build ./cmd/server/          # 编译
go test ./... -v                # 全部测试
go test ./internal/tools/ -v    # 单个包测试

# Vue
cd web && npm run dev           # 开发
npx vue-tsc --noEmit            # 类型检查
npm run build                   # 构建
```

## 代码查阅

项目已初始化 `.codegraph/` 索引。查代码优先用 `codegraph_*` MCP 工具（`codegraph_context` / `codegraph_search` / `codegraph_trace` / `codegraph_callers` / `codegraph_callees`），比 grep/find 更快且结构化。找不到再 fallback 到文件系统。

## 实施状态

项目处于 **v1.5 收口 + Eino Phase 1-5B** 阶段。所有设计文档在 `docs/` 下，架构专题在 `docs/architecture/supervisor/` 下，实施状态见 `docs/implementation.md`。

实施按领域递增（八字 → 奇门 → 紫微），验收标准见 `docs/acceptance-criteria.md`。

**开始开发前必读：**
- `docs/architecture.md`（入口）和 `docs/architecture/supervisor/`（专题） — 架构总图、调用链路、ADR
- `docs/acceptance-criteria.md` — 当前模块的验收用例
- `docs/implementation.md` — 模块依赖关系


## 关键设计决策
1. 八字引擎用 `lunar-go`（开源成熟方案），不自研
2. 路由层用 Go ADK RouteEngine（三层防御：ADK structured → textDecide → safeFallback），不做 Python 推理层
3. 执行层用 ADK ChatModelAgent + AgentAsTool + Specialist Agent，Eino 承载路由和 Agent 运行时
4. RAG 通过 MCP 调本地知识库服务，不内嵌
5. SSE 6 种结构化事件（thinking/tool_call/component/text/error/done），前端按类型渲染
6. 后续统一入口采用 `LLM Supervisor + Go Runtime + bounded specialists`

## Agent 模块边界

```
Orchestrator → RouteAdvisor（Go ADK，路由决策）
  → Policy Gate（策略门控）
  → Preflight（确定性硬判断，可能短路返回澄清/缺资料）
  → Supervisor Agent（Go ADK，每轮动态构建）+ AgentTool specialists
  → AgentEventBridge → SSE 推送

Specialists: bazi / qimen / ziwei（各自挂载领域工具）
降级链: ADK structured → textDecide → fallbackExtract → safeFallback
```

## 质量门禁

以下为提交前必检项，不满足不得合入：
- **中文表达：** 面向用户的信息（SSE 事件文本、错误提示、UI 文案）必须使用中文，且语义完整、无歧义。
- **重要逻辑必注释：** 核心算法、状态机转换、架构设计点、非直观的边界处理，必须有注释说明「为什么这样做」。
- **注释格式：** 遵循 Go godoc 规范（详见下方「注释」章节）。
- **测试通过：** `go test ./...` 全部通过，新增功能必须有对应测试用例。
- **无残留调试代码：** 不得提交 `fmt.Println` / `console.log` 等调试输出。

## 编码规范

**想清楚再写。** 不确定就明确说出来，不要隐藏困惑。有多种解读时列出来，不要默默选一种。有更简单的方案就说。

**极简实现。** 只写被要求的功能。不为单次使用建抽象。不为不可能的场景加错误处理。200 行能缩成 50 行就重写。

**精准改动。** 只动必须改的代码。不顺手优化相邻代码、注释、格式。不重构没坏的东西。匹配已有代码风格。你改动造成的孤儿引用要清理，但不删除原本就存在的死代码。

**目标驱动。** 把任务变成可验证的目标。多步骤任务先列步骤 + 每步的验证方式。弱标准（"让它工作"）不如强标准（"测试通过"）。

## 注释

### 导出的必须写

每个导出标识符（函数、类型、常量）必须有 doc comment，以标识符名开头。写清楚做什么、什么情况下用、返回什么 error。

### 关键决策必须留痕迹

以下场景必须写注释说明**为什么这么做**——不留注释的决策等于随机行为，后来者会以为是 bug 或遗留代码：

- 架构选择（为什么用 A 模式而不是 B）
- 不直观的算法或性能优化（直接读代码猜不到意图）
- 边界处理（会 panic 的条件、并发约定、重试策略、超时原因）
- 对外部依赖的特殊处理（为什么这个 MCP 调用有 3 次重试上限）

### 不写

- 代码本身说得清的（命名已表达含义的）
- 注释掉的代码（用 git，代码库里不留尸体）
- `/* */` 做文档（go doc 不解析，写了白写）

| 技能 | 路径 | 用途 |
|------|------|------|
| `eino-guide` | `.claude/skills/eino-guide/SKILL.md` | Eino 框架概述、概念和导航 |
| `eino-component` | `.claude/skills/eino-component/SKILL.md` | Eino 组件选择、配置和使用（ChatModel/Tool/Embedding/Retriever 等） |
| `eino-compose` | `.claude/skills/eino-compose/SKILL.md` | Eino 编排：Graph、Chain、Workflow |
| `eino-agent` | `.claude/skills/eino-agent/SKILL.md` | Eino ADK Agent 构建、中间件、Runner |
| `agent-test-suites` | `.claude/skills/agent-test-suites/SKILL.md` | Agent 行为回归测试 |

**Claude Code 用户：** 以上技能通过 `Skill` 工具调用，技能名与目录名一致（如 `eino-agent`、`agent-test-suites`）。
**Codex 用户：** 请从对应路径加载技能文件作为指令上下文。

## 参考资源

**`eino-agent/`** — Eino 框架源码与示例。开发时参考：
- `eino-agent/eino-examples/quickstart/` — 快速入门示例
- `eino-agent/eino-examples/adk/` — Agent Development Kit 示例
- `eino-agent/eino-examples/compose/` — 编排示例 (Chain/Graph/Workflow)
- `eino-agent/eino-examples/flow/` — Flow 引擎示例
- `eino-agent/eino/adk/` — ADK 源码（Supervisor/PlanExecute/ReAct）
- `eino-agent/eino/components/` — 组件源码 (Tool/ChatModel)
- `eino-agent/eino/compose/` — 编排引擎源码

## 文档索引

| 文档 | 用途 |
|------|------|
| `docs/product.md` | 产品定义和功能范围 |
| `docs/architecture.md`（入口）和 `docs/architecture/supervisor/`（专题） | 架构总图、调用链路、容错、ADR |
| `docs/checklist-agent-engineering.md` | Agent 工程能力自检（43 项） |
| `docs/acceptance-criteria.md` | 验收标准 |
| `docs/implementation.md` | 实施总览和模块依赖 |
| `docs/data-flow.md` | 数据链路：用户消息 → AI 回答的完整调用链 |
