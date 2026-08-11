// Package runtime 包含 Manager 拥有的执行主链。
//
// 本文件保护 Graph 控制状态与 runtime 八字领域 payload 之间的适配边界。
package runtime

import (
	"context"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/repair"
	bazigraph "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/graph"
)

func TestRunBaziGraphNodeKeepsGraphPhaseAuthoritative(t *testing.T) {
	graphState := &bazigraph.State{
		Phase:           "static",
		LoopStep:        3,
		MaxRunSteps:     bazigraph.MaxRunSteps,
		StaticAttempted: true,
		StaticAccepted:  true,
		Payload:         &baziInternalGraphState{},
	}

	var observedPhase string
	err := (&Executor{}).runBaziGraphNode(context.Background(), graphState, func(_ context.Context, in *baziInternalGraphState) (*baziInternalGraphState, error) {
		observedPhase = in.Phase
		// 领域节点可以在分类失败时使用本地 phase，但不能把它变成 Graph 的下一条边。
		in.Phase = baziPhaseRepair
		in.ChartState.Input.BaziResult = map[string]any{"ready": true}
		in.ChartState.AnalysisPlan = baziAnalysisPlan{Mode: "full", NeedDynamic: true}
		in.FactCapsule.CurrentPeriodRef = "dayun:1"
		in.EvidenceAttempts = 2
		in.EvidenceValidated = true
		in.Output = "rendered"
		return in, nil
	})
	if err != nil {
		t.Fatalf("runBaziGraphNode() error = %v", err)
	}
	if observedPhase != "static" {
		t.Fatalf("node phase = %q, want graph phase static", observedPhase)
	}
	if graphState.Phase != "static" {
		t.Fatalf("graph phase = %q, want static", graphState.Phase)
	}
	if !graphState.ChartReady || !graphState.AnalysisPlanned || !graphState.NeedDynamic || !graphState.CurrentPeriodReady {
		t.Fatalf("projected derived control state = %+v", graphState)
	}
	if graphState.EvidenceNeedsAction {
		t.Fatal("evidence should not request another action after two attempts")
	}
	if graphState.Output != "rendered" {
		t.Fatalf("output = %q, want rendered", graphState.Output)
	}
}

func TestApplyBaziGraphControlCopiesStateSafeFailureProjection(t *testing.T) {
	graphState := &bazigraph.State{
		Phase:             "dynamic",
		LoopStep:          7,
		MaxRunSteps:       bazigraph.MaxRunSteps,
		RecoveryState:     "dynamic_facts_only_degraded",
		RecoveryPolicy:    "dynamic_facts_only",
		Failure:           bazigraph.Failure{Class: "schema_error", Stage: "dynamic_synthesis", Code: "DYNAMIC_SCHEMA"},
		RepairAction:      "fallback",
		TerminationReason: "graph_step_limit_degraded",
		Payload:           &baziInternalGraphState{},
	}
	in := &baziInternalGraphState{}

	applyBaziGraphControl(in, graphState)

	if in.Phase != "dynamic" || in.LoopStep != 7 || in.MaxRunSteps != bazigraph.MaxRunSteps {
		t.Fatalf("copied graph counters = phase=%q step=%d max=%d", in.Phase, in.LoopStep, in.MaxRunSteps)
	}
	if in.RecoveryState != "dynamic_facts_only_degraded" || in.RecoveryPolicy != "dynamic_facts_only" {
		t.Fatalf("copied recovery = state=%q policy=%q", in.RecoveryState, in.RecoveryPolicy)
	}
	if in.Failure.FailureCode != "DYNAMIC_SCHEMA" || in.FailureStage != "dynamic_synthesis" {
		t.Fatalf("copied failure = %+v stage=%q", in.Failure, in.FailureStage)
	}
	if in.RepairAction != repair.ActionFallback {
		t.Fatalf("repair action = %q, want %q", in.RepairAction, repair.ActionFallback)
	}
	if in.TerminationReason != "graph_step_limit_degraded" {
		t.Fatalf("termination reason = %q, want graph_step_limit_degraded", in.TerminationReason)
	}
}

func TestBaziGraphResultCarriesTerminalPayloadAudit(t *testing.T) {
	terminal := &baziInternalGraphState{}
	terminal.ChartState.StaticSynthesis.ContractAudit = baziContractAudit{Compliant: true}
	result := bazigraph.Result{Payload: terminal}
	got, ok := result.Payload.(*baziInternalGraphState)
	if !ok || got != terminal || !got.ChartState.StaticSynthesis.ContractAudit.Compliant {
		t.Fatalf("terminal payload audit = %#v, want clean terminal audit", result.Payload)
	}
}
