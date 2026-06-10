# 命理大师 — 架构设计

**版本：** v1.3  
**日期：** 2026-06-10  
**原则：** 本文档是系统架构的单一事实来源。主线实现、实施顺序和验收标准都以此为准。

---

## 1. 架构结论

本项目分成两条线：

- **v1 主线：Go / Eino 实现**
  - 目标是先做出稳定、可演示、可讲清楚的命理咨询产品
  - 状态管理、工具执行、SSE、流式输出全部在 Go
  - 知识检索通过**项目自己的知识库 MCP / skill 知识库 MCP**
- **v2 对照版：LangGraph 增强实现**
  - 目标是学习 LangGraph 在复杂状态编排、条件边、reasoning flow 上的价值
  - 保持同一产品域，对比 v1 的实现边界和复杂度

结论很明确：

- **v1 不把 LangGraph 放进主链路**
- **LangGraph 不是否定，而是延后到更适合学习的位置**

这样做的原因是：当前八字聊天 MVP 的主业务复杂度，还不足以证明双栈主线的成本是值得的。

---

## 2. v1 主线架构

```text
Vue 3
  │
  │ POST /api/chat
  ▼
Gin / Go (:8080)
  ├── Session State Store
  ├── Conversation State Machine
  ├── Orchestrator
  ├── Tools
  │    ├── bazi_calc        → lunar-go
  │    └── knowledge_search → Project Knowledge MCP
  ├── LLM Client            → DeepSeek v4
  └── SSE Writer
```

### 2.1 Go 负责什么

- 接收用户请求
- 管理会话状态
- 解析当前会话阶段
- 决定是否追问、是否排盘、是否复用已有命盘
- 围绕用户当前问题组织咨询回答
- 调用工具和 LLM Client
- 把结构化结果转成 SSE
- 流式输出最终解答

### 2.2 Go 不负责什么

- 不做跨进程复杂编排
- 不为了“像 Agent”而引入多 agent 闭环
- 不展示原始 chain-of-thought

---

## 3. v1 状态模型

Go 侧持有唯一会话状态：

```json
{
  "session_id": "uuid",
  "profile": {
    "calendar_type": "solar",
    "year": 1990,
    "month": 5,
    "day": 20,
    "hour": 8,
    "gender": "男",
    "timezone": "Asia/Shanghai"
  },
  "bazi_result": {},
  "conversation_stage": "collecting | ready | completed",
  "conversation_summary": "",
  "last_user_question": ""
}
```

说明：

- `profile` 是出生资料
- `bazi_result` 是可复用的排盘结果
- `conversation_stage` 用于控制追问和复用
- `last_user_question` 是当前咨询问题的归一化结果
- v1 默认内存存储
- 后续可平滑替换成 SQLite / Redis

---

## 4. 出生信息输入契约

MVP 统一收集以下字段：

```json
{
  "calendar_type": "solar | lunar",
  "year": 1990,
  "month": 5,
  "day": 20,
  "hour": 8,
  "gender": "男 | 女",
  "timezone": "Asia/Shanghai"
}
```

规则：

- 用户明确说“农历/阴历/正月/腊月”等，再标为 `lunar`
- 未明确说明时，默认按 `solar`
- `hour` 必须精确到 0-23
- “早上/晚上/子时”这类模糊表达一律追问确认
- MVP 固定 `timezone=Asia/Shanghai`
- 如果用户明确表示不知道出生时辰，MVP 不做完整四柱排盘，并直接说明能力边界

---

## 5. v1 对话状态机

v1 用 Go 内部确定性状态机处理对话：

```text
START
  ↓
extract profile patch
  ↓
merge profile
  ↓
check missing fields
  ├── missing → ask user
  ├── complete and no bazi_result → full_reading
  ├── complete and profile changed → recalc_reading
  └── complete and bazi_result reusable → followup_reading
```

状态机负责回答 4 个问题：

1. 现在信息够不够排盘
2. 需不需要重算命盘
3. 当前问题是“首次排盘”还是“基于已有命盘的咨询追问”
4. 当前回答是否需要围绕特定问题重新组织知识检索

---

## 6. v1 执行流水线

### 6.1 full_reading / recalc_reading

1. `bazi_calc(profile)`
2. `build_knowledge_query(bazi_result, user_message)`
3. `knowledge_search(query)`
4. `llm_generate(profile, bazi_result, passages, user_message)`

### 6.2 followup_reading

1. 复用 `bazi_result`
2. `build_knowledge_query(bazi_result, user_message)`
3. `knowledge_search(query)`，不可用时可跳过
4. `llm_generate(...)`

关键约束：

- 知识检索 query 必须基于 `bazi_result` 生成
- 不允许在排盘前拍脑袋查“年份+日主+运势”
- `knowledge_search` 的数据源是**项目知识库 MCP / skill 知识库 MCP**

### 6.3 `llm_generate` 的实现边界

v1 中的 `llm_generate` 是一个**逻辑阶段**，不是必须注册进 Tool Registry 的 Eino Tool。

原因：

- `bazi_calc / knowledge_search` 是确定性工具，适合统一走 Tool 接口
- `llm_generate` 需要绑定当前请求的 SSE 输出上下文
- 如果把 `onChunk` 在应用启动时注入 Tool，会丢失“当前连接”这个运行时上下文

因此 v1 采用：

- Orchestrator 负责组装 prompt 和 messages
- Orchestrator 直接调用 `internal/llm.Client`
- Orchestrator 在 `onText` 回调里推送 `text` SSE

这样做更简单，也更符合流式输出的实际边界。

---

## 7. 项目知识库 MCP

v1 的 RAG 不是一个抽象“本地 RAG 服务”，而是明确指向：

- **项目自己的知识库 MCP**
- **skill 知识库 / 命理资料知识页**

Go 侧只需要一个稳定的 `KnowledgeClient` 抽象：

```text
knowledge_search(query, top_k) -> passages[]
```

知识库 MCP 的职责：

- 检索命理典籍或规则页
- 返回结构化引用
- 为最终问题解答提供出处和 supporting evidence

v1 不要求把 MCP 生命周期做得很重，先保证：

- 接口稳定
- 返回格式稳定
- 失败时可优雅降级

---

## 8. SSE 协议

固定 6 种事件：

- `thinking`
- `tool_call`
- `component`
- `text`
- `error`
- `done`

语义：

- `thinking`：展示结构化阶段信息，例如“正在校验出生信息”“开始排盘”
- `tool_call`：展示工具调用开始
- `component`：展示命盘卡片、知识引用卡片
- `text`：展示最终流式解读
- `error`：展示本轮错误
- `done`：本轮结束

注意：

- 不展示模型原始 CoT
- DeepSeek v4 默认关闭 `thinking` 输出，避免把 reasoning content 暴露到产品界面
- 只展示对用户有意义的结构化推理过程

---

## 9. 失败与降级

| 环节 | 策略 |
|------|------|
| `bazi_calc` 失败 | 终止本轮，返回错误 |
| 知识库 MCP 不可用 | 返回空 passages，继续回答 |
| `llm_generate` 失败 | 返回 `error` 事件并结束 |
| 用户信息不全 | 继续追问，不进行伪排盘 |
| 用户资料变更 | 清除旧 `bazi_result`，重新排盘 |

---

## 10. 为什么 v1 不上 LangGraph

因为当前主业务只有一层确定性对话状态机：

- 收集出生信息
- 排盘
- 查知识
- 围绕用户问题解答
- 追问复用

这套流程用 Go 写清楚，比双栈更有利于：

- 快速落地
- 保持调试简单
- 明确 Eino 的真实边界
- 为后续 LangGraph 对照版提供基线

---

## 11. v2 LangGraph 对照版

LangGraph 不是取消，而是延后到更适合学习的版本。

v2 目标：

- 保持同样的产品域
- 只增强 Eino/手写状态机开始吃力的部分
- 明确展示 LangGraph 的收益和代价

### 11.1 v2 值得引入 LangGraph 的场景

- 农历/公历/模糊时辰等复杂信息收集
- 多主题咨询路由：基础排盘 / 年运 / 婚恋 / 事业
- 条件边增多：是否重排、是否查知识库、是否复用已有结果
- 需要图形化展示 reasoning flow

### 11.2 v2 预期边界

- Go 仍然可以继续做 tool host
- LangGraph 负责 planner / conditional routing
- 是否保留双栈，到 v1 完成后再评估

重点是：**v2 用来回答“LangGraph 什么时候值得上”**，而不是为了履历先把它塞进 v1。

---

## 12. 技术选型判断

- **Gin**：合适
- **Eino**：合适，主线就该先把它吃透
- **lunar-go**：合适，强烈建议直接复用
- **Vue 3 + Naive UI**：合适，MVP 成本低
- **项目知识库 MCP**：合适，能把检索能力和知识资产直接串起来
- **LangGraph**：适合放在 v2 对照版，而不是当前主线

---

## 13. ADR

| # | 决策 | 理由 |
|---|------|------|
| ADR-1 | v1 主线采用 Go / Eino | 先保证产品稳定落地 |
| ADR-2 | LangGraph 延后到 v2 对照版 | 让学习收益更可对比 |
| ADR-3 | Go 持有唯一会话状态 | 避免双栈状态同步成本 |
| ADR-4 | 知识检索走项目知识库 MCP | 对齐现有 skill / 知识资产 |
| ADR-5 | 只展示结构化 reasoning flow | 不展示原始 CoT |

---

## 14. 当前实施建议

v1 主线顺序：

1. M0 项目脚手架
2. M1 八字引擎
3. M3 Go Orchestrator
4. M4 项目知识库 MCP + LLM
5. M5 Vue 前端
6. M6 集成联调

v2 对照版：

7. `docs/v2/implementation/m2-langgraph.md`

先把 v1 做成，再决定是否把 v2 接入同仓库主分支，还是单独开对照目录实现。

---

## 15. 学习路线图

本项目的 Agent 学习顺序单独整理在：

- [docs/learning-roadmap.md](/Users/wikiglobal/workSapce/suanming-agent/docs/learning-roadmap.md)
- [docs/long-term-consulting-evolution.md](/Users/wikiglobal/workSapce/suanming-agent/docs/long-term-consulting-evolution.md)
- [docs/v2/README.md](/Users/wikiglobal/workSapce/suanming-agent/docs/v2/README.md)

简要原则：

- v1 学 Tool / State Machine / MCP
- v2 学 LangGraph / Conditional Edges / Bounded Loop
- v3 再考虑 Skill-based / Manager / 多 Agent
