# 11 Graph 编排方案（排盘 + 执行骨架）

> **Status:** Design (pre-implementation)
> **Date:** 2026-06-25（修订：加入执行骨架 Graph）
> **关联:** [2026-06-24-eino-native-capabilities.md](../../superpowers/plans/2026-06-24-eino-native-capabilities.md) — Eino 原生能力增补方案

## 0. 方案边界：什么不 Graph 化

原方案（2026-06-24 初版）将 Graph 用于 dispatch 层，意图将 Supervisor Agent 的专家调度从 LLM 自由选择（L3）降为 Go 确定性 pipeline（L2）。该方案**已驳回**。

### 驳回原因

聊天型命理系统的对话形态不适合 Graph 做 dispatch：

- 用户追问是上下文相关的——"那我财运呢"依赖上轮排盘结果
- 用户可能中途修正信息——"等等我记错时辰了"
- 用户可能跨领域提问——"八字和紫微哪个准"

这些都不是固定流程，每一轮 dispatch 决策需要理解**当前对话上下文和用户意图变化**。LLM 在 ApprovedRoute 约束内做 specialist 选择不是浪费——它就是缓存了"理解上下文"这一步。强行 Graph 化等于把灵活性条文化。

**结论：dispatch 层保持 L3（AgentAsTool），不迁移到 Graph。specialist 内部保持 L4（ReAct）。**

---

## 1. 自主性级别

| 层 | 级别 | Graph? | 说明 |
|---|---|---|---|
| **执行骨架** | **L2（Graph）** | **是** | preflight → prefill → agent → final_guard，确定性关卡编排 |
| dispatch 路由 | L3（不动） | 否 | AgentAsTool，LLM 在 ApprovedRoute 约束内选 specialist |
| **排盘** | **L2（Graph）** | **是** | birthTime → 节气 → 四柱 → 十神 → 大运，确定性计算管道 |
| specialist 内部 | L4（不动） | 否 | ReAct 循环，LLM 自主选工具 |

**2026-06-25 修订新增"执行骨架"层**：之前只规划了排盘 Graph，现在把 executor.go 的 `Execute` 流程也显式 Graph 化。两层 Graph 嵌套：执行骨架的 `prefill` 节点 = 排盘子图（`compose.AddGraphNode`）。

**一句话：外层 Graph 定骨架（确定性关卡），内层 Agent 做推理（LLM 自主区间）。关卡走到 Agent 区间时交出决策权，Agent 完成后回到关卡做防护。**

---

## 2. 排盘 Graph（确定性计算管道）

排盘是纯确定性数据流：birthInfo → 节气计算 → 四柱 → 十神 → 大运。无 LLM 参与，天然适合 Graph。

### Graph 拓扑

```
START → birthTime 解析 → 节气计算 → 四柱排定 → 十神推算 → 大运起运 → END
```

每个节点是一个 `compose.InvokableLambda`，调用 lunar-go 的对应函数。节点间通过代码门控（非空校验）传递。

### 伪代码

```go
func buildPaipanGraph() (*compose.Graph[*paipanInput, *paipanOutput], error) {
    g := compose.NewGraph[*paipanInput, *paipanOutput]()

    g.AddLambdaNode("parse_time",    compose.InvokableLambda(parseBirthTime),
        compose.WithNodeName("paipan.parse_time"))
    g.AddLambdaNode("solar_term",    compose.InvokableLambda(calcSolarTerm),
        compose.WithNodeName("paipan.solar_term"))
    g.AddLambdaNode("sizhu",         compose.InvokableLambda(buildSiZhu),
        compose.WithNodeName("paipan.sizhu"))
    g.AddLambdaNode("shishen",       compose.InvokableLambda(calcShiShen),
        compose.WithNodeName("paipan.shishen"))
    g.AddLambdaNode("dayun",         compose.InvokableLambda(calcDaYun),
        compose.WithNodeName("paipan.dayun"))

    g.AddEdge(compose.START, "parse_time")
    g.AddEdge("parse_time", "solar_term")
    g.AddEdge("solar_term", "sizhu")
    g.AddEdge("sizhu", "shishen")
    g.AddEdge("shishen", "dayun")
    g.AddEdge("dayun", compose.END)

    return g.Compile(ctx,
        compose.WithGraphName("paipan"),
        compose.WithNodeTriggerMode(compose.AllPredecessor),
    )
}
```

### 为何 Graph 而不是纯函数调用

当前 prefill 方法已经做了同样的事。Graph 化的增量价值：

1. **Trace 覆盖** — 每个阶段自动产出 span（`paipan.parse_time`、`paipan.sizhu` 等），不需要手工埋点
2. **编译期拓扑保证** — AllPredecessor 保证阶段顺序，不会出现 sizhu 在 solar_term 之前执行
3. **可插拔** — 后续新增"真太阳时校正"节点只需 AddLambdaNode + 插入边，不动现有节点
4. **Checkpoint 就绪** — 如果需要暂停等用户确认，Graph 中断点天然支持

---

## 3. 执行骨架 Graph（外层编排）

### 现状

`executor.go` 的 `Execute` / `runAgentRoute` 已经在做 Graph 节点的事，只是用嵌套 if + 顺序调用表达：

```
当前实际流程：
  preflight 校验 ──→ 短路返回（澄清/缺资料）
       │
       └──→ prefill 确定性排盘（Go 原生，不走 LLM）
                │
                └──→ buildSupervisor + AgentTool specialists（Agent Loop）
                         │
                         └──→ agentEventBridge + final_guard
```

这是一个天然的"分支 + 线性链"骨架。用 `compose.NewGraph` 显式表达。

### 拓扑

```
START → preflight ──branch──┬─ short_circuit → END（emit 澄清/缺资料文本）
                             └─ prefill → agent → final_guard → END
```

- **preflight**：确定性校验（八字没生日 → 直接问）。分支节点，`compose.NewGraphBranch`。
- **prefill**：确定性排盘（`lunar-go`），不经过 LLM。**此节点 = 排盘子图**（§2 的 paipanGraph，通过 `AddGraphNode` 嵌入）。
- **agent**：LLM 自主推理（supervisor + AgentTool specialists）。**`compose.StreamableLambda`**，包装 ADK Runner。
- **final_guard**：输出质量校验、领域边界检查。确定性 Lambda。

### 状态共享

`compose.WithGenLocalState` 生成 per-request `orchestrationState{ SessionState, ApprovedRoute, vals, sink }`。各节点通过 `WithStatePreHandler` / `WithStatePostHandler` 读写，不走 Graph 边的类型传递。

### 流式

Graph 以 `r.Stream(ctx, input)` 调用，所有节点跑 Transform 模式。`agent` 节点是 `StreamableLambda[string]`：

1. 内部调 `runner.Run(ctx, msgs, opts...)` → `*AsyncIterator[*AgentEvent]`
2. 消费事件：`agentEventBridge` 逻辑路由到 SSE sink（侧信道捕获于 state）+ 累积 final text
3. 写 final text chunks 到 `*schema.StreamWriter[string]`

**关键：`agentEventBridge` 的 295 行业务规则（specialist 去重、chart 派发、`<analysis>/<response>` 拆分）不下放——仍在 agent Lambda 内。** Graph 只提供骨架，不替代业务规则。这与 [eino-native-capabilities.md] 的"框架原生能力盘点"结论一致。

### Checkpoint（连接 C1）

节点边界天然是中断点。配 `WithCheckPointStore` 后：

- prefill 后、agent 前可中断 → "请您确认出生时间是否为真太阳时" → 用户回复后 `runner.ResumeWithParams` 继续
- 这是 [eino-native-capabilities.md] C1 的落地路径——之前 C1 标注"排盘 Graph 化后评估"，执行骨架 Graph 化后这条路打通

### 诚实评估

| 维度 | Graph 化的收益 | 现状是否够用 |
|------|--------------|-------------|
| 流程清晰度 | 显式拓扑 > 嵌套 if | **够用**——当前 executor.go 可读 |
| Trace 覆盖 | 节点自动 span | 够用——现有手工 span |
| 状态共享 | State Graph 显式 | 够用——closure 传递 vals |
| **Checkpoint 中断** | **节点边界 = 中断点** | **不够**——这是 C1 的前提 |
| **未来复杂分支** | **AddBranch 比 if 清晰** | **前瞻**——qimen 补充/紫微闰月路径 |

**结论：当前流程（单分支 + 线性链）Graph 化主要是结构清晰度收益，cosmetic 成分高。真实收益在 Checkpoint（C1 前提）和未来复杂分支。建议分阶段推进，Phase 1 纯重构验证拓扑，不急着加 Checkpoint。**

### 分阶段

| 阶段 | 目标 | 行为变化 | 前置 |
|------|------|---------|------|
| Phase 1 | executor.go 重构为 Graph，prefill 节点嵌入排盘子图 | **无**（纯重构，验证拓扑 + 回归） | 排盘 Graph（§2）落地 |
| Phase 2 | 配 CheckPointStore，prefill→agent 边界可中断 | 新增 C1 能力 | Phase 1 稳定 |
| Phase 3 | qimen 补充信息路径、紫微闰月处理作为分支 | 新增领域分支 | Phase 1 + 领域需求驱动 |

### 伪代码

```go
type orchestrationState struct {
    st    *state.SessionState
    route policy.ApprovedRoute
    vals  map[string]any
    sink  runtime.EventSink
}

func buildOrchestrationGraph(paipanGraph compose.AnyGraph) (compose.Runnable[string, string], error) {
    g := compose.NewGraph[string, string](
        compose.WithGenLocalState(func(ctx context.Context) *orchestrationState {
            return &orchestrationState{}
        }),
    )

    // preflight: 确定性校验 + 分支
    g.AddLambdaNode("preflight", compose.InvokableLambda(preflightNode))
    g.AddBranch("preflight", compose.NewGraphBranch(
        func(ctx context.Context, _ string) (string, error) {
            st := getOrchestrationState(ctx)
            result := preflight(st.st, st.route, st.userMsg)
            if result.ShortCircuit {
                return "short_circuit", nil
            }
            return "main", nil
        },
        map[string]bool{"short_circuit": true, "main": true},
    ))

    // short_circuit: 直接 emit 文本
    g.AddLambdaNode("short_circuit", compose.InvokableLambda(emitShortCircuitNode))
    g.AddEdge("short_circuit", compose.END)

    // main: prefill（排盘子图）→ agent → final_guard
    g.AddGraphNode("prefill", paipanGraph, compose.WithNodeName("orchestration.prefill"))
    g.AddEdge("main", "prefill")
    g.AddLambdaNode("agent", compose.StreamableLambda(agentNode), compose.WithNodeName("orchestration.agent"))
    g.AddEdge("prefill", "agent")
    g.AddLambdaNode("final_guard", compose.InvokableLambda(guardNode), compose.WithNodeName("orchestration.guard"))
    g.AddEdge("agent", "final_guard")
    g.AddEdge("final_guard", compose.END)

    return g.Compile(ctx, compose.WithGraphName("orchestration"))
}
```

`agent` 节点内部桥接（伪代码）：

```go
func agentNode(ctx context.Context, in string) (*schema.StreamReader[string], error) {
    st := getOrchestrationState(ctx)
    supervisor, _ := b.BuildSupervisor(ctx, st.route, st.st, allowed)
    runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: supervisor, EnableStreaming: true})
    iter := runner.Run(ctx, msgs, adk.WithSessionValues(st.vals))

    sr, sw := schema.Pipe[string](64)
    go func() {
        defer sw.Close()
        finalText, _ := agentEventBridge(ctx, st.sink, iter, st.saveToolResult, st.labelFor, st.bufferFinal)
        sw.Send(finalText, nil)
    }()
    return sr, nil
}
```

---

## 4. 边界定义

| 组件 | 归属 | 说明 |
|------|------|------|
| **执行骨架 Graph** | **新文件 `orchestration_graph.go`** | preflight/prefill/agent/guard 节点编排 |
| 排盘 Graph | `paipan_graph.go` | 作为执行骨架的 prefill 子图嵌入（`AddGraphNode`） |
| dispatch 路由 | 不动 | AgentAsTool，现有逻辑保持 |
| specialist 内部 | 不动 | ChatModelAgent + ReAct |
| `agentEventBridge` | 移入 agent 节点 Lambda | 业务规则保留，不下放 |
| SSE 事件流 | agent 节点侧信道 sink（state 内） | Graph 边传 final text，sink 不走边 |
| trace | eino_callback.go | 执行骨架节点 span 由 Graph callback 自动产出 |

执行骨架 Graph 的调用点：`executor.Execute` 改为构建（或复用预编译）orchestration Runnable，调 `r.Stream(ctx, message)`。`runAgentRoute` 方法体拆为各节点 Lambda。

---

## 5. 实施计划

| 阶段 | 任务 | 范围 | 风险 |
|------|------|------|------|
| **前置** | 排盘 Graph（§2）落地 | `paipan_graph.go` + prefill 接入 | **低**（纯函数） |
| **Phase 1** | 新建 `orchestration_graph.go` | 节点 Lambda + 拓扑编译 | **中**（重构 executor） |
| **Phase 1** | 排盘子图嵌入 prefill 节点 | `AddGraphNode` | 低 |
| **Phase 1** | `executor.Execute` 改调 `graph.Stream` | 替换 `runAgentRoute` | **中**（回归测试，行为须不变） |
| **Phase 1** | 回归测试 | agent test suite + SSE 事件序列断言 | **中** |
| **Phase 2** | CheckPointStore 实现（Redis） | C1 前提 | **中** |
| **Phase 2** | prefill→agent 边界中断 + Resume | C1 落地 | **中** |
| **Phase 3** | qimen 补充/紫微闰月分支 | 领域扩展 | 低（增量） |

---

## 6. 决策记录

| 决策 | 选择 | 理由 |
|------|------|------|
| dispatch 层 Graph 化 | **驳回** | 聊天系统需要上下文感知路由，Graph 条文化丧失灵活性 |
| 排盘 Graph 化 | **保留** | 确定性管道，天然适合 Graph |
| **执行骨架 Graph 化** | **保留（分阶段）** | 外层骨架确定性 + Checkpoint 前提（C1）+ 未来分支可扩展 |
| specialist 层 | **不动**（L4 ReAct） | LLM 自主选工具是价值所在 |
| AgentAsTool | **不动** | 在 ApprovedRoute 约束下，这是 dispatch 的正确方案 |
| `agentEventBridge` 下放 | **不下放** | 295 行业务规则（specialist 去重/chart 派发/XML 拆分），Graph 只提供骨架 |
| Phase 1 是否加 Checkpoint | **不加** | 先纯重构验证拓扑 + 回归，Checkpoint 留给 Phase 2 |

---

## Related

- [01-overview.md](01-overview.md) — 架构总图
- [2026-06-24-eino-native-capabilities.md](../../superpowers/plans/2026-06-24-eino-native-capabilities.md) — Eino 原生能力增补方案（B1/B2 已落地，A2 TurnLoop + C1 Checkpoint 待做）
