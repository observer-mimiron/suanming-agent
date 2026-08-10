// Package runtime contains the manager-owned execution flow.
//
// This file wires the outer Eino orchestration graph:
// preflight -> decide_next -> prefill/dispatch_batch/terminal.
package runtime

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

type orchestrationCtx struct {
	GS   *orchestrationGraphState
	Init *orchestrationInit
	RT   *orchestrationRuntime
}

// loadOrchestrationCtx gathers the three state channels every graph node needs.
// Eino local state is read through ProcessState; request inputs and runtime
// services stay in context because they are not graph checkpoint payload.
func loadOrchestrationCtx(ctx context.Context) (*orchestrationCtx, error) {
	init := getOrchestrationInit(ctx)
	rt := getOrchestrationRuntime(ctx)
	if init == nil || rt == nil {
		return nil, fmt.Errorf("orchestration request context is incomplete")
	}
	var gs *orchestrationGraphState
	err := compose.ProcessState[*orchestrationGraphState](ctx, func(_ context.Context, s *orchestrationGraphState) error {
		gs = s
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load orchestration graph state: %w", err)
	}
	if gs == nil {
		return nil, fmt.Errorf("orchestration graph state is nil")
	}
	return &orchestrationCtx{GS: gs, Init: init, RT: rt}, nil
}

// preflightNode runs deterministic front-door checks before any model-owned work.
// It writes only graph-local PreflightResult; session mutation stays in Executor
// so clarification, guided fallback, and normal execution share one owner.
func preflightNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}

	preflightSpan := tracing.SpanFromContext(ctx, "preflight", tracing.KindChain)
	preflightSpan.SetAttribute("primary_domain", oc.Init.Route.PrimaryDomain)
	preflightSpan.SetAttribute("task_intent", oc.Init.Route.TaskIntent)
	if oc.Init.Plan.FollowupMode != "" {
		preflightSpan.SetAttribute("followup_mode", oc.Init.Plan.FollowupMode)
	}

	result := preflightWithPlan(oc.Init.St, oc.Init.Plan, oc.Init.UserMsg, oc.RT.Router)
	preflightSpan.SetAttribute("short_circuit", result.ShortCircuit)
	if result.TurnType != "" {
		preflightSpan.SetAttribute("turn_type", result.TurnType)
	}
	preflightSpan.End()

	oc.GS.PreflightResult = result
	return in, nil
}

// prefillNode prepares deterministic assets required by the Manager's plan.
// Guided fallback can replace the route after preflight, so this node performs
// the single allowed plan rebuild and stores the effective plan in graph state.
func prefillNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}

	route := oc.GS.Route
	plan := oc.GS.Plan
	if oc.GS.PreflightResult.ForcedRoute != nil {
		route = *oc.GS.PreflightResult.ForcedRoute
		plan = oc.RT.Executor.manager.BuildExecutionPlanForTurn(oc.Init.St, route, oc.Init.UserMsg, oc.Init.Plan.TurnContext)
		oc.GS.Plan = plan
		oc.GS.Route = route
		// 强制路由会重建计划；待执行步骤必须同时替换，否则 dispatch 会按旧领域
		// 调度，令当前计划的资产合同与 worker 领域发生错配。
		oc.GS.PendingDomainSteps = append([]contracts.DomainStep(nil), plan.DomainSteps...)
		oc.RT.Executor.syncExecutionRoute(ctx, oc.Init.St, route, plan)
	}
	oc.GS.PrefillAttempts++
	oc.GS.Failure = graphFailure{}
	oc.RT.Executor.prefill(ctx, oc.RT.Sink, oc.Init.St, plan, oc.Init.Vals)
	if err := validatePlanArtifacts(oc.Init.St, plan); err != nil {
		if recordErr := recordGraphFailure(ctx, &oc.GS.Failure, route.PrimaryDomain, failureStagePrefill, err); recordErr != nil {
			return "", recordErr
		}
		oc.GS.PrefillCompleted = false
		return in, nil
	}
	oc.GS.PrefillCompleted = true
	oc.GS.DynamicFacts = dynamicFactsForPlan(oc.Init.St, plan)
	tracing.SetTraceAttributes(ctx, map[string]any{"dynamic_facts": oc.GS.DynamicFacts})
	return in, nil
}

// buildOrchestrationGraph compiles the bounded outer runtime graph once per
// Executor. The graph owns retry, fan-out/fan-in and termination; final guard
// runs after Invoke so it can remain the single user-visible text boundary.
func buildOrchestrationGraph() (compose.Runnable[string, string], error) {
	g := compose.NewGraph[string, string](compose.WithGenLocalState(genOrchestrationState))

	if err := g.AddLambdaNode("preflight",
		compose.InvokableLambda(preflightNode),
		compose.WithNodeName("orchestration.preflight")); err != nil {
		return nil, fmt.Errorf("add preflight node: %w", err)
	}
	if err := g.AddLambdaNode("short_circuit",
		compose.InvokableLambda(shortCircuitNode),
		compose.WithNodeName("orchestration.short_circuit")); err != nil {
		return nil, fmt.Errorf("add short_circuit node: %w", err)
	}
	if err := g.AddLambdaNode("decide_next",
		compose.InvokableLambda(decideNextNode),
		compose.WithNodeName("orchestration.decide_next")); err != nil {
		return nil, fmt.Errorf("add decide_next node: %w", err)
	}
	if err := g.AddLambdaNode("prefill",
		compose.InvokableLambda(prefillNode),
		compose.WithNodeName("orchestration.prefill")); err != nil {
		return nil, fmt.Errorf("add prefill node: %w", err)
	}
	if err := g.AddLambdaNode("dispatch_batch",
		compose.InvokableLambda(dispatchBatchNode),
		compose.WithNodeName("orchestration.dispatch_batch")); err != nil {
		return nil, fmt.Errorf("add dispatch_batch node: %w", err)
	}
	if err := g.AddLambdaNode("aggregate",
		compose.InvokableLambda(aggregateNode),
		compose.WithNodeName("orchestration.aggregate")); err != nil {
		return nil, fmt.Errorf("add aggregate node: %w", err)
	}
	if err := g.AddLambdaNode("terminal",
		compose.InvokableLambda(terminalNode),
		compose.WithNodeName("orchestration.terminal")); err != nil {
		return nil, fmt.Errorf("add terminal node: %w", err)
	}
	if err := g.AddLambdaNode("terminal_error",
		compose.InvokableLambda(terminalErrorNode),
		compose.WithNodeName("orchestration.terminal_error")); err != nil {
		return nil, fmt.Errorf("add terminal_error node: %w", err)
	}

	if err := g.AddBranch("decide_next", compose.NewGraphBranch(
		orchestrationBranch,
		map[string]bool{
			"short_circuit":  true,
			"prefill":        true,
			"dispatch_batch": true,
			"terminal":       true,
			"terminal_error": true,
		},
	)); err != nil {
		return nil, fmt.Errorf("add decide_next branch: %w", err)
	}

	if err := g.AddEdge(compose.START, "preflight"); err != nil {
		return nil, fmt.Errorf("edge START->preflight: %w", err)
	}
	if err := g.AddEdge("preflight", "decide_next"); err != nil {
		return nil, fmt.Errorf("edge preflight->decide_next: %w", err)
	}
	if err := g.AddEdge("short_circuit", "terminal"); err != nil {
		return nil, fmt.Errorf("edge short_circuit->terminal: %w", err)
	}
	if err := g.AddEdge("prefill", "decide_next"); err != nil {
		return nil, fmt.Errorf("edge prefill->decide_next: %w", err)
	}
	if err := g.AddEdge("dispatch_batch", "aggregate"); err != nil {
		return nil, fmt.Errorf("edge dispatch_batch->aggregate: %w", err)
	}
	if err := g.AddEdge("aggregate", "decide_next"); err != nil {
		return nil, fmt.Errorf("edge aggregate->decide_next: %w", err)
	}
	if err := g.AddEdge("terminal", compose.END); err != nil {
		return nil, fmt.Errorf("edge terminal->END: %w", err)
	}
	if err := g.AddEdge("terminal_error", compose.END); err != nil {
		return nil, fmt.Errorf("edge terminal_error->END: %w", err)
	}

	return g.Compile(context.Background(),
		compose.WithNodeTriggerMode(compose.AnyPredecessor),
		compose.WithMaxRunSteps(orchestrationMaxRunSteps),
		compose.WithGraphName("orchestration"))
}
