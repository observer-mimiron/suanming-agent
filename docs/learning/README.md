# Agent 学习笔记目录

这个目录用于记录：

- 主流 Agent 架构
- 本项目实际采用的架构
- 做项目过程中遇到的取舍
- 每个版本的设计变化和学习结论

原则：

- **先记概念，再记项目映射**
- **先记为什么用，再记怎么实现**
- **优先记录 trade-off，不只记录定义**

---

## 当前目录

- [01-agent-architectures.md](/Users/wikiglobal/workSapce/suanming-agent/docs/learning/01-agent-architectures.md)
  - 主流 Agent 架构综述
- [TEMPLATE.md](/Users/wikiglobal/workSapce/suanming-agent/docs/learning/TEMPLATE.md)
  - 后续学习记录模板

---

## 建议记录方式

每学完一个主题，至少补 3 件事：

1. 这个架构解决什么问题
2. 它比更简单方案多了什么复杂度
3. 它在本项目里有没有必要

---

## 和本项目的关系

推荐顺序：

1. 先做 v1：Go / Eino / Tool / State Machine / MCP
2. 再做 v2：LangGraph / Conditional Edges / Bounded Loop
3. 最后再评估 Skill-based / Manager / Multi-Agent

可结合：

- [docs/architecture.md](/Users/wikiglobal/workSapce/suanming-agent/docs/architecture.md)
- [docs/learning-roadmap.md](/Users/wikiglobal/workSapce/suanming-agent/docs/learning-roadmap.md)
