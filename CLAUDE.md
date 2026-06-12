# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 进度维护

**PROGRESS.md 是本项目的上下文恢复文件。** 新对话中将 PROGRESS.md 内容作为上下文输入，即可独立继续开发。

**必须更新 PROGRESS.md 的时机：** 完成模块、完成验收节点、架构变更、环境变量变更、解决阻塞性问题。

**会话开始时：** 先读 PROGRESS.md，再读对应模块实施文档。

## 项目概述

命理大师 — AI 八字命理咨询聊天应用。v1 主线用 Go/Eino 单栈实现。

## 架构

```
Vue 3 → SSE → Gin (:8080) → Session State → Tools → lunar-go / MCP / Claude
```

**Go 单一后端。** 会话状态、工具执行、SSE 推送全部在 Go。架构文档见 `docs/architecture.md`。

## 启动

```bash
# 启动知识库（先启动，端口 3100）
make knowledge-start

# Go 后端 (:8080)
LLM_API_KEY=sk-xxx go run ./cmd/server/

# Vue 前端 (:5173)
cd web && npm run dev
```

环境变量：`LLM_API_KEY`（必填）、`LLM_BASE_URL`、`LLM_MODEL`、`KNOWLEDGE_MCP_URL`（默认 http://localhost:3100）。

## 知识库

独立知识库实例运行在端口 3100（数据目录 `knowledge/wiki/`），与 lisense 知识库（端口 3000）完全隔离。已导入 19 个页面，涵盖古籍原文、八字基础、格局用神等模块，含权威分级和交叉引用。

| 命令 | 用途 |
|------|------|
| `make knowledge-start` | 启动知识库 |
| `make knowledge-stop` | 停止知识库 |
| `make knowledge-status` | 查看状态 |
| `make knowledge-import` | 重新导入资料 |

Go 后端的 `knowledge_search` 工具自动连接 `KNOWLEDGE_MCP_URL`（默认 :3100），检索知识库返回命理资料注入到 LLM 解读中。

## 开发命令

```bash
go build ./cmd/server/          # 编译
go test ./... -v                # 全部测试
go test ./internal/tools/ -v    # 单个包测试
cd web && npx vue-tsc --noEmit  # 前端类型检查
cd web && npm run build         # 前端构建
```

## 代码查阅

项目已初始化 `.codegraph/` 索引。查代码优先用 `codegraph_*` MCP 工具（`codegraph_context` / `codegraph_search` / `codegraph_trace` / `codegraph_callers` / `codegraph_callees`），比 grep/find 更快且结构化。找不到再 fallback 到文件系统。

## 实施

项目处于**设计完成、待实施**阶段。实施按 M0→M1→M3→M4→M5→M6 顺序。

**开发前必读：**
- `docs/architecture.md` — 架构单一事实来源
- `docs/v1/acceptance-criteria.md` — 验收用例
- `docs/v1/implementation/<module>.md` — 模块实施步骤

## 关键决策

1. v1 主线 Go/Eino 单栈，LangGraph 延后到 v2 对照版
2. 八字引擎用 `lunar-go`，不自研
3. 会话状态在 Go 侧管理，确定性状态机
4. 知识检索走项目知识库 MCP
5. SSE 6 种事件：thinking / tool_call / component / text / error / done
6. 不展示原始 CoT，只展示结构化推理过程

## 编码规范

来源于 Karpathy Guidelines，补充系统提示。

**想清楚再写。** 不确定就明确说出来，不要隐藏困惑。有多种解读时列出来，不要默默选一种。有更简单的方案就说。

**极简实现。** 只写被要求的功能。不为单次使用建抽象。不为不可能的场景加错误处理。200 行能缩成 50 行就重写。

**精准改动。** 只动必须改的代码。不顺手优化相邻代码、注释、格式。不重构没坏的东西。匹配已有代码风格。你改动造成的孤儿引用要清理，但不删除原本就存在的死代码。

**目标驱动。** 把任务变成可验证的目标。多步骤任务先列步骤 + 每步的验证方式。弱标准（"让它工作"）不如强标准（"测试通过"）。

## 参考资源

**`eino-agent/`** — Eino 框架源码与示例。开发时参考：
- `eino-agent/eino-examples/quickstart/` — 快速入门示例
- `eino-agent/eino-examples/adk/` — Agent Development Kit 示例
- `eino-agent/eino-examples/compose/` — 编排示例 (Chain/Graph/Workflow)
- `eino-agent/eino-examples/flow/` — Flow 引擎示例
- `eino-agent/eino/adk/` — ADK 源码（Supervisor/PlanExecute/ReAct）
- `eino-agent/eino/components/` — 组件源码 (Tool/ChatModel)
- `eino-agent/eino/compose/` — 编排引擎源码

## 文档

| 文档 | 用途 |
|------|------|
| `docs/product.md` | 产品定义 |
| `docs/architecture.md` | 架构总图、状态机、ADR |
| `docs/v1/tech-backend.md` | Go 后端技术细节 |
| `docs/v1/tech-frontend.md` | Vue 前端技术细节 |
| `docs/v1/checklist-agent-engineering.md` | Agent 工程自检 |
| `docs/v1/acceptance-criteria.md` | 验收标准 |
| `docs/v1/implementation.md` | 实施总览 |
| `docs/learning/learning-roadmap.md` | Agent 学习路线 |
| `docs/v2/` | v2 LangGraph 对照版 |
