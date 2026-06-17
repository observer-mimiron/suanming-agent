# Agent 架构模式 — Anthropic/OpenAI 生产实践收敛

## 行业收敛

Anthropic 和 OpenAI 在 2025-2026 年独立得出相同核心架构结论。

## Three-Agent 模式

Anthropic、Stripe、OpenAI 独立收敛到同一架构：

| 角色 | 职责 |
|------|------|
| **Planner** | 任务分解 + 验收标准，代码生成前进行 |
| **Generator** | 履行合约 — 写代码、执行工具、维护状态 |
| **Evaluator** | 独立评判输出，使用测试/工具/遥测 |

> GAN 类比：生成器与评估器处于"生产性张力"。让同一 agent 自评 → "agent 自信赞美自己平庸的工作"。评估者分离是**结构性**问题，即使模型再进步也应保留。

## 六种可组合设计模式（Anthropic）

所有模式构建在 **Augmented LLM**（LLM + 检索 + 工具 + 记忆）之上：

### 1. Prompt Chaining
顺序步骤 + 程序化门控。适用：任务可分解为固定子任务，准确性 > 速度。

### 2. Routing
分类输入 → 专门处理器。适用：不同类别需要不同处理。

### 3. Parallelization
- **分段**: 独立并行子任务（速度）
- **投票**: 同任务多次不同方法，聚合结果（置信度）

### 4. Orchestrator-Workers
中央 LLM 动态分解 + 委派 + 合成。适用：子任务无法预知。

### 5. Evaluator-Optimizer
生成→评估→反馈循环，达到质量阈值。适用：有清晰评估标准。

### 6. Autonomous Agent
自主规划执行 + 可选人工检查点。风险：高成本、错误累积。需要沙箱+护栏。

### 核心原则

> **单次 API 调用能搞定的，不要上 Agent。从最简单的方案开始。**

## 四大生产模式

### 1. 沙箱边界
- Anthropic: Harness 在沙箱**内**
- OpenAI: Harness 与计算**分离**

### 2. 结构约束 > 指令约束
ESLint 规则禁止坏模式 > prompt 说"请遵循最佳实践"

### 3. Context Reset > Context Compaction
清空上下文 + 文件产物交接 > 总结追加

### 4. 更少工具 = 更好性能
OpenAI + Stripe 独立发现：减少 ~80% 工具提升质量

## 上下文工程五策略

| 策略 | 描述 |
|------|------|
| CLAUDE.md/AGENTS.md | ≤60 行，每会话加载 |
| JIT 检索 | 轻量引用，按需加载 |
| Compaction | 近限制时总结 + git commit 重建状态 |
| 子 Agent 隔离 | 各自干净上下文窗口 |
| 文件系统扩展 | 大型观察卸载到沙箱，上下文仅保留引用 |

## Agent Loop 生产模式

```
while task_not_complete:
    state = read_files() + read_test_output() + read_errors()
    context = harness.select_context(state, task)
    plan = model.reason(context, task)
    result = harness.dispatch_tool(plan.next_action)
    outcome = harness.evaluate(result)

    if outcome.needs_retry:   → 带错误追踪重试
    if outcome.needs_human:   → 升级
    harness.checkpoint(result)  # git commit
```

Claude Code、Cursor、Codex、Cline、Aider 区别不在 loop，在于 Harness 如何管理每一步。

## 架构永续性（Google Bake-Off 2026）

> "以永续心态构建。今天的复杂 harness 可能几周内被模型进步取代。"

每个 Harness 组件编码了"模型不能独立做什么"的假设。模型进步时移除对应组件。

## Anthropic vs OpenAI

| 维度 | Anthropic | OpenAI |
|------|-----------|--------|
| 策略 | 开发者向上 | 企业向下 |
| SDK 模型 | 沙箱内 | 分离 |
| 多 Agent | Opus 编排 + Sonnet 执行 | `agent.as_tool()` / `handoffs` |
| 标准 | MCP | AGENTS.md |
| 托管 | Managed Agents | Frontier |

## 生产验证

| 团队 | 成果 |
|------|------|
| Anthropic 内部 | ~100 万行代码，~1,500 PR，3 人运营 |
| Stripe "Minions" | 1,300+ PR/周，5 层流水线 |
| OpenAI Codex | 5 月 100 万行生产代码，零手写 |

## 必读资源

- [Building Effective Agents](https://www.anthropic.com/engineering/building-effective-agents) (Anthropic)
- [Effective Harnesses](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents) (Anthropic)
- [Agentic Engineering Handbook](https://github.com/keyuchen21/agentic-engineering-handbook)
- [Three Teams One Pattern](https://dev.to/kuro_agent/three-teams-one-pattern-what-anthropic-stripe-and-openai-discovered-about-ai-agent-b53)
- [Disciplined Agentic Engineering](https://github.com/swingerman/disciplined-agentic-engineering)
- [Agent Spec](https://github.com/Superfleys/agent-spec)
- [Best Practices (InfoWorld)](https://www.infoworld.com/article/4154570/best-practices-for-building-agentic-systems.html)
