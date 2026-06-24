# agentic-product-standard 交叉验证分析

> 将 AGENT_STANDARD.md + STANDARD.md 的主要论断，与 docs/agent-engineering/ 中收集的权威资料逐一比对。

## 一、核心论断来源验证

### ✅ 有明确权威来源支撑的（高可信度）

| 论断 | 来源 | 我们资料中的印证 |
|------|------|-----------------|
| 五种组合模式 (L0-L4 自主性阶梯) | Anthropic "Building Effective Agents" (2024.12) | `01-overview`、`04-architecture` 均收录，六种模式本质一致 |
| 单/多 agent 决策规则 | Cognition "Don't Build Multi-Agents" + Anthropic 多 agent 研究 | `04-architecture` 有 Anthropic/Stripe/OpenAI 收敛分析 |
| Harness > Model，98% 代码是 harness | OpenAI "Harness Engineering" + Claude Code 架构论文 (arXiv:2604.14228) | `03-harness-loop` 有十层栈和 Addy Osmani 的 Ratchet 原则 |
| 多 agent ~90% 提升、~15× token 成本 | Anthropic 多 agent 研究系统论文 (Hadfield, Zhang et al.) | 我们未收录这个具体数据点，但模式吻合 |
| Eval 三层金字塔 | Hamel Husain 方法论 + Shreya Shankar 验证研究 | `08-advisory-skills` 有 Anthropic skill-creator 四 agent 评估管线 |
| Replit 删库事件 | Fortune 2025.07 报道，真实事件 | 我们未收录 |
| Lethal trifecta（私有数据×不可信内容×外部通信） | Simon Willison (2025.06) | 我们未收录 |
| 40% 上下文窗口规则 | HumanLayer (Dex Horthy)，有 Chroma/Databricks 研究支撑 | 我们提到了 context management 但未给出具体数字 |
| RAG-over-tools 3.2× 准确率提升 | arXiv:2505.03275 | 我们未收录 |
| GEPA 提示词进化算法 | ICLR 2026 Oral | `08-advisory-skills` 已收录 |
| MCP 供应链安全 | OWASP Top 10 for Agentic Applications (2026) + MCP 安全框架论文 | 我们未收录 |

### ⚠️ 合理推断但缺乏独立验证的（中等可信度）

| 论断 | 来源 | 评估 |
|------|------|------|
| ≥90% pass rate 才能升级自主性 | 未找到独立出处，可能是作者自己的规则 | 合理但武断——为什么不是 85% 或 95%？ |
| 活跃工具 <20 个 | HumanLayer "12 Factor Agents" 提到 | 我们资料中有"减少 80% 工具"的结论，方向一致 |
| bitter-pill maintenance（模型进步则裁剪 harness） | Daniel Miessler PAI/ISA 框架 | 我们在 `05-june-2026` 有类似结论（LLM-as-Code 挑战） |
| 多 agent 必须用 orchestrator-subagent，禁用 peer-to-peer | 作者声称是 2026 共识 | 我们资料中 Anthropic/OpenAI 确实都用 orchestrator 模式 |

### ❓ 可能是作者原创但合理的（低/无独立验证）

| 论断 | 评估 |
|------|------|
| Agent Contract 13 字段格式 | 作者的模板设计，看起来合理但无外部验证 |
| Message Envelope 的 TypeScript 类型定义 | 工程实践总结，实用但非标准 |
| 禁止动作必须配 code-asserted anti-criterion | 好原则，来自作者的安全哲学 |
| 12 周构建路线图 | 经验总结，合理但无数据支撑 |

## 二、与我们收集资料的互补关系

### agentic-product-standard 比我们强的地方

| 领域 | 我们 | agentic-product-standard |
|------|------|------------------------|
| **安全模型** | 基本未覆盖 | 深度覆盖：lethal trifecta、MCP 供应链、OAuth 2.1、租户隔离、间接注入 |
| **成本治理** | 未覆盖 | Token 预算上限、prompt caching、模型级联路由 |
| **Agent 合约模板** | 未覆盖 | 完整的 13 字段 Agent Contract + Tool Contract + Handoff Contract |
| **生产就绪检查清单** | 未覆盖 | 15 点 DoD + M0-M3 自评表 |
| **实施细节** | 高层模式 | Agent Runner 伪代码、目录结构、Trace Schema |
| **源引附录** | 分散在各文档中 | 集中附录，每个论断都能追溯来源 |

### 我们比 agentic-product-standard 强的地方

| 领域 | agentic-product-standard | 我们 |
|------|------------------------|------|
| **Skill 设计** | 未涉及 | `02-skill-design-guide` 完整覆盖 agentskills.io 规范 |
| **RAG 生产细节** | 一笔带过 | `06-rag-production-guide` 深入分块策略、混合检索、语义缓存 |
| **领域专用 agent** | 未涉及 | `07-domain-specific-agents` 法律/医疗/金融 |
| **2026.06 前沿** | 截止到 2026.06 但不含最新论文 | `05-june-2026-frontier` 有 LLM-as-Code、自进化 agent |
| **Loop Engineering** | 未涉及 | `03-harness-loop` 有 Loop 六大要素和成熟度模型 |
| **Anthropic vs OpenAI 对比** | 未深入 | `04-architecture` 有详细对比表 |

## 三、总体评价

**质量判断：中上。** 作者做了扎实的文献工作——附录中每个论断都标注了来源，大部分来源是我们独立验证过的权威资料。没有发现"编造数据"或"以讹传讹"的迹象。

**最有价值的三个部分：**
1. **Agent Contract 模板和 DoD 检查清单** — 这俩可以拿来直接用
2. **安全模型** — lethal trifecta、MCP 供应链、租户隔离，这些是我们的资料完全没覆盖的
3. **Source appendix** — 每个论断都能追溯，这是严谨性的信号

**主要局限：**
- 13 stars，社区零验证，本质上是"一个人的好总结"
- 部分数字（90% pass rate、40% context）看起来合理但缺乏独立实验支撑
- 安全部分引用了 OWASP 和 Willison，但落到具体实现时偏重 TypeScript 生态
- 对 skill 设计、RAG、领域定制等话题覆盖很浅

**最终建议：值得研究，作为方法论参考框架使用，但不要当教条。** 它的源引附录恰恰是最有用的部分——帮你快速定位到真正的权威来源。
