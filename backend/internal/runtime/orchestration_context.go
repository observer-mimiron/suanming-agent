// Package runtime 包含 Manager 拥有的执行主链。
//
// 本文件负责一次请求在 Graph 节点间传递的 context 携带器；
// 不负责持久化 SessionState、推进 Graph 状态或发送 HTTP/SSE。
package runtime

import (
	"context"

	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// orchestrationInit 携带本轮 Graph 的请求输入和会话引用。
// 会话指针只供 runtime 节点执行既有读写合同，不进入 Graph checkpoint；跨轮状态仍由 PersistentStore 保存。
type orchestrationInit struct {
	Session       *state.SessionState
	Route         policy.ApprovedRoute
	Plan          ExecutionPlan
	UserMessage   string
	SessionValues map[string]any
}

// orchestrationRuntime 携带 Graph 节点所需的不可序列化运行时服务。
type orchestrationRuntime struct {
	Sink     EventSink
	Executor *Executor
	Router   intent.Router
}

// orchestrationResult 保存终端节点回传给 Execute 或 Resume 的本轮结果。
type orchestrationResult struct {
	TurnType          string
	PrimaryDomain     string
	ReplyDomain       string
	Specialist        specialists.Result
	RawFinalText      string
	GraphState        *orchestrationGraphState
	Failure           graphFailure
	TerminationReason string
}

type orchestrationInitCtxKey struct{}
type orchestrationRuntimeCtxKey struct{}
type orchestrationResultCtxKey struct{}

// withOrchestrationInit 将请求范围的 Graph 输入附加到 context。
func withOrchestrationInit(ctx context.Context, init *orchestrationInit) context.Context {
	return context.WithValue(ctx, orchestrationInitCtxKey{}, init)
}

// getOrchestrationInit 读取请求范围的 Graph 输入。
func getOrchestrationInit(ctx context.Context) *orchestrationInit {
	init, _ := ctx.Value(orchestrationInitCtxKey{}).(*orchestrationInit)
	return init
}

// withOrchestrationRuntime 将 Graph 节点所需的运行时服务附加到 context。
func withOrchestrationRuntime(ctx context.Context, rt *orchestrationRuntime) context.Context {
	return context.WithValue(ctx, orchestrationRuntimeCtxKey{}, rt)
}

// getOrchestrationRuntime 读取 Graph 节点所需的运行时服务。
func getOrchestrationRuntime(ctx context.Context) *orchestrationRuntime {
	rt, _ := ctx.Value(orchestrationRuntimeCtxKey{}).(*orchestrationRuntime)
	return rt
}

// withOrchestrationResult 创建终端节点使用的结果容器。
func withOrchestrationResult(ctx context.Context) (context.Context, *orchestrationResult) {
	r := &orchestrationResult{}
	return context.WithValue(ctx, orchestrationResultCtxKey{}, r), r
}

// getOrchestrationResult 读取终端节点填充的结果容器。
func getOrchestrationResult(ctx context.Context) *orchestrationResult {
	r, _ := ctx.Value(orchestrationResultCtxKey{}).(*orchestrationResult)
	return r
}
