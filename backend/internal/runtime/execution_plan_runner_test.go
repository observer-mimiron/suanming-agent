// This test file belongs to the manager-owned runtime layer.
// It verifies execution plan runner behavior and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

type recordingRunner struct {
	result specialists.Result
	calls  *[]string
}

func (r recordingRunner) Run(_ context.Context, req specialists.Request) (specialists.Result, error) {
	if r.calls != nil {
		*r.calls = append(*r.calls, req.Route.PrimaryDomain)
	}
	return r.result, nil
}

type blockingRunner struct {
	name    string
	started chan<- string
	release <-chan struct{}
}

func (r blockingRunner) Run(_ context.Context, _ specialists.Request) (specialists.Result, error) {
	if r.started != nil {
		r.started <- r.name
	}
	if r.release != nil {
		<-r.release
	}
	return specialists.Result{Domain: r.name, Summary: r.name + "-summary"}, nil
}

type sinkAwareRunner struct{}

func (sinkAwareRunner) Run(ctx context.Context, req specialists.Request) (specialists.Result, error) {
	if eventSinkFromContext(ctx) == nil {
		return specialists.Result{}, context.Canceled
	}
	return specialists.Result{Domain: req.Route.PrimaryDomain, Summary: req.Route.PrimaryDomain + "-summary"}, nil
}

type errorResultRunner struct {
	err error
}

func (r errorResultRunner) Run(context.Context, specialists.Request) (specialists.Result, error) {
	return specialists.Result{}, r.err
}

func TestExecutor_RunExecutionPlan_DispatchesBoundedSpecialistRunnersForMultiDomain(t *testing.T) {
	registry := specialists.NewRegistry()
	var calls []string
	registry.Register(specialists.Config{Domain: "bazi", Name: "bazi_specialist"}, recordingRunner{
		calls:  &calls,
		result: specialists.Result{Domain: "bazi", Summary: "八字结论"},
	})
	registry.Register(specialists.Config{Domain: "ziwei", Name: "ziwei_specialist"}, recordingRunner{
		calls:  &calls,
		result: specialists.Result{Domain: "ziwei", Summary: "紫微补充"},
	})

	executor := &Executor{specialistRegistry: registry}
	st := state.NewSession("s1")
	st.BaziResult = map[string]any{"ready": true}
	st.ZiWeiResult = map[string]any{"ready": true}
	plan := ExecutionPlan{
		Route: policy.ApprovedRoute{
			PrimaryDomain:    "bazi",
			SecondaryDomains: []string{"ziwei"},
		},
		DomainSteps: []contracts.DomainStep{
			{Domain: "bazi", Role: executionStepRolePrimary},
			{Domain: "ziwei", Role: executionStepRoleSupport},
		},
		Domains: []string{"bazi", "ziwei"},
	}

	result, err := executor.runExecutionPlan(context.Background(), nil, st, plan, "全面看看")
	if err != nil {
		t.Fatalf("runExecutionPlan() error = %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("runner call count = %d, want 2", len(calls))
	}
	callSet := map[string]bool{}
	for _, domain := range calls {
		callSet[domain] = true
	}
	if !callSet["bazi"] || !callSet["ziwei"] {
		t.Fatalf("runner calls = %v, want both bazi and ziwei", calls)
	}
	if result.Domain != "bazi+ziwei" {
		t.Fatalf("result.Domain = %q, want bazi+ziwei", result.Domain)
	}
	if result.Summary != "八字结论\n\n紫微补充" {
		t.Fatalf("result.Summary = %q, want aggregated domain summary", result.Summary)
	}
}

func TestExecutor_RunExecutionPlan_BlocksWhenRequiredArtifactMissing(t *testing.T) {
	registry := specialists.NewRegistry()
	registry.Register(specialists.Config{Domain: "qimen", Name: "qimen_specialist"}, recordingRunner{
		result: specialists.Result{Domain: "qimen", Summary: "qimen result"},
	})

	executor := &Executor{specialistRegistry: registry}
	st := state.NewSession("s-missing-artifact")
	plan := ExecutionPlan{
		Route: policy.ApprovedRoute{
			PrimaryDomain: "qimen",
			TaskIntent:    "fortune_followup",
		},
		DomainSteps: []contracts.DomainStep{
			{Domain: "qimen", Role: executionStepRolePrimary},
		},
		Domains:      []string{"qimen"},
		Requirements: selectArtifactRequirements(st, []string{"qimen"}),
	}

	_, err := executor.runExecutionPlan(context.Background(), nil, st, plan, "check current situation")
	if err == nil {
		t.Fatal("runExecutionPlan() error = nil, want missing artifact error")
	}
	if !strings.Contains(err.Error(), "required artifact qimen_case_chart") {
		t.Fatalf("runExecutionPlan() error = %q, want missing qimen artifact", err.Error())
	}
}

func TestExecutor_RunExecutionPlan_StartsBoundedSpecialistRunnersConcurrentlyAndKeepsPlanOrder(t *testing.T) {
	registry := specialists.NewRegistry()
	started := make(chan string, 2)
	release := make(chan struct{})
	registry.Register(specialists.Config{Domain: "bazi", Name: "bazi_specialist"}, blockingRunner{
		name:    "bazi",
		started: started,
		release: release,
	})
	registry.Register(specialists.Config{Domain: "ziwei", Name: "ziwei_specialist"}, blockingRunner{
		name:    "ziwei",
		started: started,
		release: release,
	})

	executor := &Executor{specialistRegistry: registry}
	st := state.NewSession("s-concurrent")
	st.BaziResult = map[string]any{"ready": true}
	st.ZiWeiResult = map[string]any{"ready": true}
	plan := ExecutionPlan{
		Route: policy.ApprovedRoute{
			PrimaryDomain:    "bazi",
			SecondaryDomains: []string{"ziwei"},
		},
		DomainSteps: []contracts.DomainStep{
			{Domain: "bazi", Role: executionStepRolePrimary},
			{Domain: "ziwei", Role: executionStepRoleSupport},
		},
		Domains: []string{"bazi", "ziwei"},
	}

	resultCh := make(chan specialists.Result, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := executor.runExecutionPlan(context.Background(), nil, st, plan, "鍏ㄩ潰鐪嬬湅")
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	var seen []string
	timeout := time.After(200 * time.Millisecond)
	for len(seen) < 2 {
		select {
		case domain := <-started:
			seen = append(seen, domain)
		case err := <-errCh:
			t.Fatalf("runExecutionPlan() error = %v", err)
		case <-timeout:
			t.Fatalf("expected both runners to start before any completed, got starts %v", seen)
		}
	}

	close(release)

	select {
	case err := <-errCh:
		t.Fatalf("runExecutionPlan() error = %v", err)
	case result := <-resultCh:
		if result.Domain != "bazi+ziwei" {
			t.Fatalf("result.Domain = %q, want bazi+ziwei", result.Domain)
		}
		if result.Summary != "bazi-summary\n\nziwei-summary" {
			t.Fatalf("result.Summary = %q, want plan-ordered summary", result.Summary)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("runExecutionPlan() did not finish after releasing runners")
	}
}

func TestExecutor_RunExecutionPlan_ProvidesSharedEventSinkWithoutLegacyDeps(t *testing.T) {
	registry := specialists.NewRegistry()
	registry.Register(specialists.Config{Domain: "bazi", Name: "bazi_specialist"}, sinkAwareRunner{})

	executor := &Executor{specialistRegistry: registry}
	st := state.NewSession("s-sink")
	st.BaziResult = map[string]any{"calendar_rule_version": currentBaziCalendarRule()}
	plan := ExecutionPlan{
		Route:       policy.ApprovedRoute{PrimaryDomain: "bazi"},
		DomainSteps: []contracts.DomainStep{{Domain: "bazi", Role: executionStepRolePrimary}},
		Domains:     []string{"bazi"},
	}

	result, err := executor.runExecutionPlan(context.Background(), &recordingSink{}, st, plan, "看看事业")
	if err != nil {
		t.Fatalf("runExecutionPlan() error = %v", err)
	}
	if result.Summary != "bazi-summary" {
		t.Fatalf("result.Summary = %q, want bazi-summary", result.Summary)
	}
}

func TestExecutor_RunExecutionPlan_UsesDomainStepRolesAndDegradesSupport(t *testing.T) {
	registry := specialists.NewRegistry()
	registry.Register(specialists.Config{Domain: "bazi", Name: "bazi_specialist"}, recordingRunner{
		result: specialists.Result{Domain: "bazi", Summary: "八字主线"},
	})
	registry.Register(specialists.Config{Domain: "ziwei", Name: "ziwei_specialist"}, errorResultRunner{
		err: errors.New("ziwei unavailable"),
	})

	executor := &Executor{specialistRegistry: registry}
	plan := ExecutionPlan{
		// Deliberately put support first; role, not slice order, defines the main line.
		DomainSteps: []contracts.DomainStep{
			{Domain: "ziwei", Role: executionStepRoleSupport},
			{Domain: "bazi", Role: executionStepRolePrimary},
		},
	}

	result, err := executor.runExecutionPlan(context.Background(), nil, state.NewSession("s-role"), plan, "本月运势")
	if err != nil {
		t.Fatalf("runExecutionPlan() error = %v, want support degradation only", err)
	}
	if result.Summary != "八字主线" {
		t.Fatalf("result.Summary = %q, want primary summary only", result.Summary)
	}
	outcomes, ok := result.DomainContextPatch["execution_outcomes"].([]executionStepOutcome)
	if !ok {
		t.Fatalf("execution_outcomes = %T, want typed outcomes", result.DomainContextPatch["execution_outcomes"])
	}
	for _, outcome := range outcomes {
		if outcome.Domain == "ziwei" && (outcome.Role != executionStepRoleSupport || outcome.Status != executionStepStatusDegraded) {
			t.Fatalf("ziwei outcome = %+v, want support/degraded", outcome)
		}
		if outcome.Domain == "bazi" && outcome.Role != executionStepRolePrimary {
			t.Fatalf("bazi outcome = %+v, want primary", outcome)
		}
	}
	if degraded, _ := result.DomainContextPatch["support_degraded"].(bool); !degraded {
		t.Fatal("support_degraded = false, want true")
	}
}

func TestExecutor_RunExecutionPlan_PrimaryFailureBlocksComposition(t *testing.T) {
	registry := specialists.NewRegistry()
	registry.Register(specialists.Config{Domain: "bazi", Name: "bazi_specialist"}, errorResultRunner{
		err: errors.New("bazi unavailable"),
	})
	registry.Register(specialists.Config{Domain: "ziwei", Name: "ziwei_specialist"}, recordingRunner{
		result: specialists.Result{Domain: "ziwei", Summary: "紫微复核"},
	})

	executor := &Executor{specialistRegistry: registry}
	plan := ExecutionPlan{DomainSteps: []contracts.DomainStep{
		{Domain: "bazi", Role: executionStepRolePrimary},
		{Domain: "ziwei", Role: executionStepRoleSupport},
	}}

	_, err := executor.runExecutionPlan(context.Background(), nil, state.NewSession("s-primary-failure"), plan, "本月运势")
	if err == nil || !strings.Contains(err.Error(), "primary specialist bazi failed") {
		t.Fatalf("runExecutionPlan() error = %v, want primary failure", err)
	}
}

func TestStoreFollowupArtifact_PersistsSingleDomainInterpretation(t *testing.T) {
	st := state.NewSession("s-followup-artifact")
	st.MergeProfile(map[string]any{
		"year": 1991.0, "month": 5.0, "day": 20.0, "hour": 8.0, "gender": "男",
	})
	st.StoreChart(state.AssetKindBaziChart, map[string]any{"calendar_rule_version": currentBaziCalendarRule()}, "test")
	result := specialists.Result{
		Domain:          "bazi",
		DirectAnswer:    "整体以稳步推进为主。",
		KeyPoints:       []string{"事业主线可走稳", "不宜激进加码"},
		EvidenceSummary: "原局主轴可承接，当前岁运不宜躁进。",
	}
	route := policy.ApprovedRoute{PrimaryDomain: "bazi"}

	storeFollowupArtifact(st, route, result, "整体以稳步推进为主。", "事业怎么走", "agent_reading")

	artifact, ok := loadFollowupArtifact(st, "bazi")
	if !ok {
		t.Fatal("loadFollowupArtifact() = false, want true")
	}
	if artifact.Domain != "bazi" {
		t.Fatalf("artifact.Domain = %q, want bazi", artifact.Domain)
	}
	if artifact.DirectAnswer != "整体以稳步推进为主。" {
		t.Fatalf("artifact.DirectAnswer = %q, want direct answer", artifact.DirectAnswer)
	}
	if len(artifact.KeyPoints) != 2 {
		t.Fatalf("len(artifact.KeyPoints) = %d, want 2", len(artifact.KeyPoints))
	}
}
