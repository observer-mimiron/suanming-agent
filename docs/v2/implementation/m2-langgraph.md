# M2: LangGraph 推理层（v2 对照版）

**定位：** 本文档不再描述 v1 主链路实现，而是保留为 v2 对照版入口。

**当前状态：** 暂缓实施。只有在 v1 Go/Eino 主线完成后才重新展开。

---

## 为什么暂缓

当前主线已经明确为：

- v1：Go / Eino + 项目知识库 MCP
- v2：LangGraph 对照版

所以这里不再保留旧的“双栈主链路”实施步骤，避免误导后续开发。

---

## v2 的学习目标

LangGraph 在 v2 主要用来验证这些点：

1. 复杂出生资料收集是否比 Go 手写状态机更清晰
2. 条件边是否更适合图形化表达
3. planner / routing 是否真的带来收益
4. bounded loop 是否让多轮问答更自然

---

## v2 推荐实现边界

推荐节点：

- `extractor`
- `profile_normalizer`
- `intent_router`
- `missing_info_planner`
- `execution_planner`

推荐输出：

```json
{
  "action": "ask | execute",
  "mode": "full_reading | followup_reading",
  "need_recalc": true,
  "knowledge_topics": ["事业", "流年", "日主"]
}
```

---

## v2 启动条件

只有满足以下条件，才恢复本模块详细实施：

1. v1 主线已跑通
2. 已能明确指出 Go 手写状态机的痛点
3. 已确定要对比 LangGraph 的 planner / conditional edges / bounded loop

---

## 重新展开时的参考文档

- [docs/architecture.md](/Users/wikiglobal/workSapce/suanming-agent/docs/architecture.md)
- [docs/v2/tech-reasoning.md](/Users/wikiglobal/workSapce/suanming-agent/docs/v2/tech-reasoning.md)
- [docs/learning-roadmap.md](/Users/wikiglobal/workSapce/suanming-agent/docs/learning-roadmap.md)
- [docs/learning/01-agent-architectures.md](/Users/wikiglobal/workSapce/suanming-agent/docs/learning/01-agent-architectures.md)
