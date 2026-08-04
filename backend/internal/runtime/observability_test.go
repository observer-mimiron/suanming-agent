// This test file belongs to the manager-owned runtime layer.
// It verifies runtime observability behavior and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/schemas"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
	bazitool "github.com/observer-mimiron/suanming-agent/internal/tools/bazi"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
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
	calls  *int
}

func (t stubTool) Name() string        { return t.name }
func (t stubTool) Description() string { return t.name }
func (t stubTool) Label() string       { return t.name }
func (t stubTool) Execute(_ context.Context, _ map[string]any) (any, error) {
	if t.calls != nil {
		(*t.calls)++
	}
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
	graph, err := buildOrchestrationGraph()
	if err != nil {
		t.Fatalf("buildOrchestrationGraph: %v", err)
	}
	exec := &Executor{orchestrationGraph: graph}

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

func TestExecutor_SyncExecutionRoute_UpdatesSnapshotManagerContextAndTrace(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	exec := &Executor{manager: &Manager{}}
	st := state.NewSession("sess-route-sync")
	route := policy.ApprovedRoute{
		PrimaryDomain: "qimen",
		TaskIntent:    "fortune_followup",
	}
	plan := ExecutionPlan{
		Route:        route,
		Domains:      []string{"qimen"},
		Requirements: selectArtifactRequirements(st, []string{"qimen"}),
	}
	plan.Snapshot.PrimaryDomain = "qimen"
	plan.Snapshot.TaskIntent = "fortune_followup"
	plan.Snapshot.Domains = []string{"qimen"}
	plan.Snapshot.RequiredArtifacts = []string{artifactQimenChart}

	exec.syncExecutionRoute(ctx, st, route, plan)

	if st.Routing.PrimaryDomain != "qimen" {
		t.Fatalf("Routing.PrimaryDomain = %q, want qimen", st.Routing.PrimaryDomain)
	}
	if st.Routing.TaskIntent != "fortune_followup" {
		t.Fatalf("Routing.TaskIntent = %q, want fortune_followup", st.Routing.TaskIntent)
	}
	if st.ManagerContext.ActiveDomain != "qimen" {
		t.Fatalf("ManagerContext.ActiveDomain = %q, want qimen", st.ManagerContext.ActiveDomain)
	}
	if st.ManagerContext.CurrentTopic != "fortune_followup" {
		t.Fatalf("ManagerContext.CurrentTopic = %q, want fortune_followup", st.ManagerContext.CurrentTopic)
	}
	if st.Execution.PrimaryDomain != "qimen" {
		t.Fatalf("Execution.PrimaryDomain = %q, want qimen", st.Execution.PrimaryDomain)
	}

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	if got := tr.Attributes["approved_route.primary_domain"]; got != "qimen" {
		t.Fatalf("approved_route.primary_domain = %v, want qimen", got)
	}
	if got := tr.Attributes["task_intent"]; got != "fortune_followup" {
		t.Fatalf("task_intent = %v, want fortune_followup", got)
	}
}

func TestRunControlledBaziRetrieval_RecordsKnowledgeSearchSpan(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(stubTool{
		name: "knowledge_search",
		result: map[string]any{
			"passages": []any{
				map[string]any{
					"source":  "子平真诠/正官格",
					"content": "正官格以月令取之。",
				},
			},
		},
	})

	exec := &Executor{
		builder: NewAgentBuilder(nil, reg, nil, nil, AgentBuilderConfig{}),
	}

	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	plan := baziEvidencePlan{
		NeedRetrieval: true,
		Stage:         "static",
		QueryPackets: []baziQueryPacket{
			{
				Topic:      "geju",
				Query:      "子平真诠 格局 月令 取格",
				SourceTier: "A",
			},
		},
	}

	bundle, err := exec.runControlledBaziRetrieval(ctx, plan)
	if err != nil {
		t.Fatalf("runControlledBaziRetrieval() error = %v", err)
	}
	if len(bundle.Citations) == 0 {
		t.Fatal("expected citations to be returned")
	}

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}

	var retrievalSpan *tracing.TraceSpan
	for i := range tr.Spans {
		if tr.Spans[i].Name == "knowledge_search" {
			retrievalSpan = &tr.Spans[i]
			break
		}
	}
	if retrievalSpan == nil {
		t.Fatal("expected knowledge_search span to be recorded")
	}
	if got := retrievalSpan.Attributes["query"]; got != "子平真诠 格局 月令 取格" {
		t.Fatalf("knowledge_search query = %v", got)
	}
	if got := retrievalSpan.Attributes["hits"]; got != 1 {
		t.Fatalf("knowledge_search hits = %v, want 1", got)
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
	plan := ExecutionPlan{
		Route:        route,
		Domains:      []string{"bazi"},
		Requirements: selectArtifactRequirements(st, []string{"bazi"}),
	}
	vals := map[string]any{}

	exec.prefill(ctx, nil, st, plan, vals)

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

func TestPrefillBazi_EmitsEnrichedChartPayload(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(stubTool{
		name: "bazi_calc",
		result: map[string]any{
			"dayGan": "甲",
			"dayZhi": "子",
			"dayun": []map[string]any{
				{"startAge": 1, "endAge": 10, "ganZhi": "乙丑"},
			},
			"pillars": []map[string]any{
				{"name": "年柱", "stem": "甲", "branch": "子"},
				{"name": "月柱", "stem": "丙", "branch": "寅"},
				{"name": "日柱", "stem": "甲", "branch": "午"},
				{"name": "时柱", "stem": "庚", "branch": "申"},
			},
			"birthday": "1990-05-20 08:00",
		},
	})
	reg.Register(stubTool{
		name: "yongshen",
		result: map[string]any{
			"geju":           "建禄格",
			"geju_qing_zhuo": "浊中有清",
			"yong_shen":      []string{"火", "土"},
			"xi_shen":        []string{"木"},
			"ji_shen":        []string{"金", "水"},
		},
	})
	reg.Register(stubTool{
		name: "dayun_analyzer",
		result: map[string]any{
			"summary": "顺行",
		},
	})
	reg.Register(stubTool{
		name: "bazi_liunian",
		result: map[string]any{
			"target_year": 2026,
			"summary":     "流年观察",
		},
	})

	exec := &Executor{reg: reg}
	st := state.NewSession("sess-bazi-chart")
	st.MergeProfile(map[string]any{
		"year":   1990.0,
		"month":  5.0,
		"day":    20.0,
		"hour":   8.0,
		"gender": "男",
	})
	sink := &recordingSink{}
	vals := map[string]any{}

	ok := exec.prefillBazi(context.Background(), sink, st, vals)
	if !ok {
		t.Fatal("prefillBazi() = false, want true")
	}
	if len(sink.events) != 1 {
		t.Fatalf("events = %d, want 1", len(sink.events))
	}
	if sink.events[0].Type != "component" {
		t.Fatalf("event type = %q, want component", sink.events[0].Type)
	}

	eventData, ok := sink.events[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("event data type = %T, want map[string]any", sink.events[0].Data)
	}
	if got := eventData["type"]; got != "bazi-chart" {
		t.Fatalf("event data type field = %v, want bazi-chart", got)
	}

	payload, ok := eventData["payload"].(map[string]any)
	if !ok {
		t.Fatalf("payload type = %T, want map[string]any", eventData["payload"])
	}
	yongshen, ok := payload["yongshen"].(map[string]any)
	if !ok {
		t.Fatalf("payload yongshen type = %T, want map[string]any", payload["yongshen"])
	}
	if got := yongshen["geju"]; got != "建禄格" {
		t.Fatalf("payload yongshen.geju = %v, want 建禄格", got)
	}
	if _, ok := payload["dayun_analyzed"].(map[string]any); !ok {
		t.Fatalf("payload dayun_analyzed type = %T, want map[string]any", payload["dayun_analyzed"])
	}
	if _, ok := payload["liunian"].(map[string]any); !ok {
		t.Fatalf("payload liunian type = %T, want map[string]any", payload["liunian"])
	}
}

func TestPrefill_FortuneFollowupDoesNotReemitCachedBaziChart(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(stubTool{
		name: "yongshen",
		result: map[string]any{
			"geju":      "建禄格",
			"yong_shen": []string{"火", "土"},
		},
	})
	reg.Register(stubTool{
		name: "dayun_analyzer",
		result: map[string]any{
			"summary": "顺行",
		},
	})
	reg.Register(stubTool{
		name: "bazi_liunian",
		result: map[string]any{
			"target_year": 2026,
			"summary":     "流年观察",
		},
	})

	exec := &Executor{reg: reg}
	st := state.NewSession("sess-bazi-followup")
	st.MergeProfile(map[string]any{
		"year":   1990.0,
		"month":  5.0,
		"day":    20.0,
		"hour":   8.0,
		"gender": "男",
	})
	st.BaziResult = map[string]any{
		"calendar_rule_version": bazitool.CalendarRuleVersion,
		"dayGan":                "甲",
		"dayZhi":                "子",
		"birthday":              "1990-05-20 08:00",
		"dayun": []map[string]any{
			{"startAge": 1, "endAge": 10, "ganZhi": "乙丑"},
		},
		"pillars": []map[string]any{
			{"name": "年柱", "stem": "甲", "branch": "子"},
			{"name": "月柱", "stem": "丙", "branch": "寅"},
			{"name": "日柱", "stem": "甲", "branch": "午"},
			{"name": "时柱", "stem": "庚", "branch": "申"},
		},
	}
	sink := &recordingSink{}
	vals := map[string]any{}
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
	}
	plan := ExecutionPlan{
		Route:        route,
		Domains:      []string{"bazi"},
		Requirements: selectArtifactRequirements(st, []string{"bazi"}),
	}

	exec.prefill(context.Background(), sink, st, plan, vals)

	if len(sink.events) != 0 {
		t.Fatalf("events = %d, want 0 for cached bazi follow-up", len(sink.events))
	}
	if got := vals["bazi_result"]; got == nil {
		t.Fatal("expected cached bazi_result to remain available in vals")
	}
	if got := vals["yongshen"]; got == nil {
		t.Fatal("expected deterministic enrichments to remain available in vals")
	}
}

func TestPrefill_CrossDomainFollowupPrefillsSecondaryZiwei(t *testing.T) {
	ziweiCalls := 0
	liunianCalls := 0

	reg := tools.NewRegistry()
	reg.Register(stubTool{
		name: "bazi_calc",
		result: map[string]any{
			"calendar_rule_version": bazitool.CalendarRuleVersion,
			"dayGan":                "甲",
			"dayZhi":                "子",
			"birthday":              "1990-05-20 08:00",
			"dayun": []map[string]any{
				{"startAge": 1, "endAge": 10, "ganZhi": "乙丑"},
			},
			"pillars": []map[string]any{
				{"name": "年柱", "stem": "甲", "branch": "子"},
				{"name": "月柱", "stem": "丙", "branch": "寅"},
				{"name": "日柱", "stem": "甲", "branch": "午"},
				{"name": "时柱", "stem": "庚", "branch": "申"},
			},
		},
	})
	reg.Register(stubTool{name: "yongshen", result: map[string]any{"yong_shen": []string{"火"}}})
	reg.Register(stubTool{name: "dayun_analyzer", result: map[string]any{"summary": "顺行"}})
	reg.Register(stubTool{name: "bazi_liunian", result: map[string]any{"target_year": 2026}})
	reg.Register(stubTool{
		name:  "ziwei_calc",
		calls: &ziweiCalls,
		result: map[string]any{
			"mingGong": "天机",
		},
	})
	reg.Register(stubTool{
		name:  "ziwei_liunian",
		calls: &liunianCalls,
		result: map[string]any{
			"summary": "紫微流年",
		},
	})

	exec := &Executor{reg: reg}
	st := state.NewSession("sess-cross-domain-prefill")
	st.MergeProfile(map[string]any{
		"year":   1990.0,
		"month":  5.0,
		"day":    20.0,
		"hour":   8.0,
		"gender": "男",
	})
	route := policy.ApprovedRoute{
		PrimaryDomain:    "bazi",
		SecondaryDomains: []string{"ziwei"},
		TaskIntent:       "fortune_followup",
	}
	plan := ExecutionPlan{
		Route:        route,
		Domains:      []string{"bazi", "ziwei"},
		Requirements: selectArtifactRequirements(st, []string{"bazi", "ziwei"}),
	}
	vals := map[string]any{}

	exec.prefill(context.Background(), nil, st, plan, vals)

	if ziweiCalls != 1 {
		t.Fatalf("ziwei_calc calls = %d, want 1", ziweiCalls)
	}
	if liunianCalls != 1 {
		t.Fatalf("ziwei_liunian calls = %d, want 1", liunianCalls)
	}
	if st.ZiWeiResult == nil {
		t.Fatal("expected ZiWeiResult to be prefetched for cross-domain follow-up")
	}
	if got := vals["ziwei_result"]; got == nil {
		t.Fatal("expected ziwei_result in vals after cross-domain prefill")
	}
}

func TestPrefillBazi_RecalculatesLegacyCalendarRuleCache(t *testing.T) {
	calcCalls := 0
	yongshenCalls := 0
	dayunCalls := 0
	liunianCalls := 0

	reg := tools.NewRegistry()
	reg.Register(stubTool{
		name:  "bazi_calc",
		calls: &calcCalls,
		result: map[string]any{
			"calendar_rule_version": bazitool.CalendarRuleVersion,
			"dayGan":                "癸",
			"dayZhi":                "未",
			"birthday":              "2025-11-10 23:00",
			"gender":                "男",
			"dayun": []map[string]any{
				{"startAge": 1, "endAge": 10, "ganZhi": "甲子"},
			},
			"pillars": []map[string]any{
				{"name": "年柱", "stem": "乙", "branch": "巳"},
				{"name": "月柱", "stem": "丁", "branch": "亥"},
				{"name": "日柱", "stem": "癸", "branch": "未"},
				{"name": "时柱", "stem": "壬", "branch": "子"},
			},
		},
	})
	reg.Register(stubTool{
		name:  "yongshen",
		calls: &yongshenCalls,
		result: map[string]any{
			"day_master": "癸",
		},
	})
	reg.Register(stubTool{
		name:  "dayun_analyzer",
		calls: &dayunCalls,
		result: map[string]any{
			"summary": "顺行",
		},
	})
	reg.Register(stubTool{
		name:  "bazi_liunian",
		calls: &liunianCalls,
		result: map[string]any{
			"target_year": 2026,
		},
	})

	exec := &Executor{reg: reg}
	st := state.NewSession("sess-bazi-legacy-cache")
	st.MergeProfile(map[string]any{
		"year":   2025.0,
		"month":  11.0,
		"day":    10.0,
		"hour":   23.0,
		"gender": "男",
	})
	st.BaziResult = map[string]any{
		"calendar_rule_version": "late_zi_next_day_v1",
		"dayGan":                "甲",
		"dayZhi":                "申",
		"birthday":              "2025-11-10 23:00",
		"gender":                "男",
		"dayun": []map[string]any{
			{"startAge": 1, "endAge": 10, "ganZhi": "甲子"},
		},
		"pillars": []map[string]any{
			{"name": "年柱", "stem": "乙", "branch": "巳"},
			{"name": "月柱", "stem": "丁", "branch": "亥"},
			{"name": "日柱", "stem": "甲", "branch": "申"},
			{"name": "时柱", "stem": "甲", "branch": "子"},
		},
	}

	ok := exec.prefillBazi(context.Background(), nil, st, map[string]any{})
	if !ok {
		t.Fatal("prefillBazi() = false, want true")
	}
	if calcCalls != 1 {
		t.Fatalf("bazi_calc calls = %d, want 1", calcCalls)
	}
	if yongshenCalls != 1 {
		t.Fatalf("yongshen calls = %d, want 1", yongshenCalls)
	}
	if dayunCalls != 1 {
		t.Fatalf("dayun_analyzer calls = %d, want 1", dayunCalls)
	}
	if liunianCalls != 1 {
		t.Fatalf("bazi_liunian calls = %d, want 1", liunianCalls)
	}
	if got := st.BaziResult["calendar_rule_version"]; got != bazitool.CalendarRuleVersion {
		t.Fatalf("calendar_rule_version = %v, want %s", got, bazitool.CalendarRuleVersion)
	}
	if got := st.BaziResult["dayGan"]; got != "癸" {
		t.Fatalf("dayGan = %v, want 癸", got)
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

func TestGuardFinalAnswerWithTrace_BlocksMissingBaziArtifact(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	st := state.NewSession("sess-bazi-guard")
	route := policy.ApprovedRoute{PrimaryDomain: "bazi"}

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
	if got := tr.Attributes["failure.class"]; got != "specialist_contract_violation" {
		t.Fatalf("failure.class = %v, want specialist_contract_violation", got)
	}
	if got := tr.Attributes["failure.domain"]; got != "bazi" {
		t.Fatalf("failure.domain = %v, want bazi", got)
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
	if got := tr.Attributes["decision_source"]; got != "supervisor" {
		t.Fatalf("decision_source = %v, want supervisor", got)
	}
}

func TestAnnotateApprovedRouteTrace_ExposesCheapGateSignals(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	st := state.NewSession("sess-cheap-gate-root")
	st.BaziResult = map[string]any{"dayGan": "甲"}
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
		Gate: contracts.GateContract{
			Reason:              "cheap_followup_reuse",
			ExecutionMode:       "reuse_followup",
			FollowupPolicy:      "allow",
			ReuseCachedResult:   true,
			ReuseSessionProfile: true,
		},
	}

	annotateApprovedRouteTrace(ctx, st, route)

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	if got := tr.Attributes["decision_source"]; got != "cheap_followup_reuse" {
		t.Fatalf("decision_source = %v, want cheap_followup_reuse", got)
	}
	if got := tr.Attributes["reuse_cached_result"]; got != true {
		t.Fatalf("reuse_cached_result = %v, want true", got)
	}
	if got := tr.Attributes["reuse_session_profile"]; got != true {
		t.Fatalf("reuse_session_profile = %v, want true", got)
	}
	if got := tr.Attributes["gate.reason"]; got != "cheap_followup_reuse" {
		t.Fatalf("gate.reason = %v, want cheap_followup_reuse", got)
	}
}
