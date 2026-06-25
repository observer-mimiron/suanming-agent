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
