// Package runtime 包含 Manager 拥有的执行主链。
//
// 本文件负责外层 Eino Graph 的单轮状态和状态生成；
// 不负责请求 context 携带器、会话持久化或 Graph 节点业务决策。
package runtime

import (
	"context"

	"github.com/cloudwego/eino/schema"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
)

// orchestrationGraphState 是外层 Graph 拥有的单轮状态机。
// 它只保存可安全传递给 Graph 的状态值，runtime 服务引用仍由请求 context 携带。
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

// genOrchestrationState 用 Graph 开始时已确认的路由和计划初始化单轮状态。
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

// init 注册 Graph 本地状态类型，使 Eino 可以序列化和恢复它。
func init() {
	schema.Register[orchestrationGraphState]()
}
