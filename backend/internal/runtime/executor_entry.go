// Package runtime 包含 Manager 拥有的执行主链。
//
// 本文件属于 runtime 入口层，负责 Executor wiring、Graph 调用和 final guard 后的会话收口；
// 不负责确定性排盘工具实现、领域解释或 Graph 拓扑定义。
package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"

	einomodel "github.com/cloudwego/eino/components/model"

	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/llm"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
)

// Executor 负责执行已批准的路由，驱动 manager-owned execution plan、
// prefill、bounded specialist runner 和 final guard 主链。
type Executor struct {
	reg                *tools.Registry
	toolRunner         *tools.ToolRunner
	flashChat          llm.Chat
	summarizerModel    einomodel.ToolCallingChatModel
	specialistRegistry *specialists.Registry
	builder            *AgentBuilder
	manager            *Manager
	llmModel           string
	historyLimit       int
	orchestrationGraph compose.Runnable[string, string] // 预编译 Graph
	router             intent.Router                    // semantic router，供 preflight/guidance_gate 用；nil 走 regex
}

// ExecutorConfig defines runtime wiring that is stable for the Executor lifetime.
// Router may be nil; preflight then falls back to deterministic regex guidance.
type ExecutorConfig struct {
	LLMModel     string
	HistoryLimit int
	Router       intent.Router
	Builder      AgentBuilderConfig
}

// NewExecutor 创建运行时执行器。
// summarizerModel 用于 specialist 的 summarization 中间件压缩长对话历史，传 nil 则不启用压缩。
func NewExecutor(reg *tools.Registry, sr *specialists.Registry, model einomodel.ToolCallingChatModel, flashChat llm.Chat, summarizerModel einomodel.ToolCallingChatModel, cfg ExecutorConfig) (*Executor, error) {
	graph, err := buildOrchestrationGraph()
	if err != nil {
		return nil, fmt.Errorf("compile orchestration graph: %w", err)
	}
	return &Executor{
		reg:                reg,
		toolRunner:         tools.NewToolRunner(reg),
		flashChat:          flashChat,
		summarizerModel:    summarizerModel,
		specialistRegistry: sr,
		builder:            NewAgentBuilder(model, reg, flashChat, summarizerModel, cfg.Builder),
		llmModel:           cfg.LLMModel,
		historyLimit:       cfg.HistoryLimit,
		manager:            NewManager(flashChat),
		orchestrationGraph: graph,
		router:             cfg.Router,
	}, nil
}

// Execute 执行已批准的路由，并返回本轮 turn type 与最终文本。
//
// 流程：
//  1. 解析资产焦点，合并 supervisor 新抽取的出生资料。
//  2. 由 Manager 生成 ExecutionPlan，并同步 route/debug snapshot。
//  3. 注入 graph init/runtime/result state 到 ctx。
//  4. 调用预编译 bounded Graph（preflight → decide → prefill/dispatch/aggregate）。
//  5. 在 Graph 返回后执行唯一 final guard，再保存 follow-up 资产。
func (e *Executor) Execute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (turnType string, assistantText string, err error) {
	turnContext := captureTurnContext(route)
	route = resolveArtifactFocus(st, route, message)
	// ProfileRevision is part of the artifact owner contract. Merge supervisor
	// extracted birth data before building the plan, otherwise prefill writes a
	// chart under a newer owner than the ArtifactRequirement expects.
	if len(route.Slots.Profile) > 0 {
		st.MergeProfile(route.Slots.Profile)
	}
	plan := e.manager.BuildExecutionPlanForTurn(st, route, message, turnContext)
	route = plan.Route

	e.syncExecutionRoute(ctx, st, route, plan)

	// 构造本轮 specialist 运行所需的 SessionValues。
	vals := e.buildSessionValues(sessionViewFromState(st), route)

	// 注入 init + runtime + result 到 ctx
	// Graph state（PreflightResult/Route）由 WithGenLocalState 管理，节点 Lambda 用 ProcessState 读写
	ctx = withOrchestrationInit(ctx, &orchestrationInit{
		St:      st,
		Route:   route,
		Plan:    plan,
		UserMsg: message,
		Vals:    vals,
	})
	ctx = withOrchestrationRuntime(ctx, &orchestrationRuntime{
		Sink:     sink,
		Executor: e,
		Router:   e.router,
	})
	ctx, result := withOrchestrationResult(ctx)

	finalText, err := e.orchestrationGraph.Invoke(ctx, message)
	if err != nil {
		annotateRuntimeFailureTrace(ctx, err)
		return "agent_error", finalText, classifyRuntimeFailure(route.PrimaryDomain, failureStageAgent, err)
	}

	if result == nil || result.GraphState == nil {
		invariant := fmt.Errorf("orchestration graph returned no terminal state")
		annotateRuntimeFailureTrace(ctx, invariant)
		return "agent_error", "", classifyRuntimeFailure(route.PrimaryDomain, failureStageAgent, invariant)
	}
	if result.Failure.hasFailure() {
		graphErr := graphFailureError(result.Failure)
		if graphErr == nil {
			graphErr = fmt.Errorf("orchestration graph terminated with failure")
		}
		failure := &RuntimeFailure{
			Class:       firstNonEmpty(result.Failure.FailureClass, failureClassInvariantFailure),
			Stage:       firstNonEmpty(result.Failure.FailureStage, failureStageAgent),
			Domain:      firstNonEmpty(result.Failure.Domain, route.PrimaryDomain),
			Code:        firstNonEmpty(result.Failure.FailureCode, "ORCHESTRATION_FAILED"),
			Retryable:   result.Failure.Retryable,
			Degraded:    result.Failure.Degraded,
			UserVisible: true,
			Message:     firstNonEmpty(result.Failure.Message, "本轮执行未形成可安全展示的结果。"),
			Cause:       graphErr,
		}
		annotateRuntimeFailureTrace(ctx, failure)
		return "agent_error", "", failure
	}

	// Short-circuit 文本是前置澄清或资料提示，不需要 artifact guard；普通
	// specialist 结果必须在 Invoke 返回后统一通过 final guard。
	if result.TerminationReason == "short_circuit" {
		result.TurnType = firstNonEmpty(result.TurnType, "clarification")
		finalText = result.RawFinalText
		e.updateGuidanceState(st, route, message, result.GraphState.PreflightResult)
		if strings.TrimSpace(finalText) != "" {
			_ = emitEventWithTrace(ctx, sink, Event{Type: "text", Data: map[string]any{"content": finalText}}, map[string]any{"buffer_final": true, "turn_type": result.TurnType})
		}
		result.Specialist = specialists.Result{Domain: route.PrimaryDomain, Summary: finalText}
		storeFollowupArtifact(st, route, result.Specialist, finalText, message, result.TurnType)
		e.manager.FinishTurn(st, route, result.TurnType)
		return result.TurnType, finalText, nil
	}

	// Execute 只负责把 graph 原始结果经 final guard 后整理成统一的
	// follow-up 资产与 Manager 状态；Graph 内不发送最终 text。
	rawGraphText := firstNonEmpty(result.RawFinalText, finalText)
	// guided fallback 可能在 Graph 内重建计划；最终 guard 必须消费终态计划，
	// 否则前置和 dispatch 已切到新领域，guard 仍会按旧领域错误拦截结果。
	effectivePlan := result.GraphState.Plan
	guardedTurnType, guardedText := guardFinalAnswerWithPlan(ctx, effectivePlan, st, rawGraphText)
	result.TurnType = guardedTurnType
	finalText = guardedText
	if strings.TrimSpace(guardedText) != "" {
		_ = emitEventWithTrace(ctx, sink, Event{
			Type: "text",
			Data: map[string]any{"content": guardedText},
		}, map[string]any{"buffer_final": true, "turn_type": guardedTurnType})
	}

	finalRoute := route
	if result.PrimaryDomain != "" {
		finalRoute.PrimaryDomain = result.PrimaryDomain
	}
	finalResult := result.Specialist
	// Guard 已经是本轮唯一的用户可见文本边界；这里仅补齐 follow-up
	// 资产的结构化元数据，不能再次 compose，否则会出现发送文本与存档文本分叉。
	if finalResult.Domain == "" {
		finalResult.Domain = firstNonEmpty(result.ReplyDomain, finalRoute.PrimaryDomain)
	}
	finalResult.Summary = finalText
	if strings.TrimSpace(finalResult.Domain) == "" {
		finalResult.Domain = firstNonEmpty(result.ReplyDomain, finalRoute.PrimaryDomain)
	}
	storeFollowupArtifact(st, finalRoute, finalResult, finalText, message, result.TurnType)
	e.manager.FinishTurn(st, finalRoute, result.TurnType)
	return result.TurnType, finalText, nil
}
