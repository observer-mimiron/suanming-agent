# Agent 面试资料

结论：主目录现在按 `3 份主文档 + 1 份强化题库 + 1 份索引` 收口，已经比上一版更适合系统备战一面、二面和部分深挖面。

这套资料基于两类事实重写：

- 仓库当前真实主链：`RouteAdvisor -> Policy Gate -> Manager -> ExecutionPlan -> Prefill -> specialist runner(s) -> manager compose -> final guard -> SSE`
- 截至 **2026-07-18** 的公开技术资料，重点参考 OpenAI《A practical guide to building agents》、Anthropic《Building effective agents》、Langfuse observability 文档，以及 RAG 相关一手工程资料

## 学习顺序

1. [01-agent-learning-map.md](./01-agent-learning-map.md)
   先补 Agent 基础盘，知道什么该讲、什么不该讲，并学会结构化答题模板。
2. [02-suanming-agent-project-playbook.md](./02-suanming-agent-project-playbook.md)
   再把这个项目讲顺，重点是主链、边界、失败模式、你的贡献和工程化深挖。
3. [03-agent-topics-rag-eval-guardrails.md](./03-agent-topics-rag-eval-guardrails.md)
   最后把 RAG、Eval、Guardrails、Observability 这些高频知识点补齐，尤其把 RAG 单独练透。
4. [04-agent-interview-question-bank.md](./04-agent-interview-question-bank.md)
   收口训练，用强化题库背答法、练场景题、补工程化高频问答。

## 每份文件负责什么

| 文件 | 作用 | 学完标准 |
|---|---|---|
| `01-agent-learning-map.md` | 建立知识地图、学习路径、结构化回答模板 | 你能分清 agent、workflow、RAG、memory、state，并按固定骨架作答 |
| `02-suanming-agent-project-playbook.md` | 把仓库讲成一个可信的工程项目 | 你能稳定讲 30 秒、2 分钟、5 分钟版本，并接住工程化追问 |
| `03-agent-topics-rag-eval-guardrails.md` | 补市场高频追问点 | 你能把 RAG、Eval、Guardrails、Observability 讲到调优和评测层 |
| `04-agent-interview-question-bank.md` | 做面试训练 | 你能快速回答高频题、工程化题、场景题，不飘空话 |

## 使用方法

- 如果时间只有 2 天：先看 `02 -> 04`
- 如果时间有 5 到 7 天：按 `01 -> 02 -> 03 -> 04`
- 如果时间有 7 到 10 天：按 `01 -> 02 -> 03 -> 04` 全量过一遍，再录音复述项目和 RAG 专项
- 如果你准备的是 Agent / AI Engineer：4 份都要看
- 如果你准备的是后端偏 Agent 平台：重点看 `02` 和 `03`

## 这次重点补强了什么

- 回答从“有知识点”升级成“有固定答题结构”，多处统一成 `短答 / 展开 / 项目映射 / 常见追问` 或相近骨架
- 工程化高频问题补厚了，包括并发、状态隔离、时延、成本、幂等、失败恢复、可观测性、评测闭环
- `RAG` 从原来的一个专题，升级成单独重点页，覆盖 pipeline、调优、评测、0-hit 降级、成本权衡
- 题库扩成强化版，目前共有 `85` 题，覆盖一面高频、二面深挖和部分 system design 场景题

## 这套资料的讲法边界

- 重点讲工程化 Agent，不讲“神奇智能体”
- 重点讲可控、可观测、可回归，不讲空泛框架名
- 重点讲真实代码主链，不把仓库包装成你没做过的东西
