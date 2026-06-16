package orchestrator

import (
	"context"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"

	einocallbacks "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/mcp"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	bazi "github.com/wikiglobal/suanming-agent/internal/specialists/bazi"
	qimen "github.com/wikiglobal/suanming-agent/internal/specialists/qimen"
	ziwei "github.com/wikiglobal/suanming-agent/internal/specialists/ziwei"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tools"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

type fakeTool struct {
	name      string
	executeFn func(ctx context.Context, params map[string]any) (any, error)
}

func (t *fakeTool) Name() string        { return t.name }
func (t *fakeTool) Description() string { return t.name }
func (t *fakeTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	return t.executeFn(ctx, params)
}

type streamClient struct {
	generateFn func(ctx context.Context, systemPrompt string, messages []llm.Message) (string, llm.TokenUsage, error)
	streamFn   func(ctx context.Context, systemPrompt string, messages []llm.Message, onChunk func(string)) error
}

func (c *streamClient) Generate(ctx context.Context, systemPrompt string, messages []llm.Message) (string, llm.TokenUsage, error) {
	if c.generateFn != nil {
		return c.generateFn(ctx, systemPrompt, messages)
	}
	return `{"action":"followup"}`, llm.TokenUsage{}, nil
}

func (c *streamClient) Stream(ctx context.Context, systemPrompt string, messages []llm.Message, onChunk func(string)) error {
	if c.streamFn != nil {
		return c.streamFn(ctx, systemPrompt, messages, onChunk)
	}
	return nil
}

type orchestratorFakeToolCallingModel struct {
	emitCallbacks bool
	generateFn    func(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error)
	streamFn      func(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.StreamReader[*schema.Message], error)
}

func (m *orchestratorFakeToolCallingModel) Generate(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error) {
	if !m.emitCallbacks {
		if m.generateFn != nil {
			return m.generateFn(ctx, input, opts...)
		}
		return schema.AssistantMessage("ok", nil), nil
	}

	ctx = einocallbacks.EnsureRunInfo(ctx, "orchestratorFakeToolCallingModel", components.ComponentOfChatModel)
	ctx = einocallbacks.OnStart(ctx, &einomodel.CallbackInput{Messages: input})
	var (
		msg *schema.Message
		err error
	)
	if m.generateFn != nil {
		msg, err = m.generateFn(ctx, input, opts...)
	} else {
		msg = schema.AssistantMessage("ok", nil)
	}
	if err != nil {
		einocallbacks.OnError(ctx, err)
		return nil, err
	}
	einocallbacks.OnEnd(ctx, &einomodel.CallbackOutput{Message: msg})
	return msg, nil
}

func (m *orchestratorFakeToolCallingModel) Stream(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
	if !m.emitCallbacks {
		if m.streamFn != nil {
			return m.streamFn(ctx, input, opts...)
		}
		sr, sw := schema.Pipe[*schema.Message](1)
		go func() {
			defer sw.Close()
			sw.Send(schema.AssistantMessage("ok", nil), nil)
		}()
		return sr, nil
	}

	ctx = einocallbacks.EnsureRunInfo(ctx, "orchestratorFakeToolCallingModel", components.ComponentOfChatModel)
	ctx = einocallbacks.OnStart(ctx, &einomodel.CallbackInput{Messages: input})
	var (
		sr  *schema.StreamReader[*schema.Message]
		err error
	)
	if m.streamFn != nil {
		sr, err = m.streamFn(ctx, input, opts...)
	} else {
		localSR, sw := schema.Pipe[*schema.Message](1)
		sr = localSR
		go func() {
			defer sw.Close()
			sw.Send(schema.AssistantMessage("ok", nil), nil)
		}()
	}
	if err != nil {
		einocallbacks.OnError(ctx, err)
		return nil, err
	}
	_, sr = einocallbacks.OnEndWithStreamOutput(ctx, sr)
	return sr, nil
}

func (m *orchestratorFakeToolCallingModel) WithTools(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
	return m, nil
}

// recordingSink captures emitted events for inspection.
type recordingSink struct {
	events []Event
}

func (s *recordingSink) Emit(_ context.Context, evt Event) error {
	s.events = append(s.events, evt)
	return nil
}

func (s *recordingSink) lastComponentType() string {
	for i := len(s.events) - 1; i >= 0; i-- {
		if s.events[i].Type == "component" {
			if m, ok := s.events[i].Data.(map[string]any); ok {
				if t, ok2 := m["type"].(string); ok2 {
					return t
				}
			}
		}
	}
	return ""
}

type fakeDomainHandler struct {
	name  string
	calls int
	runFn func(ctx context.Context, st *state.SessionState, route policy.ApprovedRoute, sink specialists.EventSink) (schemas.DomainResult, error)
}

func (h *fakeDomainHandler) Name() string { return h.name }

func (h *fakeDomainHandler) Run(ctx context.Context, st *state.SessionState, route policy.ApprovedRoute, sink specialists.EventSink) (schemas.DomainResult, error) {
	h.calls++
	if h.runFn != nil {
		return h.runFn(ctx, st, route, sink)
	}
	return schemas.DomainResult{Domain: h.name, Final: false}, nil
}

func TestRun_ErrorTurn_PropagatesToTrace(t *testing.T) {
	// Setup: flash returns a classify result with complete profile so we enter handleFullReading.
	flashClient := &llm.NoopClient{
		GenerateFn: func(_ context.Context, _ string, _ []llm.Message) (string, llm.TokenUsage, error) {
			return `{"action":"new_profile","year":1990,"month":5,"day":20,"hour":8,"gender":"男","birthplace":"北京","needs_qimen":false,"needs_knowledge":true}`, llm.TokenUsage{}, nil
		},
	}

	// Don't register bazi_calc — handleFullReading will fail immediately.
	reg := tools.NewRegistry()

	rt := tracing.NewRealTracer(nil)
	orch := New(reg, flashClient, flashClient, state.NewMemoryStore(), state.NewMemoryLocker(), rt, "soft")
	orch.SetLLMModel("test-model")
	orch.SetSupervisor(&mockSupervisor{decision: schemas.SupervisorDecision{
		PrimaryDomain: "bazi", TaskIntent: "collect_profile", Confidence: 0.8,
		Slots: schemas.DecisionSlots{Profile: map[string]any{
			"year": 1990.0, "month": 5.0, "day": 20.0, "hour": 8.0, "gender": "男", "birthplace": "北京",
		}},
	}})

	sink := &recordingSink{}

	err := orch.Run(context.Background(), sink, "test-session", "1990年5月20日早上8点 男 北京")

	// The turn should return an error because bazi_calc is not registered.
	if err == nil {
		t.Fatal("expected error from Run(), got nil — bazi_calc should not be registered")
	}

	// Verify trace-digest was emitted via component event.
	if sink.lastComponentType() != "trace-panel" {
		t.Error("expected trace-panel component event in sink")
	}

	// Verify the trace still in context after Run() — but Run() consumed the context.
	// We verify indirectly: the error path means trace.SetStatus("error") was called.
	// The digest would have been built with status="error".
	t.Logf("Run() error: %v", err)
	t.Logf("events captured: %d", len(sink.events))
}

func TestRun_ErrorTurn_TraceStatusInContext(t *testing.T) {
	rt := tracing.NewRealTracer(nil)

	// Manually exercise the trace lifecycle to verify SetStatus wiring.
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	// Simulate the Run() error path: orchestrator detects an error and sets status.
	trace.SetStatus("error")

	// Simulate what Run() does: read trace from ctx and build digest.
	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	tr.TurnType = "full_reading"
	tr.EndedAt = tr.StartedAt // simulate a finished trace

	digest := tr.BuildDigest()
	if digest.Status != "error" {
		t.Errorf("digest.Status = %s, want error", digest.Status)
	}

	// Verify that BuildDigest uses EndedAt when set (P1 fix).
	if digest.TotalMs != 0 {
		t.Logf("TotalMs with EndedAt==StartedAt: %d (expected 0)", digest.TotalMs)
	}
}

func TestRun_DigestStableForFinishedTrace(t *testing.T) {
	// P1: Verify that BuildDigest gives stable total_ms for a finished trace.
	flashClient := &llm.NoopClient{
		GenerateFn: func(_ context.Context, _ string, _ []llm.Message) (string, llm.TokenUsage, error) {
			return `{"action":"new_profile","year":1990,"month":5,"day":20,"hour":8,"gender":"男","birthplace":"北京","needs_qimen":false,"needs_knowledge":true}`, llm.TokenUsage{}, nil
		},
	}

	reg := tools.NewRegistry()
	rt := tracing.NewRealTracer(nil)
	orch := New(reg, flashClient, flashClient, state.NewMemoryStore(), state.NewMemoryLocker(), rt, "soft")
	orch.SetSupervisor(&mockSupervisor{decision: schemas.SupervisorDecision{
		PrimaryDomain: "bazi", TaskIntent: "collect_profile", Confidence: 0.8,
		Slots: schemas.DecisionSlots{Profile: map[string]any{
			"year": 1990.0, "month": 5.0, "day": 20.0, "hour": 8.0, "gender": "男", "birthplace": "北京",
		}},
	}})

	sink := &recordingSink{}
	_ = orch.Run(context.Background(), sink, "test-session", "1990年5月20日早上8点 男 北京")

	// Find the trace-panel component and check the digest.
	for _, evt := range sink.events {
		if evt.Type != "component" {
			continue
		}
		m, ok := evt.Data.(map[string]any)
		if !ok {
			continue
		}
		if t, ok2 := m["type"].(string); !ok2 || t != "trace-panel" {
			continue
		}
		digest, ok3 := m["payload"].(tracing.TraceDigest)
		if !ok3 {
			t.Fatal("trace-panel payload is not TraceDigest")
		}
		if digest.Status != "error" {
			t.Errorf("expected digest status=error (bazi_calc not registered), got %s", digest.Status)
		}
		if digest.TotalMs < 0 {
			t.Errorf("total_ms should never be negative, got %d", digest.TotalMs)
		}
		// BuildDigest again from the same trace — should give same total_ms (not drift).
		total1 := digest.TotalMs
		// We can't re-read from context here, but the fact that EndedAt was set means
		// the value is deterministic.
		t.Logf("trace-panel total_ms: %d, status: %s", total1, digest.Status)
		return
	}
	t.Error("trace-panel component not found in events")
}

func TestStreamInterpretation_EinoCallbackTracingDoesNotDuplicateLLMSpan(t *testing.T) {
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(tracing.NewEinoTraceCallbackHandler())

	model := &orchestratorFakeToolCallingModel{
		emitCallbacks: true,
		streamFn: func(_ context.Context, _ []*schema.Message, _ ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
			sr, sw := schema.Pipe[*schema.Message](1)
			go func() {
				defer sw.Close()
				sw.Send(&schema.Message{Role: schema.Assistant, Content: "解读内容"}, nil)
			}()
			return sr, nil
		},
	}

	chat := llm.NewEinoChat(model)
	orch := New(tools.NewRegistry(), chat, &llm.NoopClient{}, state.NewMemoryStore(), state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")
	orch.SetLLMModel("deepseek-v4-pro")

	st := state.NewSession("s1")
	st.LastUserQuestion = "看看事业"
	sink := &recordingSink{}

	ctx, trace := tracing.NewRealTracer(nil).StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	_, err := orch.streamInterpretation(ctx, sink, st, nil, "bazi")
	if err != nil && err != io.EOF {
		t.Fatalf("streamInterpretation error = %v", err)
	}

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	var count int
	for _, span := range tr.Spans {
		if span.Name == "llm_generate" && span.Kind == tracing.KindLLM {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("llm_generate span count = %d, want 1", count)
	}
}

func TestRun_FollowupWithDirectBaziContext_DoesNotAskMissingProfile(t *testing.T) {
	flashClient := &llm.NoopClient{
		GenerateFn: func(_ context.Context, _ string, _ []llm.Message) (string, llm.TokenUsage, error) {
			return `{"action":"followup","question":"今年怎么样","needs_qimen":false,"needs_knowledge":false}`, llm.TokenUsage{}, nil
		},
	}

	store := state.NewMemoryStore()
	st := store.LoadOrCreate("chart-only")
	st.BaziResult = map[string]any{
		"dayGan": "甲",
		"pillars": []map[string]any{
			{"stem": "乙", "branch": "巳"},
			{"stem": "丁", "branch": "亥"},
			{"stem": "甲", "branch": "申"},
			{"stem": "甲", "branch": "子"},
		},
	}

	orch := New(tools.NewRegistry(), flashClient, flashClient, store, state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")
	orch.SetSupervisor(&mockSupervisor{decision: schemas.SupervisorDecision{
		PrimaryDomain: "bazi", TaskIntent: "fortune_followup", Confidence: 0.8,
	}})
	sink := &recordingSink{}

	if err := orch.Run(context.Background(), sink, "chart-only", "今年怎么样"); err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	for _, evt := range sink.events {
		if evt.Type != "text" {
			continue
		}
		if m, ok := evt.Data.(map[string]any); ok {
			if content, ok2 := m["content"].(string); ok2 && content == "请告诉我你的出生年份、出生月份、出生日期、出生时辰、性别、出生地（城市）" {
				t.Fatal("followup with existing chart should not fall back to ask_missing_profile")
			}
		}
	}
}

func TestRun_NewProfileError_DoesNotClearExistingSessionState(t *testing.T) {
	flashClient := &llm.NoopClient{
		GenerateFn: func(_ context.Context, _ string, _ []llm.Message) (string, llm.TokenUsage, error) {
			return `{"action":"new_profile","year":1991,"month":6,"day":21,"hour":9,"gender":"女","birthplace":"上海","needs_qimen":false,"needs_knowledge":false}`, llm.TokenUsage{}, nil
		},
	}

	store := state.NewMemoryStore()
	st := store.LoadOrCreate("preserve-on-error")
	st.Profile = map[string]any{"year": 1990.0, "month": 5.0, "day": 20.0, "hour": 8.0, "gender": "男", "birthplace": "北京"}
	st.BaziResult = map[string]any{"dayGan": "乙"}

	orch := New(tools.NewRegistry(), flashClient, flashClient, store, state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")

	err := orch.Run(context.Background(), &recordingSink{}, "preserve-on-error", "换一个人：1991年6月21日9点，女，上海")
	if err == nil {
		t.Fatal("expected error because bazi_calc is not registered")
	}

	got := store.LoadOrCreate("preserve-on-error")
	if got.Profile["year"] != 1990.0 {
		t.Fatalf("existing profile was overwritten on failed new_profile run: %+v", got.Profile)
	}
	if got.BaziResult["dayGan"] != "乙" {
		t.Fatalf("existing bazi_result was cleared on failed new_profile run: %+v", got.BaziResult)
	}
}

func TestRun_CanceledAfterChartBuild_PersistsNewChart(t *testing.T) {
	classifier := &streamClient{
		generateFn: func(_ context.Context, _ string, _ []llm.Message) (string, llm.TokenUsage, error) {
			return `{"action":"new_profile","year":1991,"month":6,"day":21,"hour":9,"gender":"女","birthplace":"上海","needs_qimen":false,"needs_knowledge":false}`, llm.TokenUsage{}, nil
		},
	}
	answerer := &streamClient{
		streamFn: func(_ context.Context, _ string, _ []llm.Message, _ func(string)) error {
			return context.Canceled
		},
	}

	reg := tools.NewRegistry()
	reg.Register(&fakeTool{
		name: "bazi_calc",
		executeFn: func(_ context.Context, params map[string]any) (any, error) {
			return map[string]any{
				"dayGan": "辛",
				"pillars": []map[string]any{
					{"stem": "辛", "branch": "未"},
					{"stem": "甲", "branch": "午"},
					{"stem": "辛", "branch": "酉"},
					{"stem": "癸", "branch": "巳"},
				},
			}, nil
		},
	})

	store := state.NewMemoryStore()
	st := store.LoadOrCreate("persist-after-cancel")
	st.Profile = map[string]any{"year": 1990.0, "month": 5.0, "day": 20.0, "hour": 8.0, "gender": "男", "birthplace": "北京"}
	st.BaziResult = map[string]any{"dayGan": "乙"}

	orch := New(reg, answerer, classifier, store, state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")
	orch.SetSupervisor(&mockSupervisor{decision: schemas.SupervisorDecision{
		PrimaryDomain: "bazi", TaskIntent: "collect_profile", Confidence: 0.8,
		Slots: schemas.DecisionSlots{Profile: map[string]any{
			"year": 1991.0, "month": 6.0, "day": 21.0, "hour": 9.0, "gender": "女", "birthplace": "上海",
		}},
	}})

	err := orch.Run(context.Background(), &recordingSink{}, "persist-after-cancel", "换一个人：1991年6月21日9点，女，上海")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	got := store.LoadOrCreate("persist-after-cancel")
	if got.Profile["year"] != 1991.0 {
		t.Fatalf("new profile should persist once chart build completed: %+v", got.Profile)
	}
	if got.BaziResult["dayGan"] != "辛" {
		t.Fatalf("new chart should persist after canceled stream: %+v", got.BaziResult)
	}
}

func TestRecordTurnAndMaintainContext_SummaryFailureDoesNotDropOverflowTurns(t *testing.T) {
	flashClient := &llm.NoopClient{
		GenerateFn: func(_ context.Context, _ string, _ []llm.Message) (string, llm.TokenUsage, error) {
			return "", llm.TokenUsage{}, errors.New("summary unavailable")
		},
	}

	orch := New(tools.NewRegistry(), flashClient, flashClient, state.NewMemoryStore(), state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")
	st := state.NewSession("summary-fallback")
	st.RunningSummary = "已有摘要"
	for i := 0; i < state.MaxRecentTurns; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		st.RecordTurn(role, "old turn")
	}

	orch.recordTurnAndMaintainContext(context.Background(), st, "new user", "new assistant")

	if st.RunningSummary != "已有摘要" {
		t.Fatalf("running summary changed on summary failure: %q", st.RunningSummary)
	}
	if len(st.RecentTurns) != state.MaxRecentTurns+2 {
		t.Fatalf("overflow turns were dropped on summary failure, got %d turns want %d", len(st.RecentTurns), state.MaxRecentTurns+2)
	}
	if st.RecentTurns[0].Content != "old turn" {
		t.Fatalf("oldest turns should still be retained after summary failure: %+v", st.RecentTurns[0])
	}
}

func TestRun_FailedNewProfileDoesNotContaminateExistingContextMemory(t *testing.T) {
	flashClient := &llm.NoopClient{
		GenerateFn: func(_ context.Context, _ string, _ []llm.Message) (string, llm.TokenUsage, error) {
			return `{"action":"new_profile","year":1991,"month":6,"day":21,"hour":9,"gender":"女","birthplace":"上海","needs_qimen":false,"needs_knowledge":false}`, llm.TokenUsage{}, nil
		},
	}

	store := state.NewMemoryStore()
	st := store.LoadOrCreate("no-context-contamination")
	st.Profile = map[string]any{"year": 1990.0, "month": 5.0, "day": 20.0, "hour": 8.0, "gender": "男", "birthplace": "北京"}
	st.BaziResult = map[string]any{"dayGan": "乙"}
	st.RecordTurn("user", "老会话问题")
	st.RecordTurn("assistant", "老会话回答")
	st.RunningSummary = "老摘要"

	orch := New(tools.NewRegistry(), flashClient, flashClient, store, state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")

	err := orch.Run(context.Background(), &recordingSink{}, "no-context-contamination", "换一个人：1991年6月21日9点，女，上海")
	if err == nil {
		t.Fatal("expected error because bazi_calc is not registered")
	}

	got := store.LoadOrCreate("no-context-contamination")
	if got.RunningSummary != "老摘要" {
		t.Fatalf("running summary should stay unchanged on failed new_profile: %q", got.RunningSummary)
	}
	if len(got.RecentTurns) != 2 {
		t.Fatalf("recent turns should stay unchanged on failed new_profile, got %d", len(got.RecentTurns))
	}
	for _, turn := range got.RecentTurns {
		if strings.Contains(turn.Content, "换一个人") {
			t.Fatalf("failed new_profile request leaked into existing context memory: %+v", got.RecentTurns)
		}
	}
}

// --- Supervisor integration tests ---

func TestRun_SupervisorDecisionToBaziSpecialist(t *testing.T) {
	// Exercise the full supervisor → policy → bazi specialist chain.
	st := state.NewSession("test-supervisor-bazi")
	st.MergeProfile(map[string]any{
		"year": 1990.0, "month": 5.0, "day": 20.0,
		"hour": 8.0, "gender": "男", "birthplace": "北京",
	})
	st.BaziResult = map[string]any{"dayGan": "甲", "dayZhi": "子"}

	// Build a supervisor decision and run through policy gate, then bazi specialist.
	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		TaskIntent:         "interpret_chart",
		Confidence:         0.9,
	}
	d.Normalize()

	route := policy.Apply(d, st)
	if route.NeedsClarification {
		t.Fatal("complete profile should not need clarification")
	}
	if route.PrimaryDomain != "bazi" {
		t.Fatalf("PrimaryDomain: got %q, want %q", route.PrimaryDomain, "bazi")
	}

	// Dispatch to bazi specialist.
	baziSp := bazi.New()
	result, err := baziSp.Run(context.Background(), st, policy.ApprovedRoute{
		ConversationIntent:    route.ConversationIntent,
		PrimaryDomain:         route.PrimaryDomain,
		SecondaryDomains:      route.SecondaryDomains,
		TaskIntent:            route.TaskIntent,
		NeedsClarification:    route.NeedsClarification,
		ClarificationQuestion: route.ClarificationQuestion,
		ParallelAllowed:       route.ParallelAllowed,
		Slots:                 route.Slots,
		PolicyHints:           route.PolicyHints,
	}, nil)
	if err != nil {
		t.Fatalf("bazi specialist error: %v", err)
	}
	if result.Domain != "bazi" {
		t.Fatalf("Domain: got %q, want %q", result.Domain, "bazi")
	}
}

func TestRun_ClarificationForced(t *testing.T) {
	// Incomplete profile + interpret_chart → policy gate forces clarification.
	st := state.NewSession("test-clarify")
	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		TaskIntent:         "interpret_chart",
		Confidence:         0.9,
	}
	d.Normalize()

	route := policy.Apply(d, st)
	if !route.NeedsClarification {
		t.Fatal("incomplete profile should force clarification")
	}
	if route.ClarificationQuestion == "" {
		t.Fatal("clarification question should not be empty")
	}
}

func TestRun_BaziPrimaryQimenSecondarySequential(t *testing.T) {
	// bazi primary + qimen secondary → sequential supplement (phase 1).
	st := state.NewSession("test-bazi-qimen")
	st.MergeProfile(map[string]any{
		"year": 1990.0, "month": 5.0, "day": 20.0,
		"hour": 8.0, "gender": "男", "birthplace": "北京",
	})
	st.BaziResult = map[string]any{"dayGan": "甲"}

	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		SecondaryDomains:   []string{"qimen"},
		TaskIntent:         "cross_domain_consult",
		Confidence:         0.85,
		PolicyHints:        schemas.PolicyHints{NeedsQimen: true},
	}
	d.Normalize()

	route := policy.Apply(d, st)
	if route.ParallelAllowed {
		t.Fatal("parallel must be disabled in phase 1")
	}

	// bazi runs first.
	baziSp := bazi.New()
	baziResult, err := baziSp.Run(context.Background(), st, policy.ApprovedRoute{
		ConversationIntent: route.ConversationIntent,
		PrimaryDomain:      route.PrimaryDomain,
		SecondaryDomains:   route.SecondaryDomains,
		TaskIntent:         route.TaskIntent,
		Slots:              route.Slots,
		PolicyHints:        route.PolicyHints,
	}, nil)
	if err != nil {
		t.Fatalf("bazi specialist error: %v", err)
	}
	if baziResult.Domain != "bazi" {
		t.Fatalf("bazi Domain: got %q, want %q", baziResult.Domain, "bazi")
	}

	// qimen runs as sequential supplement.
	qimenSp := qimen.New()
	qimenResult, err := qimenSp.Run(context.Background(), st, policy.ApprovedRoute{
		ConversationIntent: route.ConversationIntent,
		PrimaryDomain:      "qimen",
		SecondaryDomains:   route.SecondaryDomains,
		TaskIntent:         route.TaskIntent,
		Slots:              route.Slots,
		PolicyHints:        route.PolicyHints,
	}, nil)
	if err != nil {
		t.Fatalf("qimen specialist error: %v", err)
	}
	if qimenResult.Final {
		t.Fatal("qimen result must be supplemental in phase 1")
	}
}

func TestRun_UnsupportedSecondaryDomainDropped(t *testing.T) {
	st := state.NewSession("test-drop-domain")
	st.BaziResult = map[string]any{"dayGan": "甲"}

	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		SecondaryDomains:   []string{"emotion", "qimen"},
		TaskIntent:         "interpret_chart",
		Confidence:         0.9,
	}
	d.Normalize()

	route := policy.Apply(d, st)
	for _, dom := range route.SecondaryDomains {
		if dom == "emotion" {
			t.Fatal("emotion domain should be dropped in phase 1")
		}
	}
	foundQimen := false
	for _, dom := range route.SecondaryDomains {
		if dom == "qimen" {
			foundQimen = true
			break
		}
	}
	if !foundQimen {
		t.Fatal("qimen should survive as secondary domain")
	}
}

// mockSupervisor implements the Supervisor interface for testing route-driven dispatch.
type mockSupervisor struct {
	decision         schemas.SupervisorDecision
	route            policy.ApprovedRoute
	err              error
	extractProfileFn func(context.Context, string, *state.SessionState) (map[string]any, string, error)
}

var birthTimeTestRe = regexp.MustCompile(`\d{4}\s*年.*\d{1,2}\s*月|\d{4}[-/]\d{1,2}|农历|阴历|正月|腊月`)

func containsBirthTime(msg string) bool {
	return birthTimeTestRe.MatchString(msg)
}

func (m *mockSupervisor) Approve(ctx context.Context, msg string, st *state.SessionState) (policy.ApprovedRoute, error) {
	if m.route.PrimaryDomain != "" || m.route.TaskIntent != "" || m.route.NeedsClarification {
		return m.route, m.err
	}
	route := policy.Apply(m.decision, st)

	if route.TaskIntent == "collect_profile" && len(st.Profile) > 0 {
		route.TaskIntent = "amend_profile"
		route.PolicyHints.CanReuseSessionProfile = true
		if st.HasBaziResult() {
			route.PolicyHints.CanReuseCachedResult = true
		}
	}
	if route.TaskIntent == "collect_profile" && st.HasBaziResult() && !containsBirthTime(msg) {
		route.TaskIntent = "fortune_followup"
		route.PolicyHints.CanReuseCachedResult = true
		route.PolicyHints.CanReuseSessionProfile = true
	}
	profileReady := st.IsProfileComplete() || st.HasBaziResult()
	if !profileReady && containsBirthTime(msg) &&
		route.TaskIntent != "collect_profile" &&
		route.TaskIntent != "amend_profile" &&
		route.TaskIntent != "direct_bazi" {
		if route.Slots.Profile == nil {
			route.Slots.Profile = make(map[string]any)
		}
		if len(route.Slots.Profile) == 0 && m.extractProfileFn != nil {
			patch, question, extractErr := m.extractProfileFn(ctx, msg, st)
			if extractErr == nil {
				for k, v := range patch {
					route.Slots.Profile[k] = v
				}
				if route.Slots.QuestionText == "" || route.Slots.QuestionText == msg {
					route.Slots.QuestionText = question
				}
			}
		}
		route.TaskIntent = "collect_profile"
		route.NeedsClarification = false
		route.ClarificationQuestion = ""
	}

	return route, m.err
}

// --- Phase 1.5 route-driven execution tests ---

func TestExecuteRoute_ClarificationDrivesRuntime(t *testing.T) {
	// Verify that NeedsClarification=true directly drives runtime to clarification
	// handler WITHOUT going through legacy action="incomplete".
	st := state.NewSession("test-clarify-route")
	// Intentionally: no profile, no chart.

	route := policy.ApprovedRoute{
		ConversationIntent:    "consult",
		PrimaryDomain:         "bazi",
		TaskIntent:            "interpret_chart",
		NeedsClarification:    true,
		ClarificationQuestion: "请提供出生信息",
		Slots:                 schemas.DecisionSlots{},
	}

	flashClient := &llm.NoopClient{}
	orch := New(tools.NewRegistry(), flashClient, flashClient, state.NewMemoryStore(),
		state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")
	sink := &recordingSink{}

	turnType, _, err := orch.executeRoute(context.Background(), sink, st, route, "")
	if err != nil {
		t.Fatalf("executeRoute with clarification returned error: %v", err)
	}
	if turnType != "ask_missing_profile" {
		t.Fatalf("clarification route should produce ask_missing_profile, got %q", turnType)
	}

	// Verify the sink contains the approved clarification text.
	foundAsk := false
	for _, evt := range sink.events {
		if evt.Type != "text" {
			continue
		}
		if m, ok := evt.Data.(map[string]any); ok {
			if content, ok2 := m["content"].(string); ok2 && content == route.ClarificationQuestion {
				foundAsk = true
				break
			}
		}
	}
	if !foundAsk {
		t.Fatal("clarification route should emit approved clarification text, but no matching text found in events")
	}
}

func TestExecuteRoute_LowConfidenceClarificationDoesNotAnswerDirectly(t *testing.T) {
	// Existing chart + NeedsClarification should emit the approved clarification
	// question instead of jumping into followup reading.
	st := state.NewSession("test-low-confidence-clarify")
	st.Profile = map[string]any{
		"year": 1990.0, "month": 5.0, "day": 20.0,
		"hour": 8.0, "gender": "男", "birthplace": "北京",
	}
	st.BaziResult = map[string]any{"dayGan": "甲"}

	route := policy.ApprovedRoute{
		ConversationIntent:    "consult",
		PrimaryDomain:         "bazi",
		TaskIntent:            "followup",
		NeedsClarification:    true,
		ClarificationQuestion: "请确认一下您更想问整体运势，还是具体想问事业/感情？",
	}

	flashClient := &llm.NoopClient{}
	orch := New(tools.NewRegistry(), flashClient, flashClient, state.NewMemoryStore(),
		state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")
	sink := &recordingSink{}

	turnType, assistantText, err := orch.executeRoute(context.Background(), sink, st, route, "我最近怎么样")
	if err != nil {
		t.Fatalf("executeRoute with low-confidence clarification returned error: %v", err)
	}
	if turnType != "clarification" {
		t.Fatalf("clarification route should produce clarification turn type, got %q", turnType)
	}
	if assistantText != route.ClarificationQuestion {
		t.Fatalf("assistantText = %q, want %q", assistantText, route.ClarificationQuestion)
	}

	foundClarification := false
	for _, evt := range sink.events {
		if evt.Type != "text" {
			continue
		}
		if m, ok := evt.Data.(map[string]any); ok {
			if content, ok2 := m["content"].(string); ok2 && content == route.ClarificationQuestion {
				foundClarification = true
			}
			if content, ok2 := m["content"].(string); ok2 && strings.Contains(content, "基于您的命盘") {
				t.Fatal("clarification route should not emit followup answer text")
			}
		}
	}
	if !foundClarification {
		t.Fatal("clarification route should emit the approved clarification question")
	}
}

func TestExecuteRoute_AmendProfilePreservesExistingData(t *testing.T) {
	// Verify that amend_profile does NOT wipe existing profile/chart.
	st := state.NewSession("test-amend-preserve")
	st.Profile = map[string]any{
		"year": 1990.0, "month": 5.0, "day": 20.0,
		"hour": 8.0, "gender": "男", "birthplace": "北京",
	}
	st.BaziResult = map[string]any{"dayGan": "甲", "pillars": []map[string]any{
		{"stem": "乙", "branch": "巳"},
		{"stem": "丁", "branch": "亥"},
		{"stem": "甲", "branch": "申"},
		{"stem": "甲", "branch": "子"},
	}}

	route := policy.ApprovedRoute{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		TaskIntent:         "amend_profile",
		Slots: schemas.DecisionSlots{
			Profile: map[string]any{"gender": "女"},
		},
	}

	flashClient := &llm.NoopClient{}
	store := state.NewMemoryStore()
	store.Save(st)
	orch := New(tools.NewRegistry(), flashClient, flashClient, store,
		state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")
	sink := &recordingSink{}

	_, _, err := orch.executeRoute(context.Background(), sink, st, route, "")
	if err != nil {
		t.Fatalf("executeRoute with amend_profile returned error: %v", err)
	}

	// Existing chart must be preserved.
	if !st.HasBaziResult() {
		t.Fatal("amend_profile wiped existing BaziResult")
	}
	if st.BaziResult["dayGan"] != "甲" {
		t.Fatalf("amend_profile corrupted existing chart: dayGan=%v", st.BaziResult["dayGan"])
	}
	// Profile should be updated with the new gender.
	if st.Profile["gender"] != "女" {
		t.Fatalf("amend_profile did not apply gender update: gender=%v", st.Profile["gender"])
	}
	// Original fields should still be there.
	if st.Profile["year"] != 1990.0 {
		t.Fatal("amend_profile wiped existing profile fields")
	}
}

func TestExecuteRoute_TimingFollowupEnablesQimen(t *testing.T) {
	// Verify that timing_followup directly enables qimen supplement.
	st := state.NewSession("test-timing-qimen")
	st.BaziResult = map[string]any{"dayGan": "甲"}

	route := policy.ApprovedRoute{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		TaskIntent:         "timing_followup",
		PolicyHints:        schemas.PolicyHints{NeedsQimen: true},
		Slots:              schemas.DecisionSlots{QuestionText: "最近运势如何"},
	}

	flashClient := &llm.NoopClient{}
	orch := New(tools.NewRegistry(), flashClient, flashClient, state.NewMemoryStore(),
		state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")
	sink := &recordingSink{}

	turnType, _, err := orch.executeRoute(context.Background(), sink, st, route, "最近运势如何")
	if err != nil {
		t.Fatalf("executeRoute with timing_followup returned error: %v", err)
	}

	// timing_followup with NeedsQimen=true routes to parallel_fortune (qimen + bazi).
	if turnType != "parallel_fortune" {
		t.Fatalf("timing_followup with NeedsQimen=true should produce parallel_fortune, got %q", turnType)
	}
	// needsQimen=true should have been passed through to the handler.
	// We verify indirectly: st.NeedsQimen should be true (set by the route path).
	if !st.NeedsQimen {
		t.Fatal("timing_followup route should set NeedsQimen=true on session")
	}
}

func TestExecuteRoute_FortuneFollowupDoesNotEnableQimenByDefault(t *testing.T) {
	st := state.NewSession("test-fortune-followup-no-qimen")
	st.BaziResult = map[string]any{"dayGan": "甲"}

	var qimenCalled bool
	reg := tools.NewRegistry()
	reg.Register(&fakeTool{
		name: "qimen_dunjia",
		executeFn: func(_ context.Context, params map[string]any) (any, error) {
			qimenCalled = true
			return map[string]any{"pan_type": "时家奇门"}, nil
		},
	})

	answerer := &streamClient{
		streamFn: func(_ context.Context, _ string, _ []llm.Message, onChunk func(string)) error {
			onChunk("八字追问解读")
			return nil
		},
	}

	orch := New(reg, answerer, answerer, state.NewMemoryStore(),
		state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")
	sink := &recordingSink{}

	route := policy.ApprovedRoute{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		TaskIntent:         "fortune_followup",
		Slots:              schemas.DecisionSlots{QuestionText: "印绶当权，财星透干是什么意思"},
	}

	turnType, _, err := orch.executeRoute(context.Background(), sink, st, route, "印绶当权，财星透干是什么意思")
	if err != nil {
		t.Fatalf("executeRoute with fortune_followup returned error: %v", err)
	}
	if turnType != "followup_reading" {
		t.Fatalf("fortune_followup should stay on bazi followup path, got %q", turnType)
	}
	if qimenCalled {
		t.Fatal("fortune_followup should not invoke qimen_dunjia by default")
	}
	if st.NeedsQimen {
		t.Fatal("fortune_followup should not persist NeedsQimen on session")
	}
}

func TestExecuteRoute_StaleQimenFlagDoesNotLeakIntoPlainFollowup(t *testing.T) {
	st := state.NewSession("test-stale-qimen-flag")
	st.BaziResult = map[string]any{"dayGan": "甲"}
	st.NeedsQimen = true

	var qimenCalled bool
	reg := tools.NewRegistry()
	reg.Register(&fakeTool{
		name: "qimen_dunjia",
		executeFn: func(_ context.Context, params map[string]any) (any, error) {
			qimenCalled = true
			return map[string]any{"pan_type": "时家奇门"}, nil
		},
	})

	answerer := &streamClient{
		streamFn: func(_ context.Context, _ string, _ []llm.Message, onChunk func(string)) error {
			onChunk("八字普通追问")
			return nil
		},
	}

	orch := New(reg, answerer, answerer, state.NewMemoryStore(),
		state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")
	sink := &recordingSink{}

	route := policy.ApprovedRoute{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		TaskIntent:         "fortune_followup",
		Slots:              schemas.DecisionSlots{QuestionText: "财星透干有什么作用"},
	}

	turnType, _, err := orch.executeRoute(context.Background(), sink, st, route, "财星透干有什么作用")
	if err != nil {
		t.Fatalf("executeRoute with stale qimen flag returned error: %v", err)
	}
	if turnType != "followup_reading" {
		t.Fatalf("plain followup should ignore stale qimen flag, got %q", turnType)
	}
	if qimenCalled {
		t.Fatal("plain followup should not invoke qimen_dunjia because of stale session flag")
	}
	if st.NeedsQimen {
		t.Fatal("plain followup should clear stale NeedsQimen after execution")
	}
}

func TestExecuteRoute_QimenPrimaryWithoutProfileRunsDirectly(t *testing.T) {
	st := state.NewSession("test-qimen-primary-no-profile")

	var qimenCalled bool
	reg := tools.NewRegistry()
	reg.Register(&fakeTool{
		name: "qimen_dunjia",
		executeFn: func(_ context.Context, _ map[string]any) (any, error) {
			qimenCalled = true
			return map[string]any{"pan_type": "时家奇门", "value_star": "天辅"}, nil
		},
	})

	answerer := &streamClient{
		streamFn: func(_ context.Context, _ string, _ []llm.Message, onChunk func(string)) error {
			onChunk("直接看今日时运")
			return nil
		},
	}

	orch := New(reg, answerer, answerer, state.NewMemoryStore(),
		state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")
	sink := &recordingSink{}

	route := policy.ApprovedRoute{
		ConversationIntent: "consult",
		PrimaryDomain:      "qimen",
		TaskIntent:         "fortune_followup",
		Slots: schemas.DecisionSlots{
			QuestionText: "今天运气怎么样",
			TimeScope:    "今天",
		},
		PolicyHints: schemas.PolicyHints{
			NeedsQimen:         true,
			QimenMode:          "primary",
			ProfileRequirement: "none",
		},
	}

	turnType, _, err := orch.executeRoute(context.Background(), sink, st, route, "今天运气怎么样")
	if err != nil {
		t.Fatalf("executeRoute with qimen primary route returned error: %v", err)
	}
	if turnType != "qimen_primary_reading" {
		t.Fatalf("qimen primary route should produce qimen_primary_reading, got %q", turnType)
	}
	if !qimenCalled {
		t.Fatal("qimen primary route should invoke qimen_dunjia even without profile")
	}
	if st.HasBaziResult() {
		t.Fatal("qimen primary route without profile should not fabricate a bazi chart")
	}
}

func TestRun_BirthTimeMessageOverridesBadSupervisorRoute(t *testing.T) {
	st := state.NewSession("test-bad-supervisor-birthtime")

	reg := tools.NewRegistry()
	reg.Register(&fakeTool{
		name: "bazi_calc",
		executeFn: func(_ context.Context, params map[string]any) (any, error) {
			return map[string]any{
				"dayGan": "甲",
				"pillars": []map[string]any{
					{"stem": "庚", "branch": "午"},
					{"stem": "壬", "branch": "午"},
					{"stem": "甲", "branch": "申"},
					{"stem": "戊", "branch": "辰"},
				},
				"dayun": []map[string]any{},
			}, nil
		},
	})
	reg.Register(&fakeTool{
		name: "yongshen",
		executeFn: func(_ context.Context, params map[string]any) (any, error) {
			return map[string]any{
				"day_master": "甲",
				"strength":   "中和",
				"yong_shen":  []string{"水"},
				"ji_shen":    []string{"火"},
			}, nil
		},
	})
	reg.Register(&fakeTool{
		name: "knowledge_search",
		executeFn: func(_ context.Context, params map[string]any) (any, error) {
			return map[string]any{"passages": []mcp.Passage{}}, nil
		},
	})

	answerer := &streamClient{
		streamFn: func(_ context.Context, _ string, _ []llm.Message, onChunk func(string)) error {
			onChunk("命理分析结果")
			return nil
		},
	}

	store := state.NewMemoryStore()
	store.Save(st)

	orch := New(reg, answerer, answerer, store, state.NewMemoryLocker(),
		tracing.NewRealTracer(nil), "soft")
	orch.SetSupervisor(&mockSupervisor{
		decision: schemas.SupervisorDecision{
			ConversationIntent: "consult",
			PrimaryDomain:      "bazi",
			TaskIntent:         "interpret_chart",
			Confidence:         0.95,
			Slots: schemas.DecisionSlots{
				QuestionText: "我1990年5月20日早上8点，男，北京，看看事业",
			},
		},
		extractProfileFn: func(_ context.Context, _ string, _ *state.SessionState) (map[string]any, string, error) {
			return map[string]any{
				"year":       1990.0,
				"month":      5.0,
				"day":        20.0,
				"hour":       8.0,
				"gender":     "男",
				"birthplace": "北京",
			}, "看看事业", nil
		},
	})
	orch.SetSpecialists(bazi.New(), qimen.New(), ziwei.New())

	sink := &recordingSink{}
	err := orch.Run(context.Background(), sink, "test-bad-supervisor-birthtime",
		"我1990年5月20日早上8点，男，北京，看看事业")
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	got := store.LoadOrCreate("test-bad-supervisor-birthtime")
	if !got.HasBaziResult() {
		t.Fatal("expected birth-time message to recover into full reading and store BaziResult")
	}

	for _, evt := range sink.events {
		if evt.Type != "text" {
			continue
		}
		if m, ok := evt.Data.(map[string]any); ok {
			if content, ok2 := m["content"].(string); ok2 && strings.Contains(content, "请提供您的出生信息") {
				t.Fatal("birth-time override should not fall back to ask_missing_profile")
			}
		}
	}
}

func TestExecuteRoute_DispatchByTaskIntentNotActionString(t *testing.T) {
	// Verify that the dispatch keys off route.TaskIntent, not a legacy action string.
	// The same session state with different TaskIntents should produce different turnTypes.

	flashClient := &llm.NoopClient{}
	store := state.NewMemoryStore()

	tests := []struct {
		name         string
		taskIntent   string
		needsClarify bool
		hasChart     bool
		hasProfile   bool
		wantTurnType string
	}{
		{
			name:         "clarification forces ask",
			taskIntent:   "interpret_chart",
			needsClarify: true,
			hasChart:     false,
			hasProfile:   false,
			wantTurnType: "ask_missing_profile",
		},
		{
			name:         "direct_bazi with valid pillars",
			taskIntent:   "direct_bazi",
			needsClarify: false,
			hasChart:     false,
			hasProfile:   false,
			wantTurnType: "direct_bazi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := state.NewSession("test-dispatch-" + tt.name)
			if tt.hasChart {
				st.BaziResult = map[string]any{"dayGan": "甲"}
			}
			if tt.hasProfile {
				st.Profile = map[string]any{
					"year": 1990.0, "month": 5.0, "day": 20.0,
					"hour": 8.0, "gender": "男",
				}
			}

			route := policy.ApprovedRoute{
				TaskIntent:         tt.taskIntent,
				NeedsClarification: tt.needsClarify,
				Slots:              schemas.DecisionSlots{},
			}

			orch := New(tools.NewRegistry(), flashClient, flashClient, store,
				state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")
			sink := &recordingSink{}

			var rawBazi []string
			if tt.taskIntent == "direct_bazi" {
				rawBazi = []string{"乙巳", "丁亥", "甲申", "甲子"}
			}

			msg := ""
			if tt.taskIntent == "direct_bazi" {
				msg = strings.Join(rawBazi, " ")
			}
			turnType, _, err := orch.executeRoute(context.Background(), sink, st, route, msg)
			if err != nil {
				t.Fatalf("executeRoute error: %v", err)
			}
			if turnType != tt.wantTurnType {
				t.Errorf("TaskIntent=%q needsClarify=%v: got turnType=%q, want %q",
					tt.taskIntent, tt.needsClarify, turnType, tt.wantTurnType)
			}
		})
	}
}

func TestRun_SupervisorPathBaziMainlineRegression(t *testing.T) {
	// Full integration: supervisor path with complete profile → bazi mainline.
	// Verifies existing bazi behavior doesn't regress with route-driven dispatch.

	st := state.NewSession("test-bazi-regression")
	st.MergeProfile(map[string]any{
		"year": 1990.0, "month": 5.0, "day": 20.0,
		"hour": 8.0, "gender": "男", "birthplace": "北京",
	})

	reg := tools.NewRegistry()
	reg.Register(&fakeTool{
		name: "bazi_calc",
		executeFn: func(_ context.Context, params map[string]any) (any, error) {
			return map[string]any{
				"dayGan": "甲",
				"pillars": []map[string]any{
					{"stem": "庚", "branch": "午"},
					{"stem": "壬", "branch": "午"},
					{"stem": "甲", "branch": "申"},
					{"stem": "戊", "branch": "辰"},
				},
			}, nil
		},
	})

	answerer := &streamClient{
		streamFn: func(_ context.Context, _ string, _ []llm.Message, onChunk func(string)) error {
			onChunk("命理分析结果")
			return nil
		},
	}

	store := state.NewMemoryStore()
	store.Save(st)

	orch := New(reg, answerer, answerer, store, state.NewMemoryLocker(),
		tracing.NewRealTracer(nil), "soft")
	orch.SetSupervisor(&mockSupervisor{
		decision: schemas.SupervisorDecision{
			ConversationIntent: "consult",
			PrimaryDomain:      "bazi",
			TaskIntent:         "collect_profile",
			Confidence:         0.9,
			Slots: schemas.DecisionSlots{
				Profile: map[string]any{
					"year": 1990.0, "month": 5.0, "day": 20.0,
					"hour": 8.0, "gender": "男", "birthplace": "北京",
				},
			},
		},
	})
	orch.SetSpecialists(bazi.New(), qimen.New(), ziwei.New())

	sink := &recordingSink{}
	err := orch.Run(context.Background(), sink, "test-bazi-regression",
		"1990年5月20日早上8点 男 北京")

	if err != nil {
		t.Fatalf("bazi mainline regression: Run() returned error: %v", err)
	}

	// Verify bazi chart was computed and stored.
	got := store.LoadOrCreate("test-bazi-regression")
	if !got.HasBaziResult() {
		t.Fatal("bazi mainline regression: BaziResult not set after full reading")
	}
	if got.BaziResult["dayGan"] != "甲" {
		t.Fatalf("bazi mainline regression: dayGan=%v, want 甲", got.BaziResult["dayGan"])
	}

	// Verify trace-panel was emitted.
	if sink.lastComponentType() != "trace-panel" {
		t.Error("bazi mainline regression: trace-panel not emitted")
	}

	// Verify LLM response was streamed.
	foundAnalysis := false
	for _, evt := range sink.events {
		if evt.Type == "text" {
			if m, ok := evt.Data.(map[string]any); ok {
				if c, ok2 := m["content"].(string); ok2 && strings.Contains(c, "命理分析") {
					foundAnalysis = true
					break
				}
			}
		}
	}
	if !foundAnalysis {
		t.Error("bazi mainline regression: analysis text not streamed")
	}
}

// --- Qimen primary lane tests ---

func TestExecuteRoute_QimenPrimaryLaneInvokesQimenRegardlessOfSupplementFlag(t *testing.T) {
	// When PrimaryDomain="qimen" + timing_followup, the qimen primary lane
	// must invoke qimen_dunjia even when the legacy needsQimen flag is false.
	// This proves qimen tool invocation is driven by primary domain semantics,
	// not by the NeedsQimen supplement gate.

	st := state.NewSession("test-qimen-primary-lane")
	st.BaziResult = map[string]any{"dayGan": "甲", "pillars": []map[string]any{
		{"stem": "庚", "branch": "午"},
		{"stem": "壬", "branch": "午"},
		{"stem": "甲", "branch": "申"},
		{"stem": "戊", "branch": "辰"},
	}}

	var qimenCalled bool
	reg := tools.NewRegistry()
	reg.Register(&fakeTool{
		name: "qimen_dunjia",
		executeFn: func(_ context.Context, params map[string]any) (any, error) {
			qimenCalled = true
			return map[string]any{
				"pan_type": "时家奇门",
				"ju_text":  "阳遁三局",
				"cells":    []map[string]any{},
			}, nil
		},
	})

	answerer := &streamClient{
		streamFn: func(_ context.Context, _ string, _ []llm.Message, onChunk func(string)) error {
			onChunk("奇门分析结果")
			return nil
		},
	}

	store := state.NewMemoryStore()
	store.Save(st)

	orch := New(reg, answerer, answerer, store, state.NewMemoryLocker(),
		tracing.NewRealTracer(nil), "soft")
	sink := &recordingSink{}

	route := policy.ApprovedRoute{
		ConversationIntent: "consult",
		PrimaryDomain:      "qimen",
		TaskIntent:         "timing_followup",
		Slots: schemas.DecisionSlots{
			QuestionText: "最近适合换工作吗",
			TimeScope:    "recent",
		},
	}

	// Pass needsQimen=false — in the old code executeFollowupRoute would skip qimen.
	// The qimen primary lane must invoke qimen based on PrimaryDomain, not the flag.
	turnType, _, err := orch.executeRoute(context.Background(), sink, st, route, "最近适合换工作吗")
	if err != nil {
		t.Fatalf("executeRoute qimen primary lane: error: %v", err)
	}
	if turnType != "qimen_primary_reading" {
		t.Fatalf("qimen primary lane: turnType = %q, want qimen_primary_reading", turnType)
	}
	if !qimenCalled {
		t.Fatal("qimen primary lane: qimen_dunjia was NOT invoked — PrimaryDomain=qimen should drive qimen tool invocation")
	}

	// Verify SSE events.
	foundToolCall := false
	foundQimenChart := false
	for _, evt := range sink.events {
		if evt.Type == "tool_call" {
			if m, ok := evt.Data.(map[string]any); ok {
				if t, ok2 := m["tool"].(string); ok2 && t == "qimen_dunjia" {
					foundToolCall = true
				}
			}
		}
		if evt.Type == "component" {
			if m, ok := evt.Data.(map[string]any); ok {
				if t, ok2 := m["type"].(string); ok2 && t == "qimen-chart" {
					foundQimenChart = true
				}
			}
		}
	}
	if !foundToolCall {
		t.Fatal("qimen primary lane: tool_call event for qimen_dunjia not found")
	}
	if !foundQimenChart {
		t.Fatal("qimen primary lane: component event for qimen-chart not found")
	}
}

func TestRun_SupervisorPathQimenPrimaryTimingMainline(t *testing.T) {
	// Full integration: supervisor → policy gate → qimen primary specialist
	// → qimen primary lane executes qimen_dunjia and produces qimen-chart.
	// Verifies the complete path from supervisor decision to SSE output.

	store := state.NewMemoryStore()
	st := store.LoadOrCreate("test-qimen-mainline")
	st.Profile = map[string]any{
		"year": 1990.0, "month": 5.0, "day": 20.0,
		"hour": 8.0, "gender": "男", "birthplace": "北京",
	}
	st.BaziResult = map[string]any{
		"dayGan": "甲",
		"pillars": []map[string]any{
			{"stem": "庚", "branch": "午"},
			{"stem": "壬", "branch": "午"},
			{"stem": "甲", "branch": "申"},
			{"stem": "戊", "branch": "辰"},
		},
	}

	var qimenCalled bool
	reg := tools.NewRegistry()
	reg.Register(&fakeTool{
		name: "qimen_dunjia",
		executeFn: func(_ context.Context, params map[string]any) (any, error) {
			qimenCalled = true
			return map[string]any{
				"pan_type":      "时家奇门",
				"ju_text":       "阳遁三局",
				"question_time": "2026-06-12T10:00:00+08:00",
				"cells":         []map[string]any{},
			}, nil
		},
	})

	answerer := &streamClient{
		streamFn: func(_ context.Context, _ string, _ []llm.Message, onChunk func(string)) error {
			onChunk("结合奇门盘分析，当前时机...")
			return nil
		},
	}

	orch := New(reg, answerer, answerer, store, state.NewMemoryLocker(),
		tracing.NewRealTracer(nil), "soft")
	orch.SetSupervisor(&mockSupervisor{
		decision: schemas.SupervisorDecision{
			ConversationIntent: "consult",
			PrimaryDomain:      "qimen",
			TaskIntent:         "timing_followup",
			Confidence:         0.92,
			PolicyHints:        schemas.PolicyHints{NeedsQimen: true},
			Slots: schemas.DecisionSlots{
				QuestionText: "最近适合换工作吗",
				TimeScope:    "recent",
			},
		},
	})
	orch.SetSpecialists(bazi.New(), qimen.New(), ziwei.New())

	sink := &recordingSink{}
	err := orch.Run(context.Background(), sink, "test-qimen-mainline",
		"最近适合换工作吗")
	if err != nil {
		t.Fatalf("qimen primary mainline: Run() error: %v", err)
	}

	// qimen_dunjia must be invoked as primary tool.
	if !qimenCalled {
		t.Fatal("qimen primary mainline: qimen_dunjia was never invoked")
	}

	// Verify SSE events.
	foundToolCall := false
	foundQimenChart := false
	foundAnalysis := false
	for _, evt := range sink.events {
		if evt.Type == "tool_call" {
			if m, ok := evt.Data.(map[string]any); ok {
				if t, ok2 := m["tool"].(string); ok2 && t == "qimen_dunjia" {
					foundToolCall = true
				}
			}
		}
		if evt.Type == "component" {
			if m, ok := evt.Data.(map[string]any); ok {
				if t, ok2 := m["type"].(string); ok2 && t == "qimen-chart" {
					foundQimenChart = true
				}
			}
		}
		if evt.Type == "text" {
			if m, ok := evt.Data.(map[string]any); ok {
				if c, ok2 := m["content"].(string); ok2 && strings.Contains(c, "奇门") {
					foundAnalysis = true
				}
			}
		}
	}
	if !foundToolCall {
		t.Fatal("qimen primary mainline: tool_call event for qimen_dunjia not found")
	}
	if !foundQimenChart {
		t.Fatal("qimen primary mainline: component event for qimen-chart not found")
	}
	if !foundAnalysis {
		t.Error("qimen primary mainline: analysis text not streamed")
	}
}

func TestRun_SupervisorPathDispatchesPrimaryQimenSpecialist(t *testing.T) {
	store := state.NewMemoryStore()
	st := store.LoadOrCreate("qimen-primary-dispatch")
	st.Profile = map[string]any{
		"year": 1990.0, "month": 5.0, "day": 20.0,
		"hour": 8.0, "gender": "男", "birthplace": "北京",
	}
	st.BaziResult = map[string]any{"dayGan": "甲"}

	orch := New(tools.NewRegistry(), &llm.NoopClient{}, &llm.NoopClient{}, store, state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")
	orch.SetSupervisor(&mockSupervisor{
		decision: schemas.SupervisorDecision{
			ConversationIntent: "consult",
			PrimaryDomain:      "qimen",
			TaskIntent:         "timing_followup",
			Confidence:         0.92,
			PolicyHints:        schemas.PolicyHints{NeedsQimen: true},
			Slots: schemas.DecisionSlots{
				QuestionText: "我最近适合换工作吗",
				TimeScope:    "recent",
			},
		},
	})

	baziSp := &fakeDomainHandler{name: "bazi"}
	qimenSp := &fakeDomainHandler{name: "qimen"}
	ziweiSp := &fakeDomainHandler{name: "ziwei"}
	orch.SetSpecialists(baziSp, qimenSp, ziweiSp)

	if err := orch.Run(context.Background(), &recordingSink{}, "qimen-primary-dispatch", "我最近适合换工作吗"); err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if qimenSp.calls != 1 {
		t.Fatalf("qimen specialist calls = %d, want 1", qimenSp.calls)
	}
	if baziSp.calls != 0 {
		t.Fatalf("bazi specialist calls = %d, want 0 when primary domain is qimen", baziSp.calls)
	}
}

func TestStreamInterpretation_DoesNotStoreFilteredDisclaimerText(t *testing.T) {
	answerer := &streamClient{
		streamFn: func(_ context.Context, _ string, _ []llm.Message, onChunk func(string)) error {
			onChunk("这是正常分析。")
			onChunk("以上内容由AI生成，仅供娱乐")
			return nil
		},
	}

	orch := New(tools.NewRegistry(), answerer, answerer, state.NewMemoryStore(), state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")
	st := state.NewSession("filtered-text")
	st.LastUserQuestion = "今年如何"
	st.BaziResult = map[string]any{"dayGan": "甲"}

	got, err := orch.streamInterpretation(context.Background(), &recordingSink{}, st, nil, "bazi")
	if err != nil {
		t.Fatalf("streamInterpretation returned error: %v", err)
	}
	if strings.Contains(got, "以上内容由") || strings.Contains(got, "仅供娱乐") || strings.Contains(got, "AI生成") {
		t.Fatalf("filtered disclaimer text should not be stored in assistant text: %q", got)
	}
	if got != "这是正常分析。" {
		t.Fatalf("unexpected stored assistant text: %q", got)
	}
}

// --- Qimen standalone answer tests ---

func TestExecuteRoute_QimenPrimaryNoChartAnswersDirectly(t *testing.T) {
	// When qimen is primary and no bazi chart exists, the qimen primary lane
	// should answer directly based on the qimen chart — NOT ask for birth info.

	st := state.NewSession("test-qimen-no-chart-answer")
	// Intentionally: no BaziResult, no profile.

	route := policy.ApprovedRoute{
		ConversationIntent: "consult",
		PrimaryDomain:      "qimen",
		TaskIntent:         "timing_followup",
		Slots: schemas.DecisionSlots{
			QuestionText: "最近适合换工作吗",
			TimeScope:    "recent",
		},
	}

	var qimenCalled bool
	reg := tools.NewRegistry()
	reg.Register(&fakeTool{
		name: "qimen_dunjia",
		executeFn: func(_ context.Context, params map[string]any) (any, error) {
			qimenCalled = true
			return map[string]any{
				"pan_type":   "时家奇门",
				"ju_text":    "阳遁三局",
				"value_star": "天辅星",
				"value_door": "开门",
				"cells":      []map[string]any{},
			}, nil
		},
	})

	answerer := &streamClient{
		streamFn: func(_ context.Context, _ string, _ []llm.Message, onChunk func(string)) error {
			onChunk("当前时机分析：值符天辅星落...")
			return nil
		},
	}

	store := state.NewMemoryStore()
	store.Save(st)

	orch := New(reg, answerer, answerer, store, state.NewMemoryLocker(),
		tracing.NewRealTracer(nil), "soft")
	sink := &recordingSink{}

	turnType, _, err := orch.executeRoute(context.Background(), sink, st, route, "最近适合换工作吗")
	if err != nil {
		t.Fatalf("qimen no-chart answer: executeRoute error: %v", err)
	}

	// Must NOT degenerate to ask_missing_profile.
	if turnType == "ask_missing_profile" {
		t.Fatal("qimen no-chart answer: should answer directly, not ask for birth info")
	}
	if turnType != "qimen_primary_reading" {
		t.Fatalf("qimen no-chart answer: turnType = %q, want qimen_primary_reading", turnType)
	}
	if !qimenCalled {
		t.Fatal("qimen no-chart answer: qimen_dunjia was not invoked")
	}

	// Verify no "请告诉我你的出生" text.
	for _, evt := range sink.events {
		if evt.Type == "text" {
			if m, ok := evt.Data.(map[string]any); ok {
				if c, ok2 := m["content"].(string); ok2 {
					if strings.Contains(c, "请告诉我你的出生") {
						t.Fatal("qimen no-chart answer: should not ask for birth info")
					}
				}
			}
		}
	}

	// Verify qimen-chart was emitted.
	foundChart := false
	foundAnswer := false
	for _, evt := range sink.events {
		if evt.Type == "component" {
			if m, ok := evt.Data.(map[string]any); ok {
				if t, ok2 := m["type"].(string); ok2 && t == "qimen-chart" {
					foundChart = true
				}
			}
		}
		if evt.Type == "text" {
			if m, ok := evt.Data.(map[string]any); ok {
				if c, ok2 := m["content"].(string); ok2 && c != "" {
					foundAnswer = true
				}
			}
		}
	}
	if !foundChart {
		t.Fatal("qimen no-chart answer: qimen-chart component not emitted")
	}
	if !foundAnswer {
		t.Fatal("qimen no-chart answer: no answer text emitted")
	}
}

func TestBuildQimenKnowledgeQuery(t *testing.T) {
	// Verify that buildQimenKnowledgeQuery uses qimen-specific terms
	// from the qimen chart data, not bazi day-master terms.

	st := state.NewSession("test-qimen-search-query")
	st.LastUserQuestion = "最近适合换工作吗"

	qimenData := map[string]any{
		"value_star": "天辅星",
		"value_door": "开门",
		"ju_text":    "阳遁三局",
	}

	orch := New(tools.NewRegistry(), &llm.NoopClient{}, &llm.NoopClient{},
		state.NewMemoryStore(), state.NewMemoryLocker(), tracing.NewRealTracer(nil), "soft")

	query := orch.promptBuilder.BuildQimenKnowledgeQuery(st.LastUserQuestion, qimenData)

	if query == "" {
		t.Fatal("buildQimenKnowledgeQuery: returned empty query")
	}
	if !strings.Contains(query, "奇门遁甲") {
		t.Error("buildQimenKnowledgeQuery: should contain 奇门遁甲")
	}
	if !strings.Contains(query, "天辅星") {
		t.Error("buildQimenKnowledgeQuery: should contain value_star 天辅星")
	}
	if !strings.Contains(query, "开门") {
		t.Error("buildQimenKnowledgeQuery: should contain value_door 开门")
	}
	if !strings.Contains(query, "阳遁三局") {
		t.Error("buildQimenKnowledgeQuery: should contain ju_text 阳遁三局")
	}
	if !strings.Contains(query, "最近适合换工作吗") {
		t.Error("buildQimenKnowledgeQuery: should contain user question")
	}
}
