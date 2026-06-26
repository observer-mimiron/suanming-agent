# 执行骨架 Graph 实施方案

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `executor.go` 的执行流程从嵌套 if + 顺序调用重构为 Eino Graph 拓扑，并叠加 Checkpoint 中断-恢复能力（C1：排盘后追问确认再继续 agent）。

**Architecture:** 外层 Graph 定骨架 + 内层 Agent 做推理。Graph 节点承担确定性关卡（preflight 校验 / prefill 排盘 / final_guard 输出防护），agent 节点承载 LLM 自主推理（Supervisor + AgentTool specialists）。`agentEventBridge` 345 行业务规则整体保留在 agent 节点的 `StreamableLambda` 内，**不下放**到 Graph 边。dispatch 路由（L3 AgentAsTool）和 specialist 内部（L4 ReAct）不在本方案范围。

**Tech Stack:** Go, Eino compose (`NewGraph` / `AddBranch` / `NewGraphBranch` / `InvokableLambda` / `StreamableLambda` / `WithCheckPointStore` / `WithInterruptBeforeNodes`), ADK Runner, Redis（Checkpoint 存储）, 现有测试套件。

---

## 0. 前置准备

执行前必须完成：

- [ ] **环境:** 在 `/Users/wikiglobal/workSapce/suanming-agent` 目录下工作（不是 lisense 工作区）
- [ ] **分支:** 不在 `dev` / `master` / `main` / `prd` / `production` 受保护分支上。若是，先切功能分支：`git checkout -b feat/orchestration-graph`
- [ ] **依赖:** 本地 Redis 运行在 `localhost:6379`（Phase 2 需要）。`redis-cli ping` 返回 PONG
- [ ] **Eino 源码可读:** `eino-agent/eino/` 目录存在，是项目本地 fork（go.mod 标 v0.9.6）
- [ ] **现有测试基线:** `cd /Users/wikiglobal/workSapce/suanming-agent && go test ./internal/runtime/ -v -count=1` 全绿——记录这个 baseline，重构后必须保持全绿
- [ ] **CodeGraph 索引:** `.codegraph/` 存在，用 `codegraph_*` MCP 工具查代码结构（比 grep 快）
- [ ] **阅读现有代码:**
  - [executor.go](../../../internal/runtime/executor.go)（458 行，要重构的对象）
  - [bridge.go](../../../internal/runtime/bridge.go)（345 行，`agentEventBridge` 不下放但要迁入 agent Lambda）
  - [preflight.go](../../../internal/runtime/preflight.go)（180 行，纯函数迁入 preflight Lambda）
  - [observability.go:35](../../../internal/runtime/observability.go:35)（`guardFinalAnswerWithTrace`，迁入 guard Lambda）

---

## 1. 范围

| 项目 | 状态 | 说明 |
|------|------|------|
| 执行骨架 Graph（preflight / prefill / agent / guard） | **本方案** | 外层确定性关卡编排 |
| Checkpoint 中断-恢复（C1） | **本方案**（Phase 2） | prefill 后、agent 前可中断 |
| paipan 子图 | **不做** | 底层工具 `bazi_calc` / `ziwei_calc` / `qimen_dunjia` 返回完整命盘，不暴露 parse_time → 节气 → 四柱 → 十神 → 大运 的中间阶段。Graph 化需先分解工具（不在范围）。prefill 节点直接调现有 `executor.prefill` |
| dispatch 路由（L3） | **不动** | AgentAsTool，LLM 在 ApprovedRoute 约束内选 specialist |
| specialist 内部（L4） | **不动** | ReAct 循环，LLM 自主选工具 |
| `agentEventBridge` | **不下放** | 345 行业务规则（specialist 去重、chart 派发、XML 拆分）留在 agent Lambda 内 |

---

## 2. 文件结构

| 文件 | 操作 | 责任 |
|------|------|------|
| `internal/runtime/orchestration_state.go` | **新建** | `orchestrationState` 结构、`getOrchestrationState` / `withOrchestrationState` ctx helper |
| `internal/runtime/orchestration_graph.go` | **新建** | Graph 拓扑编译、节点 Lambda 定义 |
| `internal/runtime/checkpoint_store.go` | **新建**（Phase 2） | Redis-backed `CheckPointStore` 实现 |
| `internal/runtime/executor.go` | **修改** | `Execute` 改调 `r.Stream`；`runAgentRoute` 方法体拆为节点 Lambda；`prefill*` / `preflight` / `guardFinalAnswerWithTrace` 保留为节点 Lambda 内部调用的纯函数 |
| `internal/runtime/bridge.go` | **不动** | `agentEventBridge` 整体移入 agent 节点 `StreamableLambda` 内调用 |
| `internal/runtime/orchestration_graph_test.go` | **新建** | 拓扑编译烟雾测试 |
| `internal/runtime/checkpoint_test.go` | **新建**（Phase 2） | 中断-恢复端到端测试 |
| `internal/runtime/executor_agent_route_test.go` | **现有** | 回归测试必须全绿（行为不变） |

---

## 3. 实施任务

> **执行顺序:** Task 1-7 为 Phase 1（纯重构，行为不变），Task 8 为回归验收门，Task 9-10 为 Phase 2（Checkpoint 能力）。Phase 1 全绿后才进 Phase 2。

### Task 1: orchestration_state.go — 状态结构 + ctx helper

**Files:**
- Create: `internal/runtime/orchestration_state.go`

- [ ] **Step 1: 定义状态结构**

```go
package runtime

import (
	"context"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// orchestrationState 是执行骨架 Graph 的 per-request 状态。
//
// 通过 ctx.Value 注入，各节点 Lambda 通过 getOrchestrationState(ctx) 读取。
// agentEventBridge 的 345 行业务规则保留在 agent Lambda 内，不下放到 Graph 边。
//
// Phase 1 用 ctx.Value 简单直接。Phase 2 Checkpoint 要求状态可序列化存储到 Redis，
// 届时需要重设计：移除 executor / sink 等非序列化引用，启用 compose.WithGenLocalState
// 让 Graph 接管 state 生命周期（见 Task 9）。
type orchestrationState struct {
	st              *state.SessionState
	route           policy.ApprovedRoute
	userMsg         string
	vals            map[string]any
	sink            EventSink
	executor        *Executor
	preflightResult preflightResult
	guardedTurnType string
}

type orchestrationStateCtxKey struct{}

func withOrchestrationState(ctx context.Context, s *orchestrationState) context.Context {
	return context.WithValue(ctx, orchestrationStateCtxKey{}, s)
}

func getOrchestrationState(ctx context.Context) *orchestrationState {
	s, _ := ctx.Value(orchestrationStateCtxKey{}).(*orchestrationState)
	return s
}
```

- [ ] **Step 2: 编译验证**

Run: `cd /Users/wikiglobal/workSapce/suanming-agent && go build ./internal/runtime/`
Expected: 编译通过。

- [ ] **Step 3: Commit**

```bash
git add internal/runtime/orchestration_state.go
git commit -m "feat(runtime): add orchestrationState struct for Graph state injection"
```

---

### Task 2: preflight 节点 Lambda + Branch

**Files:**
- Create: `internal/runtime/orchestration_graph.go`

- [ ] **Step 1: 定义 preflightNode Lambda 和 branch condition**

```go
package runtime

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
)

// preflightNode 是 Graph 的 preflight 节点：执行确定性校验，结果写入 state。
// Branch 节点根据 state 中的 result 决定走 short_circuit 还是 main。
func preflightNode(ctx context.Context, in string) (string, error) {
	s := getOrchestrationState(ctx)
	result := preflight(s.st, s.route, s.userMsg)
	s.preflightResult = result
	return in, nil
}

// preflightBranch 根据 preflightResult 决定分支。
// short_circuit: preflight 短路（澄清/缺资料）
// main: 进入 prefill → agent → guard 主路径
func preflightBranch(ctx context.Context, _ string) (string, error) {
	s := getOrchestrationState(ctx)
	if s.preflightResult.ShortCircuit {
		return "short_circuit", nil
	}
	return "main", nil
}
```

- [ ] **Step 2: 编译验证**

Run: `go build ./internal/runtime/`
Expected: 编译通过。

- [ ] **Step 3: Commit**

```bash
git add internal/runtime/orchestration_graph.go
git commit -m "feat(runtime): add preflight node Lambda and branch condition"
```

---

### Task 3: short_circuit / prefill / guard 节点 Lambda

**Files:**
- Modify: `internal/runtime/orchestration_graph.go`

- [ ] **Step 1: 实现 short_circuit 节点（emit 澄清/缺资料文本）**

```go
// emitShortCircuitNode 处理 preflight 短路：emit text 事件并返回。
// executor.go 原 Execute:80-86 的 updateGuidanceState + emit 逻辑整体移入此处。
func emitShortCircuitNode(ctx context.Context, in string) (string, error) {
	s := getOrchestrationState(ctx)
	s.executor.updateGuidanceState(s.st, s.route, s.userMsg, s.preflightResult)
	_ = emitEventWithTrace(ctx, s.sink, Event{
		Type: "text",
		Data: map[string]any{"content": s.preflightResult.Text},
	}, map[string]any{"turn_type": s.preflightResult.TurnType})
	return s.preflightResult.Text, nil
}
```

- [ ] **Step 2: 实现 prefill 节点（调现有 e.prefill，不走子图）**

```go
// prefillNode 调用现有 executor.prefill，结果写入 state.vals 和 session state。
// 不使用 AddGraphNode 嵌入子图——底层工具不暴露中间阶段，子图无意义。
func prefillNode(ctx context.Context, in string) (string, error) {
	s := getOrchestrationState(ctx)
	s.executor.prefill(ctx, s.sink, s.st, s.route, s.vals)
	return in, nil
}
```

- [ ] **Step 3: 实现 guard 节点（调现有 guardFinalAnswerWithTrace）**

```go
// guardNode 调用现有 guardFinalAnswerWithTrace，emit 最终 text。
// shouldBufferFinalAnswer()=true 时 guard 后的文本走 bufferFinal emit 路径。
func guardNode(ctx context.Context, finalText string) (string, error) {
	s := getOrchestrationState(ctx)
	turnType, guardedText := guardFinalAnswerWithTrace(ctx, s.route, s.st, finalText)
	if shouldBufferFinalAnswer() && guardedText != "" {
		_ = emitEventWithTrace(ctx, s.sink, Event{
			Type: "text",
			Data: map[string]any{"content": guardedText},
		}, map[string]any{"buffer_final": true, "turn_type": turnType})
	}
	s.guardedTurnType = turnType
	return guardedText, nil
}
```

- [ ] **Step 4: 编译验证**

Run: `go build ./internal/runtime/`
Expected: 编译通过。

- [ ] **Step 5: Commit**

```bash
git add internal/runtime/orchestration_graph.go
git commit -m "feat(runtime): add short_circuit/prefill/guard node Lambdas"
```

---

### Task 4: agent 节点 StreamableLambda（最复杂，345 行 bridge 整体迁入）

**Files:**
- Modify: `internal/runtime/orchestration_graph.go`

- [ ] **Step 1: 实现 agentNode 为 StreamableLambda**

在 `orchestration_graph.go` 顶部 import 块新增（如未有）：

```go
import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/wikiglobal/suanming-agent/internal/tracing"
)
```

然后实现 agentNode：

```go
// agentNode 是 Graph 的 agent 节点：构建 Supervisor + AgentTool specialists，
// 启动 ADK Runner，通过 agentEventBridge 桥接事件到 SSE。
//
// agentEventBridge 的 345 行业务规则（specialist 去重、chart 派发、XML 拆分、
// AgentAsTool 内联检测）整体保留在此 Lambda 内，不下放到 Graph 边。
// Graph 只提供骨架，不替代业务规则。
func agentNode(ctx context.Context, in string) (*schema.StreamReader[string], error) {
	s := getOrchestrationState(ctx)

	// ForcedRoute 覆盖（preflight 返回 ForcedRoute 时）
	route := s.route
	if s.preflightResult.ForcedRoute != nil {
		route = *s.preflightResult.ForcedRoute
	}

	// emit transition text（ForcedRoute 场景）
	if s.preflightResult.ForcedRoute != nil && s.preflightResult.Text != "" {
		_ = emitEventWithTrace(ctx, s.sink, Event{
			Type: "text",
			Data: map[string]any{"content": s.preflightResult.Text},
		}, map[string]any{"turn_type": s.preflightResult.TurnType})
	}

	// updateGuidanceState（非短路路径）
	s.executor.updateGuidanceState(s.st, route, s.userMsg, s.preflightResult)

	// 构建 Supervisor
	allConfigs := s.executor.specialistRegistry.All()
	allowed := allowedSpecialists(route, allConfigs)
	supervisor, err := s.executor.builder.BuildSupervisor(ctx, route, s.st, allowed)
	if err != nil {
		return nil, fmt.Errorf("build supervisor agent: %w", err)
	}

	// tracing callback span
	ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
		Name: "adk_supervisor_agent", Kind: tracing.KindChain,
		Attributes: map[string]any{"model": s.executor.llmModel, "domain": route.PrimaryDomain},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: supervisor, EnableStreaming: true})
	msgs := s.executor.buildConversationMessages(s.st, s.userMsg)
	iter := runner.Run(ctx, msgs, adk.WithSessionValues(s.vals))

	// Pipe: agentEventBridge 写 finalText 到 sw，Graph 边读 sr
	sr, sw := schema.Pipe[string](64)
	go func() {
		defer sw.Close()
		finalText, err := agentEventBridge(ctx, s.sink, iter, func(toolName, resultJSON string) {
			s.executor.saveToolResult(s.st, toolName, resultJSON)
		}, s.executor.reg.DisplayName, shouldBufferFinalAnswer())
		if err != nil {
			sw.Send("", err)
			return
		}
		sw.Send(finalText, nil)
	}()
	return sr, nil
}
```

- [ ] **Step 2: 编译验证**

Run: `go build ./internal/runtime/`
Expected: 编译通过。

- [ ] **Step 3: Commit**

```bash
git add internal/runtime/orchestration_graph.go
git commit -m "feat(runtime): add agent node StreamableLambda with agentEventBridge"
```

---

### Task 5: buildOrchestrationGraph — 拓扑编译

**Files:**
- Modify: `internal/runtime/orchestration_graph.go`

- [ ] **Step 1: 实现 buildOrchestrationGraph**

```go
// buildOrchestrationGraph 编译执行骨架 Graph Runnable。
//
// 拓扑:
//   START → preflight ──branch──┬─ short_circuit → END
//                                └─ prefill → agent → final_guard → END
//
// 状态通过 ctx.Value 注入 orchestrationState（见 orchestration_state.go），
// 不使用 compose.WithGenLocalState——后者创建的 state 与外部注入的 state 是两个对象，
// 会导致节点 Lambda 拿不到真实字段。ctx.Value 方式简单直接，Phase 1 够用。
// Phase 2 Checkpoint 需要真正的 State Graph 时再重设计（见 Task 9）。
func buildOrchestrationGraph() (compose.Runnable[string, string], error) {
	g := compose.NewGraph[string, string]()

	// preflight 节点 + 分支
	if err := g.AddLambdaNode("preflight",
		compose.InvokableLambda(preflightNode),
		compose.WithNodeName("orchestration.preflight")); err != nil {
		return nil, fmt.Errorf("add preflight node: %w", err)
	}
	if err := g.AddBranch("preflight", compose.NewGraphBranch(
		preflightBranch,
		map[string]bool{"short_circuit": true, "main": true},
	)); err != nil {
		return nil, fmt.Errorf("add preflight branch: %w", err)
	}

	// short_circuit 路径
	if err := g.AddLambdaNode("short_circuit",
		compose.InvokableLambda(emitShortCircuitNode),
		compose.WithNodeName("orchestration.short_circuit")); err != nil {
		return nil, fmt.Errorf("add short_circuit node: %w", err)
	}
	if err := g.AddEdge("short_circuit", compose.END); err != nil {
		return nil, fmt.Errorf("edge short_circuit->END: %w", err)
	}

	// main 路径: prefill → agent → guard
	if err := g.AddLambdaNode("prefill",
		compose.InvokableLambda(prefillNode),
		compose.WithNodeName("orchestration.prefill")); err != nil {
		return nil, fmt.Errorf("add prefill node: %w", err)
	}
	if err := g.AddEdge("main", "prefill"); err != nil {
		return nil, fmt.Errorf("edge main->prefill: %w", err)
	}

	if err := g.AddLambdaNode("agent",
		compose.StreamableLambda(agentNode),
		compose.WithNodeName("orchestration.agent")); err != nil {
		return nil, fmt.Errorf("add agent node: %w", err)
	}
	if err := g.AddEdge("prefill", "agent"); err != nil {
		return nil, fmt.Errorf("edge prefill->agent: %w", err)
	}

	if err := g.AddLambdaNode("final_guard",
		compose.InvokableLambda(guardNode),
		compose.WithNodeName("orchestration.guard")); err != nil {
		return nil, fmt.Errorf("add guard node: %w", err)
	}
	if err := g.AddEdge("agent", "final_guard"); err != nil {
		return nil, fmt.Errorf("edge agent->guard: %w", err)
	}
	if err := g.AddEdge("final_guard", compose.END); err != nil {
		return nil, fmt.Errorf("edge guard->END: %w", err)
	}

	return g.Compile(context.Background(), compose.WithGraphName("orchestration"))
}
```

- [ ] **Step 2: 编译验证**

Run: `go build ./internal/runtime/`
Expected: 编译通过。

- [ ] **Step 3: Commit**

```bash
git add internal/runtime/orchestration_graph.go
git commit -m "feat(runtime): add buildOrchestrationGraph topology compile"
```

---

### Task 6: executor.Execute 改调 graph.Stream

**Files:**
- Modify: `internal/runtime/executor.go`

- [ ] **Step 1: 在 Executor 结构添加预编译 Runnable 字段**

修改 `executor.go:25-33`：

```go
type Executor struct {
	reg                *tools.Registry
	flashChat          llm.Chat
	summarizerModel    einomodel.ToolCallingChatModel
	specialistRegistry *specialists.Registry
	builder            *AgentBuilder
	llmModel           string
	historyLimit       int
	orchestrationGraph compose.Runnable[string, string] // 预编译 Graph
}
```

- [ ] **Step 2: NewExecutor 中编译 Graph**

修改 `executor.go:37-45` 的 `NewExecutor`：

```go
func NewExecutor(reg *tools.Registry, sr *specialists.Registry, model einomodel.ToolCallingChatModel, flashChat llm.Chat, summarizerModel einomodel.ToolCallingChatModel) (*Executor, error) {
	graph, err := buildOrchestrationGraph()
	if err != nil {
		return nil, fmt.Errorf("compile orchestration graph: %w", err)
	}
	return &Executor{
		reg:                reg,
		summarizerModel:    summarizerModel,
		specialistRegistry: sr,
		builder:            NewAgentBuilder(model, reg, flashChat, summarizerModel),
		orchestrationGraph: graph,
	}, nil
}
```

- [ ] **Step 3: 重写 Execute 调 graph.Stream**

替换 `executor.go:61-102` 的 `Execute` 方法体：

```go
func (e *Executor) Execute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (turnType string, assistantText string, err error) {
	updateRoutingSnapshot(st, route)
	if len(route.Slots.Profile) > 0 {
		st.MergeProfile(route.Slots.Profile)
	}
	annotateApprovedRouteTrace(ctx, st, route)

	// 构造 per-request vals（原 runAgentRoute:153-165 的逻辑）
	vals := map[string]any{"profile": st.Profile, "domain": route.PrimaryDomain}
	if st.BaziResult != nil {
		vals["bazi_result"] = st.BaziResult
		if bj, err := json.Marshal(st.BaziResult); err == nil {
			vals["bazi_json"] = string(bj)
		}
	}
	if st.QimenResult != nil {
		vals["qimen_result"] = st.QimenResult
	}
	if st.ZiWeiResult != nil {
		vals["ziwei_result"] = st.ZiWeiResult
	}

	// 注入 state 到 ctx——节点 Lambda 通过 getOrchestrationState(ctx) 读取。
	// 不使用 compose.WithGenLocalState（后者创建的 state 与外部 state 是两个对象，
	// 节点 Lambda 拿不到真实字段）。ctx.Value 简单直接，Phase 1 够用。
	// Phase 2 Checkpoint 需要真正的 State Graph 时再重设计（见 Task 9）。
	state := &orchestrationState{
		st:       st,
		route:    route,
		userMsg:  message,
		vals:     vals,
		sink:     sink,
		executor: e,
	}
	ctx = withOrchestrationState(ctx, state)

	// preflight span（保留原 tracing）
	preflightSpan := tracing.SpanFromContext(ctx, "preflight", tracing.KindChain)
	preflightSpan.SetAttribute("primary_domain", route.PrimaryDomain)
	preflightSpan.SetAttribute("task_intent", route.TaskIntent)
	preflightSpan.End()

	finalText, err := e.orchestrationGraph.Stream(ctx, message)
	if err != nil {
		return "agent_error", finalText, err
	}

	// guardedTurnType 由 guardNode 写入 state（通过 ctx.Value 共享）
	return state.guardedTurnType, finalText, nil
}
```

- [ ] **Step 4: 删除 runAgentRoute 方法**

`executor.go:151-198` 的 `runAgentRoute` 方法体已拆入各节点 Lambda，删除该方法。保留 `prefill` / `prefillBazi` / `prefillQimen` / `prefillZiWei` / `callTool` / `buildToolParams` / `saveToolResult` / `buildConversationMessages` / `updateGuidanceState` / `shouldPreserveGuidanceOnExecution`——这些方法被节点 Lambda 调用。

- [ ] **Step 5: 编译验证**

Run: `go build ./internal/runtime/`
Expected: 编译通过。

- [ ] **Step 6: Commit**

```bash
git add internal/runtime/executor.go
git commit -m "refactor(runtime): rewrite Execute to call orchestrationGraph.Stream"
```

---

### Task 7: 回归测试（Phase 1 验收门）

**Files:**
- Test: `internal/runtime/executor_agent_route_test.go`（现有，必须全绿）
- Test: `internal/runtime/bridge_test.go`（现有，必须全绿）
- Test: `internal/runtime/preflight_test.go`（现有，必须全绿）
- Test: `internal/runtime/observability_test.go`（现有，必须全绿）
- Create: `internal/runtime/orchestration_graph_test.go`

- [ ] **Step 1: 运行现有测试套件（行为不变验证）**

Run: `cd /Users/wikiglobal/workSapce/suanming-agent && go test ./internal/runtime/ -v -count=1`
Expected: 所有现有测试 PASS。任何 FAIL 视为回归，必须修复后才能进入 Phase 2。

- [ ] **Step 2: 新增拓扑编译烟雾测试**

```go
package runtime

import (
	"testing"
)

// TestOrchestrationGraphTopology 验证 Graph 拓扑结构正确编译。
// 不验证行为（行为由现有回归测试覆盖），只验证 Runnable 可编译。
func TestOrchestrationGraphTopology(t *testing.T) {
	r, err := buildOrchestrationGraph()
	if err != nil {
		t.Fatalf("buildOrchestrationGraph failed: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil Runnable")
	}
}
```

- [ ] **Step 3: 运行新增测试**

Run: `go test ./internal/runtime/ -run TestOrchestrationGraph -v -count=1`
Expected: PASS。

- [ ] **Step 4: 全套回归再跑一次**

Run: `go test ./internal/runtime/ -v -count=1`
Expected: 全绿。

- [ ] **Step 5: Commit**

```bash
git add internal/runtime/orchestration_graph_test.go
git commit -m "test(runtime): add topology compile smoke test"
```

---

### Task 8: Phase 1 验收门（Stop Point）

**Phase 1 完成条件（全部满足才进 Phase 2）:**

- [ ] `go build ./...` 通过
- [ ] `go test ./internal/runtime/ -v -count=1` 全绿（与 Task 0 baseline 一致）
- [ ] 手动跑一遍八字主路径（有资料 / 无资料 / 澄清 / amend_profile / 真太阳时闰月）行为与重构前一致
- [ ] 手动跑一遍紫微主路径（有资料 / 闰月处理）行为一致
- [ ] 手动跑一遍奇门主路径（无资料 primary profileless）行为一致
- [ ] SSE 事件序列（thinking / tool_call / component / text / done）与重构前一致——前端无感知
- [ ] tracing span 层级正确（preflight / prefill / adk_supervisor_agent / guard）

**若任一不满足:** 回滚到 Phase 1 开始前的 commit，分析原因，不进 Phase 2。

#### 手动验证步骤

启动服务（三窗口）：

```bash
# 窗口 1: 知识库
cd /Users/wikiglobal/workSapce/suanming-agent && make knowledge-start

# 窗口 2: 后端
cd /Users/wikiglobal/workSapce/suanming-agent && LLM_API_KEY=sk-xxx go run ./cmd/server/

# 窗口 3: 前端
cd /Users/wikiglobal/workSapce/suanming-agent/web && npm run dev
```

打开 `http://localhost:5173`，按以下场景测试：

| 场景 | 输入 | 预期 |
|------|------|------|
| 八字有资料 | "我1990年5月5日午时生，帮我看看事业" | emit bazi-chart → thinking → final text 解读 |
| 八字无资料 | "帮我算算事业"（无出生信息） | emit text 提示要出生信息，无 chart 事件 |
| 澄清 | 上轮问事业，本轮 "那我财运呢" | 复用上轮 bazi，新角度解读 |
| amend_profile | "不是1990年是1989年" | 重新排盘，新解读 |
| 紫微有资料 | "我1990年5月5日午时生，紫微看看感情" | emit ziwei-chart → thinking → final text |
| 奇门无资料 | "今天运势怎么样" | emit qimen-chart（当前时间排盘）→ final text |

每个场景在浏览器 DevTools Network 标签查看 SSE 事件流，对比重构前后事件序列（类型 + 顺序 + payload 结构）一致。

---

### Task 9: Phase 2 — Checkpoint Store + State 序列化重设计

**前置:** Task 8 全部通过。

**Files:**
- Create: `internal/runtime/checkpoint_store.go`
- Modify: `internal/runtime/orchestration_state.go`（state 拆分可序列化字段）

- [ ] **Step 1: 定义 CheckPointStore 接口和 Redis 实现**

接口签名按 [eino-agent/eino/internal/core/interrupt.go:27](../../../eino-agent/eino/internal/core/interrupt.go:27) 定义：`Get(ctx, checkPointID) ([]byte, bool, error)`——三个返回值。

```go
package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/redis/go-redis/v9"
)

// redisCheckPointStore 是 Graph Checkpoint 的 Redis 实现。
// 用于 prefill 后、agent 前的中断-恢复交互（C1 能力）。
type redisCheckPointStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCheckPointStore(addr string) (compose.CheckPointStore, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &redisCheckPointStore{client: client, ttl: 24 * time.Hour}, nil
}

// Get 返回 (checkpoint bytes, found, error)。
func (s *redisCheckPointStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	val, err := s.client.Get(ctx, checkPointID).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return val, true, nil
}

func (s *redisCheckPointStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	return s.client.Set(ctx, checkPointID, checkPoint, s.ttl).Err()
}

// Delete 实现 compose.CheckPointDeleter（可选接口，用于显式清理 stale checkpoint）。
func (s *redisCheckPointStore) Delete(ctx context.Context, checkPointID string) error {
	return s.client.Del(ctx, checkPointID).Err()
}
```

- [ ] **Step 2: orchestrationState 拆分可序列化字段**

Phase 1 的 `orchestrationState` 含 `executor *Executor` / `sink EventSink` 等非序列化引用，无法存入 Redis。Phase 2 拆分：

```go
// orchestrationStateSerializable 是可序列化的状态字段，存入 Checkpoint。
// 节点 Lambda 通过 Graph state API 读写。
type orchestrationStateSerializable struct {
	St              *state.SessionState
	Route           policy.ApprovedRoute
	UserMsg         string
	Vals            map[string]any
	PreflightResult preflightResult
	GuardedTurnType string
}

// orchestrationRuntime 保留非序列化引用，通过 ctx 传递（不进 Checkpoint）。
type orchestrationRuntime struct {
	Sink     EventSink
	Executor *Executor
}
```

启用 `compose.WithGenLocalState` 创建 `orchestrationStateSerializable`，节点 Lambda 改用 Graph state API 读写。`orchestrationRuntime` 仍走 ctx.Value。

- [ ] **Step 3: 编译验证**

Run: `go build ./internal/runtime/`
Expected: 编译通过（需要 `go get github.com/redis/go-redis/v9` 如果未引入）。

- [ ] **Step 4: Commit**

```bash
git add internal/runtime/checkpoint_store.go internal/runtime/orchestration_state.go go.mod go.sum
git commit -m "feat(runtime): add redisCheckPointStore and serializable state split"
```

---

### Task 10: Phase 2 — 中断-恢复交互

> **API 已核实（见附录 B）:**
> - `compose.WithCheckPointID(id string) Option` — Stream/Invoke 选项，指定 checkpoint ID
> - `compose.ExtractInterruptInfo(err) (*InterruptInfo, bool)` — 从错误中提取中断信息
> - `info.InterruptContexts[0].ID` — 中断上下文 ID（resume 时用）
> - `compose.ResumeWithData(ctx, interruptID, data) context.Context` — 包装 ctx 携带 resume 数据
> - Resume = 再次调用 `Stream(rCtx, input, WithCheckPointID(cpID))`，cpID 与首次相同

**Files:**
- Modify: `internal/runtime/orchestration_graph.go`（Graph 编译加 `WithCheckPointStore` + `WithInterruptBeforeNodes`）
- Modify: `internal/runtime/executor.go`（Execute 识别 InterruptError + 新增 `Resume` 方法）
- Test: `internal/runtime/checkpoint_test.go`

- [ ] **Step 1: buildOrchestrationGraph 接受 CheckpointStore 参数，加中断节点**

```go
// buildOrchestrationGraph 编译执行骨架 Graph。
// cpStore 非空时启用 Checkpoint，并在 agent 节点前中断（用于 C1 真太阳时确认类交互）。
func buildOrchestrationGraph(cpStore compose.CheckPointStore) (compose.Runnable[string, string], error) {
	g := compose.NewGraph[string, string]()
	// ... 已有节点和边（Task 5）...

	compileOpts := []compose.GraphCompileOption{
		compose.WithGraphName("orchestration"),
	}
	if cpStore != nil {
		compileOpts = append(compileOpts,
			compose.WithCheckPointStore(cpStore),
			// 在 agent 节点前中断——prefill 完成后、LLM 推理前
			// 用户可在此确认"出生时间是否为真太阳时"等交互
			compose.WithInterruptBeforeNodes([]string{"agent"}),
		)
	}
	return g.Compile(context.Background(), compileOpts...)
}
```

同步修改 `executor.go` 的 `NewExecutor`：`buildOrchestrationGraph()` 改为 `buildOrchestrationGraph(e.cpStore)`（`Executor` 新增 `cpStore compose.CheckPointStore` 字段，启动时根据配置注入；nil 表示 Phase 1 模式不启用 Checkpoint）。

- [ ] **Step 2: Execute 处理 InterruptError**

修改 `executor.go` 的 `Execute`，识别 Graph 返回的 InterruptError。checkpoint ID 用 session ID + turn ID 拼接（用户可控），不从 error 中提取：

```go
// Execute 内部生成 checkpoint ID（每次调用唯一）
cpID := fmt.Sprintf("%s-turn-%d", st.SessionID, st.TurnCount)

finalText, err := e.orchestrationGraph.Stream(ctx, message, compose.WithCheckPointID(cpID))
if err != nil {
	// 检查是否为 InterruptError（Graph 在 agent 节点前中断）
	if info, ok := compose.ExtractInterruptInfo(err); ok {
		// info.InterruptContexts[0].ID 是中断上下文 ID，resume 时回传
		interruptID := ""
		if len(info.InterruptContexts) > 0 {
			interruptID = info.InterruptContexts[0].ID
		}
		// 返回特殊 turnType，前端展示"请您确认出生时间是否为真太阳时"
		// 前端调 Resume 时回传 interruptID + 用户确认数据
		return "awaiting_confirm", finalText, &InterruptError{
			CheckPointID: cpID,
			InterruptID:  interruptID,
			Reason:       "solar_time_confirm",
		}
	}
	return "agent_error", finalText, err
}

return state.guardedTurnType, finalText, nil
```

新增 `InterruptError` 类型（`executor.go` 或新建 `errors.go`）：

```go
// InterruptError 表示 Graph 在 agent 节点前中断，等待用户确认后继续。
type InterruptError struct {
	CheckPointID string
	InterruptID string
	Reason      string
}

func (e *InterruptError) Error() string {
	return fmt.Sprintf("graph interrupted: cpID=%s interruptID=%s reason=%s",
		e.CheckPointID, e.InterruptID, e.Reason)
}
```

- [ ] **Step 3: Executor 新增 Resume 方法**

```go
// Resume 在 Checkpoint 中断后由用户回复触发，继续执行 Graph。
// 典型场景: prefill 后追问"出生时间是否为真太阳时"，用户回复后调用此方法。
//
// 参数:
//   - cpID: Execute 返回的 InterruptError.CheckPointID
//   - interruptID: Execute 返回的 InterruptError.InterruptID
//   - userMessage: 用户的回复文本（如"是的，真太阳时"）
func (e *Executor) Resume(ctx context.Context, sink EventSink, st *state.SessionState, cpID, interruptID, userMessage string) (string, string, error) {
	state := &orchestrationState{
		st:       st,
		userMsg:  userMessage,
		sink:     sink,
		executor: e,
	}
	ctx = withOrchestrationState(ctx, state)

	// 用 ResumeWithData 包装 ctx，携带 interruptID + 用户回复数据
	// Graph 内部节点可通过 InterruptCtx 读取 userMessage
	rCtx := compose.ResumeWithData(ctx, interruptID, userMessage)

	finalText, err := e.orchestrationGraph.Stream(rCtx, userMessage, compose.WithCheckPointID(cpID))
	if err != nil {
		// resume 后仍可能再次中断（多轮确认）
		if info, ok := compose.ExtractInterruptInfo(err); ok {
			newInterruptID := ""
			if len(info.InterruptContexts) > 0 {
				newInterruptID = info.InterruptContexts[0].ID
			}
			return "awaiting_confirm", finalText, &InterruptError{
				CheckPointID: cpID,
				InterruptID:  newInterruptID,
				Reason:       "solar_time_confirm",
			}
		}
		return "agent_error", finalText, err
	}

	return state.guardedTurnType, finalText, nil
}
```

- [ ] **Step 4: 编译验证**

Run: `go build ./internal/runtime/`
Expected: 编译通过。

- [ ] **Step 5: 新增中断-恢复端到端测试**

参考 [eino-agent/eino/compose/checkpoint_test.go:80-117](../../../eino-agent/eino/compose/checkpoint_test.go:80) 的模式：

```go
package runtime

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/compose"
)

// TestCheckpoint_SolarTimeConfirm 验证 prefill 后中断、用户回复后恢复。
func TestCheckpoint_SolarTimeConfirm(t *testing.T) {
	// 1. 构造 Executor + Redis CheckPointStore + mock SessionState
	//    st.BaziResult = {"needs_solar_time_confirm": true, ...}
	// 2. 第一次调 Execute：
	//    turnType, _, err := e.Execute(ctx, sink, st, route, "帮我看看事业")
	//    预期 err 是 *InterruptError，turnType="awaiting_confirm"
	//    cpID := err.(*InterruptError).CheckPointID
	//    interruptID := err.(*InterruptError).InterruptID
	// 3. 验证 sink 已收到 chart 事件（prefill 完成），但未收到 final text（agent 未跑）
	// 4. 第二次调 Resume：
	//    turnType, finalText, err := e.Resume(ctx, sink, st, cpID, interruptID, "是的，真太阳时")
	//    预期 err=nil，finalText 非空，turnType 为 guard 后的类型
	// 5. 验证 sink 收到 final text 事件
	//
	// 具体 mock 构造参考 eino-agent/eino/compose/checkpoint_test.go:80
	t.Skip("Phase 2 实施时填写：mock 构造参考 checkpoint_test.go:80")
}
```

- [ ] **Step 6: 运行 Phase 2 测试**

Run: `go test ./internal/runtime/ -run TestCheckpoint -v -count=1`
Expected: PASS（需要本地 Redis 实例 `redis://localhost:6379`）。

- [ ] **Step 7: 全套回归**

Run: `go test ./internal/runtime/ -v -count=1`
Expected: 全绿（包括 Phase 1 现有测试）。

- [ ] **Step 8: Commit**

```bash
git add internal/runtime/orchestration_graph.go internal/runtime/executor.go internal/runtime/checkpoint_test.go
git commit -m "feat(runtime): add Checkpoint interrupt-resume for solar time confirm"
```

---

## 4. 当前实施状态与已知问题（2026-06-25 手动测试）

### 4.1 已完成

| 阶段 | 任务 | 状态 | commit |
|------|------|------|--------|
| Phase 1 | Task 1-7 拓扑重构 | ✅ 完成 | 48d0f1f → ceda97c |
| Phase 1 | Task 8 验收门（自动化测试） | ✅ 通过 | 0c3fd62 |
| Phase 2 | Task 9 fileCheckPointStore + 可序列化 state | ✅ 完成 | 9494ac5 |
| Phase 2 | Task 10 Checkpoint 中断-恢复 | ✅ 基础设施完成 | fef4508 |
| Phase 2 | 端到端中断-恢复测试 | ⚠️ t.Skip | 需 mock LLM |

### 4.2 自动化测试

- `TestOrchestrationGraphTopology` — Graph 编译 ✅
- `TestFileCheckPointStore_SetGetDelete` — 存储基础 ✅
- `TestFileCheckPointStore_PersistAcrossInstances` — 文件持久化 ✅
- `TestOrchestrationGraphCompilesWithCheckPointStore` — 带 cpStore 编译 ✅
- `TestCheckpoint_SolarTimeConfirm` — 端到端中断-恢复 ⚠️ t.Skip（需 mock LLM + specialist registry）
- `TestExecute_RecordsPreflightAndSSETraceOnShortCircuit` — 短路路径 ✅

### 4.3 手动测试遇到的问题

测试输入：`1992年12月1日12点 男 北京`（三轮对话）

#### 问题 1：bazi_specialist LLM 幻觉——无视 instruction 里注入的真实命盘

**现象**：用户输入 1992 男北京，supervisor 正确提取 profile（1992-12-01 12:00 男 北京），prefillBazi 正确排盘（1992 辛金，壬申年辛亥月辛亥日），session state 的 `BaziResult.dayGan=辛` 正确。但 bazi_specialist 最终回复"您好，我们继续。上一轮我们分析了您1988年女命的命盘，确认了戊土身弱、财生杀生印的吉象循环，以及当前甲寅大运杀印相生的格局..."——**完全无视 instruction 里注入的 1992 辛金命盘，编造 1988 年戊土女命**。

**证据**：
- session state `data/sessions/dfc7ac9e-*.json`：`Profile={year:1992,...}`、`BaziResult.dayGan=辛` ✅
- trace `logs/debug/172543-dfc7ac9e.jsonl`：supervisor_model 输出 `slots.profile={year:1992,...}` ✅
- trace "八字专家" TOOL 节点：`args.request="1992年12月1日12点 男 北京"`，`response="您好，我们继续。上一轮我们分析了您1988年女命的命盘..."` ❌

**根因**：bazi_specialist 的 instruction 通过 [BuildSpecialist](../../../internal/runtime/agent_route.go:63) + [buildBaziDataBlock](../../../internal/runtime/agent_route.go:244) 注入了 1992 辛金命盘数据，但 LLM（deepseek-v4-pro）无视了这些数据，编造了 1988 年戊土女命。instruction 里明确写了"命盘结果...是唯一权威来源。严禁根据出生资料自行推算四柱"（[prompts/interpret.md:26](../../../prompts/interpret.md:26)），但 LLM 不遵守。

#### 问题 2：supervisor 路由幻觉——把八字问题路由到奇门

**现象**：第二轮用户追问"我问的是1992年12月1日12点 男 北京 我们啥时候继续了？"，supervisor 路由结果变成 `primary_domain="qimen"`、`task_intent="fortune_followup"`、`qimen_mode="primary"`（第一轮是 `primary_domain="bazi"`）。prefillQimen 排了奇门盘，qimen_specialist 输出"奇门遁甲 × 八字 综合解读"。

**根因**：supervisor 路由 LLM（deepseek-v4-flash）看到"继续"两个字，把 domain 从 bazi 切到 qimen。用户根本没问奇门。这是 supervisor 路由 LLM 的判断错误。

#### 问题 3：bazi_specialist ReAct loop 陷入循环，exceeds max iterations

**现象**：第三轮用户再追问，bazi_specialist 报错 `[NodeRunError] run node[ChatModel] pre processor fail: exceeds max iterations`。后端 stderr：
```
[handler] orchestrator.Run session=c870997a error: [NodeRunError] run node[ChatModel] pre processor fail: exceeds max iterations
node path: [agent, node_1, ToolNode, node_1, ChatModel]
```

bazi_specialist 6 次都只调 `knowledge_catalog`（目录探索），**从不调 `knowledge_search`**（检索），每次 thinking 都说"让我先检索古籍依据"，但下一步又调 catalog。陷入循环，超过 `MaxIterations=6`（[agent_route.go:94](../../../internal/runtime/agent_route.go:94)）报错。

**根因**：
1. bazi_specialist 收到矛盾上下文：instruction 说 1992 辛金，conversation history（[buildConversationMessages](../../../internal/runtime/executor.go:497) 传 st.RecentTurns）里有前两轮的幻觉回复（1988 戊土、奇门交叉）。LLM 困惑，thinking 明确说"我注意到摘要中的排盘数据与系统提供的权威命盘结果存在差异...而非摘要中的戊土"。
2. **知识库上限 3 次没有硬性限制**：[adapter.go:271](../../../internal/runtime/adapter.go:271) 的"每轮限3次调用"只是 tool description 里的提示词，LLM 可以无视。[buildSpecialistHandlers](../../../internal/runtime/agent_route.go:108) 只有 reduction（截断结果）和 summarization（压缩历史）两种中间件，**没有限制 tool 调用次数的中间件**。所以 bazi_specialist 调 6 次 catalog 也没被拦截。

### 4.4 trace 数字异常说明

用户质疑"14 次主控调度"和"6 次目录检索"——这不是幻觉，是实际架构问题：

- **14 次主控调度**：[eino_callback.go:50](../../../internal/tracing/eino_callback.go:50) 的 `ChatModel.OnStart` 每次创建 span，名字取自 `cfg.Name`（"adk_supervisor_agent"）。bazi_specialist 作为 supervisor 的子 agent，**继承了 supervisor 的 ctx**（包含 einoCallbackConfigKey），所以 bazi_specialist 的 LLM 调用也被标记为"主控调度"。按 input_tokens 模式拆分：8 次小 input（supervisor）+ 6 次大 input（bazi_specialist，instruction 含命盘数据）= 14 次。supervisor MaxIterations=10 没超，bazi_specialist MaxIterations=6 刚好卡上限。
- **6 次目录检索**：knowledge_catalog 没有"每轮限3次"的提示（只有 knowledge_search 有），而且提示词本身也没有硬性限制。

### 4.5 与重构的关系

git diff 确认重构（commit 510643b → ceda97c）**没有改**：
- [buildConversationMessages](../../../internal/runtime/executor.go:497)（传 st.RecentTurns 给 specialist）
- [BuildSpecialist](../../../internal/runtime/agent_route.go:63) / [buildBaziDataBlock](../../../internal/runtime/agent_route.go:244)（instruction 注入命盘数据）
- [buildSpecialistHandlers](../../../internal/runtime/agent_route.go:108)（中间件链）
- `MaxIterations` 配置（supervisor=10，specialist=6）
- [adapter.go:271](../../../internal/runtime/adapter.go:271) 的 tool description
- [eino_callback.go](../../../internal/tracing/eino_callback.go) 的 span 命名
- supervisor 路由 LLM 的 prompt（在 internal/supervisor/）
- [prompts/interpret.md](../../../prompts/interpret.md)

重构只改了 `internal/runtime/` 的拓扑结构（executor.go 从嵌套 if 改成 Graph.Invoke）。agent 调用路径（vals/messages/BuildSupervisor/runner.Run/agentEventBridge）**完全一致**。

**但是**用户反馈"重构前没这些问题"。可能的解释：
1. 重构前没触发"两轮幻觉+第三轮矛盾上下文"的场景
2. 重构后 agentEventBridge 从同步变成异步（在 goroutine 里跑，通过 schema.Pipe 传给 guardNode），ctx 经过 Graph 框架——可能影响 LLM 行为（未证实）
3. 重构前可能有某种状态清理机制（未找到）

**未验证**：没有回退到重构前 commit（510643b）跑同样三步对比。需要确认是不是重构引起的回归。

### 4.6 待决问题（给接手者）

1. **是不是重构引起的回归？** 需要回退到 510643b 跑同样三步（1992 男北京 → 追问 → 再追问）对比。如果重构前也复现幻觉+循环，那是既有 LLM 问题；如果不复现，需要查重构引入的差异（agentEventBridge 异步化、ctx 经过 Graph 框架）。

2. **bazi_specialist 为什么无视 instruction 里的命盘数据？** instruction 里明确注入了 1992 辛金命盘（buildBaziDataBlock），但 LLM 编造 1988 戊土。可能原因：
   - LLM 模型能力不足（deepseek-v4-pro）
   - instruction 太长（172 行 interpret.md + 命盘数据块），LLM 没读到命盘部分
   - conversation history 的权重高于 instruction

3. **supervisor 路由为什么把 bazi 切成 qimen？** 需要看 supervisor 路由 LLM 的 prompt（在 internal/supervisor/），看"继续"为什么触发 domain 切换。

4. **知识库调用次数没有硬性限制** —— 需要加一个中间件统计 knowledge_catalog/search 调用次数，超 3 次拒绝。

5. **bazi_specialist 的 callback span 命名** —— 应该用 "bazi_specialist" 而不是继承 supervisor 的 "adk_supervisor_agent"，trace 更清晰。

6. **Phase 2 残留风险**：
   - 端到端测试 t.Skip（需 mock LLM）
   - container.go 未接入 SetCheckPointStore（生产没启用 Checkpoint）
   - 无条件中断逻辑（WithInterruptBeforeNodes 总是中断，应该只在特定场景）
   - Resume 语义缺陷（用户回复如何映射到 interruptID）

---

## 5. 风险与回滚

| 风险 | 概率 | 影响 | 缓解 |
|------|------|------|------|
| agentEventBridge 345 行业务规则迁入 StreamableLambda 后 SSE 事件序列变化 | 中 | 高 | Task 7 Step 1 现有回归测试必须全绿；SSE 事件序列前后对比（前端无感） |
| Graph State 注入策略错误（外部 state vs Graph state 混淆） | 中 | 中 | Task 6 Step 3 架构说明已标注；Phase 2 Task 9 重设计为可序列化 state |
| Eino `compose.NewGraph` + `WithCheckPointStore` 在 fork 版本（go.mod v0.9.6）有 bug | 低 | 高 | Task 10 Step 0 先跑 Eino 自带 checkpoint 示例验证 API 可用 |
| Redis 不可用导致 Checkpoint 失败 | 低 | 中 | NewRedisCheckPointStore 在启动时 Ping，失败则启动失败（fail-fast） |
| Phase 1 回归测试不全面，遗漏边界场景 | 中 | 高 | Task 8 验收门要求手动跑 5 个主路径场景 + SSE 序列对比 |
| StreamableLambda 的 goroutine 泄漏（agentEventBridge 阻塞） | 低 | 中 | agentNode 的 `go func` 有 `defer sw.Close()`，iter.Next() 出错或结束都会关闭 |

**回滚策略:**
- Phase 1 失败：`git revert` 到 Task 1 前的 commit，executor.go 恢复嵌套 if 结构
- Phase 2 失败：`git revert` Task 9-10，保留 Phase 1 Graph 拓扑（无 Checkpoint 能力但行为不变）

---

## 6. 决策记录

| 决策 | 选择 | 理由 |
|------|------|------|
| paipan 子图 | **不做** | `bazi_calc` 等工具返回完整命盘，不暴露中间阶段；Graph 化需先分解工具（不在范围） |
| Phase 1+2 合并实施 | **采纳** | 需求清晰后拓扑一次到位，避免 Phase 1 后期返工 |
| dispatch 层 Graph 化 | **不做** | 聊天系统需要上下文感知路由，AgentAsTool L3 不动 |
| specialist 层 Graph 化 | **不做** | LLM 自主选工具是价值所在，ReAct L4 不动 |
| `agentEventBridge` 下放 | **不下放** | 345 行业务规则（specialist 去重、chart 派发、XML 拆分），Graph 只提供骨架 |
| Phase 1 State 注入 | **ctx.Value** | 简单直接，`WithGenLocalState` 创建的 state 与外部 state 是两个对象 |
| Phase 2 State 注入 | **可序列化 + WithGenLocalState** | Checkpoint 要求状态可序列化存储到 Redis |

---

## Related

- [01-overview.md](../../architecture/supervisor/01-overview.md) — 架构总图
- [executor.go](../../../internal/runtime/executor.go) — 当前实现（458 行）
- [bridge.go](../../../internal/runtime/bridge.go) — agentEventBridge（345 行，不下放）
- [preflight.go](../../../internal/runtime/preflight.go) — preflight 纯函数（180 行，迁入 Lambda）
- [observability.go:35](../../../internal/runtime/observability.go:35) — guardFinalAnswerWithTrace（迁入 guard Lambda）
- [eino-agent/eino/internal/core/interrupt.go:27](../../../eino-agent/eino/internal/core/interrupt.go:27) — CheckPointStore 接口
- [eino-agent/eino/compose/interrupt.go:31](../../../eino-agent/eino/compose/interrupt.go:31) — WithInterruptBeforeNodes
- [eino-agent/eino/adk/runner.go:147](../../../eino-agent/eino/adk/runner.go:147) — Runner.ResumeWithParams（参考）
- [eino-agent/eino/adk/agentic_react_test.go:585](../../../eino-agent/eino/adk/agentic_react_test.go:585) — resume 测试模式参考

---

## 附录 A: Eino API 速查（已核实）

所有 API 已在 `eino-agent/eino/` 源码中核实存在。引用位置见各 API 后的文件:行号。

### Graph 构造

| API | 签名 | 位置 |
|-----|------|------|
| `compose.NewGraph[I, O]` | `func NewGraph[I, O any](opts ...NewGraphOption) *Graph[I, O]` | [generic_graph.go:72](../../../eino-agent/eino/compose/generic_graph.go:72) |
| `g.AddLambdaNode` | `func (g *graph) AddLambdaNode(key string, node *Lambda, opts ...GraphAddNodeOpt) error` | [graph.go:433](../../../eino-agent/eino/compose/graph.go:433) |
| `g.AddBranch` | `func (g *graph) AddBranch(startNode string, branch *GraphBranch) error` | [graph.go:466](../../../eino-agent/eino/compose/graph.go:466) |
| `g.AddEdge` | `func (g *Graph[I, O]) AddEdge(startNode, endNode string) error` | [generic_graph.go:106](../../../eino-agent/eino/compose/generic_graph.go:106) |
| `g.Compile` | `func (g *Graph[I, O]) Compile(ctx context.Context, opts ...GraphCompileOption) (Runnable[I, O], error)` | [generic_graph.go:123](../../../eino-agent/eino/compose/generic_graph.go:123) |
| `compose.NewGraphBranch` | `func NewGraphBranch[T any](condition GraphBranchCondition[T], endNodes map[string]bool) *GraphBranch` | [branch.go:145](../../../eino-agent/eino/compose/branch.go:145) |

### Lambda 构造

| API | 签名 | 位置 |
|-----|------|------|
| `compose.InvokableLambda` | `func InvokableLambda[I, O any](i InvokeWOOpt[I, O], opts ...LambdaOpt) *Lambda` | [types_lambda.go:105](../../../eino-agent/eino/compose/types_lambda.go:105) |
| `compose.StreamableLambda` | `func StreamableLambda[I, O any](s StreamWOOpt[I, O], opts ...LambdaOpt) *Lambda` | [types_lambda.go:119](../../../eino-agent/eino/compose/types_lambda.go:119) |

### 节点/编译选项

| API | 签名 | 位置 |
|-----|------|------|
| `compose.WithNodeName` | `func WithNodeName(n string) GraphAddNodeOpt` | [graph_add_node_options.go:50](../../../eino-agent/eino/compose/graph_add_node_options.go:50) |
| `compose.WithGraphName` | `func WithGraphName(graphName string) GraphCompileOption` | [graph_compile_options.go:65](../../../eino-agent/eino/compose/graph_compile_options.go:65) |
| `compose.WithGenLocalState` | `func WithGenLocalState[S any](gls GenLocalState[S]) NewGraphOption` | [generic_graph.go:37](../../../eino-agent/eino/compose/generic_graph.go:37) |
| `compose.WithCheckPointStore` | `func WithCheckPointStore(store CheckPointStore) GraphCompileOption` | [checkpoint.go:59](../../../eino-agent/eino/compose/checkpoint.go:59) |
| `compose.WithInterruptBeforeNodes` | `func WithInterruptBeforeNodes(nodes []string) GraphCompileOption` | [interrupt.go:31](../../../eino-agent/eino/compose/interrupt.go:31) |
| `compose.WithInterruptAfterNodes` | `func WithInterruptAfterNodes(nodes []string) GraphCompileOption` | [interrupt.go:?](../../../eino-agent/eino/compose/interrupt.go) |

### Runnable 调用 + Checkpoint

| API | 签名 | 位置 |
|-----|------|------|
| `compose.Runnable[I, O]` | interface，方法 `Invoke` / `Stream` / `Collect` | [runnable.go:32](../../../eino-agent/eino/compose/runnable.go:32) |
| `compose.WithCheckPointID` | `func WithCheckPointID(checkPointID string) Option` — Stream/Invoke 选项 | [checkpoint.go:73](../../../eino-agent/eino/compose/checkpoint.go:73) |
| `compose.ExtractInterruptInfo` | `func ExtractInterruptInfo(err error) (info *InterruptInfo, existed bool)` | [interrupt.go:299](../../../eino-agent/eino/compose/interrupt.go:299) |
| `compose.ResumeWithData` | `func ResumeWithData(ctx context.Context, interruptID string, data any) context.Context` | [resume.go:106](../../../eino-agent/eino/compose/resume.go:106) |
| `compose.InterruptInfo` | struct，字段 `State` / `BeforeNodes` / `AfterNodes` / `InterruptContexts []*InterruptCtx` / `SubGraphs` | [interrupt.go:258](../../../eino-agent/eino/compose/interrupt.go:258) |

### Stream + Schema

| API | 签名 | 位置 |
|-----|------|------|
| `schema.Pipe[T]` | `func Pipe[T any](cap int) (*StreamReader[T], *StreamWriter[T])` | [stream.go:99](../../../eino-agent/eino/schema/stream.go:99) |
| `compose.START` / `compose.END` | 常量，`AddEdge` 的特殊节点名 | [compose 包] |

### CheckPointStore 接口

```go
// 位置: eino-agent/eino/internal/core/interrupt.go:27
type CheckPointStore interface {
    Get(ctx context.Context, checkPointID string) ([]byte, bool, error)
    Set(ctx context.Context, checkPointID string, checkPoint []byte) error
}

// 可选删除接口
type CheckPointDeleter interface {
    Delete(ctx context.Context, checkPointID string) error
}
```

### 典型中断-恢复调用模式

参考 [eino-agent/eino/compose/checkpoint_test.go:80-117](../../../eino-agent/eino/compose/checkpoint_test.go:80)：

```go
// 1. 首次调用，传入 checkpoint ID
_, err := r.Invoke(ctx, "start", compose.WithCheckPointID("1"))
// err 非 nil，是 interrupt error

// 2. 提取中断信息
info, ok := compose.ExtractInterruptInfo(err)
// ok == true
// info.InterruptContexts[0].ID 是中断上下文 ID

// 3. 用 ResumeWithData 包装 ctx，携带回复数据
rCtx := compose.ResumeWithData(ctx, info.InterruptContexts[0].ID, &testStruct{A: "state"})

// 4. 再次调用，使用相同 checkpoint ID
result, err := r.Invoke(rCtx, "start", compose.WithCheckPointID("1"))
// err == nil, result 是最终输出
```

---

## 附录 B: 现有测试套件参考

| 测试文件 | 覆盖范围 | 用途 |
|---------|---------|------|
| `executor_agent_route_test.go` | executor 主路径回归 | Phase 1 必须全绿 |
| `bridge_test.go` | `agentEventBridge` 事件桥接 | Phase 1 必须全绿（agent Lambda 迁入后） |
| `preflight_test.go` | preflight 短路逻辑 | Phase 1 必须全绿 |
| `observability_test.go` | `guardFinalAnswerWithTrace` | Phase 1 必须全绿 |
| `agent_route_middleware_test.go` | agent 路由中间件 | Phase 1 必须全绿 |
| `guidance_gate_test.go` | guidance 门控 | Phase 1 必须全绿 |
| `adapter_test.go` | adapter | Phase 1 必须全绿 |

**测试夹具参考:** `executor_agent_route_test.go` 中有 mock Executor / SessionState / EventSink 的构造模式，新测试 `orchestration_graph_test.go` / `checkpoint_test.go` 可复用。

**断言模式参考:** `bridge_test.go` 的 `drainEvents` / `findEvent` helper 函数可复用于 checkpoint 测试的事件序列断言。

---

## 附录 C: 关键代码片段速查

### preflightResult 结构（preflight.go:11）

```go
type preflightResult struct {
    ShortCircuit bool
    TurnType     string
    Text         string
    GuidanceNext *state.GuidanceState
    ForcedRoute  *policy.ApprovedRoute
}
```

### guardFinalAnswerWithTrace 签名（observability.go:35）

```go
func guardFinalAnswerWithTrace(ctx context.Context, route policy.ApprovedRoute, st *state.SessionState, finalText string) (turnType string, text string)
```

### shouldBufferFinalAnswer（final_guard.go:7）

```go
func shouldBufferFinalAnswer() bool { return true } // 永远返回 true
```

### emitEventWithTrace 签名（observability.go）

```go
func emitEventWithTrace(ctx context.Context, sink EventSink, event Event, attrs map[string]any) error
```

### updateGuidanceState 签名（executor.go:104）

```go
func (e *Executor) updateGuidanceState(st *state.SessionState, route policy.ApprovedRoute, message string, result preflightResult)
```

### EventSink 接口（event.go）

```go
type EventSink interface {
    // 具体方法见 internal/runtime/event.go
}
```

### ADK Runner 调用模式（executor.go:181-184，重构前的现状）

```go
runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: supervisor, EnableStreaming: true})
msgs := e.buildConversationMessages(st, message)
iter := runner.Run(ctx, msgs, adk.WithSessionValues(vals))
finalText, err := agentEventBridge(ctx, sink, iter, saveFn, labelFn, bufferFinal)
```

这段在 Task 4 的 `agentNode` 内**原样保留**，只是包了一层 `StreamableLambda` + `schema.Pipe`。
