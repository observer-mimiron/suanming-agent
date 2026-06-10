# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## 进度维护规则

**PROGRESS.md 是本项目的上下文恢复文件。** 其作用是：在新对话中，将 PROGRESS.md 的内容作为上下文输入给任何 AI 编码助手，即可独立继续开发。

**必须更新 PROGRESS.md 的时机：**
- 完成一个模块（M0-M6 任一）
- 完成一个验收用例节点
- 做出会影响后续开发的架构变更
- 新增或修改环境变量/启动方式
- 遇到并解决了一个阻塞性问题（记录原因和解决方案）

**更新内容：** 更新进度总览表的状态、完成日期；如果正在进行的任务变了，更新「正在进行的任务」；如果做出了新决策，追加到「关键决策记录」。

**会话开始时：** 先读取 PROGRESS.md 了解当前进度，再读对应模块的实施文档。

## 项目概述

命理大师 — AI 八字算命 Agent 聊天应用。学习项目，覆盖 Multi-Agent、RAG、Plan-Execute、SSE 等企业 Agent 开发技巧。

## 架构核心

**推理与执行分离：** LangGraph (Python) 管有状态推理，Go 管高性能工具执行，HTTP 通信。

```
Vue 3 → SSE → Gin (:8080) → HTTP → LangGraph (:8000)
                   │
            lunar-go / MCP→RAG / Codex
```

**架构单一事实来源：** `docs/architecture.md`。任何架构决策变更必须先更新该文档。

## 三服务启动

```bash
# 推理层 (Python :8000)
source reasoning/venv/bin/activate
LLM_API_KEY=sk-xxx uvicorn reasoning.server:app --port 8000 --reload

# 执行层 (Go :8080)
LLM_API_KEY=sk-xxx go run ./cmd/server/

# 前端 (Vue :5173)
cd web && npm run dev
```

环境变量：`LLM_API_KEY`（必填）、`LLM_BASE_URL`、`LLM_MODEL`、`LANGRAPH_URL`、`RAG_MCP_URL`。

## 开发命令

```bash
# Go
go build ./cmd/server/          # 编译
go test ./... -v                # 全部测试
go test ./internal/tools/ -v    # 单个包测试

# Python
cd reasoning && source venv/bin/activate
pip install -r requirements.txt
python -m pytest . -v

# Vue
cd web && npm run dev           # 开发
npx vue-tsc --noEmit            # 类型检查
npm run build                   # 构建
```

## 实施状态

项目处于**设计完成、待实施**阶段。所有设计文档在 `docs/` 下，实施方案在 `docs/implementation/` 下。

实施按 M0→M6 顺序，每模块独立可验证。验收标准见 `docs/acceptance-criteria.md`（42 个用例）。

**开始开发前必读：**
- `docs/architecture.md` — 架构总图、调用链路、ADR
- `docs/acceptance-criteria.md` — 当前模块的验收用例
- `docs/implementation.md` — 模块依赖关系
- `docs/implementation/m<N>-*.md` — 当前模块的详细步骤

## 关键设计决策

1. 八字引擎用 `lunar-go`（开源成熟方案），不自研
2. 推理层用 LangGraph StateGraph（条件边 + TypedState），不用 Eino 的编排能力
3. Go 只做工具执行，Eino 只用 Tool 接口 + ChatModel + Session Memory 三个底层组件
4. RAG 通过 MCP 调本地服务，不内嵌
5. SSE 5 种结构化事件（thinking/tool_call/component/text/done），前端按类型渲染

## Agent 模块边界

```
Go Orchestrator → LangGraph POST /reason → 返回 action
  action=ask  → SSE text 追问 → 等待下一轮
  action=plan → 按序调 Tool → SSE component/text → done

Tool 注册: bazi_calc / rag_search / llm_generate
LangGraph 降级: 不可用时跳过推理层，直接 llm_generate
```

## 文档索引

| 文档 | 用途 |
|------|------|
| `docs/product.md` | 产品定义和功能范围 |
| `docs/architecture.md` | 架构总图、调用链路、容错、ADR |
| `docs/tech-backend.md` | Go 执行层技术细节 |
| `docs/tech-reasoning.md` | LangGraph 推理层技术细节 |
| `docs/tech-frontend.md` | Vue 前端技术细节 |
| `docs/checklist-agent-engineering.md` | Agent 工程能力自检（43 项） |
| `docs/acceptance-criteria.md` | 验收标准（42 用例） |
| `docs/implementation.md` | 实施总览和模块依赖 |
| `docs/implementation/m0-*.md` ~ `m6-*.md` | 各模块详细步骤 |
