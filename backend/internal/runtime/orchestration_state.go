package runtime

import (
	"context"

	"github.com/cloudwego/eino/schema"

	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// orchestrationGraphState holds the serialized graph-local state that must
// survive checkpoint resume.
type orchestrationGraphState struct {
	PreflightResult preflightResult
	Route           policy.ApprovedRoute
}

// orchestrationInit carries per-request runtime state through context. This is
// not serialized into checkpoints; session state already lives in PersistentStore.
type orchestrationInit struct {
	St      *state.SessionState
	Route   policy.ApprovedRoute
	Plan    ExecutionPlan
	UserMsg string
	Vals    map[string]any
}

// orchestrationRuntime carries non-serializable references needed by graph nodes.
type orchestrationRuntime struct {
	Sink     EventSink
	Executor *Executor
	Router   intent.Router
}

// orchestrationResult lets graph nodes report the final turn type/domain back to
// Execute or Resume without extending the graph output payload.
type orchestrationResult struct {
	TurnType      string
	PrimaryDomain string
	ReplyDomain   string
	Specialist    specialists.Result
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

func withOrchestrationResult(ctx context.Context) (context.Context, *orchestrationResult) {
	r := &orchestrationResult{}
	return context.WithValue(ctx, orchestrationResultCtxKey{}, r), r
}

func getOrchestrationResult(ctx context.Context) *orchestrationResult {
	r, _ := ctx.Value(orchestrationResultCtxKey{}).(*orchestrationResult)
	return r
}

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
