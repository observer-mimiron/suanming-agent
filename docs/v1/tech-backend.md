# 命理大师 — Go 执行层技术方案

**版本：** v1.2  
**定位：** Go 服务是 v1 主线的唯一后端。负责状态、工具执行、SSE 和流式输出。

---

## 1. 技术选型

| 层 | 选型 | 原因 |
|---|------|------|
| 语言 | Go 1.22+ | 主力语言，适合主线实现 |
| HTTP 框架 | Gin | 轻量，SSE 处理简单 |
| 工具抽象 | Eino Tool System | 统一工具接口 |
| 八字引擎 | lunar-go | 成熟可靠，不重复造轮子 |
| 知识检索 | 项目知识库 MCP Client | 直接复用项目知识资产 |
| LLM | DeepSeek v4 API | 负责最终流式解读 |

---

## 2. 职责边界

Go 做什么：

- 管理会话状态
- 解析当前对话阶段
- 决定是否追问或排盘
- 执行 `bazi_calc / knowledge_search`
- 直接调用 LLM Client 做流式生成
- 通过 SSE 输出结构化事件

Go 不做什么：

- 不接入 LangGraph 主链路
- 不做多 agent 闭环
- 不展示原始 CoT

---

## 3. 核心模块

```text
cmd/server/main.go
internal/
  orchestrator/
  state/
  tools/
    bazi_calc.go
    knowledge_search.go
  llm/
    client.go
  mcp/
    knowledge_client.go
  sse/
    writer.go
```

---

## 4. 会话状态

最小状态模型：

```go
type SessionState struct {
    SessionID           string
    Profile             map[string]any
    BaziResult          map[string]any
    ConversationStage   string
    ConversationSummary string
}
```

约束：

- 同一 `session_id` 串行处理
- 出生资料变化时，旧 `BaziResult` 失效
- v1 先用内存实现

---

## 5. 工具设计

### `bazi_calc`

- 输入：出生资料
- 输出：四柱、五行、十神、大运
- 失败时终止本轮

### `knowledge_search`

- 输入：基于 `bazi_result` 生成的检索 query
- 输出：`passages[]`
- 数据源：项目知识库 MCP / skill 知识库 MCP
- 失败时返回空结果并继续

### `llm_generate`

- 输入：用户问题 + 命盘 + 知识引用
- 输出：流式文本
- 由 Orchestrator 调用 `llm.Client.ChatStream(...)`
- 由 Orchestrator 通过 SSE 推送 `text` 事件

---

## 6. SSE 事件

固定输出：

- `thinking`
- `tool_call`
- `component`
- `text`
- `error`
- `done`

---

## 7. 与 v2 的关系

v2 如果引入 LangGraph：

- Go 仍可继续做 tool host 和 SSE host
- 但 v1 不为此预埋复杂双栈协议

当前阶段先把 Go/Eino 主线做扎实，再决定是否抽出 LangGraph planner。
