// This test file belongs to the route approval layer.
// It verifies cheap follow-up gate behavior and protects the related contract from regressions.
// It approves routes; execution contracts are built later by Manager.
package supervisor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/observability"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

func seedBaziAsset(st *state.SessionState) {
	st.Profile = map[string]any{
		"year": 1990.0, "month": 5.0, "day": 5.0, "hour": 14.0,
		"gender": "男", "birthplace": "北京",
	}
	st.StoreChart(state.AssetKindBaziChart, map[string]any{"dayGan": "甲"}, "test")
}

func TestTryCheapFollowupRoute_ReusesExistingExecutionContract(t *testing.T) {
	client := &Client{}
	st := state.NewSession("sess-1")
	seedBaziAsset(st)
	st.Execution = contracts.ExecutionSnapshot{
		ConsultationKind:   contracts.ConsultationKindPeriodFortune,
		PrimaryDomain:      "bazi",
		TaskIntent:         "fortune_followup",
		ConversationIntent: "consult",
		QimenMode:          "none",
		TargetSubject:      "自己",
		Gate: contracts.GateContract{
			Admitted:            true,
			ReuseSessionProfile: true,
			ReuseCachedResult:   true,
			ExecutionMode:       "execute",
			FollowupPolicy:      "allow",
			ProfileRequirement:  "full",
		},
	}

	route, ok := client.tryCheapFollowupRoute("那事业呢", st)
	if !ok {
		t.Fatal("ok = false, want cheap gate hit")
	}
	if route.TaskIntent != "fortune_followup" {
		t.Fatalf("TaskIntent = %q, want fortune_followup", route.TaskIntent)
	}
	if route.PrimaryDomain != "bazi" {
		t.Fatalf("PrimaryDomain = %q, want bazi", route.PrimaryDomain)
	}
	if route.Gate.Reason != "cheap_followup_reuse" {
		t.Fatalf("Gate.Reason = %q, want cheap_followup_reuse", route.Gate.Reason)
	}
	if route.Gate.ExecutionMode != "reuse_followup" {
		t.Fatalf("Gate.ExecutionMode = %q, want reuse_followup", route.Gate.ExecutionMode)
	}
}

func TestTryCheapFollowupRoute_RejectsTimingQuestion(t *testing.T) {
	client := &Client{}
	st := state.NewSession("sess-2")
	seedBaziAsset(st)
	st.Execution = contracts.ExecutionSnapshot{
		ConsultationKind: contracts.ConsultationKindPeriodFortune,
		PrimaryDomain:    "bazi",
		TaskIntent:       "fortune_followup",
		Gate: contracts.GateContract{
			Admitted:       true,
			ExecutionMode:  "execute",
			FollowupPolicy: "allow",
		},
	}

	if _, ok := client.tryCheapFollowupRoute("最近时机怎么样", st); ok {
		t.Fatal("cheap gate should reject timing question")
	}
}

func TestTryCheapFollowupRoute_RejectsConcreteEventQuestion(t *testing.T) {
	client := &Client{}
	st := state.NewSession("sess-event")
	seedBaziAsset(st)
	st.Execution = contracts.ExecutionSnapshot{
		ConsultationKind: contracts.ConsultationKindPeriodFortune,
		PrimaryDomain:    "bazi",
		TaskIntent:       "fortune_followup",
		Gate: contracts.GateContract{
			Admitted:       true,
			ExecutionMode:  "execute",
			FollowupPolicy: "allow",
		},
	}

	if _, ok := client.tryCheapFollowupRoute("这个面试能不能成", st); ok {
		t.Fatal("cheap gate should reject concrete event question")
	}
}

func TestTryCheapFollowupRoute_RejectsHealthQuestion(t *testing.T) {
	client := &Client{}
	st := state.NewSession("sess-health")
	seedBaziAsset(st)
	st.Execution = contracts.ExecutionSnapshot{
		ConsultationKind: contracts.ConsultationKindPeriodFortune,
		PrimaryDomain:    "bazi",
		TaskIntent:       "fortune_followup",
		Gate: contracts.GateContract{
			Admitted:       true,
			ExecutionMode:  "execute",
			FollowupPolicy: "allow",
		},
	}

	if _, ok := client.tryCheapFollowupRoute("健康如何", st); ok {
		t.Fatal("cheap gate should reject health question")
	}
}

func TestTryCheapFollowupRoute_RejectsBirthInfoAmend(t *testing.T) {
	client := &Client{}
	st := state.NewSession("sess-3")
	seedBaziAsset(st)
	st.Execution = contracts.ExecutionSnapshot{
		ConsultationKind: contracts.ConsultationKindPeriodFortune,
		PrimaryDomain:    "bazi",
		TaskIntent:       "fortune_followup",
		Gate: contracts.GateContract{
			Admitted:       true,
			ExecutionMode:  "execute",
			FollowupPolicy: "allow",
		},
	}

	if _, ok := client.tryCheapFollowupRoute("我其实是1992年5月20日早上8点生的", st); ok {
		t.Fatal("cheap gate should reject birth-info amendment")
	}
}

func TestTryCheapFollowupRoute_RejectsExplicitMethodSwitch(t *testing.T) {
	client := &Client{}
	st := state.NewSession("sess-method-switch")
	seedBaziAsset(st)
	st.Execution = contracts.ExecutionSnapshot{
		ConsultationKind: contracts.ConsultationKindPeriodFortune,
		PrimaryDomain:    "bazi",
		TaskIntent:       "fortune_followup",
		Gate: contracts.GateContract{
			Admitted:       true,
			ExecutionMode:  "execute",
			FollowupPolicy: "allow",
		},
	}

	if _, ok := client.tryCheapFollowupRoute("换成紫微一起看看感情", st); ok {
		t.Fatal("cheap gate should reject explicit method switch")
	}
}

func TestTryCheapFollowupRoute_RejectsCrossDomainAsk(t *testing.T) {
	client := &Client{}
	st := state.NewSession("sess-cross-domain")
	seedBaziAsset(st)
	st.Execution = contracts.ExecutionSnapshot{
		ConsultationKind: contracts.ConsultationKindPeriodFortune,
		PrimaryDomain:    "bazi",
		TaskIntent:       "fortune_followup",
		Gate: contracts.GateContract{
			Admitted:       true,
			ExecutionMode:  "execute",
			FollowupPolicy: "allow",
		},
	}

	if _, ok := client.tryCheapFollowupRoute("你综合全面看看事业和感情", st); ok {
		t.Fatal("cheap gate should reject cross-domain ask")
	}
}

func TestTryCheapFollowupRoute_AllowsSingleDomainInterpretChartReuse(t *testing.T) {
	client := &Client{}
	st := state.NewSession("sess-interpret")
	seedBaziAsset(st)
	st.Execution = contracts.ExecutionSnapshot{
		ConsultationKind:   contracts.ConsultationKindNatalChart,
		PrimaryDomain:      "bazi",
		TaskIntent:         "interpret_chart",
		ConversationIntent: "consult",
		Gate: contracts.GateContract{
			Admitted:          true,
			ExecutionMode:     "execute",
			FollowupPolicy:    "allow",
			ReuseCachedResult: true,
		},
	}

	route, ok := client.tryCheapFollowupRoute("那这块能展开解释一下吗", st)
	if !ok {
		t.Fatal("single-domain interpret_chart should reuse via cheap gate")
	}
	if route.Gate.Reason != "cheap_followup_reuse" {
		t.Fatalf("Gate.Reason = %q, want cheap_followup_reuse", route.Gate.Reason)
	}
	if route.TaskIntent != "fortune_followup" {
		t.Fatalf("TaskIntent = %q, want fortune_followup", route.TaskIntent)
	}
}

func TestTryCheapFollowupRoute_RejectsInterpretChartWithSecondaryDomain(t *testing.T) {
	client := &Client{}
	st := state.NewSession("sess-interpret-mixed")
	seedBaziAsset(st)
	st.Execution = contracts.ExecutionSnapshot{
		ConsultationKind:   contracts.ConsultationKindPeriodFortune,
		PrimaryDomain:      "bazi",
		SecondaryDomains:   []string{"ziwei"},
		TaskIntent:         "interpret_chart",
		ConversationIntent: "consult",
		Gate: contracts.GateContract{
			Admitted:          true,
			ExecutionMode:     "execute",
			FollowupPolicy:    "allow",
			ReuseCachedResult: true,
		},
	}

	if _, ok := client.tryCheapFollowupRoute("那这块能展开解释一下吗", st); ok {
		t.Fatal("mixed-domain interpret_chart should not reuse via cheap gate")
	}
}

func TestApprove_CheapFollowupRouteSkipsSupervisorDecide(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	client := &Client{}
	st := state.NewSession("sess-4")
	seedBaziAsset(st)
	st.Execution = contracts.ExecutionSnapshot{
		ConsultationKind:   contracts.ConsultationKindPeriodFortune,
		PrimaryDomain:      "bazi",
		TaskIntent:         "fortune_followup",
		ConversationIntent: "consult",
		Gate: contracts.GateContract{
			Admitted:       true,
			ExecutionMode:  "execute",
			FollowupPolicy: "allow",
		},
	}

	route, err := client.Approve(ctx, "那感情呢", st)
	if err != nil {
		t.Fatalf("Approve() error = %v, want nil", err)
	}
	if route.Gate.Reason != "cheap_followup_reuse" {
		t.Fatalf("Gate.Reason = %q, want cheap_followup_reuse", route.Gate.Reason)
	}

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	found := false
	for _, span := range tr.Spans {
		if span.Name == "supervisor_decision" && span.Attributes["decision_source"] == "cheap_followup_reuse" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected supervisor_decision trace span for cheap followup reuse")
	}
}

func TestApprove_CheapFollowupRouteWritesSampleReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hits.jsonl")
	reporter := observability.NewCheapGateReporter(path)
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()
	tracing.SetTraceAttribute(ctx, "session_id", "sess-report-write")

	client := &Client{reporter: reporter}
	st := state.NewSession("sess-report-write")
	seedBaziAsset(st)
	st.Execution = contracts.ExecutionSnapshot{
		ConsultationKind:   contracts.ConsultationKindPeriodFortune,
		PrimaryDomain:      "bazi",
		TaskIntent:         "fortune_followup",
		ConversationIntent: "consult",
		Gate: contracts.GateContract{
			Admitted:       true,
			ExecutionMode:  "execute",
			FollowupPolicy: "allow",
		},
	}

	if _, err := client.Approve(ctx, "那感情呢", st); err != nil {
		t.Fatalf("Approve() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected cheap gate report file to contain data")
	}
}
