package runtime

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// orchestrationCtx 组合 Graph state + init + runtime，节点 Lambda 通过 loadOrchestrationCtx 一次性加载。
// Graph 顺序执行，节点内修改 gs 指针字段是安全的（无并行节点）。
type orchestrationCtx struct {
	GS   *orchestrationGraphState
	Init *orchestrationInit
	RT   *orchestrationRuntime
}

func loadOrchestrationCtx(ctx context.Context) (*orchestrationCtx, error) {
	init := getOrchestrationInit(ctx)
	rt := getOrchestrationRuntime(ctx)
	var gs *orchestrationGraphState
	err := compose.ProcessState[*orchestrationGraphState](ctx, func(_ context.Context, s *orchestrationGraphState) error {
		gs = s
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load orchestration graph state: %w", err)
	}
	return &orchestrationCtx{GS: gs, Init: init, RT: rt}, nil
}

// preflightNode 是 Graph 的 preflight 节点：执行确定性校验，结果写入 Graph state。
// Branch 节点根据 Graph state 中的 PreflightResult 决定走 short_circuit 还是 prefill。
//
// preflight tracing span 整段保留在此节点内（原 executor.go:71-79 的逻辑），
// 保证 short_circuit / turn_type 属性不丢失。
func preflightNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	preflightSpan := tracing.SpanFromContext(ctx, "preflight", tracing.KindChain)
	preflightSpan.SetAttribute("primary_domain", oc.Init.Route.PrimaryDomain)
	preflightSpan.SetAttribute("task_intent", oc.Init.Route.TaskIntent)
	result := preflight(oc.Init.St, oc.Init.Route, oc.Init.UserMsg)
	preflightSpan.SetAttribute("short_circuit", result.ShortCircuit)
	if result.TurnType != "" {
		preflightSpan.SetAttribute("turn_type", result.TurnType)
	}
	preflightSpan.End()
	oc.GS.PreflightResult = result
	return in, nil
}

// preflightBranch 根据 PreflightResult 决定分支。
// short_circuit: preflight 短路（澄清/缺资料）
// prefill: 进入 prefill → agent → guard 主路径
func preflightBranch(ctx context.Context, _ string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	if oc.GS.PreflightResult.ShortCircuit {
		return "short_circuit", nil
	}
	return "prefill", nil
}

// emitShortCircuitNode 处理 preflight 短路：emit text 事件并返回。
// executor.go 原 Execute:80-86 的 updateGuidanceState + emit 逻辑整体移入此处。
//
// guardedTurnType 同步赋值，让 Execute 通过 Graph state 拿到短路路径的 turnType，
// 与 guardNode 路径共用 GuardedTurnType 字段作为返回通道。
func emitShortCircuitNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	span := tracing.SpanFromContext(ctx, "short_circuit", tracing.KindChain)
	span.SetAttribute("turn_type", oc.GS.PreflightResult.TurnType)
	span.SetAttribute("short_circuit", oc.GS.PreflightResult.ShortCircuit)
	defer span.End()

	oc.RT.Executor.updateGuidanceState(oc.Init.St, oc.Init.Route, oc.Init.UserMsg, oc.GS.PreflightResult)
	_ = emitEventWithTrace(ctx, oc.RT.Sink, Event{
		Type: "text",
		Data: map[string]any{"content": oc.GS.PreflightResult.Text},
	}, map[string]any{"turn_type": oc.GS.PreflightResult.TurnType})
	getOrchestrationResult(ctx).TurnType = oc.GS.PreflightResult.TurnType
	return oc.GS.PreflightResult.Text, nil
}

// prefillNode 调用现有 executor.prefill，结果写入 init.Vals 和 session state。
// 不使用 AddGraphNode 嵌入子图——底层工具不暴露中间阶段，子图无意义。
func prefillNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	oc.RT.Executor.prefill(ctx, oc.RT.Sink, oc.Init.St, oc.Init.Route, oc.Init.Vals)
	return in, nil
}

// guardNode 调用现有 guardFinalAnswerWithTrace，emit 最终 text。
// shouldBufferFinalAnswer()=true 时 guard 后的文本走 bufferFinal emit 路径。
func guardNode(ctx context.Context, finalText string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	span := tracing.SpanFromContext(ctx, "final_guard", tracing.KindChain)
	defer span.End()

	turnType, guardedText := guardFinalAnswerWithTrace(ctx, oc.GS.Route, oc.Init.St, finalText)
	span.SetAttribute("turn_type", turnType)
	if shouldBufferFinalAnswer() && guardedText != "" {
		_ = emitEventWithTrace(ctx, oc.RT.Sink, Event{
			Type: "text",
			Data: map[string]any{"content": guardedText},
		}, map[string]any{"buffer_final": true, "turn_type": turnType})
	}
	getOrchestrationResult(ctx).TurnType = turnType
	return guardedText, nil
}

// agentNode 是 Graph 的 agent 节点：构建 Supervisor + AgentTool specialists，
// 启动 ADK Runner，通过 agentEventBridge 桥接事件到 SSE。
//
// agentEventBridge 的业务规则（specialist 去重、chart 派发、XML 拆分、
// AgentAsTool 内联检测）整体保留在此 Lambda 内，不下放到 Graph 边。
// Graph 只提供骨架，不替代业务规则。
func agentNode(ctx context.Context, in string) (*schema.StreamReader[string], error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return nil, err
	}
	nodeSpan := tracing.SpanFromContext(ctx, "agent", tracing.KindChain)
	nodeSpan.SetAttribute("primary_domain", oc.GS.Route.PrimaryDomain)

	// ForcedRoute 覆盖（preflight 返回 ForcedRoute 时）
	route := oc.GS.Route
	if oc.GS.PreflightResult.ForcedRoute != nil {
		route = *oc.GS.PreflightResult.ForcedRoute
		oc.GS.Route = route // 同步到 Graph state，guardNode 用
		nodeSpan.SetAttribute("forced_route", true)
	}

	// emit transition text（ForcedRoute 场景）
	if oc.GS.PreflightResult.ForcedRoute != nil && oc.GS.PreflightResult.Text != "" {
		_ = emitEventWithTrace(ctx, oc.RT.Sink, Event{
			Type: "text",
			Data: map[string]any{"content": oc.GS.PreflightResult.Text},
		}, map[string]any{"turn_type": oc.GS.PreflightResult.TurnType})
	}

	// updateGuidanceState（非短路路径）
	oc.RT.Executor.updateGuidanceState(oc.Init.St, route, oc.Init.UserMsg, oc.GS.PreflightResult)

	// 构建 Supervisor
	allConfigs := oc.RT.Executor.specialistRegistry.All()
	allowed := allowedSpecialists(route, allConfigs)
	supervisor, err := oc.RT.Executor.builder.BuildSupervisor(ctx, route, oc.Init.St, allowed)
	if err != nil {
		return nil, fmt.Errorf("build supervisor agent: %w", err)
	}

	// tracing callback span
	ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
		Name: "adk_supervisor_agent", Kind: tracing.KindChain,
		Attributes: map[string]any{"model": oc.RT.Executor.llmModel, "domain": route.PrimaryDomain},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: supervisor, EnableStreaming: true})
	msgs := oc.RT.Executor.buildConversationMessages(oc.Init.St, oc.Init.UserMsg)
	iter := runner.Run(ctx, msgs, adk.WithSessionValues(oc.Init.Vals))

	// Pipe: agentEventBridge 写 finalText 到 sw，Graph 边读 sr
	sr, sw := schema.Pipe[string](64)
	go func() {
		defer sw.Close()
		defer nodeSpan.End()
		finalText, err := agentEventBridge(ctx, oc.RT.Sink, iter, func(toolName, resultJSON string) {
			oc.RT.Executor.saveToolResult(oc.Init.St, toolName, resultJSON)
		}, oc.RT.Executor.reg.DisplayName, shouldBufferFinalAnswer())
		if err != nil {
			nodeSpan.RecordError(err)
			nodeSpan.SetStatus("error")
			sw.Send("", err)
			return
		}
		nodeSpan.SetAttribute("final_text_len", len([]rune(finalText)))
		sw.Send(finalText, nil)
	}()
	return sr, nil
}

// buildOrchestrationGraph 编译执行骨架 Graph Runnable。
//
// 拓扑:
//   START → preflight ──branch──┬─ short_circuit → END
//                                └─ prefill → agent → final_guard → END
//
// 状态通过 compose.WithGenLocalState 管理 orchestrationGraphState，
// 节点 Lambda 用 compose.ProcessState 读写。中断-恢复时由 Eino Checkpoint 序列化。
// St/UserMsg/Vals 走 ctx.Value（init），不进 Checkpoint。
//
// cpStore 非空时启用 Checkpoint，并在 agent 节点前中断（用于 C1 真太阳时确认类交互）。
// cpStore 为 nil 时是 Phase 1 模式，不启用 Checkpoint。
func buildOrchestrationGraph(cpStore compose.CheckPointStore) (compose.Runnable[string, string], error) {
	g := compose.NewGraph[string, string](compose.WithGenLocalState(genOrchestrationState))

	// 节点：先添加所有节点，再 AddBranch（branch 的 endNodes 必须已在图中）
	if err := g.AddLambdaNode("preflight",
		compose.InvokableLambda(preflightNode),
		compose.WithNodeName("orchestration.preflight")); err != nil {
		return nil, fmt.Errorf("add preflight node: %w", err)
	}
	if err := g.AddLambdaNode("short_circuit",
		compose.InvokableLambda(emitShortCircuitNode),
		compose.WithNodeName("orchestration.short_circuit")); err != nil {
		return nil, fmt.Errorf("add short_circuit node: %w", err)
	}
	if err := g.AddLambdaNode("prefill",
		compose.InvokableLambda(prefillNode),
		compose.WithNodeName("orchestration.prefill")); err != nil {
		return nil, fmt.Errorf("add prefill node: %w", err)
	}
	if err := g.AddLambdaNode("agent",
		compose.StreamableLambda(agentNode),
		compose.WithNodeName("orchestration.agent")); err != nil {
		return nil, fmt.Errorf("add agent node: %w", err)
	}
	if err := g.AddLambdaNode("final_guard",
		compose.InvokableLambda(guardNode),
		compose.WithNodeName("orchestration.guard")); err != nil {
		return nil, fmt.Errorf("add guard node: %w", err)
	}

	// preflight 分支：short_circuit / prefill（endNodes 必须是实际节点名）
	if err := g.AddBranch("preflight", compose.NewGraphBranch(
		preflightBranch,
		map[string]bool{"short_circuit": true, "prefill": true},
	)); err != nil {
		return nil, fmt.Errorf("add preflight branch: %w", err)
	}

	// 边
	if err := g.AddEdge(compose.START, "preflight"); err != nil {
		return nil, fmt.Errorf("edge START->preflight: %w", err)
	}
	if err := g.AddEdge("short_circuit", compose.END); err != nil {
		return nil, fmt.Errorf("edge short_circuit->END: %w", err)
	}
	if err := g.AddEdge("prefill", "agent"); err != nil {
		return nil, fmt.Errorf("edge prefill->agent: %w", err)
	}
	if err := g.AddEdge("agent", "final_guard"); err != nil {
		return nil, fmt.Errorf("edge agent->guard: %w", err)
	}
	if err := g.AddEdge("final_guard", compose.END); err != nil {
		return nil, fmt.Errorf("edge guard->END: %w", err)
	}

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
