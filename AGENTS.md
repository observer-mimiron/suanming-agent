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

## 默认协作模式

本项目默认采用**平衡模式**：

- 对**单文件文档改写**、**README/注释整理**、**轻量机械改动**、**局部格式收口**这类低风险任务，可优先委托给其他 AI 先出第一稿。
- Codex 在这类任务中默认只做**最小必要读取**，避免先大范围阅读仓库上下文。
- 其他 AI 完成后，Codex 主要负责**轻量 review**：检查目标文件 diff、抽查关键段落或关键函数、执行最窄验证命令，并在必要时做小范围收口修正。
- 若任务已经明确为“只改一个文件”或“只收口现有文档”，默认不要扩展为 repo 级探索。

以下任务**不默认下放**，除非用户明确要求：

- 跨多文件的逻辑改动
- 需要理解现有业务流后才能判断的 bug
- 架构设计、边界调整、协议变更
- 高风险发布项、迁移项、状态一致性问题
- 任何需要 Codex 自己做完整判断和端到端验证的任务

原则：**简单任务先委托，复杂任务由 Codex 主做；能轻审就不深审，但发现边界、事实或行为风险后，Codex 必须接手收口。**

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
- **注释格式：** 遵循 Go godoc 规范（详见下方「注释规范」章节）。
- **测试通过：** `go test ./...` 全部通过，新增功能必须有对应测试用例。
- **无残留调试代码：** 不得提交 `fmt.Println` / `console.log` 等调试输出。

## 编码规范

**想清楚再写。** 不确定就明确说出来，不要隐藏困惑。有多种解读时列出来，不要默默选一种。有更简单的方案就说。

**极简实现。** 只写被要求的功能。不为单次使用建抽象。不为不可能的场景加错误处理。200 行能缩成 50 行就重写。

**精准改动。** 只动必须改的代码。不顺手优化相邻代码、注释、格式。不重构没坏的东西。匹配已有代码风格。你改动造成的孤儿引用要清理，但不删除原本就存在的死代码。

**目标驱动。** 把任务变成可验证的目标。多步骤任务先列步骤 + 每步的验证方式。弱标准（"让它工作"）不如强标准（"测试通过"）。

## 注释规范

遵循 Go 官方 godoc 标准，注释即文档。

### 总则

- 使用 `//` 单行注释，不使用 `/* */` 块注释。
- 注释必须紧邻声明上方，中间不留空行。
- 导出标识符（首字母大写）必须有注释，注释以标识符名开头。
- 注释用完整中文或英文句子，结尾带句号。首句为摘要（≤80 字符为宜）。

### 包注释

每个包必须有包注释，建议放在 `doc.go` 文件中：

```go
// Package orchestrator 实现命理咨询的对话编排逻辑。
//
// 负责管理会话状态机、调度工具执行、生成 SSE 事件流。
// 支持多轮对话上下文保持和 supervisor 三层降级。
package orchestrator
```

### 函数/方法注释

```go
// CalculateBazi 根据出生时间计算八字四柱。
//
// 参数 birthTime 为公历时间，会自动转换为农历后起卦。
// 返回的四柱包含年柱、月柱、日柱、时柱及对应的天干地支。
// 若输入时间早于 1900 年，返回 ErrUnsupportedYear。
func CalculateBazi(birthTime time.Time) (*SiZhu, error) {
```

**要点：**
- 以函数名开头，描述做什么（而非怎么做）。
- 说明参数约束、非显而易见的副作用、可能 panic 或返回 error 的条件。
- 并发安全的函数需标注「此函数可并发调用」。

### 类型/接口注释

```go
// SessionState 表示会话的确定性状态机。
//
// 状态转换：Idle → Asking → Planning → Executing → Done
// 所有转换通过 Transition 方法原子完成，可并发访问。
type SessionState struct {
```

### 架构与重要逻辑注释

以下场景必须有详细注释，说明「为什么」而非「是什么」：

- **架构设计点：** 为什么选择这个模式（如「用状态机而非 if-else 链，因为后续有 7 种状态扩展」）。
- **非直观算法：** 如八字排盘中的节气计算、天干地支转换，需注明参考的历法规则。
- **容错与降级：** 如 supervisor route engine 不可用时经三层降级（ADK → textDecide → safeFallback），注释说明降级触发条件和恢复方式。
- **并发边界：** 共享状态的锁策略、channel 关闭时机、goroutine 生命周期。
- **外部依赖调用：** MCP 调用、LLM API 调用的重试策略和超时原因。

### 禁止的注释

| 类型 | 反例 | 原因 |
|------|------|------|
| 废话注释 | `// increment i by 1` 跟着 `i++` | 代码自明 |
| 注释掉的代码 | `// oldFunc()` | 用 git 管理历史 |
| 日期署名 | `// 2024-01-01 by zhangsan` | git blame 可追溯 |
| 情绪化注释 | `// 这里很 hacky，没办法` | 应改为说明为什么必须 hacky |

## 项目技能

本项目的 AI 技能定义在 `.claude/skills/` 目录下。所有 AI 编码助手启动时应自动加载该目录中的技能文件。

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
