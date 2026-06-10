# 命理大师 — Agent 学习路线图

**版本：** v1.0  
**日期：** 2026-06-10

---

## 1. 目标

这个项目的学习目标不是“一次性把所有 Agent 架构都用上”，而是：

- 先做出一个可运行、可解释、可演示的 Agent 产品
- 再逐步引入更复杂的 Agent 架构
- 每一层都能明确回答：
  - 它解决了什么问题
  - 为什么上一层已经不够了
  - 它带来了什么额外复杂度

---

## 2. 总体路线

```text
v1  Tool + State Machine + MCP
v2  Planner + Conditional Edges + Bounded Loop
v3  Skill-based / Multi-domain / Optional Manager
```

原则：

- **先做确定性，再做智能编排**
- **先做单域，再做多域**
- **先做 bounded loop，再考虑更开放的 agent loop**

---

## 3. v1：Go / Eino 主线

### 3.1 要学什么

- Tool-based 架构
- 会话状态机
- MCP 集成
- SSE 结构化输出
- LLM + 工具结合的基本编排

### 3.2 在本项目里对应什么

- `bazi_calc`
- `knowledge_search`
- `llm_generate`
- 出生信息收集与追问
- 已有命盘复用
- 项目知识库 MCP / skill 知识库 MCP

### 3.3 为什么先学这一层

因为这是最接近真实工程落地的基础层：

- 简单
- 稳定
- 容易调试
- 容易面试时讲清楚

### 3.4 这一层完成后应能回答

- Tool 和普通函数有什么区别
- 为什么知识检索要走 MCP
- 如何设计一个可复用的会话状态机
- 为什么当前产品不需要多 Agent

---

## 4. v2：LangGraph 对照版

### 4.1 要学什么

- Planner
- Conditional edges
- Structured reasoning flow
- Bounded agent loop

### 4.2 在本项目里对应什么

- 农历/公历识别
- 模糊时辰追问
- 用户修正资料后的分支跳转
- 首次排盘 / 追问复用 / 重排盘的路由
- reasoning flow 可视化

### 4.3 为什么放到 v2

因为只有先做完 v1，才能真正判断：

- 哪些地方 Go 手写状态机已经开始变重
- LangGraph 是否真的更清晰
- 双栈增加的复杂度是否值得

### 4.4 v2 推荐边界

- `extractor`
- `profile_normalizer`
- `intent_router`
- `missing_info_planner`
- `execution_planner`

### 4.5 v2 只建议做 bounded loop

推荐：

- 最多 2-3 轮补信息
- 最多 1 次重算
- 最多 1 次知识检索 fallback

不推荐：

- 开放式自主循环
- 无限 replan
- 自反思闭环

### 4.6 这一层完成后应能回答

- LangGraph 比 Go 手写状态机强在哪里
- 哪些条件边值得图形化表达
- bounded loop 为什么比开放式 loop 更适合业务系统

---

## 5. v3：Skill-based 与多域扩展

### 5.1 要学什么

- Skill-based architecture
- 多能力域拆分
- 可选的 manager / supervisor

### 5.2 适合什么前提

只有当产品扩展到多个独立能力域时才值得，例如：

- 八字
- 六爻
- 姓名分析
- 年运专项分析

### 5.3 为什么现在不做

因为当前只有一个核心领域：八字咨询。

这时上 manager / supervisor 型多 Agent，通常只是增加：

- prompt 复杂度
- 调试成本
- 上下文污染风险
- 演示噪音

### 5.4 什么时候才值得上 manager

当你真的出现下面这些情况时：

- 有多个相对独立的 specialist
- 需要跨 specialist 路由
- 不同 specialist 有不同上下文和能力边界
- 单一 planner 已经难以维护

---

## 6. 关于 Agent Loop 的判断

### 6.1 值得学

值得学的是：

- **bounded loop**
- **可解释的 loop**
- **有退出条件的 loop**

例如：

- 信息不全时继续追问
- 信息变化时允许一次重排
- 检索失败时走一次 fallback

### 6.2 现在不值得学

不建议在本项目当前阶段学习：

- 开放式 autonomous loop
- 无限工具尝试
- 自发反思再规划再反思

原因很简单：

- 难测
- 难控
- 难演示
- 对当前产品收益不大

---

## 7. 关于 Skill-based 架构的判断

### 7.1 值得学的部分

现在就值得学的是“轻量 skill 思维”：

- 把能力做成边界清晰的模块
- 把知识检索策略做成可替换单元
- 把解读模板做成可配置资源

### 7.2 暂时不用学太重的部分

当前不需要：

- 完整 skill runtime
- skill marketplace
- skill discovery / installation 体系
- 跨 skill 动态编排

---

## 8. 结论

这个项目的推荐学习路径是：

1. **v1**
   - Go / Eino
   - Tool system
   - State machine
   - 项目知识库 MCP
   - SSE 可观测输出
2. **v2**
   - LangGraph planner
   - Conditional edges
   - Bounded loop
   - reasoning flow
3. **v3**
   - Skill-based architecture
   - 多领域扩展
   - 可选 manager / supervisor

一句话总结：

- **Tool / MCP / 状态机：现在就学**
- **LangGraph / 条件边 / bounded loop：v2 学**
- **manager / 多 Agent / 重型 skill 架构：等多域之后再学**

---

## 9. 相关学习文档

配套学习目录：

- [docs/learning/README.md](/Users/wikiglobal/workSapce/suanming-agent/docs/learning/README.md)
- [docs/learning/01-agent-architectures.md](/Users/wikiglobal/workSapce/suanming-agent/docs/learning/01-agent-architectures.md)
- [docs/learning/TEMPLATE.md](/Users/wikiglobal/workSapce/suanming-agent/docs/learning/TEMPLATE.md)
