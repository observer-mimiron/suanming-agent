package graph

import (
	"context"
	"strings"
	"testing"
)

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

func TestCompileRejectsEveryMissingDependency(t *testing.T) {
	missing := []struct {
		name  string
		clear func(*Deps)
	}{
		{"bootstrap", func(deps *Deps) { deps.Bootstrap = nil }},
		{"analysis_plan", func(deps *Deps) { deps.AnalysisPlan = nil }},
		{"evidence", func(deps *Deps) { deps.Evidence = nil }},
		{"validate_evidence", func(deps *Deps) { deps.ValidateEvidence = nil }},
		{"static", func(deps *Deps) { deps.Static = nil }},
		{"lifetime", func(deps *Deps) { deps.Lifetime = nil }},
		{"dynamic", func(deps *Deps) { deps.Dynamic = nil }},
		{"contract_check", func(deps *Deps) { deps.ContractCheck = nil }},
		{"repair", func(deps *Deps) { deps.Repair = nil }},
		{"recover_facts", func(deps *Deps) { deps.RecoverFacts = nil }},
		{"render", func(deps *Deps) { deps.Render = nil }},
		{"hard_error", func(deps *Deps) { deps.HardError = nil }},
	}

	for _, test := range missing {
		t.Run(test.name, func(t *testing.T) {
			deps := noOpDeps()
			test.clear(&deps)
			if _, err := Compile(context.Background(), deps); err == nil || !strings.Contains(err.Error(), test.name) {
				t.Fatalf("Compile() error = %v, want missing dependency %q", err, test.name)
			}
		})
	}
}

func TestRunCompletesStaticPipeline(t *testing.T) {
	var calls []string
	deps := noOpDeps()
	deps.Bootstrap = recordCallback(&calls, "bootstrap", nil)
	deps.AnalysisPlan = recordCallback(&calls, "analysis_plan", func(state *State) {
		state.AnalysisPlanned = true
	})
	deps.Static = recordCallback(&calls, "static_judgment", func(state *State) {
		state.StaticAttempted = true
		state.StaticAccepted = true
	})
	deps.ContractCheck = recordCallback(&calls, "contract_check", nil)
	deps.Render = recordCallback(&calls, "render", func(state *State) {
		state.Output = "rendered"
	})

	result, err := Run(context.Background(), deps, &State{ChartReady: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Text != "rendered" || result.TerminationReason != "completed" {
		t.Fatalf("result = %+v, want rendered/completed", result)
	}
	want := []string{"bootstrap", "analysis_plan", "static_judgment", "contract_check", "render"}
	if strings.Join(calls, ",") != strings.Join(want, ",") {
		t.Fatalf("callback order = %v, want %v", calls, want)
	}
}

func TestRunReturnsTerminalPayload(t *testing.T) {
	payload := &struct{ Marker string }{Marker: "terminal"}
	deps := noOpDeps()
	deps.Bootstrap = recordCallback(&[]string{}, "bootstrap", func(state *State) { state.Payload = payload })
	deps.Render = recordCallback(&[]string{}, "render", nil)
	result, err := Run(context.Background(), deps, &State{ChartReady: true, AnalysisPlanned: true, StaticAttempted: true, StaticAccepted: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Payload != payload {
		t.Fatalf("terminal payload = %#v, want %#v", result.Payload, payload)
	}
}

func TestRunMissingChartReachesHardErrorTerminal(t *testing.T) {
	var calls []string
	deps := noOpDeps()
	deps.Bootstrap = recordCallback(&calls, "bootstrap", nil)
	deps.HardError = recordCallback(&calls, "hard_error", nil)

	result, err := Run(context.Background(), deps, &State{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.TerminationReason != "hard_error" {
		t.Fatalf("termination reason = %q, want hard_error", result.TerminationReason)
	}
	if result.Failure.Code != "BAZI_CHART_MISSING" {
		t.Fatalf("failure code = %q, want BAZI_CHART_MISSING", result.Failure.Code)
	}
	if strings.Join(calls, ",") != "bootstrap,hard_error" {
		t.Fatalf("callback order = %v, want bootstrap,hard_error", calls)
	}
}

func TestRunStepLimitDegradesBeforeRender(t *testing.T) {
	var calls []string
	deps := noOpDeps()
	deps.Bootstrap = recordCallback(&calls, "bootstrap", nil)
	deps.RecoverFacts = recordCallback(&calls, "recover_facts", nil)
	deps.Render = recordCallback(&calls, "render", func(state *State) {
		state.Output = "facts-only"
	})

	result, err := Run(context.Background(), deps, &State{
		ChartReady:         true,
		AnalysisPlanned:    true,
		StaticAttempted:    true,
		StaticAccepted:     true,
		NeedDynamic:        true,
		CurrentPeriodReady: true,
		MaxRunSteps:        1,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Text != "facts-only" || result.TerminationReason != "graph_step_limit_degraded" {
		t.Fatalf("result = %+v, want facts-only/graph_step_limit_degraded", result)
	}
	if strings.Join(calls, ",") != "bootstrap,recover_facts,render" {
		t.Fatalf("callback order = %v, want bootstrap,recover_facts,render", calls)
	}
}

func TestRunLifetimeCompletesBeforeDynamic(t *testing.T) {
	var calls []string
	deps := noOpDeps()
	deps.Bootstrap = recordCallback(&calls, "bootstrap", nil)
	deps.Lifetime = recordCallback(&calls, "lifetime_dayun_judgment", func(state *State) {
		state.LifetimeAttempted = true
		state.LifetimeAccepted = true
	})
	deps.Dynamic = recordCallback(&calls, "dynamic_judgment", func(state *State) {
		state.DynamicAttempted = true
		state.DynamicAccepted = true
	})
	deps.ContractCheck = recordCallback(&calls, "contract_check", nil)
	deps.Render = recordCallback(&calls, "render", func(state *State) {
		state.Output = "complete"
	})

	result, err := Run(context.Background(), deps, &State{
		ChartReady:         true,
		AnalysisPlanned:    true,
		StaticAttempted:    true,
		StaticAccepted:     true,
		NeedLifetimeDayun:  true,
		NeedDynamic:        true,
		CurrentPeriodReady: true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Text != "complete" || result.TerminationReason != "completed" {
		t.Fatalf("result = %+v, want complete/completed", result)
	}
	want := []string{"bootstrap", "lifetime_dayun_judgment", "contract_check", "dynamic_judgment", "contract_check", "render"}
	if strings.Join(calls, ",") != strings.Join(want, ",") {
		t.Fatalf("callback order = %v, want %v", calls, want)
	}
}

func noOpDeps() Deps {
	noop := func(context.Context, *State) error { return nil }
	return Deps{
		Bootstrap:        noop,
		AnalysisPlan:     noop,
		Evidence:         noop,
		ValidateEvidence: noop,
		Static:           noop,
		Lifetime:         noop,
		Dynamic:          noop,
		ContractCheck:    noop,
		Repair:           noop,
		RecoverFacts:     noop,
		Render:           noop,
		HardError:        noop,
	}
}

func recordCallback(calls *[]string, name string, mutate func(*State)) func(context.Context, *State) error {
	return func(_ context.Context, state *State) error {
		*calls = append(*calls, name)
		if mutate != nil {
			mutate(state)
		}
		return nil
	}
}
