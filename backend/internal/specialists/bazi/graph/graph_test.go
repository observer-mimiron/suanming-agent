package graph

import "testing"

func TestChooseAction_DynamicWithoutCurrentPeriodUsesFactsOnly(t *testing.T) {
	state := &State{
		ChartReady:      true,
		AnalysisPlanned: true,
		StaticAttempted: true,
		StaticAccepted:  true,
		NeedDynamic:     true,
	}

	if got := chooseAction(state); got != ActionRecoverFacts {
		t.Fatalf("chooseAction() = %q, want %q", got, ActionRecoverFacts)
	}
	if state.Phase != phaseDynamic {
		t.Fatalf("phase = %q, want %q", state.Phase, phaseDynamic)
	}
}

func TestChooseAction_DynamicWithCurrentPeriodRunsJudgment(t *testing.T) {
	state := &State{
		ChartReady:         true,
		AnalysisPlanned:    true,
		StaticAttempted:    true,
		StaticAccepted:     true,
		NeedDynamic:        true,
		CurrentPeriodReady: true,
	}

	if got := chooseAction(state); got != ActionDynamic {
		t.Fatalf("chooseAction() = %q, want %q", got, ActionDynamic)
	}
}

func TestChooseAction_LifetimeRunsBeforeCurrentDynamic(t *testing.T) {
	state := &State{
		ChartReady: true, AnalysisPlanned: true, StaticAttempted: true, StaticAccepted: true,
		NeedLifetimeDayun: true, NeedDynamic: true, CurrentPeriodReady: true,
	}
	if got := chooseAction(state); got != ActionLifetime {
		t.Fatalf("chooseAction() = %q, want %q", got, ActionLifetime)
	}
	if state.Phase != phaseLifetime {
		t.Fatalf("phase = %q, want %q", state.Phase, phaseLifetime)
	}
}
