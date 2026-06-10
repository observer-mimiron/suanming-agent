# 命理大师 — LangGraph 推理层技术方案

**版本：** v2 draft  
**定位：** 本文档不再描述 v1 主链路，而是描述未来 LangGraph 对照版的设计方向。

---

## 1. 为什么把 LangGraph 放到 v2

v1 主线已经可以用 Go/Eino 完成：

- 出生信息收集
- 排盘
- 知识检索
- 流式解读
- 追问复用

因此，LangGraph 的价值不应通过“强塞进主线”来体现，而应通过对照版来回答：

- 哪些状态编排开始变复杂
- 哪些条件边用 Go 手写会变重
- LangGraph 是否真的更清晰

---

## 2. v2 适用场景

当产品升级到以下复杂度时，再引入 LangGraph：

- 农历/公历、模糊时辰、资料修正等复杂资料采集
- 多主题咨询路由：基础排盘 / 年运 / 婚恋 / 事业
- 条件边增多：是否重排、是否查知识库、是否复用已有结果
- 需要更强的 reasoning flow 展示

---

## 3. v2 推荐边界

推荐的 LangGraph 边界：

- `extractor`
- `profile_normalizer`
- `intent_router`
- `missing_info_planner`
- `execution_planner`

LangGraph 输出的是结构化 planner 决策，例如：

```json
{
  "action": "ask | execute",
  "mode": "full_reading | followup_reading",
  "need_recalc": true,
  "knowledge_topics": ["日主", "年运"]
}
```

不建议在 v2 一开始就做：

- tool result callback 闭环
- 多 agent 自反思循环
- 复杂中断恢复

---

## 4. 与 Go 的协作方式

如果 v2 落地，推荐边界如下：

- Go：状态、工具、SSE、LLM 流式输出
- LangGraph：planner / routing / conditional edges

这样能保留：

- Go 的稳定执行层
- LangGraph 的状态编排学习价值

---

## 5. 当前状态

本文档暂时只是 v2 草案。

启用条件：

1. v1 主线完成
2. 已能明确指出 Go 手写状态机的痛点
3. 有足够理由证明 LangGraph 会让实现更清晰，而不是更重
