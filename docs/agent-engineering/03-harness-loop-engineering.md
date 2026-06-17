# Harness Engineering 与 Loop Engineering — 2026 核心范式

## Harness Engineering

**核心公式**: `Agent = Model + Harness`

### 十层标准栈

| 层 | 组件 | 说明 |
|----|------|------|
| 1 | System Prompt & Skills | 渐进式加载，规则追溯到历史失败 |
| 2 | Tool Registry | MCP servers；10 聚焦工具 > 50 重叠工具 |
| 3 | Sandbox & Execution | 容器/浏览器/文件系统边界 |
| 4 | Permission Model | 最小权限 + 人工审批检查点 |
| 5 | Memory & State | 上下文内→草稿→情景→语义→程序 |
| 6 | Context Management | 压缩、重置+产物交接、渐进加载 |
| 7 | Sub-agent Orchestration | 大模型规划 + 小模型执行 |
| 8 | Hooks & Middleware | 确定性执行包围非确定性模型调用 |
| 9 | Observability | 日志、指标、分布式追踪 |
| 10 | Eval Loop | 与生成 agent 分离的独立评估 |

### Harness Ratchet 原则（Addy Osmani）

> "Every time an agent makes a mistake, engineer a solution so it never makes that mistake again."

AGENTS.md 每条规则都应追溯到具体历史失败。规则是挣来的，不是拍脑袋想的。

### IMPACT 框架（swyx, 2025）

| 组件 | 描述 |
|------|------|
| **I** - Intent | 目标编码 + eval 验证 |
| **M** - Memory | Skill 库 + 可复用模式 |
| **P** - Planning | 多步骤**可编辑**计划 |
| **A** - Authority | 权限模型 + 审批门控 |
| **C** - Control Flow | LLM 动态执行路径（区分 agent vs 工作流） |
| **T** - Tools | RAG、搜索、沙箱执行、浏览器自动化 |

## Loop Engineering（2026.06）

从"人驱动 Agent"到"系统驱动 Agent"。

- **Boris Cherny** (Claude Code): "I don't prompt Claude anymore. I have loops running. My job is to write loops."
- **Peter Steinberger** (OpenClaw): "You should be designing loops."

### Loop 六大要素

| 要素 | 作用 |
|------|------|
| Automations | 心跳触发器，定时/事件驱动 |
| Worktrees | 独立工作目录隔离 |
| Skills | 项目约定沉淀为可复用文件 |
| Connectors/MCP | 接入真实工具（GitHub、DB、Slack） |
| Sub-Agents | 生产者+检查者分离 |
| Memory | 状态写磁盘，跨会话持久 |

### Loop 成熟度模型

| L0 | L1 | L2 | L3 | L4 | L5 | L6 |
|----|----|----|----|----|----|----|
| 手动提示 | 脚本重试 | 定时循环 | 有状态循环 | 自验证循环 | 多Agent循环 | 生产监督循环 |

## 九大生产实践（Mindflow 2025-2026）

1. **工具优先设计** — 先定义有界操作再接入 MCP
2. **纯函数工具调用** — 无状态、幂等、可预测
3. **单一职责 Agent** — 可调试性 > 优雅性
4. **工作流分解** — 显式错误边界
5. **外部化 prompt 管理** — 版本化配置，与代码分离
6. **模型联盟设计** — 快/便宜路由 + 推理优化决策
7. **工作流/MCP 分离** — 编排不含工具实现细节
8. **第一天容器化** — 无状态水平扩展
9. **KISS 作为运营纪律** — 简单不可妥协

## 错误恢复层级

1. 带上下文重试
2. 回滚到检查点（git reset）
3. 分解为更小子任务
4. 升级给人工

## 三大风险（Addy Osmani）

1. **验证仍是你的责任** — "完成了"只是声明，不是证明
2. **理解债务滚雪球** — 产出越快，理解越少
3. **认知投降** — 别变成只会按"开始键"的人

## 必读资源

- [Loop Engineering 橙皮书](https://github.com/alchaincyf/loop-engineering-orange-book)
- [loop-engineering](https://github.com/cobusgreyling/loop-engineering)
- [awesome-loop-engineering](https://github.com/ChaoYue0307/awesome-loop-engineering)
- [awesome-agent-harness](https://github.com/AutoJunjie/awesome-agent-harness)
- [Agentic Engineering Methodology](https://github.com/abbabiavati-bit/Agentic_Engineering_Methodology)
- [9 Engineering Practices That Actually Work](https://mindflow.io/blog/the-production-ai-agent-reality-check-9-engineering-practices-that-actually-work)
- [innobu Harness Engineering](https://www.innobu.com/en/agentic-harness-engineering.html)
