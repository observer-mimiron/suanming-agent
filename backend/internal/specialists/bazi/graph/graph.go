// Package graph 包含八字领域拥有的有界执行图。
//
// 本包负责单轮状态机、动作选择和图拓扑；
// runtime 只适配模型、检索、追踪和 SSE 能力，不选择八字下一步或持有图内状态。
package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"

	"github.com/observer-mimiron/suanming-agent/internal/repair"
)

const (
	// MaxRunSteps 限制业务状态机的一轮决策次数，防止补证、修复或降级无限循环。
	MaxRunSteps = 24
	// maxFrameworkRunSteps 限制框架节点访问次数。一次业务决策最多经过决定、动作和合同校验三个节点，
	// 因而不能复用 MaxRunSteps；否则合法的补证和降级路径会在终态渲染前被框架截断。
	maxFrameworkRunSteps = MaxRunSteps*3 + 2
)

// Action is the closed set of BaZi graph transitions.
type Action string

const (
	ActionAnalysisPlan Action = "analysis_plan"
	ActionEvidence     Action = "evidence_action"
	ActionStatic       Action = "static_judgment"
	ActionLifetime     Action = "lifetime_dayun_judgment"
	ActionDynamic      Action = "dynamic_judgment"
	ActionRepair       Action = "repair"
	ActionRecoverFacts Action = "recover_facts"
	ActionRender       Action = "render"
	ActionHardError    Action = "hard_error"
)

const (
	phaseAnalysisPlan = "analysis_plan"
	phaseEvidence     = "evidence"
	phaseStatic       = "static"
	phaseLifetime     = "lifetime"
	phaseDynamic      = "dynamic"
	phaseRepair       = "repair"
)

// Failure is the serializable failure shape exchanged by domain nodes.
// It deliberately stores no Go error, client, executor or event sink.
type Failure struct {
	Class       string   `json:"failure_class,omitempty"`
	Stage       string   `json:"failure_stage,omitempty"`
	Code        string   `json:"failure_code,omitempty"`
	Domain      string   `json:"domain,omitempty"`
	Retryable   bool     `json:"retryable,omitempty"`
	Degraded    bool     `json:"degraded,omitempty"`
	Message     string   `json:"message,omitempty"`
	MissingRefs []string `json:"missing_refs,omitempty"`
	AllowedRefs []string `json:"allowed_refs,omitempty"`
}

// HasFailure reports whether a node wrote a recoverable or terminal failure.
func (f Failure) HasFailure() bool {
	return strings.TrimSpace(f.Class) != "" || strings.TrimSpace(f.Code) != "" || strings.TrimSpace(f.Message) != ""
}

// State contains the graph-owned control facts for one BaZi turn. Payload is a
// domain-data carrier owned by the adapter; it never contains model clients,
// executors or SSE sinks and is not used for action selection.
type State struct {
	Phase       string `json:"phase,omitempty"`
	NextAction  Action `json:"next_action,omitempty"`
	LoopStep    int    `json:"loop_step"`
	MaxRunSteps int    `json:"max_run_steps"`

	ChartReady          bool `json:"chart_ready"`
	AnalysisPlanned     bool `json:"analysis_planned"`
	NeedDynamic         bool `json:"need_dynamic"`
	NeedLifetimeDayun   bool `json:"need_lifetime_dayun"`
	CurrentPeriodReady  bool `json:"current_period_ready"`
	EvidenceValidated   bool `json:"evidence_validated"`
	EvidenceNeedsAction bool `json:"evidence_needs_action"`
	StaticAttempted     bool `json:"static_attempted"`
	StaticAccepted      bool `json:"static_accepted"`
	LifetimeAttempted   bool `json:"lifetime_attempted"`
	LifetimeAccepted    bool `json:"lifetime_accepted"`
	DynamicAttempted    bool `json:"dynamic_attempted"`
	DynamicAccepted     bool `json:"dynamic_accepted"`

	Failure        Failure        `json:"failure"`
	RecoveryPolicy string         `json:"recovery_policy,omitempty"`
	RepairState    repair.State   `json:"repair_state"`
	RepairFailure  repair.Failure `json:"repair_failure"`
	RepairAction   repair.Action  `json:"repair_action,omitempty"`

	EvidenceAttempts  int `json:"evidence_attempts"`
	TransportAttempts int `json:"transport_attempts"`
	RepairAttempts    int `json:"repair_attempts"`

	RecoveryState     string `json:"recovery_state,omitempty"`
	TerminationReason string `json:"termination_reason,omitempty"`
	Output            string `json:"output,omitempty"`

	Payload any `json:"-"`
}

// Result is the typed terminal projection returned by the BaZi graph.
type Result struct {
	Text              string
	RecoveryState     string
	TerminationReason string
	Failure           Failure
	ContractAudit     any
	// Payload 是 adapter 消费的终态领域数据；Graph 只负责透传，不用它选择动作。
	Payload any
}

// Deps is the narrow capability surface required by the BaZi graph. Each
// callback updates State and may use request-scoped runtime services captured by
// the adapter. It must not make an action-selection decision.
type Deps struct {
	Bootstrap        func(context.Context, *State) error
	AnalysisPlan     func(context.Context, *State) error
	Evidence         func(context.Context, *State) error
	ValidateEvidence func(context.Context, *State) error
	Static           func(context.Context, *State) error
	Lifetime         func(context.Context, *State) error
	Dynamic          func(context.Context, *State) error
	ContractCheck    func(context.Context, *State) error
	Repair           func(context.Context, *State) error
	RecoverFacts     func(context.Context, *State) error
	Render           func(context.Context, *State) error
	HardError        func(context.Context, *State) error
	TraceAttributes  func(context.Context, map[string]any)
}

// Compile builds the Pregel graph for one request-scoped dependency adapter.
func Compile(ctx context.Context, deps Deps) (compose.Runnable[*State, *State], error) {
	if err := validateDeps(deps); err != nil {
		return nil, err
	}
	g := compose.NewGraph[*State, *State]()
	add := func(key string, fn func(context.Context, *State) (*State, error)) error {
		return g.AddLambdaNode(key, compose.InvokableLambda(fn), compose.WithNodeName("bazi."+key))
	}
	if err := add("bootstrap", passthroughNode(deps.Bootstrap)); err != nil {
		return nil, fmt.Errorf("add bazi bootstrap: %w", err)
	}
	if err := add("decide_next", decideNode(deps)); err != nil {
		return nil, fmt.Errorf("add bazi decide_next: %w", err)
	}
	if err := add("analysis_plan", passthroughNode(deps.AnalysisPlan)); err != nil {
		return nil, fmt.Errorf("add bazi analysis_plan: %w", err)
	}
	if err := add("evidence_action", passthroughNode(deps.Evidence)); err != nil {
		return nil, fmt.Errorf("add bazi evidence_action: %w", err)
	}
	if err := add("validate_evidence", passthroughNode(deps.ValidateEvidence)); err != nil {
		return nil, fmt.Errorf("add bazi validate_evidence: %w", err)
	}
	if err := add("static_judgment", passthroughNode(deps.Static)); err != nil {
		return nil, fmt.Errorf("add bazi static_judgment: %w", err)
	}
	if err := add("lifetime_dayun_judgment", passthroughNode(deps.Lifetime)); err != nil {
		return nil, fmt.Errorf("add bazi lifetime_dayun_judgment: %w", err)
	}
	if err := add("dynamic_judgment", passthroughNode(deps.Dynamic)); err != nil {
		return nil, fmt.Errorf("add bazi dynamic_judgment: %w", err)
	}
	if err := add("contract_check", passthroughNode(deps.ContractCheck)); err != nil {
		return nil, fmt.Errorf("add bazi contract_check: %w", err)
	}
	if err := add("repair", passthroughNode(deps.Repair)); err != nil {
		return nil, fmt.Errorf("add bazi repair: %w", err)
	}
	if err := add("recover_facts", passthroughNode(deps.RecoverFacts)); err != nil {
		return nil, fmt.Errorf("add bazi recover_facts: %w", err)
	}
	if err := add("render", renderNode(deps)); err != nil {
		return nil, fmt.Errorf("add bazi render: %w", err)
	}
	if err := add("hard_error", hardErrorNode(deps)); err != nil {
		return nil, fmt.Errorf("add bazi hard_error: %w", err)
	}

	if err := g.AddEdge(compose.START, "bootstrap"); err != nil {
		return nil, fmt.Errorf("edge START->bootstrap: %w", err)
	}
	if err := g.AddEdge("bootstrap", "decide_next"); err != nil {
		return nil, fmt.Errorf("edge bootstrap->decide_next: %w", err)
	}
	if err := g.AddBranch("decide_next", compose.NewGraphBranch(branch, map[string]bool{
		"analysis_plan": true, "evidence_action": true, "static_judgment": true, "lifetime_dayun_judgment": true, "dynamic_judgment": true,
		"repair": true, "recover_facts": true, "render": true, "hard_error": true,
	})); err != nil {
		return nil, fmt.Errorf("add decide_next branch: %w", err)
	}
	for _, edge := range [][2]string{
		{"analysis_plan", "decide_next"}, {"evidence_action", "validate_evidence"}, {"validate_evidence", "decide_next"},
		{"static_judgment", "contract_check"}, {"lifetime_dayun_judgment", "contract_check"}, {"dynamic_judgment", "contract_check"}, {"repair", "contract_check"},
		{"contract_check", "decide_next"}, {"recover_facts", "render"}, {"hard_error", compose.END}, {"render", compose.END},
	} {
		if err := g.AddEdge(edge[0], edge[1]); err != nil {
			return nil, fmt.Errorf("edge %s->%s: %w", edge[0], edge[1], err)
		}
	}
	return g.Compile(ctx, compose.WithNodeTriggerMode(compose.AnyPredecessor), compose.WithMaxRunSteps(maxFrameworkRunSteps), compose.WithGraphName("bazi_deterministic"))
}

// Run invokes one bounded BaZi graph with a fresh in-memory state.
func Run(ctx context.Context, deps Deps, state *State) (Result, error) {
	if state == nil {
		return Result{}, fmt.Errorf("bazi graph state is nil")
	}
	if state.MaxRunSteps <= 0 {
		state.MaxRunSteps = MaxRunSteps
	}
	if state.RepairState.MaxTurnRepairAttempts == 0 {
		state.RepairState = repair.NewState()
	}
	runnable, err := Compile(ctx, deps)
	if err != nil {
		return Result{}, err
	}
	out, err := runnable.Invoke(ctx, state)
	if err != nil {
		return Result{}, err
	}
	if out == nil {
		return Result{}, fmt.Errorf("bazi graph returned nil state")
	}
	return Result{
		Text:              out.Output,
		RecoveryState:     out.RecoveryState,
		TerminationReason: out.TerminationReason,
		Failure:           out.Failure,
		Payload:           out.Payload,
	}, nil
}

// validateDeps 在编译前拒绝缺失 callback。这里保持逐项 typed 检查，因为把
// nil 函数放进 any 会隐藏 nil 值，让错误延迟到 Graph 节点真正执行时才暴露。
func validateDeps(deps Deps) error {
	for _, item := range []struct {
		name  string
		isNil bool
	}{
		{"bootstrap", deps.Bootstrap == nil}, {"analysis_plan", deps.AnalysisPlan == nil}, {"evidence", deps.Evidence == nil},
		{"validate_evidence", deps.ValidateEvidence == nil}, {"static", deps.Static == nil}, {"lifetime", deps.Lifetime == nil}, {"dynamic", deps.Dynamic == nil},
		{"contract_check", deps.ContractCheck == nil}, {"repair", deps.Repair == nil}, {"recover_facts", deps.RecoverFacts == nil},
		{"render", deps.Render == nil}, {"hard_error", deps.HardError == nil},
	} {
		if item.isNil {
			return fmt.Errorf("bazi graph dependency %s is nil", item.name)
		}
	}
	return nil
}

func passthroughNode(fn func(context.Context, *State) error) func(context.Context, *State) (*State, error) {
	return func(ctx context.Context, state *State) (*State, error) {
		if state == nil {
			return nil, fmt.Errorf("bazi graph state is nil")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := fn(ctx, state); err != nil {
			return nil, err
		}
		return state, nil
	}
}

// hardErrorNode owns the terminal reason even when an adapter only records a
// classified failure. This keeps the result contract independent of callbacks.
func hardErrorNode(deps Deps) func(context.Context, *State) (*State, error) {
	return func(ctx context.Context, state *State) (*State, error) {
		if state == nil {
			return nil, fmt.Errorf("bazi graph state is nil")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := deps.HardError(ctx, state); err != nil {
			return nil, err
		}
		if strings.TrimSpace(state.TerminationReason) == "" {
			state.TerminationReason = "hard_error"
		}
		traceTerminal(ctx, deps, state)
		return state, nil
	}
}

// renderNode makes normal completion explicit before returning from the graph.
func renderNode(deps Deps) func(context.Context, *State) (*State, error) {
	return func(ctx context.Context, state *State) (*State, error) {
		if state == nil {
			return nil, fmt.Errorf("bazi graph state is nil")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := deps.Render(ctx, state); err != nil {
			return nil, err
		}
		if strings.TrimSpace(state.TerminationReason) == "" {
			state.TerminationReason = "completed"
		}
		traceTerminal(ctx, deps, state)
		return state, nil
	}
}

func traceTerminal(ctx context.Context, deps Deps, state *State) {
	if deps.TraceAttributes == nil || state == nil {
		return
	}
	deps.TraceAttributes(ctx, map[string]any{"bazi.termination_reason": state.TerminationReason})
}

func decideNode(deps Deps) func(context.Context, *State) (*State, error) {
	return func(ctx context.Context, state *State) (*State, error) {
		if state == nil {
			return nil, fmt.Errorf("bazi graph state is nil")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		state.LoopStep++
		if state.LoopStep >= state.MaxRunSteps {
			if state.StaticAccepted {
				if state.NeedDynamic && !state.DynamicAccepted {
					state.Phase, state.NextAction = phaseDynamic, ActionRecoverFacts
				} else {
					state.NextAction = ActionRender
				}
				state.TerminationReason = "graph_step_limit_degraded"
			} else {
				state.NextAction = ActionHardError
				state.TerminationReason = "graph_step_limit"
				state.Failure = Failure{Class: "invariant_failure", Stage: "decide_next", Code: "BAZI_MAX_STEPS", Domain: "bazi", Message: "八字内部图达到本轮步数上限，未形成可安全展示的结果。"}
			}
		} else {
			state.NextAction = chooseAction(state)
		}
		if deps.TraceAttributes != nil {
			deps.TraceAttributes(ctx, map[string]any{
				"bazi.loop_step": state.LoopStep, "bazi.next_action": string(state.NextAction), "bazi.max_run_steps": state.MaxRunSteps,
				"bazi.evidence_attempts": state.EvidenceAttempts, "bazi.repair_attempts": state.RepairAttempts, "bazi.transport_attempts": state.TransportAttempts,
			})
		}
		return state, nil
	}
}

func chooseAction(state *State) Action {
	if state.Failure.HasFailure() {
		decision := repair.DefaultPolicy().Decide(state.RepairFailure, state.RepairState)
		state.RepairAction = decision.Action
		if decision.Action == repair.ActionRepairNode && !decision.Exhausted {
			state.Phase = phaseRepair
			return ActionRepair
		}
		if decision.Action == repair.ActionFallback && strings.TrimSpace(state.RepairFailure.Fallback) != "" {
			return ActionRecoverFacts
		}
		return ActionHardError
	}
	if !state.ChartReady {
		state.Failure = Failure{Class: "artifact_missing", Stage: "bootstrap", Code: "BAZI_CHART_MISSING", Domain: "bazi", Message: "八字命盘事实未就绪，无法继续内部裁断。"}
		return ActionHardError
	}
	if !state.AnalysisPlanned {
		state.Phase = phaseAnalysisPlan
		return ActionAnalysisPlan
	}
	if state.EvidenceNeedsAction {
		state.Phase = phaseEvidence
		return ActionEvidence
	}
	if !state.StaticAttempted {
		state.Phase = phaseStatic
		return ActionStatic
	}
	if state.NeedLifetimeDayun && !state.LifetimeAttempted {
		state.Phase = phaseLifetime
		return ActionLifetime
	}
	if state.NeedLifetimeDayun && !state.LifetimeAccepted {
		state.Phase = phaseLifetime
		return ActionHardError
	}
	if state.NeedDynamic && !state.DynamicAttempted {
		state.Phase = phaseDynamic
		// 当前大运未绑定时，动态模型没有合法的 period ref 可引用；直接
		// 进入 facts-only，避免适配节点执行一次无效的模型调用后再降级。
		if !state.CurrentPeriodReady {
			return ActionRecoverFacts
		}
		return ActionDynamic
	}
	if state.NeedDynamic && !state.DynamicAccepted {
		state.Phase = phaseDynamic
		return ActionRecoverFacts
	}
	return ActionRender
}

func branch(_ context.Context, state *State) (string, error) {
	if state == nil {
		return "", fmt.Errorf("bazi graph state is nil")
	}
	switch state.NextAction {
	case ActionAnalysisPlan:
		return "analysis_plan", nil
	case ActionEvidence:
		return "evidence_action", nil
	case ActionStatic:
		return "static_judgment", nil
	case ActionLifetime:
		return "lifetime_dayun_judgment", nil
	case ActionDynamic:
		return "dynamic_judgment", nil
	case ActionRepair:
		return "repair", nil
	case ActionRecoverFacts:
		return "recover_facts", nil
	case ActionRender:
		return "render", nil
	case ActionHardError:
		return "hard_error", nil
	default:
		return "", fmt.Errorf("unknown bazi action %q", state.NextAction)
	}
}
