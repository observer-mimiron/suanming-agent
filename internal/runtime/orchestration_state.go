package runtime

import (
	"context"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// orchestrationState 是执行骨架 Graph 的 per-request 状态。
//
// 通过 ctx.Value 注入，各节点 Lambda 通过 getOrchestrationState(ctx) 读取。
// agentEventBridge 的业务规则保留在 agent Lambda 内，不下放到 Graph 边。
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
