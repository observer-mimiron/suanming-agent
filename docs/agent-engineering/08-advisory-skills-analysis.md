# Agent 工程顾问型 Skill — 已有方案分析

> 这类 skill 的角色是**方法论顾问**——提供决策框架、检查清单、最佳实践，而不是自动生成一切。人工始终掌握决策权。

## 最值得关注的已有方案

### 1. agentic-product-standard（最全面）
- **仓库**: https://github.com/AlexDuchDev/agentic-product-standard
- **核心内容**:
  - 自主性阶梯 (L0-L4)：只有 ≥90% pass rate 才能升级
  - 7 层 harness 架构
  - 12 点生产就绪检查清单
  - 反模式审查（12 种已知失败模式）
  - 五大原则（来自 Anthropic/OpenAI/Cognition/Sierra/LangChain 实践提炼）
- **特点**: 不是替你生成代码，而是告诉你在每个决策点该考虑什么

### 2. architecture-review — 架构决策顾问
- **地址**: https://skillsmp.com/skills/athola-claude-night-market-plugins-pensive-skills-architecture-review-skill-md
- **核心特点**:
  - 不变量冲突检测：当改动与已有设计约束冲突，给出三个选项（保留/分层/修订）
  - ADR 合规审计、耦合分析、原则检查
  - 决策权交还给人，不自动解决
- **适用场景**: 遇到架构决策点，需要结构化框架和选项

### 3. cc-foundry: subagent-engineering + skill-engineering
- **仓库**: https://github.com/xobotyi/cc-foundry
- **核心内容**:
  - 何时用 subagent/skill/agent team 的决策矩阵
  - skill 原型决策（Workflow/Knowledge/Coding Discipline）
  - 内容架构规则 + 量化影响数据（KV 列表 vs 表格：+8.8pp 准确率）
- **适用场景**: 设计和审查 agent 系统的内部结构

### 4. Anthropic skill-creator — 提示词优化的官方方案
- **四模式循环**: Create → Eval → Improve → Benchmark
- **四 agent 评估管线**: Executor → Grader → Comparator → Analyzer
- **5 维度评分**: 任务完成(30%) + 触发准确率(25%) + 输出质量(25%) + 上下文效率(10%) + 工具使用(10%)
- **适用场景**: 系统性优化提示词和 skill

### 5. GEPA + SePO — 提示词自动优化算法
- **GEPA** (ICLR 2026 Oral): 35x 更少执行次数，高 6-20%
- **SePO** (2026.06): 优化器自己也在进化
- **适用场景**: 有 eval suite 后的自动化 prompt 调优

## 这些和 revfactory/harness 的区别

| | revfactory/harness | agentic-product-standard | architecture-review |
|---|---|---|---|
| 做什么 | 自动生成 agent 团队 | 告诉你怎么设计 agent | 帮你审查架构决策 |
| 输出 | .md 配置文件 | 方法论框架 + 检查清单 | 风险标记 + 选项 |
| 角色 | 替你干活 | 顾问 | 审查员 |
| 决策权 | 在 AI | 在人 | 在人 |

## 关键发现

顾问型 skill 的思路**已被验证有效**。最值得参考的是 `agentic-product-standard`——它覆盖了架构决策、提示词优化、测试评估、反模式检测，基本就是你描述的那几个能力方向。它不是"帮你全自动做"而是"指导你怎么做"，决策权始终在人手里。
