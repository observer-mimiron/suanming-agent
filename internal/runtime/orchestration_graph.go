package runtime

import (
	"context"

	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// preflightNode 是 Graph 的 preflight 节点：执行确定性校验，结果写入 state。
// Branch 节点根据 state 中的 result 决定走 short_circuit 还是 main。
//
// preflight tracing span 整段保留在此节点内（原 executor.go:71-79 的逻辑），
// 保证 short_circuit / turn_type 属性不丢失。
func preflightNode(ctx context.Context, in string) (string, error) {
	s := getOrchestrationState(ctx)
	preflightSpan := tracing.SpanFromContext(ctx, "preflight", tracing.KindChain)
	preflightSpan.SetAttribute("primary_domain", s.route.PrimaryDomain)
	preflightSpan.SetAttribute("task_intent", s.route.TaskIntent)
	result := preflight(s.st, s.route, s.userMsg)
	preflightSpan.SetAttribute("short_circuit", result.ShortCircuit)
	if result.TurnType != "" {
		preflightSpan.SetAttribute("turn_type", result.TurnType)
	}
	preflightSpan.End()
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

// emitShortCircuitNode 处理 preflight 短路：emit text 事件并返回。
// executor.go 原 Execute:80-86 的 updateGuidanceState + emit 逻辑整体移入此处。
//
// guardedTurnType 同步赋值，让 Execute 通过 state 拿到短路路径的 turnType，
// 与 guardNode 路径共用 state.guardedTurnType 字段作为返回通道。
func emitShortCircuitNode(ctx context.Context, in string) (string, error) {
	s := getOrchestrationState(ctx)
	s.executor.updateGuidanceState(s.st, s.route, s.userMsg, s.preflightResult)
	_ = emitEventWithTrace(ctx, s.sink, Event{
		Type: "text",
		Data: map[string]any{"content": s.preflightResult.Text},
	}, map[string]any{"turn_type": s.preflightResult.TurnType})
	s.guardedTurnType = s.preflightResult.TurnType
	return s.preflightResult.Text, nil
}

// prefillNode 调用现有 executor.prefill，结果写入 state.vals 和 session state。
// 不使用 AddGraphNode 嵌入子图——底层工具不暴露中间阶段，子图无意义。
func prefillNode(ctx context.Context, in string) (string, error) {
	s := getOrchestrationState(ctx)
	s.executor.prefill(ctx, s.sink, s.st, s.route, s.vals)
	return in, nil
}

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
