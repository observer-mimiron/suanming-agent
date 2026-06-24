# 原生实现 vs Eino 接入：学习引导

**日期：** 2026-06-13  
**适用阶段：** 当前仓库停在 `Go 主控 runtime + Eino 渐进接入`  
**一句话结论：** 现在正好适合停下来学，因为原生路径还完整可读，Eino 路径也已经接进来了，但还没有把业务主链完全包进框架。

---

## 1. 为什么现在是一个合适的学习断点

如果再继续往前推，比如把更多编排改成 `compose.Graph`，或者把更多调度逻辑塞进 ADK / callbacks，你看到的就不再是“原生代码怎么做”与“框架替你做了什么”的清晰对比，而会变成“整个系统都已经框架化”。

现在这个版本的好处是：

- 原生主链还在，Go 的控制面没有丢
- Eino 已经接到几个最有代表性的基础设施点
- 同一个仓库里能同时看到“手写做法”和“框架做法”
- 还保留了切换开关，可以自己做实验

所以如果你的目标是**先学原生，再学框架到底替你省了什么**，现在就可以先停手。

---

## 2. 先建立一个总图

先记住一句话：

**这个项目现在不是“Eino 项目”，而是“Go 主控项目，局部接入 Eino”。**

你可以把系统分成两层：

### A. Go 主控层

这层负责“业务控制”和“产品协议”：

- 会话状态怎么存
- 本轮应该走哪条业务分支
- 哪个工具什么时候调用
- SSE 给前端发什么事件
- trace digest 怎么组织给前端看

这层的关键词是：**确定性、可控、跟业务强绑定**

### B. Eino 基础设施层

这层负责“模型调用相关的通用能力”：

- ChatModel 封装
- tool schema / tool calling 兼容
- supervisor 的一层 structured route 承载
- callback 级别的 LLM tracing

这层的关键词是：**通用化、可复用、减少手写底座**

---

## 3. 原生实现主要在哪里看

如果你要先学“没有框架时，这个系统本来怎么工作”，按下面顺序读。

### 第一站：容器装配

看 `internal/container/container.go`

这里是整个系统的装配入口。你会看到：

- 主回答模型怎么建
- flash 路由模型怎么建
- tool registry 怎么注册
- orchestrator / supervisor / tracer 怎么接起来

建议你先只回答一个问题：

**系统有哪些对象是程序员自己明确创建并接线的？**

这个问题回答清楚了，后面你就不容易把“业务主控”和“框架能力”混在一起。

### 第二站：主编排

看 `internal/orchestrator/orchestrator.go`

这部分最值得学，因为它体现了原生实现的核心思想：

- route 是业务语义，不是框架节点
- 工具调用时机由 Go 明确决定
- prompt 拼装、知识检索、结果流式输出都在可见的代码里

这里你重点看：

- `Run()`
- `executeRoute(...)`
- `streamInterpretation(...)`

你要建立的认知是：

**原生实现的优势不是“代码少”，而是控制面一眼可见。**

### 第三站：原生 supervisor

看 `internal/supervisor/client.go`

这里最适合学“为什么很多团队一开始都手写 supervisor”。

你会看到三层语义：

- `structuredDecide`
- `textDecide`
- `safeFallback`

这部分很关键，因为它说明：

**真实业务里，retry 不等于 fallback。**

框架通常容易帮你做“重试”，但不一定天然等价于你业务上的“多层降级”。

### 第四站：原生工具系统

看：

- `internal/tools/registry.go`
- `internal/tools/bazi/calc.go`
- `internal/tools/yongshen.go`
- `internal/tools/dayun_analyzer.go`
- `internal/tools/qimen_tool.go`

你要理解的是：

- 工具接口很简单
- 工具调用不是 model-decide，而是 route-decide
- tool 本身是业务能力，不是 agent 自主性的象征

### 第五站：原生 tracing

看：

- `internal/tracing/real_tracer.go`
- `internal/tracing/turn_trace.go`
- `internal/tracing/middleware.go`

这部分要学的是：

**这个项目的 tracing 是产品和调试共用的一套业务 trace，不是单纯的底层 telemetry。**

---

## 4. Eino 主要接入了什么

理解完原生后，再看 Eino 接入层。这样你会更容易看出“省掉了哪些手写底座”，而不是只记住一堆框架 API。

### 第一块：LLM 底座

看：

- `internal/llm/factory.go`
- `internal/llm/eino_chat.go`

这里的学习重点是：

- `eino_chat.go` 代表“保持旧接口不变，但底层换成 Eino ChatModel”
- `factory.go` 代表“把底层模型创建集中到一个固定的 Eino 工厂里”

你要重点观察的不是语法，而是边界：

**上层依然只认 `llm.Chat`，Eino 只是底层实现之一。**

这就是一个很典型的“先换底座，不动上游调用方”做法。

### 第二块：工具兼容层（已收口）

看 `internal/tools/registry.go`

这一步在 Phase 2 已经收口：`eino_adapter.go` 被删除，工具不再对外暴露 Eino 兼容接口。

目前 tools 只保留 `registry.Get` / `registry.List`，由 Go runtime 显式调度。

值得留意的设计思路：

**Phase 2 证明了一点："框架兼容"不等于"模型自治"。**

先做兼容层是正确的阶段性选择——它让你确认 tools 的 schema 统一、调用边界清晰。
当发现现阶段不需要模型自主调度时，干干净净删掉兼容层，不会伤到上层业务。

如果你来自习惯"一步到位把全部 tools 塞给 agent"的团队，这个先接兼容层再收口的节奏很值得体会。


看：

- `internal/supervisor/client.go`
- `internal/supervisor/adk_engine.go`
- `internal/container/container.go`

这部分是目前最有学习价值的一块。

原因是它不是“全量改写”，而是“局部替换”：

- `RouteEngine` 成了一个可插拔边界
- classic path 已移除；ADK 是唯一的 route engine
- ADK path 只负责 layer-1 structured route
- Go 外层 fallback 语义还保留

这能帮助你真正理解：

**框架最适合先接在“可抽象的局部内核”上，而不是一上来吞掉整段业务语义。**

### 第四块：Tracing callback

看：

- `internal/tracing/eino_callback.go`
- `internal/orchestrator/orchestrator.go`
- `internal/supervisor/client.go`
- `internal/supervisor/adk_engine.go`

这一块要学的不是 callback API 本身，而是设计取舍：

- 现有 `TurnTrace` 不推翻
- 前端 `TracePanel` 不改协议
- 只把 Eino 底层 ChatModel span 回填进现有 trace
- 为了避免双记，还会关闭对应手工 `llm_generate` span

这里体现的是一个很成熟的思路：

**框架接入不是替换所有东西，而是优先复用已经稳定的产品数据模型。**

---

## 5. 你应该怎么理解“原生”和“框架”的区别

最简单的理解方式不是“谁代码多谁代码少”，而是比较下面四个维度。

| 维度 | 原生实现 | Eino 接入 |
|---|---|---|
| 控制面 | 全在 Go 代码里，最透明 | 抽象层更高，但会隐藏一部分细节 |
| 业务语义 | 你自己定义最清楚 | 框架能承载，但不天然懂你的降级语义 |
| 通用能力 | 需要自己补 | ChatModel / Tool / Callback 这类更省力 |
| 调试方式 | 读代码最直接 | 需要同时理解框架生命周期 |

再换成一句更直白的话：

- **原生实现更适合学“系统到底怎么跑”。**
- **框架实现更适合学“哪些重复劳动可以不自己写”。**

所以你的学习顺序最好永远是：

**先理解原生控制面，再理解框架替换点。**

---

## 6. 这个仓库里最值得你重点观察的 4 个对照点

### 对照点 1：模型调用

- Eino：`internal/llm/eino_chat.go` + `internal/llm/factory.go`

看点：

- 消息怎么转换
- stream 怎么处理
- tool choice 怎么做
- token usage 怎么拿

### 对照点 2：structured route

- 原生：`internal/supervisor/client.go`
- Eino：`internal/supervisor/adk_engine.go`

看点：

- 原生怎么组织多层 fallback
- ADK 怎么承载一层 structured route
- 为什么 `ModelRetryConfig` 不等于整个业务 fallback 语义

### 对照点 3：工具系统

- 原生：`internal/tools/registry.go` + 各 tool 的 `Execute`
- Eino：已收口（Phase 2 删除了 eino_adapter.go）；tools 只保留 Go-native interface

看点：
  - Eino：已收口（Phase 2 删除了 eino_adapter.go）；tools 只保留 Go-native interface
- 原生工具为什么足够简单
- 为什么先做兼容层，而不是立即 model-decide

### 对照点 4：trace

- 原生：`internal/tracing/real_tracer.go` / `turn_trace.go`
- Eino：`internal/tracing/eino_callback.go`

看点：

- 原生 trace 怎么围绕业务步骤建模
- callback span 怎么嵌回现有业务 trace

---

## 7. 推荐你的实际阅读顺序

如果你想高效一点，我建议这样读。

### 第一轮：只读原生主链

顺序：

1. `internal/container/container.go`
2. `internal/orchestrator/orchestrator.go`
3. `internal/supervisor/client.go`
4. `internal/tools/registry.go`
5. `internal/tracing/real_tracer.go`
6. `internal/tracing/turn_trace.go`

目标：

你能回答这三个问题就够了：

- 一条请求从进来到出去，Go 是怎么控制的？
- tool 是谁决定调用的？
- trace 为什么是业务步骤视角而不是底层 API 视角？

### 第二轮：只读 Eino 覆盖层

顺序：

1. `internal/llm/factory.go`
2. `internal/llm/eino_chat.go`
3. `internal/tools/registry.go` —— tools 已收口
4. `internal/supervisor/adk_engine.go`
5. `internal/tracing/eino_callback.go`
3. `internal/tools/registry.go` —— tools 已收口，不再维护 Eino 适配层

目标：

你能回答这三个问题就够了：

- Eino 现在到底接管了哪几块？
- 哪些业务语义仍然留在 Go 里？
- 为什么 current state 不是“全量 Eino 化”？

### 第三轮：做切换实验

这是最有帮助的一轮。

3.  —— tools 已收口，不再维护 Eino 适配层

你重点观察：
现在 `SUPERVISOR_ENGINE` 配置和 `classic|adk` 切换都已删除，路由路径只有一条。

如果你仍想深入理解 ADK route engine，可以尝试：
- 在 container.go 中注释掉 ADK route engine 的构建，观察 `textDecide → safeFallback` 降级链是否正常
- 观察 prompt `unified_router.md` 中的 output tool contract 与 ADK run 的配合


---

## 8. 你学完后应该得到的核心认识

如果这篇文档起作用，最后你应该能形成下面几个判断。

### 判断 1

框架最先替换的应该是**通用基础设施**，不是最先替换你的业务主脑。

### 判断 2

对于这种命理咨询项目，`route`、`policy gate`、`tool 调用时机`、`session state` 都是强业务语义，保留在 Go 里是合理的。

### 判断 3

Eino 现在最明显的价值，不是“让系统更智能”，而是：

- 少写模型调用底座
- 少写 tool 兼容层
- 少写 callback 接线
- 给后续更深迁移预留标准接口

### 判断 4

`ChatModelAgent` 这种能力很适合先承载一个**局部内核**，而不是直接把整个 supervisor 的多层业务语义都吞进去。

---

## 9. 我对你下一步学习方式的建议

如果你是想真正把东西学明白，而不是只想把项目继续往前推，我建议下一步不要继续开发功能，先做两件事：

### 第一件事：自己画一张“当前边界图”

就画两列：

- 左边：Go owns
- 右边：Eino owns

你只要能把这张图自己画出来，说明你已经真的理解了当前架构。

### 第二件事：自己解释一遍“为什么 Phase 3 只替换 layer-1”

如果你能不用看文档，自己讲清楚下面这句话，你就已经吃透这轮迁移了：

**“因为 retry 不是 fallback，ADK 可以先承载 structured route，但不能假装它天然等价于整个业务降级链。”**

---

## 10. 最后给你的学习建议

如果你的目标是学习，而不是立刻把项目全量框架化，那么我建议：

**现在就停在这里是对的。**

因为当前状态刚好同时满足三件事：

- 原生主链还看得见
- Eino 替换点已经足够典型
- 差异还没有被后续更复杂的 graph / agent 编排淹没

等你把这几块吃透，再继续推进 Phase 4，收获会更大。
