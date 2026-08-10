// This test file belongs to the manager-owned runtime layer.
// It verifies the single BaZi deterministic graph and its contract metadata.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"testing"

	bazigraph "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/graph"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/structured"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

func TestBaziInternalGraphCompilesAsSingleProductionGraph(t *testing.T) {
	if _, err := bazigraph.Compile(context.Background(), (&Executor{}).baziGraphDeps()); err != nil {
		t.Fatalf("compile deterministic BaZi graph: %v", err)
	}
}

func TestBaziDynamicJudgmentBeforeFirstDayunKeepsFactsOnly(t *testing.T) {
	state := baziCharterState{
		AnalysisPlan:    baziAnalysisPlan{NeedDynamic: true},
		StaticSynthesis: baziStaticSynthesis{MainAxis: "静态主轴已通过。"},
		Input: baziCharterInput{
			Dayun: map[string]any{"dayun_analyzed": []map[string]any{{
				"ganZhi": "", "startAt": "2025-11-11 00:15:00", "endAtExclusive": "2027-01-11 00:15:00",
			}}},
			Liunian: map[string]any{"liunian_target_at": "2025-11-10 23:53:00", "liunian_ganzhi": "丙午"},
		},
	}
	in := &baziInternalGraphState{ChartState: state, Canonical: baziCanonicalSynthesis{Source: "model"}}
	if _, err := (&Executor{}).baziDynamicJudgmentV2Node(context.Background(), in); err != nil {
		t.Fatalf("pre-first-dayun dynamic judgment returned error: %v", err)
	}
	if in.Canonical.Source != baziSynthesisSourceFactsOnlyDegraded {
		t.Fatalf("dynamic source = %q; want %q", in.Canonical.Source, baziSynthesisSourceFactsOnlyDegraded)
	}
	if !in.DynamicAttempted {
		t.Fatal("facts-only dynamic node must mark the dynamic action as attempted")
	}
}

func TestBaziInternalGraphInvokeRoutesMissingChartToTerminalFailure(t *testing.T) {
	executor := &Executor{}
	result, err := executor.baziGraphRuntimeResult(context.Background(), nil, state.NewSession("bazi-missing-chart"), "看看事业")
	if err != nil {
		t.Fatalf("bazi graph invoke returned Go error: %v", err)
	}
	if result.TerminationReason != "hard_error" {
		t.Fatalf("termination reason = %q, want hard_error", result.TerminationReason)
	}
	if result.Failure.FailureCode != "BAZI_CHART_MISSING" {
		t.Fatalf("failure code = %q, want BAZI_CHART_MISSING", result.Failure.FailureCode)
	}
}

func TestBaziV2RepairFeedbackPreservesReferenceCatalog(t *testing.T) {
	err := baziViolationError(
		baziViolationUndeclaredFactClaim,
		"assertions.fact_refs",
		"static.main_axis",
		"fact_ref is not declared in this runtime catalog",
		[]string{"core_chart.month_command"},
		[]string{"chart.month_branch", "fact_capsule.month_command"},
	)
	failure, ok := repairFailureFromBaziContract("static_synthesis", err)
	if !ok {
		t.Fatal("expected contract failure to be repairable")
	}
	feedback := buildBaziCanonicalRepairFeedback(failure, 1)
	refs, ok := feedback["reference_feedback"].(map[string]any)
	if !ok {
		t.Fatalf("reference feedback = %#v; want catalog feedback", feedback)
	}
	invalid, ok := refs["invalid_refs"].([]string)
	if !ok || len(invalid) != 1 || invalid[0] != "core_chart.month_command" {
		t.Fatalf("invalid refs = %#v", refs["invalid_refs"])
	}
	allowed, ok := refs["allowed_refs"].([]string)
	if !ok || len(allowed) != 2 || allowed[1] != "fact_capsule.month_command" {
		t.Fatalf("allowed refs = %#v", refs["allowed_refs"])
	}
}

func TestBaziRecordInternalFailureReplacesStaleFindingWithParseError(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()
	attrs := tracing.TraceFromContext(ctx).Attributes
	attrs["bazi.contract.finding_field"] = "assertions.fact_refs"
	attrs["bazi.contract.failure_class"] = "schema_error"

	in := &baziInternalGraphState{}
	baziRecordInternalFailure(ctx, in, "static_synthesis", &structured.Error{Schema: structuredSchemaBaziStaticSynthesis, Detail: "invalid JSON"}, "static_judgment_failed")

	attrs = tracing.TraceFromContext(ctx).Attributes
	if got := attrs["bazi.contract.finding_field"]; got != structuredSchemaBaziStaticSynthesis {
		t.Fatalf("finding field = %v; want %s", got, structuredSchemaBaziStaticSynthesis)
	}
	if got := attrs["bazi.contract.failure_class"]; got != string(RepairSchemaError) {
		t.Fatalf("failure class = %v; want %s", got, RepairSchemaError)
	}
}
