package runtime

import (
	"context"
	"strings"
	"testing"
	"time"

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
		Domains:           []string{"qimen"},
		RequiredArtifacts: []string{artifactQimenChart},
	}

	_, err := executor.runExecutionPlan(context.Background(), nil, st, plan, "check current situation")
	if err == nil {
		t.Fatal("runExecutionPlan() error = nil, want missing artifact error")
	}
	if !strings.Contains(err.Error(), "required artifact qimen_chart") {
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
	st.BaziResult = map[string]any{"calendar_rule_version": "zi_zheng_v1"}
	plan := ExecutionPlan{
		Route: policy.ApprovedRoute{PrimaryDomain: "bazi"},
		Domains: []string{"bazi"},
	}

	result, err := executor.runExecutionPlan(context.Background(), &recordingSink{}, st, plan, "看看事业")
	if err != nil {
		t.Fatalf("runExecutionPlan() error = %v", err)
	}
	if result.Summary != "bazi-summary" {
		t.Fatalf("result.Summary = %q, want bazi-summary", result.Summary)
	}
}

func TestStoreFollowupArtifact_PersistsSingleDomainInterpretation(t *testing.T) {
	st := state.NewSession("s-followup-artifact")
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
