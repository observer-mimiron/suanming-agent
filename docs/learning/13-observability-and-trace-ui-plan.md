# 可探测 Agent 方案（v1）

**日期：** 2026-06-11  
**状态：** 已修订，待评审  
**目标：** 让每个聊天回合都能被看见、被解释、被回放，而不是只看到最后一段文案。  
**参考：**

- `docs/learning/02-reference-projects-and-observability.md`
- Langfuse Observability Overview / Data Model
- OpenInference Specification / Semantic Conventions
- Phoenix Tracing Overview
- OpenTelemetry Go Manual Instrumentation

---

## 1. 先说结论

这轮不应该先做“trace UI”，也不应该先接外部 tracing 平台。

**正确顺序是：先做可探测 Agent，再把探测结果映射到 UI。**

也就是：

1. 后端先建立一份真实的 `TurnTrace`
2. 本地调试文件、前端过程面板、后续 OTLP 导出都复用这份数据
3. 前端只展示“对用户有意义的结构化步骤”，不展示原始 CoT

推荐路线：

- **v1 推荐方案：进程内 Trace Collector + 本地 JSON Trace + SSE Trace Digest**
- 不直接上 Langfuse / Phoenix
- 不直接把 OTel/OTLP 作为第一步强依赖

原因很简单：当前项目最缺的不是平台，而是**统一的执行轨迹模型**。

---

## 2. 当前代码现状与真实问题

这部分按当前代码实情修正，不再按猜测写方案。

### 2.1 乱码的根因不是 NoopTracer

当前“debug 日志中文乱码”的直接根因，不在 `internal/tracing/`，而在 SSE debug 文件写入时使用了：

```go
fmt.Fprintf(s.dbg, "[%s] %v\n", evt.Type, evt.Data)
```

`%v` 打 `map[string]any` 时不保证可读 JSON，也不适合后续分析。

所以：

- **修复乱码的第一步不是替换 NoopTracer**
- 而是先把 debug 输出改成结构化 JSON

### 2.2 tracer 现在只有壳，没有数据沉淀

当前代码里已经有：

- `Tracer`
- `Trace`
- `Span`
- Gin middleware
- Orchestrator `StartTrace("chat.run")`

但实现仍是 `NewNoopTracer()`，所以：

- 没有 trace id
- 没有 span 树
- 没有耗时
- 没有错误记录
- 没有持久化

### 2.3 前端并不是“完全没有过程展示”

当前前端已经能展示：

- `thinking`
- `tool_call`
- `component`
- `text`
- `error`

而且 `ThinkingSegment.vue` 已经是 `<details>` 折叠块。

所以这轮不是从 0 到 1 做过程展示，而是：

- 把“零散事件”升级成“同一回合的结构化轨迹”
- 让前端能展示**步骤时间线**，而不是只显示几条散乱 thinking

### 2.4 真正缺的是统一的数据通路

当前有三条彼此独立的东西：

- SSE 事件流
- debug txt 文件
- tracer 接口

它们没有共享同一个数据模型。

这会导致两个后果：

1. 落盘一套、前端一套、后续导出一套，长期必然分叉
2. 你想回答“这轮到底走了哪些步骤、哪个最慢、哪一步降级了”时，没有统一答案

---

## 3. 什么叫“可探测 Agent”

本项目里，“可探测”不是传统 APM 意义上的 metrics first。

这里的可探测，指每个聊天回合都能回答下面这些问题：

1. 这一轮被路由成了什么类型：`ask / full_reading / followup / direct_bazi`
2. 调了哪些工具，顺序是什么，是否成功
3. 每一步花了多久
4. 知识检索是否命中，命中了多少条，最终哪些引用进入回答
5. LLM 用了哪个模型，耗时多少，若拿得到则记录 token
6. 本轮是否发生降级：例如跳过知识检索、奇门失败后回退八字
7. 前端给用户展示的“过程面板”来自哪份真实执行数据

**一句话：不是把日志打得更多，而是把一次 agent 运行变成一条可以解释的执行轨迹。**

---

## 4. 外部方案调研结论

### 4.1 Langfuse 提供的是“数据模型启发”，不是你当前第一步要接的平台

官方文档强调三层核心概念：

- trace
- session
- observation

并建议：

- 多轮应用用 `session` 归组
- 给 trace 加属性
- 跟踪 token / cost / timing

对本项目的启发是：

- `1 个聊天回合 = 1 个 trace`
- `1 个 session_id = 1 条多轮会话`
- 每个工具/检索/LLM 步骤都应该成为子 span

### 4.2 OpenInference 适合作为语义约束

OpenInference 的价值不在 UI，而在语义标准。

它明确区分了这些 span kind：

- `AGENT`
- `CHAIN`
- `TOOL`
- `RETRIEVER`
- `LLM`

这非常适合本项目，因为你现在的链路刚好就有这些步骤。

所以本方案建议：

- **字段命名和 span kind 借 OpenInference 的思路**
- 但 v1 先不要求完整接入 OpenTelemetry SDK

### 4.3 Phoenix / OTel 更适合作为第二阶段导出目标

Phoenix 强调：

- 同一条 trace 里同时看 LLM / retrieval / tool / custom logic
- 通过 OTLP 接入

OpenTelemetry Go 官方建议：

- 应用侧自己初始化 tracer provider
- 通过 context 传播 span
- 手动创建嵌套 span

这些都适合第二阶段：

- 当本地 trace 模型稳定后
- 再做 exporter，把本地 trace 映射到 OTel span

**所以推荐顺序是：本地统一模型 -> UI 消费 -> OTLP 导出。**

---

## 5. 三种可选方案

### 方案 A：只修 debug 文件，继续靠现有 SSE 事件

**做法：**

- 把 txt 调试输出改成 JSON
- 保持 `thinking/tool_call/component/text` 现状

**优点：**

- 最小改动
- 很快见效

**缺点：**

- 仍然没有统一 trace 模型
- 前端过程展示仍是零散事件，不是回合级轨迹
- 后续接 OTel 还得重做

**适用：**

- 只想先解决“乱码”和“能看日志”

### 方案 B：统一 `TurnTrace` 模型，前后端共用一份轨迹

**做法：**

- 在 Go 内建立回合级 trace collector
- 产出本地 JSON trace
- 从同一条 trace 派生：
  - debug 文件
  - 前端 trace panel
  - trace summary card
  - 后续 OTLP exporter

**优点：**

- 与现有 Go orchestrator 架构最匹配
- 一次建模，三处复用
- 可逐步演进到 Langfuse / Phoenix / OTel

**缺点：**

- 比方案 A 多一层建模工作
- 需要明确 UI 展示边界

**适用：**

- 当前项目，且希望它真的成为“可探测 agent”

### 方案 C：直接全面接入 OTel + OTLP + 外部平台

**做法：**

- 直接用 OTel SDK
- 直接导到 Phoenix / Langfuse 兼容链路

**优点：**

- 标准化程度最高
- 后续平台集成成本低

**缺点：**

- 当前项目体量偏小，收益不够稳定
- 前端过程面板的数据仍然要单独处理
- 一开始就引入 exporter / collector / endpoint 配置，心智成本偏高

**适用：**

- 团队已经确定要长期运营外部 tracing 平台

### 推荐

**推荐方案 B。**

这是当前仓库最平衡的方案：既不只修表象，也不一开始就把系统复杂度拉高。

---

## 6. 推荐方案：可探测 Agent 架构

### 6.1 一句话架构

`Orchestrator` 在一次聊天回合内收集 `TurnTrace`，然后把它同时输出到：

- 本地结构化 trace 文件
- 前端 `component` 事件里的 trace digest
- 可选的调试查询接口
- 未来可选的 OTLP exporter

### 6.2 数据流

```text
POST /api/chat
  -> Gin Middleware 创建 request trace context
  -> Orchestrator 创建 turn trace (1 turn = 1 trace)
  -> 每个步骤创建 span
  -> 完成后生成：
       1. logs/traces/*.json
       2. SSE component(trace-panel / trace-summary)
       3. 可选 /api/debug/traces/:trace_id
       4. 未来 exporter: OTLP / Langfuse / Phoenix
```

### 6.3 核心原则

- **单一事实来源**：所有观测结果来自同一条 `TurnTrace`
- **不暴露原始 CoT**：只记录结构化步骤、摘要、耗时、状态
- **调试面和产品面分离**：
  - debug 面可以看更全
  - 产品面只看用户可理解的步骤
- **渐进增强**：
  - 先本地
  - 再前端
  - 最后 exporter

---

## 7. 统一 Trace 数据模型

### 7.1 回合级模型

```go
type TurnTrace struct {
    TraceID      string            `json:"trace_id"`
    SessionID    string            `json:"session_id"`
    TurnType     string            `json:"turn_type"`
    UserMessage  string            `json:"user_message"`
    StartedAt    time.Time         `json:"started_at"`
    EndedAt      time.Time         `json:"ended_at"`
    Status       string            `json:"status"`
    Attributes   map[string]any    `json:"attributes,omitempty"`
    Spans        []TraceSpan       `json:"spans"`
    Summary      TraceSummary      `json:"summary"`
}
```

### 7.2 span 模型

```go
type TraceSpan struct {
    SpanID         string         `json:"span_id"`
    ParentSpanID   string         `json:"parent_span_id,omitempty"`
    Name           string         `json:"name"`
    Kind           string         `json:"kind"`
    Status         string         `json:"status"`
    StartedAt      time.Time      `json:"started_at"`
    EndedAt        time.Time      `json:"ended_at"`
    DurationMs     int64          `json:"duration_ms"`
    InputPreview   any            `json:"input_preview,omitempty"`
    OutputPreview  any            `json:"output_preview,omitempty"`
    Error          string         `json:"error,omitempty"`
    Attributes     map[string]any `json:"attributes,omitempty"`
}
```

### 7.3 span kind 约定

借 OpenInference 语义，但先本地实现：

- `AGENT`：整个聊天回合
- `CHAIN`：路由、缺失字段判断、followup 组织
- `TOOL`：`bazi_calc / yongshen / dayun_analyzer / qimen_dunjia`
- `RETRIEVER`：`knowledge_search`
- `LLM`：`llm_generate`

### 7.4 前端展示专用 digest

前端不直接吃完整 trace，而是吃摘要后的 `TraceDigest`：

```json
{
  "type": "trace-panel",
  "payload": {
    "trace_id": "trc_123",
    "turn_type": "full_reading",
    "total_ms": 8421,
    "steps": [
      {"label":"意图识别","kind":"CHAIN","status":"ok","ms":188},
      {"label":"八字排盘","kind":"TOOL","status":"ok","ms":17},
      {"label":"用神分析","kind":"TOOL","status":"ok","ms":9},
      {"label":"知识检索","kind":"RETRIEVER","status":"ok","ms":304,"meta":{"hits":5}},
      {"label":"命理解读","kind":"LLM","status":"ok","ms":7903,"meta":{"model":"deepseek-v4-pro","output_tokens":null}}
    ]
  }
}
```

这条 digest 必须来源于 `TurnTrace`，不能手写第二套数据。

---

## 8. 当前代码中的步骤映射

基于现有 `orchestrator.go`，建议映射如下：

### 8.1 root span

- `chat.turn` (`AGENT`)

### 8.2 classify 阶段

- `classify_and_extract` (`CHAIN`)

记录：

- action
- profile patch 是否为空
- `needs_qimen`
- `user_question_present`

### 8.3 ask 分支

- `ask_missing_profile` (`CHAIN`)

记录：

- missing fields
- ask text

### 8.4 full reading 分支

- `bazi_calc` (`TOOL`)
- `yongshen` (`TOOL`, optional)
- `dayun_analyzer` (`TOOL`, optional)
- `knowledge_search` (`RETRIEVER`)
- `llm_generate` (`LLM`)

### 8.5 followup 分支

- `reuse_bazi_result` (`CHAIN`)
- `qimen_dunjia` (`TOOL`, optional)
- `knowledge_search` (`RETRIEVER`, conditional)
- `llm_generate` (`LLM`)

### 8.6 direct bazi 输入分支

- `parse_direct_bazi` (`CHAIN`)
- `knowledge_search` (`RETRIEVER`)
- `llm_generate` (`LLM`)

---

## 9. 文件级改造建议

这部分写成精确落点，便于后续实施。

| 文件 | 改造建议 |
|------|----------|
| `internal/tracing/tracing.go` | 保留接口名，但补齐可收集 trace 数据的实现能力 |
| `internal/tracing/noop.go` | 保留 noop 作为 fallback |
| `internal/tracing/` 新增实现文件 | 增加 `TurnTrace`、`TraceSpan`、collector、file sink |
| `internal/tracing/middleware.go` | 注入 request 级 trace context 和 request id |
| `internal/handler/chat.go` | 把 debug txt 改成 JSON Lines 或 JSON 文件；追加 trace id；不要再用 `%v` 打 map |
| `internal/orchestrator/orchestrator.go` | 给 classify / ask / tool / retriever / llm / fallback 全部补 span |
| `internal/llm/client.go` | 记录模型、开始结束时间；若流式协议里可解析 usage，则补 token |
| `web/src/types/chat.ts` | 增加 `trace-panel` / `trace-summary` payload 类型 |
| `web/src/composables/useSSE.ts` | 识别新的 `component` 载荷类型 |
| `web/src/components/ChatBubble.vue` | 渲染 trace panel / trace summary |
| `web/src/components/TracePanel.vue` | 新增：回合级步骤时间线 |
| `web/src/components/TraceSummaryCard.vue` | 新增：总耗时 + 核心步骤卡片 |

---

## 10. 分阶段实施

### 阶段 0：修正 debug 可读性

**目标：** 先让本地输出可读、可检索、可 diff

**做法：**

- `logs/debug/*.txt` 改为 JSON Lines 或 `logs/debug/*.json`
- 每条 SSE event 用 `json.Encoder` 写入
- 补字段：
  - timestamp
  - session_id
  - trace_id
  - event_type
  - payload

**验收：**

- 中文不再乱码
- 日志可被 `jq` 直接分析

### 阶段 1：建立 `TurnTrace`

**目标：** 把运行轨迹从“事件流”升级成“回合级 trace”

**做法：**

- 非 noop tracer 实现
- root trace + child spans
- 文件落盘到 `logs/traces/{date}/{trace_id}.json`

**验收：**

- 一次聊天对应一份 trace 文件
- trace 中有统一 trace id、时间、状态、span 树

### 阶段 2：把 trace digest 推到前端

**目标：** 让前端看到结构化步骤，而不是散乱文字

**做法：**

- SSE 仍保持 6 种事件类型不变
- 通过 `component` 增加两类 payload：
  - `trace-summary`
  - `trace-panel`

**验收：**

- assistant 回复下方有“过程摘要”
- 可展开查看步骤时间线
- 步骤来自真实 trace，不是前端临时拼接

### 阶段 3：提供本地调试接口

**目标：** 方便开发时按 `trace_id` 回看完整轨迹

**做法：**

- `GET /api/debug/traces/:trace_id`
- 仅在 debug 开关开启时暴露

**验收：**

- 可以从前端或命令行直接回看某次完整执行

### 阶段 4：预留 exporter 适配层

**目标：** 后续接 Phoenix / Langfuse / OTLP 时不推翻本地模型

**做法：**

- 增加 exporter interface
- 初期只实现 `file`
- 后续补 `otlp`

**验收：**

- 本地 trace 模型无需重写，只新增导出器

---

## 11. 环境变量建议

在现有 `DEBUG_HTTP` 基础上，建议新增：

| 变量 | 含义 | 默认值 |
|------|------|--------|
| `DEBUG_TRACE` | 是否落盘完整 turn trace | `0` |
| `TRACE_UI` | 是否通过 SSE 输出 trace panel / summary | `1`（开发期） |
| `TRACE_EXPORTER` | `none / file / otlp` | `file` |
| `TRACE_OTLP_ENDPOINT` | 第二阶段 OTLP 地址 | 空 |

说明：

- `DEBUG_HTTP` 继续控制原始事件调试输出
- `DEBUG_TRACE` 控制回合级 trace 持久化
- 生产环境可关闭 `DEBUG_HTTP`，保留 `TRACE_EXPORTER`

---

## 12. 产品面与调试面的边界

这是这轮最重要的约束之一。

### 前端产品面可以展示

- 步骤名称
- 步骤耗时
- 是否降级
- 检索命中数量
- 使用的模型名
- 是否复用已有命盘

### 前端产品面不展示

- 原始 system prompt
- 原始 chain-of-thought
- 完整工具输入参数
- 完整知识库原文
- 任何可能泄漏内部实现细节的错误堆栈

### 调试面可以额外保留

- `input_preview`
- `output_preview`
- 降级原因
- error string
- trace attributes

---

## 13. 验收标准

### 后端验收

1. 一次 `/api/chat` 请求结束后，若 `DEBUG_TRACE=1`，会生成一份 `logs/traces/*.json`
2. trace 文件中包含：
   - `trace_id`
   - `session_id`
   - `turn_type`
   - `status`
   - `spans[]`
3. `knowledge_search` 失败时，trace 能明确记录为降级，而不是静默消失
4. `qimen_dunjia` 失败回退八字时，trace 能明确记录 fallback
5. LLM span 至少记录 model 和 duration；token 拿不到时写 `null`，不伪造

### 前端验收

1. assistant 消息下方显示 trace summary
2. 用户可展开 trace panel 查看完整步骤
3. 没有知识检索、没有奇门、没有用神时，面板不会展示伪步骤
4. 错误场景下，trace panel 能显示哪个步骤失败

### 架构验收

1. 不新增新的 SSE 事件种类，继续使用既有 6 类
2. 不暴露模型原始 thinking / CoT
3. 本地 trace 模型可映射到 OpenInference / OTel 语义

---

## 14. 不做的事

- 不在这轮直接接入 Langfuse SaaS
- 不在这轮直接接入 Phoenix Collector
- 不在这轮做 metrics / dashboard / cost center
- 不在这轮做多会话对比分析 UI
- 不把 trace 文件直接暴露给前端静态读取

---

## 15. 为什么这才是“可探测 Agent”

因为它解决的不是“多一个日志目录”，而是这四个问题：

1. **解释性**：这轮 agent 到底做了什么
2. **诊断性**：失败在哪一步、降级在哪一步
3. **一致性**：debug、UI、未来 exporter 用的是同一份轨迹
4. **可演进性**：以后要接 Langfuse / Phoenix，不需要推翻已有实现

这才是当前项目真正需要的“可探测 agent 方案”。

---

## 16. 建议的落地顺序

如果下一步要实施，建议严格按下面顺序：

1. 先修 `chat.go` 的结构化 debug 输出
2. 再实现 `TurnTrace` 和 file sink
3. 再把 digest 通过 `component` 推给前端
4. 最后再补 debug trace API 和 OTLP exporter

不要反过来做。  
如果先做前端 UI，再回头补 trace 模型，最后一定会返工。
