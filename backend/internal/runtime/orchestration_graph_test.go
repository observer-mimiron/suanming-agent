// This test file belongs to the manager-owned runtime layer.
// It verifies outer runtime graph behavior and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/schemas"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
)

// TestOrchestrationGraphTopology 验证 Graph 拓扑结构正确编译。
// 不验证行为（行为由现有回归测试覆盖），只验证 Runnable 可编译。
func TestOrchestrationGraphTopology(t *testing.T) {
	r, err := buildOrchestrationGraph()
	if err != nil {
		t.Fatalf("buildOrchestrationGraph failed: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil Runnable")
	}
}

func TestExecute_ManagerOwnedPathDispatchesBoundedRunnerThroughGraph(t *testing.T) {
	graph, err := buildOrchestrationGraph()
	if err != nil {
		t.Fatalf("buildOrchestrationGraph: %v", err)
	}

	registry := specialists.NewRegistry()
	registry.Register(specialists.Config{Domain: "qimen", Name: "qimen_specialist"}, recordingRunner{
		result: specialists.Result{Domain: "qimen", Summary: "奇门结论"},
	})

	exec := &Executor{
		orchestrationGraph: graph,
		specialistRegistry: registry,
		manager:            &Manager{},
		reg:                tools.NewRegistry(),
	}
	exec.reg.Register(stubTool{
		name: "qimen_dunjia",
		result: map[string]any{
			"pan_schema":    "rotating_8",
			"symbol_system": "eight_gate_eight_god",
			"palaces":       []map[string]any{{"position": "坎"}},
		},
	})
	st := state.NewSession("sess-manager-owned")
	sink := &recordingSink{}

	turnType, text, err := exec.Execute(context.Background(), sink, st, policy.ApprovedRoute{
		PrimaryDomain:    "qimen",
		TaskIntent:       "fortune_followup",
		ConsultationKind: contracts.ConsultationKindEventQuestion,
		PolicyHints: schemas.PolicyHints{
			QimenMode:          "primary",
			ProfileRequirement: "none",
		},
		Slots: schemas.DecisionSlots{
			QuestionText: "今天运气怎么样",
		},
	}, "今天运气怎么样")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if turnType != "agent_reading" {
		t.Fatalf("turnType = %q, want agent_reading", turnType)
	}
	if text != "奇门结论" {
		t.Fatalf("text = %q, want 奇门结论", text)
	}
	if len(sink.events) == 0 {
		t.Fatal("expected buffered final text event")
	}
	textEvents := 0
	for _, event := range sink.events {
		if event.Type == "text" {
			textEvents++
		}
	}
	if textEvents != 1 {
		t.Fatalf("text event count = %d, want exactly one; events=%+v", textEvents, sink.events)
	}
	if sink.events[len(sink.events)-1].Type != "text" {
		t.Fatalf("last event type = %q, want text", sink.events[len(sink.events)-1].Type)
	}
}

type retryingPrimaryRunner struct {
	calls  *int
	result specialists.Result
}

func (r retryingPrimaryRunner) Run(context.Context, specialists.Request) (specialists.Result, error) {
	*r.calls++
	if *r.calls == 1 {
		return specialists.Result{}, errors.New("transient primary failure")
	}
	return r.result, nil
}

func TestExecute_RetriesOnlyFailedPrimaryThroughGraph(t *testing.T) {
	graph, err := buildOrchestrationGraph()
	if err != nil {
		t.Fatalf("buildOrchestrationGraph: %v", err)
	}

	var calls int
	registry := specialists.NewRegistry()
	registry.Register(specialists.Config{Domain: "qimen", Name: "qimen_specialist"}, retryingPrimaryRunner{
		calls:  &calls,
		result: specialists.Result{Domain: "qimen", Summary: "重试后的奇门结论"},
	})
	reg := tools.NewRegistry()
	reg.Register(stubTool{
		name: "qimen_dunjia",
		result: map[string]any{
			"pan_schema":    "rotating_8",
			"symbol_system": "eight_gate_eight_god",
			"palaces":       []map[string]any{{"position": "坎"}},
		},
	})
	exec := &Executor{
		orchestrationGraph: graph,
		specialistRegistry: registry,
		manager:            &Manager{},
		reg:                reg,
	}
	sink := &recordingSink{}
	turnType, text, err := exec.Execute(context.Background(), sink, state.NewSession("sess-primary-retry"), policy.ApprovedRoute{
		PrimaryDomain:    "qimen",
		TaskIntent:       "fortune_followup",
		ConsultationKind: contracts.ConsultationKindEventQuestion,
		PolicyHints: schemas.PolicyHints{
			QimenMode:          "primary",
			ProfileRequirement: "none",
		},
		Slots: schemas.DecisionSlots{QuestionText: "今天运气怎么样"},
	}, "今天运气怎么样")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if turnType != "agent_reading" || text != "重试后的奇门结论" {
		t.Fatalf("result = (%q, %q), want agent_reading and retry result", turnType, text)
	}
	if calls != 2 {
		t.Fatalf("primary runner calls = %d, want one initial call plus one retry", calls)
	}
	textEvents := 0
	for _, event := range sink.events {
		if event.Type == "text" {
			textEvents++
		}
	}
	if textEvents != 1 {
		t.Fatalf("text event count = %d, want exactly one", textEvents)
	}
}

func TestExecute_ForcedRouteReplacesPendingDomainSteps(t *testing.T) {
	graph, err := buildOrchestrationGraph()
	if err != nil {
		t.Fatalf("buildOrchestrationGraph: %v", err)
	}

	var qimenCalls []string
	registry := specialists.NewRegistry()
	registry.Register(specialists.Config{Domain: "qimen", Name: "qimen_specialist"}, recordingRunner{
		calls:  &qimenCalls,
		result: specialists.Result{Domain: "qimen", Summary: "奇门引导后的结论"},
	})
	reg := tools.NewRegistry()
	reg.Register(stubTool{
		name: "qimen_dunjia",
		result: map[string]any{
			"pan_schema":    "rotating_8",
			"symbol_system": "eight_gate_eight_god",
			"palaces":       []map[string]any{{"position": "坎"}},
		},
	})
	exec := &Executor{orchestrationGraph: graph, specialistRegistry: registry, manager: &Manager{}, reg: reg}
	st := state.NewSession("sess-forced-qimen")
	st.Guidance = &state.GuidanceState{DirectiveKind: "guided_fallback"}
	sink := &recordingSink{}

	turnType, text, err := exec.Execute(context.Background(), sink, st, policy.ApprovedRoute{
		PrimaryDomain:    "bazi",
		TaskIntent:       "collect_profile",
		ConsultationKind: contracts.ConsultationKindNatalChart,
		PolicyHints: schemas.PolicyHints{
			ProfileRequirement: "full",
		},
	}, "好，那你综合看看")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if turnType != "agent_reading" || text != "奇门引导后的结论" {
		t.Fatalf("result = (%q, %q), want guided qimen result", turnType, text)
	}
	if len(qimenCalls) != 1 || qimenCalls[0] != "qimen" {
		t.Fatalf("qimen runner calls = %v, want [qimen]", qimenCalls)
	}
	if st.Routing.PrimaryDomain != "qimen" {
		t.Fatalf("routing primary domain = %q, want qimen", st.Routing.PrimaryDomain)
	}
}

func TestChooseOrchestrationAction_SupportFailureFinishesDegraded(t *testing.T) {
	graphState := &orchestrationGraphState{
		PrefillCompleted: true,
		DomainOutcomes: []orchestrationDomainOutcome{
			{Domain: "bazi", Role: executionStepRolePrimary, Status: executionStepStatusReady, Result: specialists.Result{Summary: "主线结果"}},
			{Domain: "ziwei", Role: executionStepRoleSupport, Status: executionStepStatusDegraded},
		},
	}
	if action := chooseOrchestrationAction(graphState); action != orchestrationActionFinish {
		t.Fatalf("action = %q, want finish for degraded support", action)
	}
	if !graphState.Degraded {
		t.Fatal("support failure should mark the graph degraded")
	}
}

func TestChooseOrchestrationAction_DoesNotTreatSupportOnlyAsSafeResult(t *testing.T) {
	graphState := &orchestrationGraphState{
		PrefillCompleted: true,
		DomainOutcomes: []orchestrationDomainOutcome{
			{Domain: "ziwei", Role: executionStepRoleSupport, Status: executionStepStatusReady, Result: specialists.Result{Summary: "仅复核结果"}},
		},
	}
	if action := chooseOrchestrationAction(graphState); action != orchestrationActionHardError {
		t.Fatalf("action = %q, want hard_error without a primary result", action)
	}
}

func TestExecute_ReturnsAgentErrorWhenPrefillCannotSatisfyRequiredArtifact(t *testing.T) {
	graph, err := buildOrchestrationGraph()
	if err != nil {
		t.Fatalf("buildOrchestrationGraph: %v", err)
	}

	exec := &Executor{
		orchestrationGraph: graph,
		specialistRegistry: specialists.NewRegistry(),
		manager:            &Manager{},
		reg:                tools.NewRegistry(),
	}
	st := state.NewSession("sess-missing-qimen")

	turnType, _, err := exec.Execute(context.Background(), &recordingSink{}, st, policy.ApprovedRoute{
		PrimaryDomain:    "qimen",
		TaskIntent:       "fortune_followup",
		ConsultationKind: contracts.ConsultationKindEventQuestion,
		PolicyHints: schemas.PolicyHints{
			QimenMode:          "primary",
			ProfileRequirement: "none",
		},
	}, "今天运气怎么样")
	if err == nil {
		t.Fatal("Execute() error = nil, want artifact failure")
	}
	if turnType != "agent_error" {
		t.Fatalf("turnType = %q, want agent_error", turnType)
	}
	if !strings.Contains(err.Error(), "required artifact qimen_case_chart missing") {
		t.Fatalf("error = %q, want missing qimen artifact", err.Error())
	}
}
