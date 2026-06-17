# Agent 工程概述 — 2025-2026 权威资源地图

## 定义

**Agent Engineering** 是构建包裹在 LLM 外面的工程系统（Harness）的学科。swyx 在 2025 AI Engineer Summit 上提出核心公式：

> **Agent = Model + Harness**（模型 + 线束）

模型提供推理能力，Harness 提供工具、记忆、规划、权限、沙箱、Hook、可观测性和恢复路径。

## 三次演进

| 时代 | 时期 | 焦点 |
|------|------|------|
| **Prompt Engineering** | 2022-2024 | 如何与模型对话 |
| **Context Engineering** | 2025 | 给模型什么信息 |
| **Harness Engineering** | 2026 | 围绕模型构建什么系统 |

## P0 必读文献

### Anthropic 官方

1. **[Building Effective Agents](https://www.anthropic.com/engineering/building-effective-agents)** (2024.12) — 6 种可组合设计模式，"从最简单方案开始"的金科玉律
2. **[Effective Harnesses for Long-Running Agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)** — 长运行 agent 的 Harness 实践
3. **[Demystifying Evals for AI Agents](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents)** — Agent 评估方法论

### OpenAI 官方

4. **[New Tools for Building Agents](https://openai.com/index/new-tools-for-building-agents/)** (2025)
5. **[Harness Engineering: Leveraging Codex](https://openai.com/index/harness-engineering/)**

### 行业权威

6. **[Agent Harness Engineering](https://addyosmani.com/blog/agent-harness-engineering/)** (Addy Osmani, 2026.04) — [O'Reilly 转载](https://www.oreilly.com/radar/agent-harness-engineering/)
7. **[Agent Engineering: Harness Patterns & IMPACT Framework](https://www.morphllm.com/agent-engineering)** (2026)
8. **[Build Better AI Agents: 5 Tips from Agent Bake-Off](https://developers.googleblog.com/build-better-ai-agents-5-developer-tips-from-the-agent-bake-off/)** (Google, 2026)

## GitHub 必看仓库

| 仓库 | 说明 |
|------|------|
| [agentic-engineering-handbook](https://github.com/keyuchen21/agentic-engineering-handbook) | 114 官方资源的结构化学习路线，6 阶段 |
| [awesome-agent-harness](https://github.com/AutoJunjie/awesome-agent-harness) | Agent Harness 精选资源 |
| [awesome-loop-engineering](https://github.com/ChaoYue0307/awesome-loop-engineering) | Loop Engineering 精选资源 |
| [loop-engineering](https://github.com/cobusgreyling/loop-engineering) | Loop Engineering 实践参考 |
| [better-agents](https://github.com/langwatch/better-agents) | 生产级 agent 构建标准 |
| [disciplined-agentic-engineering](https://github.com/swingerman/disciplined-agentic-engineering) | 验收测试驱动，反"vibe coding" |
| [Agentic_Engineering_Methodology](https://github.com/abbabiavati-bit/Agentic_Engineering_Methodology) | 7 阶段实战方法论 |

## 关键协议

| 协议 | 用途 |
|------|------|
| **MCP** (Model Context Protocol) | Agent 与工具/数据的通用连接器 |
| **A2A** (Agent-to-Agent) | Agent 间通信标准 |
| **Agent Skills Open Standard** | 可复用 agent 能力开放标准 (agentskills.io) |

## 核心教训

1. **好的 Harness 比好的模型更重要** — 换模型提升 20-30%，构建 Harness 提升 10 倍
2. **每次 Agent 犯错都应工程化解决**（Harness Ratchet 原则）
3. **以模块化心态构建** — 模型进步时立即废弃不再需要的 Harness 组件
