// This file belongs to the manager-owned runtime layer.
// It owns ExecutionPlan dispatch behavior for this package.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

type eventSinkCtxKey struct{}

const (
	executionStepRolePrimary    = "primary"
	executionStepRoleSupport    = "support"
	executionStepStatusReady    = "ready"
	executionStepStatusDegraded = "degraded"
	executionStepStatusFailed   = "failed"
)

// executionStep 是一次 specialist 调度的内部角色投影。
// 它只存在于 runtime，避免把主次、降级和错误状态塞进公共 specialist 结果合同。
type executionStep struct {
	Domain string
	Role   string
}

// executionStepOutcome 保存一个领域 worker 的结构化执行结果。
// Manager 后续合成应消费该结构，而不是从 Summary 的拼接顺序猜主次。
type executionStepOutcome struct {
	Domain string
	Role   string
	Status string
	Result specialists.Result
	Err    error
}

func withEventSink(ctx context.Context, sink EventSink) context.Context {
	return context.WithValue(ctx, eventSinkCtxKey{}, sink)
}

func eventSinkFromContext(ctx context.Context) EventSink {
	sink, _ := ctx.Value(eventSinkCtxKey{}).(EventSink)
	return sink
}

// executionStepsForPlan 返回本轮唯一的角色化调度清单。
// DomainSteps 是唯一主次来源；缺少它的旧计划必须先由调用方迁移，不能按 Domains 顺序猜主次。
func executionStepsForPlan(plan ExecutionPlan) ([]executionStep, error) {
	if len(plan.DomainSteps) == 0 {
		return nil, fmt.Errorf("execution plan requires DomainSteps")
	}
	steps := make([]executionStep, 0, len(plan.DomainSteps))
	for _, domainStep := range plan.DomainSteps {
		steps = append(steps, executionStep{
			Domain: strings.TrimSpace(domainStep.Domain),
			Role:   strings.TrimSpace(domainStep.Role),
		})
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("execution plan requires at least one domain step")
	}

	seen := make(map[string]struct{}, len(steps))
	primaryCount := 0
	for _, step := range steps {
		if step.Domain == "" {
			return nil, fmt.Errorf("execution plan contains an empty domain step")
		}
		if _, ok := seen[step.Domain]; ok {
			return nil, fmt.Errorf("execution plan contains duplicate domain step %q", step.Domain)
		}
		seen[step.Domain] = struct{}{}
		switch step.Role {
		case executionStepRolePrimary:
			primaryCount++
		case executionStepRoleSupport:
		default:
			return nil, fmt.Errorf("execution plan domain %q has invalid role %q", step.Domain, step.Role)
		}
	}
	if primaryCount != 1 {
		return nil, fmt.Errorf("execution plan requires exactly one primary domain, got %d", primaryCount)
	}
	return steps, nil
}

// primaryDomainForSteps derives the overall primary domain from the role contract.
// Worker-specific route projections are kept separate from final synthesis outcomes.
func primaryDomainForSteps(steps []executionStep) string {
	for _, step := range steps {
		if step.Role == executionStepRolePrimary {
			return step.Domain
		}
	}
	return ""
}

func (e *Executor) runExecutionPlan(ctx context.Context, sink EventSink, st *state.SessionState, plan ExecutionPlan, message string) (specialists.Result, error) {
	steps, err := executionStepsForPlan(plan)
	if err != nil {
		return specialists.Result{}, err
	}
	outcomes, err := e.dispatchExecutionSteps(ctx, sink, st, plan, message, steps)
	if err != nil {
		return specialists.Result{}, err
	}
	primaryDomain := primaryDomainForSteps(steps)
	if primary := primaryFailureOutcome(outcomes); primary != nil {
		return specialists.Result{}, fmt.Errorf("primary specialist %s failed: %w", primaryDomain, primary.Err)
	}
	return aggregateExecutionOutcomes(outcomes, primaryDomain), nil
}

// dispatchExecutionSteps executes one pending batch and returns every domain
// outcome. It deliberately does not decide retry or termination; that belongs
// to the outer graph state machine.
func (e *Executor) dispatchExecutionSteps(ctx context.Context, sink EventSink, st *state.SessionState, plan ExecutionPlan, message string, steps []executionStep) ([]executionStepOutcome, error) {
	if e == nil {
		return nil, fmt.Errorf("execution plan requires executor")
	}
	if st == nil {
		return nil, fmt.Errorf("execution plan requires session state")
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("execution plan requires at least one pending domain step")
	}
	if e.specialistRegistry == nil && !(shouldUseBaziCharterGraph(plan) && len(plan.Domains) == 1) {
		return nil, fmt.Errorf("execution plan requires specialist registry")
	}

	ctx = withEventSink(ctx, sink)
	if err := validatePlanArtifacts(st, plan); err != nil {
		return nil, err
	}

	outcomes := make([]executionStepOutcome, len(steps))
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	for i, step := range steps {
		wg.Add(1)
		go func(idx int, step executionStep) {
			defer wg.Done()
			outcome := executionStepOutcome{
				Domain: step.Domain,
				Role:   step.Role,
				Status: executionStepStatusReady,
			}
			defer func() { outcomes[idx] = outcome }()

			var result specialists.Result
			var runErr error
			if e.shouldUseBaziAuthorityGraph(plan) && step.Domain == "bazi" {
				result.Summary, runErr = e.runBaziAuthorityFirstGraph(runCtx, sink, st, message)
				result.Domain = "bazi"
			} else {
				runner, ok := e.specialistRegistry.RunnerFor(step.Domain)
				if !ok {
					outcome.Err = fmt.Errorf("no specialist runner registered for %s", step.Domain)
					outcome.Status = executionStepStatusFailed
					if step.Role == executionStepRoleSupport {
						outcome.Status = executionStepStatusDegraded
					}
					return
				}

				route := plan.Route
				// 每个 worker 仍接收自己的领域视角；最终主次只由 outcome.Role 表达，
				// 避免把 specialist 请求里的 route 投影误当成合成合同。
				route.PrimaryDomain = step.Domain
				route.SecondaryDomains = secondaryDomainsForExecutionSteps(steps, step.Domain)
				result, runErr = runner.Run(runCtx, specialists.Request{
					SessionID:      st.SessionID,
					UserMessage:    message,
					Route:          route,
					ManagerContext: st.ManagerContext,
					DomainContext:  *domainContextFor(st, step.Domain),
					Session:        specialistSessionView(st, plan, step.Domain),
				})
			}
			if runErr != nil {
				outcome.Err = runErr
				if step.Role == executionStepRoleSupport {
					outcome.Status = executionStepStatusDegraded
				} else {
					outcome.Status = executionStepStatusFailed
				}
				return
			}
			outcome.Result = result
		}(i, step)
	}

	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return outcomes, nil
}

// aggregateExecutionOutcomes 保留旧的文本结果投影，同时把角色和降级状态放入结构化 runtime 元数据。
// 这样 Manager 可以按 Role 合成，support 的文本不会因拼接顺序而变成主线。
func aggregateExecutionOutcomes(outcomes []executionStepOutcome, primaryDomain string) specialists.Result {
	var primary executionStepOutcome
	allReady := true
	summaries := make([]string, 0, len(outcomes))
	domainNames := make([]string, 0, len(outcomes))
	ordered := make([]executionStepOutcome, 0, len(outcomes))
	for _, outcome := range outcomes {
		if outcome.Role == executionStepRolePrimary {
			primary = outcome
			ordered = append(ordered, outcome)
		}
	}
	for _, outcome := range outcomes {
		if outcome.Role == executionStepRoleSupport {
			ordered = append(ordered, outcome)
		}
	}
	for _, outcome := range ordered {
		if outcome.Status != executionStepStatusReady {
			allReady = false
			continue
		}
		if summary := strings.TrimSpace(outcome.Result.NormalizedSummary()); summary != "" {
			summaries = append(summaries, summary)
		}
		name := strings.TrimSpace(outcome.Result.Domain)
		if name == "" {
			name = outcome.Domain
		}
		domainNames = append(domainNames, name)
	}

	aggregated := primary.Result
	if len(summaries) > 0 {
		aggregated.Summary = strings.Join(summaries, "\n\n")
	}
	if allReady && len(domainNames) > 1 {
		aggregated.Domain = strings.Join(domainNames, "+")
	} else if strings.TrimSpace(aggregated.Domain) == "" {
		aggregated.Domain = primaryDomain
	}
	aggregated.DomainContextPatch = mergeExecutionOutcomeMetadata(primary.Result.DomainContextPatch, outcomes, primaryDomain)
	return aggregated
}

// mergeExecutionOutcomeMetadata attaches the typed role outcomes to the existing specialist patch.
// The patch is internal runtime data; user-facing text remains the legacy summary projection.
func mergeExecutionOutcomeMetadata(existing map[string]any, outcomes []executionStepOutcome, primaryDomain string) map[string]any {
	patch := make(map[string]any, len(existing)+3)
	for key, value := range existing {
		patch[key] = value
	}
	supportDegraded := false
	for _, outcome := range outcomes {
		if outcome.Role == executionStepRoleSupport && outcome.Status == executionStepStatusDegraded {
			supportDegraded = true
			break
		}
	}
	patch["execution_outcomes"] = append([]executionStepOutcome(nil), outcomes...)
	patch["primary_domain"] = primaryDomain
	patch["support_degraded"] = supportDegraded
	return patch
}

// secondaryDomainsForExecutionSteps 返回当前 worker 以外的领域投影。
// 它只服务 specialist 请求兼容性，不参与 primary/support 判定。
func secondaryDomainsForExecutionSteps(steps []executionStep, primary string) []string {
	if len(steps) <= 1 {
		return nil
	}
	secondary := make([]string, 0, len(steps)-1)
	for _, step := range steps {
		if step.Domain == "" || step.Domain == primary {
			continue
		}
		secondary = append(secondary, step.Domain)
	}
	return secondary
}

func secondaryDomainsForPlan(domains []string, primary string) []string {
	if len(domains) <= 1 {
		return nil
	}
	secondary := make([]string, 0, len(domains)-1)
	for _, domain := range domains {
		if domain == "" || domain == primary {
			continue
		}
		secondary = append(secondary, domain)
	}
	return secondary
}
