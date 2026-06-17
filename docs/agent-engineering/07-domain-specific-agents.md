# 领域专用 Agent 设计 — 法律、医疗、金融等垂直领域

> 2025-2026 年是从通用 AI 向**垂直化领域 agent** 转变的关键期。医疗、法律、金融仍处于极早期渗透（1%-5%），是蓝海。

## 领域 Agent 通用架构

### DEA — Domain Expert Agent 六核心组件

| 组件 | 职责 |
|------|------|
| **感知模块** | 跨模态数据摄入（DICOM 影像、法律文书、金融终端） |
| **领域知识图谱 (DKG)** | 实体-关系-属性结构化知识 |
| **推理引擎** | 混合：规则推理 + 案例推理 + 深度学习推理 |
| **工具调用模块** | 专业 API 编排（Westlaw、Wind、ICD-10 编码器） |
| **决策与行动模块** | 合规边界内的自主决策 |
| **记忆模块** | 历史交互、决策和用户上下文 |

### 三层混合架构（RAG + Fine-Tuning + Tools）

| 层 | 用途 |
|----|------|
| **数据层 (RAG)** | 专有文档（判例、病历）——用于**事实接地** |
| **工具层** | 专业 API ——用于**行动执行** |
| **模型层 (Fine-Tuning)** | 领域语法、语调、行为规范——用于**格式/行为** |
| **护栏层** | 严格操作边界——"绝不提供理财建议" |

> **设计原则**: RAG 用于 facts（接地），fine-tuning 用于 format/behavior（语调匹配）。高风险场景 temperature=0。

## 医疗领域

### Anthropic vs OpenAI 对决（2026.01）

| 维度 | OpenAI for Healthcare | Claude for Healthcare |
|------|----------------------|----------------------|
| 模型 | GPT-5.2 + HealthBench 微调（250+医师参与） | Claude Opus 4.5（200K 上下文） |
| 焦点 | 临床诊断、多模态影像+EHR 综合 | 行政精确性（ICD-10 编码、预授权自动化） |
| 策略 | "临床大脑"——诊断推理 | "安全优先编排器"——运营引擎 |
| 云伙伴 | Microsoft Azure + Epic Systems（180+医疗系统） | AWS + Google HealthScribe |
| 合规 | HIPAA + BAA | HIPAA + BAA |

**两方都签署了 BAA，承诺不拿患者数据训练。**

### 医疗抄写员模式（示例）

```
音频 → 专业 ASR → 提取 Agent（症状、用药）
→ 编码 Agent（映射到 ICD-10）→ 验证（"胸痛对 5 岁儿童是否合理？"）
→ EMR 输出
```

## 法律领域

### Anthropic 法律插件（2026.02）

- **逐条合同分析** — 绿/黄/红风险标记
- **NDA 快速筛查**
- **基于组织谈判手册生成修订稿 (redline)**
- **集成**: Microsoft 365、Slack、Box、Jira

### 法律特有设计模式

- **层级上下文**: 律所标准 > 通用法律知识
- **信任验证**: 防止 agent 执行合同中嵌入的恶意指令（prompt 注入攻击向量）
- **评估-优化循环**: 起草→评估→修订，直到质量达标

### 真实教训

美国一名律师因使用 AI 生成的 **6 条虚构判例引用**被罚款 $5,000。法律 agent 的 grounding 不是可选项，是刚需。

## 金融领域

### 典型架构

- **编排器-子 agent**: 编排器将"生成 X 公司竞争分析"分解为子任务（网页搜索、数据库查询、情感分析、图表生成），合成输出
- **账户对账**工作流使用自主多步骤流程
- Claude Opus 4.6 的 GDPval-AA 基准（经济价值知识工作评估）领先 GPT-5.2 144 Elo 分

## 领域 Agent 六种核心设计模式（Anthropic）

| 模式 | 领域示例 |
|------|----------|
| **Prompt Chaining** | 保险核保：提取→评估风险→生成决策→格式化合规 |
| **Routing** | 医疗接诊：症状→临床链，预约→排程器，账单→查询 |
| **Parallelization** | 地产尽调：200 页租约分章节并行处理 |
| **Orchestrator-Subagents** | 金融研究：搜索/数据库/情感/图表子 agent |
| **Evaluator-Optimizer** | 法律起草：起草 NDA→按标准评估→重写直到通过 |
| **Autonomous Agent** | IT 运维：跨日志/部署/负载均衡自主排查 504 错误 |

## 六条上下文接地原则（Agentic Context Grounding）

1. **Read Before You Write** — 生成前先查权威来源
2. **Layered Context Hierarchy** — 系统 prompt > 检索上下文 > 对话历史 > 用户输入
3. **Scoped Knowledge Domains** — 法律 agent 不应提供医疗建议
4. **Minimal Footprint** — 仅请求需要的权限和数据
5. **Prefer Reversible Actions** — 暂存变更而非直接提交
6. **Trust Verification** — 验证声称的权限，不执行来自不可信来源的指令

## 市场现状（Menlo Ventures, 2025.12）

| 公司 | 2023 企业 LLM 支出 | 2025 企业 LLM 支出 |
|------|-------------------|-------------------|
| Anthropic | 12% | **40%** |
| OpenAI | 50% | **27%** |

但垂直领域渗透率极低：医疗 1%、法律 0.9%、金融 <5%。**16 个垂直领域仍是蓝海。**

## Anthropic vs OpenAI 策略分歧

| 维度 | Anthropic | OpenAI |
|------|-----------|--------|
| 策略 | 无代码、预构建角色插件 | 强大底层多模态模型 |
| 产品 | Claude Cowork（11 个开源插件） | ChatGPT Enterprise + Microsoft 生态 |
| 定价 | $5/$25 per M tokens (Opus 4.6) | ~$1.25/$10 per M tokens (GPT-5.2) |
| 目标 | 受监管行业（安全/Constitutional AI） | 消费者应用、媒体丰富工作流 |

Anthropic 定价高 2-4 倍，但受监管行业愿为安全框架买单。

## 一个关键发现：信任差距

Anthropic 研究发现：**Claude 能解决约需 5 小时人工的任务，但 99.9 分位用户会话仅约 42 分钟**。能力与部署之间的"信任赤字"本身就是下一个产品机会。

## 权威资源

- [Unified-MAS (arXiv 2603.21475)](https://arxiv.org/abs/2603.21475) — 领域专用多 agent 自动生成
- [Two-Dimensional Framework (arXiv 2605.13850)](https://arxiv.org/abs/2605.13850) — 27 种设计模式 × 7×6 矩阵
- [Building Domain-Specific Agents (arunbaby.com)](https://arunbaby.com/ai-agents/0050-building-domain-specific-agents/)
- [Agentic AI Survey (Springer, 2026)](https://link.springer.com/article/10.1007/s10462-025-11422-4) — 90 篇研究 PRISMA 评审
- [Claude Design 6 Agentic Patterns (MindStudio)](https://www.mindstudio.ai/blog/claude-design-6-agentic-patterns-vertical-ai-apps)
- [Agentic Context Grounding (MindStudio)](https://www.mindstudio.ai/blog/agentic-context-grounding-claude-design-patterns)
- [Healthcare Duel: OpenAI vs Anthropic (Wedbush, 2026.01)](https://investor.wedbush.com/wedbush/article/tokenring-2026-1-19-the-battle-for-the-white-coat-openai-and-anthropic-reveal-dueling-healthcare-strategies)
- [Vertical Domain Blue Oceans (HTX Insights)](https://www.htx.com.gt/news/anthropic-data-nearly-half-of-ai-agent-calls-concentrated-in-QzTU49iy/)
- [EONSR: Custom Tooling for Domain Agents](https://eonsr.com/en/what-is-the-strategic-imperative-for-developing-custom-tooling-for-domain-specific-ai-agents/)
