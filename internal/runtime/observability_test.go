package runtime

import (
	"context"
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tools"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

type recordingSink struct {
	events []Event
}

func (s *recordingSink) Emit(_ context.Context, evt Event) error {
	s.events = append(s.events, evt)
	return nil
}

type stubTool struct {
	name   string
	result map[string]any
}

func (t stubTool) Name() string        { return t.name }
func (t stubTool) Description() string { return t.name }
func (t stubTool) Execute(_ context.Context, _ map[string]any) (any, error) {
	return t.result, nil
}

func TestExecute_RecordsPreflightAndSSETraceOnShortCircuit(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	st := state.NewSession("sess-preflight")
	route := policy.ApprovedRoute{
		NeedsClarification:    true,
		ClarificationQuestion: "请确认问题范围。",
	}
	sink := &recordingSink{}
	exec := &Executor{}

	turnType, text, err := exec.Execute(ctx, sink, st, route, "帮我看看")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if turnType != "clarification" {
		t.Fatalf("turnType = %q, want clarification", turnType)
	}
	if text != "请确认问题范围。" {
		t.Fatalf("text = %q, want clarification question", text)
	}

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}

	var preflightSpan, sseSpan *tracing.TraceSpan
	for i := range tr.Spans {
		switch tr.Spans[i].Name {
		case "preflight":
			preflightSpan = &tr.Spans[i]
		case "sse_emit":
			sseSpan = &tr.Spans[i]
		}
	}
	if preflightSpan == nil {
		t.Fatal("expected preflight span to be recorded")
	}
	if got := preflightSpan.Attributes["short_circuit"]; got != true {
		t.Fatalf("preflight short_circuit = %v, want true", got)
	}
	if got := preflightSpan.Attributes["turn_type"]; got != "clarification" {
		t.Fatalf("preflight turn_type = %v, want clarification", got)
	}
	if sseSpan == nil {
		t.Fatal("expected sse_emit span to be recorded")
	}
	if got := sseSpan.Attributes["event_type"]; got != "text" {
		t.Fatalf("sse_emit event_type = %v, want text", got)
	}
}

func TestExecutor_UpdateGuidanceState_ClearsOnExecutionEntry(t *testing.T) {
	exec := &Executor{}
	st := state.NewSession("sess-guidance-clear")
	st.Guidance = &state.GuidanceState{
		DirectiveKind: "collect_slot",
		ChosenTopic:   "事业",
		PendingSlot:   "birth_time",
		RetryCount:    2,
	}

	exec.updateGuidanceState(st, policy.ApprovedRoute{TaskIntent: "interpret_chart"}, "继续分析", preflightResult{})

	if st.Guidance != nil {
		t.Fatalf("Guidance = %#v, want nil", st.Guidance)
	}
}

func TestExecutor_UpdateGuidanceState_PreservesCollectProfileFlowWhenProfileIncomplete(t *testing.T) {
	exec := &Executor{}
	st := state.NewSession("sess-guidance-collect")
	st.Guidance = &state.GuidanceState{
		DirectiveKind: "collect_slot",
		ChosenTopic:   "事业",
		PendingSlot:   "birth_date",
		RetryCount:    2,
	}
	st.MergeProfile(map[string]any{
		"year":  1990.0,
		"month": 5.0,
		"day":   20.0,
	})

	exec.updateGuidanceState(st, policy.ApprovedRoute{TaskIntent: "collect_profile"}, "1990年5月20日", preflightResult{})

	if st.Guidance == nil {
		t.Fatal("Guidance = nil, want preserved guidance state")
	}
	if st.Guidance.ChosenTopic != "事业" {
		t.Fatalf("Guidance.ChosenTopic = %q, want 事业", st.Guidance.ChosenTopic)
	}
	if st.Guidance.RetryCount != 0 {
		t.Fatalf("Guidance.RetryCount = %d, want 0 (reset on effective slot advance)", st.Guidance.RetryCount)
	}
	if st.Guidance.PendingSlot != "birth_time" {
		t.Fatalf("Guidance.PendingSlot = %q, want birth_time", st.Guidance.PendingSlot)
	}
}

func TestExecutor_UpdateGuidanceState_GenericShortCircuitDoesNotMutateGuidance(t *testing.T) {
	exec := &Executor{}
	st := state.NewSession("sess-guidance-generic-short")
	st.Guidance = &state.GuidanceState{
		DirectiveKind: "offer_consult",
		ChosenTopic:   "事业",
		PendingSlot:   "",
		RetryCount:    1,
	}

	exec.updateGuidanceState(st, policy.ApprovedRoute{
		NeedsClarification:    true,
		ClarificationQuestion: "请确认一下问题范围。",
	}, "行，那你看看感情", preflightResult{ShortCircuit: true})

	if st.Guidance == nil {
		t.Fatal("Guidance = nil, want unchanged guidance state")
	}
	if st.Guidance.DirectiveKind != "offer_consult" {
		t.Fatalf("Guidance.DirectiveKind = %q, want offer_consult", st.Guidance.DirectiveKind)
	}
	if st.Guidance.ChosenTopic != "事业" {
		t.Fatalf("Guidance.ChosenTopic = %q, want 事业", st.Guidance.ChosenTopic)
	}
	if st.Guidance.PendingSlot != "" {
		t.Fatalf("Guidance.PendingSlot = %q, want empty", st.Guidance.PendingSlot)
	}
	if st.Guidance.RetryCount != 1 {
		t.Fatalf("Guidance.RetryCount = %d, want 1", st.Guidance.RetryCount)
	}
}

func TestPrefill_RecordsBaziPrefillSpan(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	reg := tools.NewRegistry()
	reg.Register(stubTool{
		name: "bazi_calc",
		result: map[string]any{
			"dayGan": "甲",
			"dayZhi": "子",
			"dayun":  []map[string]any{{"gan": "甲", "zhi": "子"}},
		},
	})
	reg.Register(stubTool{name: "yongshen", result: map[string]any{"yongshen": "木"}})
	reg.Register(stubTool{name: "dayun_analyzer", result: map[string]any{"summary": "顺行"}})

	exec := &Executor{reg: reg}
	st := state.NewSession("sess-prefill")
	st.MergeProfile(map[string]any{
		"year": 1990.0, "month": 5.0, "day": 20.0, "hour": 8.0, "gender": "男",
	})
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		PolicyHints: schemas.PolicyHints{
			ProfileRequirement: "full",
		},
	}
	vals := map[string]any{}

	exec.prefill(ctx, nil, st, route, vals)

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}

	var prefillSpan *tracing.TraceSpan
	for i := range tr.Spans {
		if tr.Spans[i].Name == "prefill" {
			prefillSpan = &tr.Spans[i]
			break
		}
	}
	if prefillSpan == nil {
		t.Fatal("expected prefill span to be recorded")
	}
	if got := prefillSpan.Attributes["domain"]; got != "bazi" {
		t.Fatalf("prefill domain = %v, want bazi", got)
	}
	if got := prefillSpan.Attributes["executed"]; got != true {
		t.Fatalf("prefill executed = %v, want true", got)
	}
}

func TestGuardFinalAnswerWithTrace_RecordsContractGate(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	st := state.NewSession("sess-guard")
	route := policy.ApprovedRoute{PrimaryDomain: "qimen"}

	turnType, text := guardFinalAnswerWithTrace(ctx, route, st, "final")
	if turnType != "guardrail_blocked" {
		t.Fatalf("turnType = %q, want guardrail_blocked", turnType)
	}
	if text == "final" {
		t.Fatal("expected guardFinalAnswerWithTrace to block original text")
	}

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}

	var guardSpan *tracing.TraceSpan
	for i := range tr.Spans {
		if tr.Spans[i].Name == "contract_gate" {
			guardSpan = &tr.Spans[i]
			break
		}
	}
	if guardSpan == nil {
		t.Fatal("expected contract_gate span to be recorded")
	}
	if got := guardSpan.Attributes["artifact_present"]; got != false {
		t.Fatalf("artifact_present = %v, want false", got)
	}
	if got := guardSpan.Attributes["guardrail_result"]; got != "blocked" {
		t.Fatalf("guardrail_result = %v, want blocked", got)
	}
}

func TestAnnotateApprovedRouteTrace_SetsRootTraceAttributes(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	st := state.NewSession("sess-root")
	st.MergeProfile(map[string]any{"year": 1990.0})
	route := policy.ApprovedRoute{
		PrimaryDomain:      "qimen",
		SecondaryDomains:   []string{"bazi"},
		TaskIntent:         "fortune_followup",
		NeedsClarification: true,
		PolicyHints: schemas.PolicyHints{
			QimenMode:          "primary",
			ProfileRequirement: "full",
		},
	}

	annotateApprovedRouteTrace(ctx, st, route)

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	if got := tr.Attributes["approved_route.primary_domain"]; got != "qimen" {
		t.Fatalf("primary_domain = %v, want qimen", got)
	}
	if got := tr.Attributes["task_intent"]; got != "fortune_followup" {
		t.Fatalf("task_intent = %v, want fortune_followup", got)
	}
	if got := tr.Attributes["qimen_mode"]; got != "primary" {
		t.Fatalf("qimen_mode = %v, want primary", got)
	}
	if got := tr.Attributes["profile_requirement"]; got != "full" {
		t.Fatalf("profile_requirement = %v, want full", got)
	}
	if got := tr.Attributes["needs_clarification"]; got != true {
		t.Fatalf("needs_clarification = %v, want true", got)
	}
	if got := tr.Attributes["profile_complete"]; got != false {
		t.Fatalf("profile_complete = %v, want false", got)
	}
}
