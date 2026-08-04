// Package runtime contains the manager-owned execution flow.
//
// This file wires the outer Eino orchestration graph:
// preflight -> short_circuit | prefill -> agent -> final_guard.
package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/observer-mimiron/suanming-agent/internal/specialists"
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

// preflightBranch selects the graph edge after preflight.
// The returned labels must match the branch map in buildOrchestrationGraph.
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

// emitShortCircuitNode sends preflight's user-visible answer and stops execution.
// This path covers clarification, missing-profile prompts, and direct follow-up reuse.
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
	getOrchestrationResult(ctx).PrimaryDomain = oc.GS.Route.PrimaryDomain
	getOrchestrationResult(ctx).ReplyDomain = oc.GS.Route.PrimaryDomain
	return oc.GS.PreflightResult.Text, nil
}

// prefillNode prepares deterministic assets required by the Manager's plan.
// Guided fallback can replace the route after preflight, so this node rebuilds
// the plan before touching artifacts when ForcedRoute is present.
func prefillNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}

	route := oc.Init.Route
	plan := oc.Init.Plan
	if oc.GS.PreflightResult.ForcedRoute != nil {
		route = *oc.GS.PreflightResult.ForcedRoute
		plan = oc.RT.Executor.manager.BuildExecutionPlan(oc.Init.St, route, oc.Init.UserMsg)
		oc.GS.Route = route
		oc.RT.Executor.syncExecutionRoute(ctx, oc.Init.St, route, plan)
	}
	oc.RT.Executor.prefill(ctx, oc.RT.Sink, oc.Init.St, plan, oc.Init.Vals)
	return in, nil
}

// guardNode applies final output contracts and emits the buffered final text.
// It is deliberately last so missing artifacts or internal leakage are blocked
// after all specialist/manager composition has finished.
func guardNode(ctx context.Context, finalText string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}

	span := tracing.SpanFromContext(ctx, "final_guard", tracing.KindChain)
	defer span.End()

	turnType, guardedText := guardFinalAnswerWithTrace(ctx, oc.GS.Route, oc.Init.St, finalText)
	span.SetAttribute("turn_type", turnType)
	if guardedText != "" {
		_ = emitEventWithTrace(ctx, oc.RT.Sink, Event{
			Type: "text",
			Data: map[string]any{"content": guardedText},
		}, map[string]any{"buffer_final": true, "turn_type": turnType})
	}

	getOrchestrationResult(ctx).TurnType = turnType
	getOrchestrationResult(ctx).PrimaryDomain = oc.GS.Route.PrimaryDomain
	getOrchestrationResult(ctx).ReplyDomain = oc.GS.Route.PrimaryDomain
	return guardedText, nil
}

// agentNode dispatches the manager-approved execution plan to the correct worker.
// Pure BaZi uses the authority-first internal graph; other domains use bounded
// specialist runners and then Manager compose, never free-form runtime tool choice.
func agentNode(ctx context.Context, in string) (*schema.StreamReader[string], error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return nil, err
	}

	nodeSpan := tracing.SpanFromContext(ctx, "agent", tracing.KindChain)
	nodeSpan.SetAttribute("primary_domain", oc.GS.Route.PrimaryDomain)

	route := oc.GS.Route
	plan := oc.Init.Plan
	if oc.GS.PreflightResult.ForcedRoute != nil {
		route = *oc.GS.PreflightResult.ForcedRoute
		plan = oc.RT.Executor.manager.BuildExecutionPlan(oc.Init.St, route, oc.Init.UserMsg)
		nodeSpan.SetAttribute("forced_route", true)
		oc.RT.Executor.syncExecutionRoute(ctx, oc.Init.St, route, plan)
	}
	if plan.FollowupMode != "" {
		nodeSpan.SetAttribute("followup_mode", plan.FollowupMode)
	}

	oc.GS.Route = route

	if oc.GS.PreflightResult.ForcedRoute != nil && oc.GS.PreflightResult.Text != "" {
		_ = emitEventWithTrace(ctx, oc.RT.Sink, Event{
			Type: "text",
			Data: map[string]any{"content": oc.GS.PreflightResult.Text},
		}, map[string]any{"turn_type": oc.GS.PreflightResult.TurnType})
	}

	oc.RT.Executor.updateGuidanceState(oc.Init.St, route, oc.Init.UserMsg, oc.GS.PreflightResult)

	route = plan.Route
	oc.GS.Route = route

	if shouldUseBaziCharterGraph(plan) {
		finalText, err := oc.RT.Executor.runBaziAuthorityFirstGraph(ctx, oc.RT.Sink, oc.Init.St, oc.Init.UserMsg)
		if err != nil {
			return nil, fmt.Errorf("run pure bazi charter graph: %w", err)
		}
		sr, sw := schema.Pipe[string](1)
		go func() {
			defer sw.Close()
			defer nodeSpan.End()
			nodeSpan.SetAttribute("inner_graph", "bazi_authority_first")
			nodeSpan.SetAttribute("dispatch_owner", "manager")
			nodeSpan.SetAttribute("dispatch_domains", strings.Join(plan.Domains, ","))
			nodeSpan.SetAttribute("final_text_len", len([]rune(finalText)))
			getOrchestrationResult(ctx).ReplyDomain = route.PrimaryDomain
			getOrchestrationResult(ctx).Specialist = specialists.Result{
				Domain:  route.PrimaryDomain,
				Summary: finalText,
			}
			sw.Send(finalText, nil)
		}()
		return sr, nil
	}

	resultCh := make(chan struct {
		result specialists.Result
		err    error
	}, 1)

	go func() {
		defer close(resultCh)
		defer nodeSpan.End()
		nodeSpan.SetAttribute("dispatch_domains", strings.Join(plan.Domains, ","))

		nodeSpan.SetAttribute("dispatch_owner", "manager")
		result, runErr := oc.RT.Executor.runExecutionPlan(ctx, oc.RT.Sink, oc.Init.St, plan, oc.Init.UserMsg)
		if runErr == nil {
			result.Summary = oc.RT.Executor.manager.ComposeFinalReply(oc.Init.UserMsg, result)
			nodeSpan.SetAttribute("final_text_len", len([]rune(result.Summary)))
			getOrchestrationResult(ctx).ReplyDomain = result.Domain
			getOrchestrationResult(ctx).Specialist = result
		}
		resultCh <- struct {
			result specialists.Result
			err    error
		}{result: result, err: runErr}
	}()

	sr, sw := schema.Pipe[string](1)
	go func() {
		defer sw.Close()
		outcome := <-resultCh
		if outcome.err != nil {
			nodeSpan.RecordError(outcome.err)
			nodeSpan.SetStatus("error")
			sw.Send("", outcome.err)
			return
		}
		sw.Send(outcome.result.Summary, nil)
	}()
	return sr, nil
}

// buildOrchestrationGraph compiles the outer runtime graph once per Executor.
// The graph owns control flow only; concrete state mutation stays in Manager,
// Executor, prefill, and final guard helpers called by the nodes.
func buildOrchestrationGraph() (compose.Runnable[string, string], error) {
	g := compose.NewGraph[string, string](compose.WithGenLocalState(genOrchestrationState))

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

	if err := g.AddBranch("preflight", compose.NewGraphBranch(
		preflightBranch,
		map[string]bool{"short_circuit": true, "prefill": true},
	)); err != nil {
		return nil, fmt.Errorf("add preflight branch: %w", err)
	}

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

	return g.Compile(context.Background(), compose.WithGraphName("orchestration"))
}
