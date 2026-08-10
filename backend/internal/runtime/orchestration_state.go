// Package runtime contains the manager-owned execution flow.
//
// This file defines the small state carriers that let Eino graph nodes share
// request data, graph-local decisions, and final node outputs without globals.
package runtime

import (
	"context"

	"github.com/cloudwego/eino/schema"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// orchestrationGraphState is the single-turn state machine owned by the outer
// graph. It contains only state-safe values; runtime services stay in context.
type orchestrationGraphState struct {
	NextAction  orchestrationNextAction `json:"next_action,omitempty"`
	LoopStep    int                     `json:"loop_step"`
	MaxRunSteps int                     `json:"max_run_steps"`

	PreflightResult preflightResult      `json:"preflight_result"`
	Route           policy.ApprovedRoute `json:"route"`
	Plan            ExecutionPlan        `json:"plan"`
	DynamicFacts    []DynamicFacts       `json:"dynamic_facts,omitempty"`

	PendingDomainSteps []contracts.DomainStep       `json:"pending_domain_steps,omitempty"`
	DomainOutcomes     []orchestrationDomainOutcome `json:"domain_outcomes,omitempty"`
	AggregatedResult   specialists.Result           `json:"aggregated_result"`
	RawFinalText       string                       `json:"raw_final_text,omitempty"`

	PrefillAttempts  int  `json:"prefill_attempts"`
	DispatchAttempts int  `json:"dispatch_attempts"`
	PrefillCompleted bool `json:"prefill_completed"`
	Degraded         bool `json:"degraded"`

	Failure           graphFailure `json:"failure"`
	TerminationReason string       `json:"termination_reason,omitempty"`
	TurnType          string       `json:"turn_type,omitempty"`
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

// withOrchestrationInit attaches request-scoped immutable inputs to the graph context.
func withOrchestrationInit(ctx context.Context, init *orchestrationInit) context.Context {
	return context.WithValue(ctx, orchestrationInitCtxKey{}, init)
}

// getOrchestrationInit reads the request-scoped graph inputs from context.
func getOrchestrationInit(ctx context.Context) *orchestrationInit {
	init, _ := ctx.Value(orchestrationInitCtxKey{}).(*orchestrationInit)
	return init
}

// withOrchestrationRuntime attaches non-serializable services needed by graph nodes.
func withOrchestrationRuntime(ctx context.Context, rt *orchestrationRuntime) context.Context {
	return context.WithValue(ctx, orchestrationRuntimeCtxKey{}, rt)
}

// getOrchestrationRuntime reads non-serializable graph services from context.
func getOrchestrationRuntime(ctx context.Context) *orchestrationRuntime {
	rt, _ := ctx.Value(orchestrationRuntimeCtxKey{}).(*orchestrationRuntime)
	return rt
}

// withOrchestrationResult creates the side-channel result container used after graph invocation.
func withOrchestrationResult(ctx context.Context) (context.Context, *orchestrationResult) {
	r := &orchestrationResult{}
	return context.WithValue(ctx, orchestrationResultCtxKey{}, r), r
}

// getOrchestrationResult reads the side-channel result container populated by terminal nodes.
func getOrchestrationResult(ctx context.Context) *orchestrationResult {
	r, _ := ctx.Value(orchestrationResultCtxKey{}).(*orchestrationResult)
	return r
}

// genOrchestrationState seeds Eino local state with the route known at graph start.
func genOrchestrationState(ctx context.Context) *orchestrationGraphState {
	init := getOrchestrationInit(ctx)
	if init != nil {
		pending := append([]contracts.DomainStep(nil), init.Plan.DomainSteps...)
		return &orchestrationGraphState{
			Route:              init.Route,
			Plan:               init.Plan,
			MaxRunSteps:        orchestrationMaxRunSteps,
			PendingDomainSteps: pending,
		}
	}
	return &orchestrationGraphState{MaxRunSteps: orchestrationMaxRunSteps}
}

// init registers the graph-local state type so Eino can serialize and restore it.
func init() {
	schema.Register[orchestrationGraphState]()
}
