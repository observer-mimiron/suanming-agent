# Agent Skill 设计指南 — 规范、模式与最佳实践

## Agent Skills 开放标准

Anthropic 于 **2025.12.18** 发布 [agentskills.io](https://agentskills.io)。Simon Willison 评价："a deliciously tiny specification."

- **规范**: https://agentskills.io/specification
- **仓库**: https://github.com/agentskills/agentskills

## Skill 目录结构

```
my-skill/
├── SKILL.md          # 必须：YAML frontmatter + Markdown 指令
├── scripts/          # 可选：确定性可执行脚本
├── references/       # 可选：按需加载的补充文档
└── assets/           # 可选：模板、图片、静态资源
```

> Skill 编码**流程（怎么做）**，Document 编码**事实（是什么）**。

## YAML Frontmatter

| 字段 | 必须 | 约束 |
|------|------|------|
| `name` | 是 | ≤64 字符，小写字母+数字+连字符，必须与目录名一致 |
| `description` | 是 | ≤1024 字符。**这是路由逻辑，不是标签** |
| `license` | 否 | 许可证名称 |
| `allowed-tools` | 否 | 预批准工具列表（实验性） |

## 渐进式加载（三层模型）

1. **启动时** — 仅加载 `name` + `description`（约 100 tokens/skill）
2. **触发时** — 完整 `SKILL.md` 正文（建议 <500 行）
3. **按需** — 具体 `references/`、`assets/` 仅在步骤明确指示时加载

### Token 预算参考

| 层级 | Token | 内容 |
|------|-------|------|
| 最小 | 500-1,000 | 核心身份 + 5 规则 + 3 反模式 |
| 标准 | 2,000-4,000 | 完整行为 prompt |
| 综合 | 6,000-10,000 | 含示例、边界、schema |

Vercel 将 40KB 文档压缩到 8KB（80%）同时保持 100% eval 通过率。

## 三条核心设计原则（Angie Jones / Block）

### 1. 决定 Agent **不应该**决定什么
确定性操作 → `scripts/`，不交给 LLM。

### 2. 决定 Agent **应该**决定什么
解释、创作、推理 → 留给 LLM。

| 区域 | 归属 | 原因 |
|------|------|------|
| 规则与执行 | 脚本/模板/硬规则 | 相同输入→相同输出 |
| 解释与行动 | Agent (LLM) | 上下文不同，需要智能 |

### 3. 写宪法而非建议
LLM 是"讨好者"——会软化坏消息、跳过步骤。必须包含：
- 永远不覆盖脚本输出
- 永远不增删检查项
- 精确遵循输出模板
- **负面触发器**："不要用于 X、Y、Z"

## 写作规范（Minko Gechev）

- `SKILL.md` < 500 行
- 第三人称祈使句："Execute the extraction script" 而非 "You should..."
- `description` 含触发条件 + 负面触发条件
- 发布前验证：让 LLM 读 description，生成 3 个应触发 + 3 个不应触发的 prompt

## 六种设计模式

| 模式 | 描述 |
|------|------|
| **Checklist** | 顺序验证步骤，pass/fail |
| **Options** | 结构化选择呈现给用户 |
| **Pipeline** | 多阶段数据转换 + 验证门控 |
| **Integration** | 按特定顺序协调多工具/API |
| **Swarm** | 并行执行 + 结果聚合 |
| **Distillation** | 从成功执行痕迹提取可复用 skill |

## 关键研究结论

- **SkillsBench (2026)**: 人工 skill 平均提升 pass rate 16.2%；模型自生成 skill **无一致收益**
- **AgentSkillOS (2026)**: 扁平库在 80+ skills 时崩溃，需要**能力树**层级结构
- **SoK 安全 (2026)**: 31,132 社区 skill 中 26.1% 含可利用漏洞
- **Contractual Skills (2026)**: Skill 是**治理层**，需合约式约束

## Skill 生命周期

发现 → 创建 → 精炼 → 评估 → 更新 → 废弃

定期测试：不用此 skill 是否已能产出同等结果？模型进步覆盖了 gap → 退役。

## 必读资源

- [skills-best-practices](https://github.com/mgechev/skills-best-practices) (Minko Gechev)
- [agent-skills-orange-book](https://github.com/alchaincyf/agent-skills-orange-book) (6 种设计模式)
- [Agent-Skills-for-Context-Engineering](https://github.com/muratcankoylan/Agent-Skills-for-Context-Engineering)
- [3 Design Principles](https://angiejones.tech/3-design-principles-for-creating-agent-skills/) (Angie Jones)
- [Most Teams Building Them Wrong](https://www.oreilly.com/radar/agent-skills-work-but-the-research-shows-most-teams-are-building-them-wrong/) (O'Reilly)
- [Skills vs Documents](https://www.fortegrp.com/insights/skills-vs-documents-your-agent-is-only-as-good-as-its-last-skill-update)
- [Contractual Skills (arXiv)](https://export.arxiv.org/abs/2605.22634)
- [Agent Skills Standard](https://github.com/HoangNguyen0403/agent-skills-standard) (259 标准，85% token 缩减)
