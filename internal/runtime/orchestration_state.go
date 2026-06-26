package runtime

import (
	"context"

	"github.com/cloudwego/eino/schema"

	"github.com/wikiglobal/suanming-agent/internal/intent"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// orchestrationGraphState 是 Graph 的 local state，可序列化存入 Checkpoint。
// 通过 compose.WithGenLocalState 创建，节点 Lambda 用 compose.ProcessState 读写。
// 中断-恢复时由 Eino Checkpoint 自动序列化/反序列化。
//
// 只存需要跨中断-恢复的字段（中断点在 prefill 后、agent 前）：
//   - PreflightResult: agentNode 用（ForcedRoute 检查 + transition text + updateGuidanceState）
//   - Route: agentNode 用（BuildSupervisor + domain），ForcedRoute 覆盖时同步更新
//
// St/UserMsg/Vals 不进 Graph state——St 在 PersistentStore，UserMsg/Vals 是 per-request 的，
// Resume 时从 ctx.Value（init）拿。Vals 是 map[string]any，Eino serializer 对 any 字段要求注册，
// 不放进 Graph state 规避序列化问题。
//
// GuardedTurnType 不进 Graph state——它只在 guardNode/emitShortCircuitNode 设置
// （都在中断点之后，Resume 时会重新执行），通过 ctx.Value 的 result 容器传回 Execute。
type orchestrationGraphState struct {
	PreflightResult preflightResult
	Route           policy.ApprovedRoute
}

// orchestrationInit 是 Execute/Resume 注入的 per-request 初始数据，通过 ctx.Value 传给
// genOrchestrationState 和节点 Lambda。不进 Checkpoint（St 在 PersistentStore）。
type orchestrationInit struct {
	St      *state.SessionState
	Route   policy.ApprovedRoute // 第一次请求的路由，genOrchestrationState 用；Resume 时 Graph state 从 Checkpoint 恢复，此字段被忽略
	UserMsg string
	Vals    map[string]any
}

// orchestrationRuntime 保留非序列化引用，通过 ctx.Value 传递（不进 Checkpoint）。
type orchestrationRuntime struct {
	Sink     EventSink
	Executor *Executor
	Router   intent.Router // 从 Executor.router 传入，供 preflight 用
}

// orchestrationResult 是 Execute/Resume 拿返回值的容器，通过 ctx.Value 传递。
// guardNode/emitShortCircuitNode 写入 TurnType，Execute 读取。
type orchestrationResult struct {
	TurnType string
}

type orchestrationInitCtxKey struct{}
type orchestrationRuntimeCtxKey struct{}
type orchestrationResultCtxKey struct{}

func withOrchestrationInit(ctx context.Context, init *orchestrationInit) context.Context {
	return context.WithValue(ctx, orchestrationInitCtxKey{}, init)
}

func getOrchestrationInit(ctx context.Context) *orchestrationInit {
	init, _ := ctx.Value(orchestrationInitCtxKey{}).(*orchestrationInit)
	return init
}

func withOrchestrationRuntime(ctx context.Context, rt *orchestrationRuntime) context.Context {
	return context.WithValue(ctx, orchestrationRuntimeCtxKey{}, rt)
}

func getOrchestrationRuntime(ctx context.Context) *orchestrationRuntime {
	rt, _ := ctx.Value(orchestrationRuntimeCtxKey{}).(*orchestrationRuntime)
	return rt
}

// withOrchestrationResult 注入一个 result 容器到 ctx，返回 ctx 和容器指针供 Execute 读取。
func withOrchestrationResult(ctx context.Context) (context.Context, *orchestrationResult) {
	r := &orchestrationResult{}
	return context.WithValue(ctx, orchestrationResultCtxKey{}, r), r
}

func getOrchestrationResult(ctx context.Context) *orchestrationResult {
	r, _ := ctx.Value(orchestrationResultCtxKey{}).(*orchestrationResult)
	return r
}

// genOrchestrationState 是 WithGenLocalState 的 state 生成函数。
// 第一次请求时从 init 拿 Route 构造初始 state；Resume 时 Graph 从 Checkpoint 恢复 state，此函数不调用。
func genOrchestrationState(ctx context.Context) *orchestrationGraphState {
	init := getOrchestrationInit(ctx)
	if init != nil {
		return &orchestrationGraphState{
			Route: init.Route,
		}
	}
	return &orchestrationGraphState{}
}

func init() {
	schema.Register[orchestrationGraphState]()
}
