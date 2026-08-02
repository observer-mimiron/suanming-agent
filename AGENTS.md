# AGENTS.md

本文件是项目的**唯一 AI 编码助手指导文件**。Codex、Claude Code 及其他 AI 编码工具均以此文件为准。

## 进度维护规则

**PROGRESS.md 是本项目的当前事实快照。** 新对话先用它恢复当前状态；它不是变更日志、实施日记或历史方案合集。

**必须更新 PROGRESS.md 的时机：**
- 完成一个验收用例节点
- 做出会影响后续开发的架构变更
- 新增或修改环境变量/启动方式
- 遇到并解决了一个阻塞性问题（记录原因和解决方案）

**更新方式：** 先替换过期项，不追加同义历史。只保留当前阶段、已验证事实、未解决阻塞、下一步、最小入口/验证命令和仍有效的关键决策。历史过程查 Git、`eval/reports/` 或专项设计文档。

**长度约束：** 通常控制在 80-120 行；超过时先合并或删除历史性描述。架构事实仍以 `docs/architecture.md` 为准。

**会话开始时：** 先读取 PROGRESS.md 了解当前进度，再读对应模块的实施文档。

## 项目概述

命理大师 — AI 八字算命 Agent 聊天应用。学习项目，覆盖 Multi-Agent、RAG、Plan-Execute、SSE 等企业 Agent 开发技巧。

## 架构核心

**全 Go 原生，两层 "supervisor" 各司其职：**

1. **RouteAdvisor**（`backend/internal/supervisor/`，Go ADK RouteEngine）— 路由决策。三层防御（ADK structured → textDecide → safeFallback），产出 `SupervisorDecision`（L0 对话意图 → L1 领域 → L2 任务 → L3 槽位）。
2. **Manager**（`backend/internal/runtime/`）— 执行主控。将 `ApprovedRoute` 转为 `ExecutionPlan`，绑定精确资产并调度受限领域 runner；它是 runtime 内唯一的对话 owner。

```
Vue 3 → SSE → Gin (:8080)
                │
         Eino ADK / lunar-go / MCP→RAG (:3100)
```

**架构单一事实来源：** `docs/architecture.md`。任何架构决策变更必须先更新该文档。

## 三服务启动

可使用 `make dev` 一键启动全部三个服务（推荐），或分别启动：
make knowledge-start      # 仅知识库
make dev-backend          # 仅后端
make dev-frontend         # 仅前端

```bash
# 知识库 (Next.js :3100)
cd knowledge && npx next dev -p 3100

# 执行层 (Go :8080)
LLM_API_KEY=sk-xxx go run ./backend/cmd/server/

# 前端 (Vue :5173)
cd web && npm run dev
```

### Docker / WSL

- 本机已实测：`docker` / `docker compose` 在 **WSL2 Ubuntu** 中可用，PowerShell 会话里不保证存在 `docker` 命令。
- 需要跑 `deploy/` 下的 Docker Compose 时，优先使用 WSL 路径执行，例如：
  当前默认是本地开发/演示用途，不是线上正式部署路径。

```bash
wsl -e sh -lc "cd /home/huang/workspace/suanming-agent/deploy/app && docker compose up -d --build"
```

- `deploy/app/` 是当前默认的本地 Docker 入口：启动 `app`（Go 后端 + 内嵌前端构建产物）和 `knowledge`；`deploy/langfuse/` 仍是可选观测栈。
- `deploy/app` 中的知识库目录采用**仓库直挂载**：`knowledge/wiki/` -> `/app/wiki`、`knowledge/raw/` -> `/app/raw`。本地通过知识库服务新增或修改的知识，默认直接落回仓库文件；不再走 seed 到 volume 的单向复制。

**端口：** 知识库 :3100，后端默认 :8080（可通过 `LISTEN_ADDR` 环境变量修改），前端 :5173。
环境变量：`LLM_API_KEY`（必填）、`LLM_BASE_URL`、`LLM_MODEL`、`KNOWLEDGE_MCP_URL`、`EMBEDDING_API_KEY`/`EMBEDDING_BASE_URL`/`EMBEDDING_MODEL`（semantic router，默认 DashScope `text-embedding-v4`）、`ROUTER_MODE`（`off`/`shadow`/`enforce`，默认 `off`）。

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
## 开发命令

```bash
# Go
go build ./backend/cmd/server/          # 编译
go test ./backend/... -v                # 全部测试
go test ./backend/internal/tools/ -v    # 单个包测试

# Vue
cd web && npm run dev           # 开发
npx vue-tsc --noEmit            # 类型检查
npm run build                   # 构建
```

## 代码查阅

项目已初始化 `.codegraph/` 索引。查代码优先用 `codegraph_*` MCP 工具（`codegraph_context` / `codegraph_search` / `codegraph_trace` / `codegraph_callers` / `codegraph_callees`），比 grep/find 更快且结构化。找不到再 fallback 到文件系统。

## 实施状态

项目处于 **v1.5 收口 + Eino 迁移完成** 阶段，orchestrationGraph 已上线运行。所有设计文档在 `docs/` 下，当前架构总览以 `docs/architecture.md` 为准，实施状态见 `PROGRESS.md`。

实施按领域递增（八字 → 奇门 → 紫微），验收标准见 `docs/acceptance-criteria.md`。

**开始开发前必读：**
- `docs/architecture.md` — 架构总图、调用链路、关键边界
- `docs/acceptance-criteria.md` — 当前模块的验收用例


## 关键设计决策
1. 八字引擎用 `lunar-go`（开源成熟方案），不自研
2. 路由层用 Go ADK RouteEngine（三层防御：ADK structured → textDecide → safeFallback），不做 Python 推理层
3. 执行层由 Manager 生成 `ExecutionPlan` 并调度 bounded specialist runners；Eino 承载路由和 Agent 运行时
4. RAG 通过 MCP 调本地知识库服务，不内嵌
5. SSE 6 种结构化事件（thinking/tool_call/component/text/error/done），前端按类型渲染
6. 后续统一入口采用 `LLM Supervisor + Go Runtime + bounded specialists`

## Agent 模块边界

```
Orchestrator → RouteAdvisor（Go ADK，路由决策）
  → Policy Gate（策略门控）
  → Manager（对象/资产解析、ExecutionPlan、最终 compose）
  → Preflight / Prefill / ToolRunner（确定性准备与硬判断）
  → bounded specialist runners（领域执行）
  → final guard（最终合同保护）
  → AgentEventBridge → SSE 推送
Specialists: bazi / qimen / ziwei（各自挂载领域工具，不拥有最终答复权）
降级链: ADK structured → textDecide → fallbackExtract → safeFallback
```

## 质量门禁

以下为提交前必检项，不满足不得合入：
- **中文表达：** 面向用户的信息（SSE 事件文本、错误提示、UI 文案）必须使用中文，且语义完整、无歧义。
- **重要逻辑必注释：** 核心算法、状态机转换、架构设计点、非直观的边界处理，必须有注释说明「为什么这样做」。
- **注释格式：** 遵循 Go godoc 规范（详见下方「注释」章节）。
- **测试通过：** `go test ./backend/...` 全部通过，新增功能必须有对应测试用例。
- **无残留调试代码：** 不得提交 `fmt.Println` / `console.log` 等调试输出。

## 编码规范

**想清楚再写。** 不确定就明确说出来，不要隐藏困惑。有多种解读时列出来，不要默默选一种。有更简单的方案就说。

**极简实现。** 只写被要求的功能。不为单次使用建抽象。不为不可能的场景加错误处理。200 行能缩成 50 行就重写。

**精准改动。** 只动必须改的代码。不顺手优化相邻代码、注释、格式。不重构没坏的东西。匹配已有代码风格。你改动造成的孤儿引用要清理，但不删除原本就存在的死代码。

**目标驱动。** 把任务变成可验证的目标。多步骤任务先列步骤 + 每步的验证方式。弱标准（"让它工作"）不如强标准（"测试通过"）。

**禁止专项 case 补丁。** 不得为单个命盘、trace、用户样例追加专用分支或不断扩张自然语言禁词表。命理输出问题优先按合同修复：事实可复算、关系已声明、profile 有来源、renderer 只展示结构化字段。单 case 只能进入回归测试，不能成为业务逻辑。

## 注释

### 目标

注释要同时服务人类维护者和 AI 编码助手：读文件先知道职责，读函数先知道合同，读核心逻辑能知道为什么这样做。能被 `go doc` 抽取的注释，本身就是项目文档的一部分。

### 分层规则

1. **文件头职责注释：** 每个非平凡源码文件顶部必须写 2-5 行职责说明，说明该文件属于哪一层、负责什么、不负责什么；架构约束文件要写成“宪法级”规则，明确不可破坏的边界。
2. **类型与接口注释：** 说明它表达的领域概念、生命周期、所有权或并发约定；不要只复述字段名。
3. **函数注释：** 每个函数都写一句简单注释；导出函数必须用 Go doc comment（Go 文档注释）格式，以函数名开头，并说明输入语义、返回结果和主要 error 条件。
4. **核心逻辑注释：** 对状态机、路由决策、合同校验、降级策略、缓存/并发、外部依赖特殊处理，必须解释“为什么这么做”和“改坏会发生什么”。
5. **局部注释：** 只在代码本身无法说明意图时写，优先解释业务约束、边界条件、历史兼容原因和安全原因。

### 文件头模板

```go
// Package runtime contains the manager-owned execution flow.
//
// This file owns ExecutionPlan construction after route approval. It must not
// call specialist LLMs directly; specialists are scheduled through bounded runners.
```

文件头不是作者、日期、变更流水，也不是整份设计文档。设计背景超过 5 行时，写到 `docs/architecture.md`，文件头只保留职责和边界。

### 函数即文档

函数注释要让读者不展开实现也能知道能不能调用：

- 做什么：函数完成的业务动作或合同判断。
- 何时用：调用前置条件、所属阶段、是否只用于 fallback / migration。
- 返回什么：关键返回值含义，特别是 nil、空集合、degraded、retryable。
- 失败怎样：主要 error 条件、是否可重试、是否会产生用户可见错误。

非导出的小函数也要有一句短注释；如果一句话写不清，优先拆函数或重命名，再补注释。

### 导出的必须写

每个导出标识符（函数、类型、常量）必须有 doc comment，以标识符名开头。写清楚做什么、什么情况下用、返回什么 error。导出结构体字段如果不是自解释字段，也要在类型注释或字段旁说明含义。

### 关键决策必须留痕迹

以下场景必须写注释说明**为什么这么做**——不留注释的决策等于随机行为，后来者会以为是 bug 或遗留代码：

- 架构选择（为什么用 A 模式而不是 B）
- 不直观的算法或性能优化（直接读代码猜不到意图）
- 边界处理（会 panic 的条件、并发约定、重试策略、超时原因）
- 对外部依赖的特殊处理（为什么这个 MCP 调用有 3 次重试上限）
- 合同保护（为什么拒绝某类输出、为什么只能降级不能猜测）
- AI 协作边界（哪些语义只能在 planner / manager / renderer 的某一层处理）

### 不写

- 代码本身说得清的（命名已表达含义的）
- 逐行翻译实现的注释（例如“遍历数组”“设置变量”）
- 过期背景、TODO 感叹、临时猜测和没有 owner 的承诺
- 注释掉的代码（用 git，代码库里不留尸体）
- `/* */` 做文档（go doc 不解析，写了白写）

| 技能 | 路径 | 用途 |
|------|------|------|
| `eino-guide` | `.claude/skills/eino-guide/SKILL.md` | Eino 框架概述、概念和导航 |
| `eino-component` | `.claude/skills/eino-component/SKILL.md` | Eino 组件选择、配置和使用（ChatModel/Tool/Embedding/Retriever 等） |
| `eino-compose` | `.claude/skills/eino-compose/SKILL.md` | Eino 编排：Graph、Chain、Workflow |
| `eino-agent` | `.claude/skills/eino-agent/SKILL.md` | Eino ADK Agent 构建、中间件、Runner |

**Claude Code 用户：** 以上技能通过 `Skill` 工具调用，技能名与目录名一致（如 `eino-agent`）。
**Codex 用户：** 请从对应路径加载技能文件作为指令上下文。

## 参考资源

**`../eino-agent/`** — Eino 框架源码与示例。开发时参考：
- `../eino-agent/eino-examples/quickstart/` — 快速入门示例
- `../eino-agent/eino-examples/adk/` — Agent Development Kit 示例
- `../eino-agent/eino-examples/compose/` — 编排示例 (Chain/Graph/Workflow)
- `../eino-agent/eino-examples/flow/` — Flow 引擎示例
- `../eino-agent/eino/adk/` — ADK 源码（Supervisor/PlanExecute/ReAct）
- `../eino-agent/eino/components/` — 组件源码 (Tool/ChatModel)
- `../eino-agent/eino/compose/` — 编排引擎源码

## 文档索引

| 文档 | 用途 |
|------|------|
| `docs/architecture.md` | 架构总图、调用链路、关键边界 |
| `docs/acceptance-criteria.md` | 验收标准 |
| `docs/data-flow.md` | 数据链路：用户消息 → AI 回答的完整调用链 |
| `docs/glossary.md` | 名词对照表，术语统一 |
| `PROGRESS.md` | 实施进度、决策记录、上下文恢复 |
