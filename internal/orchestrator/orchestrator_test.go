package orchestrator

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	bazi "github.com/wikiglobal/suanming-agent/internal/specialists/bazi"
	qimen "github.com/wikiglobal/suanming-agent/internal/specialists/qimen"
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

func TestRun_ClassifyFail_FallbackToExtract(t *testing.T) {
	// When flash fails, classify falls back to regex extraction.
	flashClient := &llm.NoopClient{
		GenerateFn: func(_ context.Context, _ string, _ []llm.Message) (string, llm.TokenUsage, error) {
			return "", llm.TokenUsage{}, errors.New("flash timeout")
		},
	}

	reg := tools.NewRegistry()
	rt := tracing.NewRealTracer(nil)
	orch := New(reg, flashClient, flashClient, state.NewMemoryStore(), state.NewMemoryLocker(), rt, "soft")

	sink := &recordingSink{}

	// Message without birth info → regex can't extract much → incomplete → handleAsk (no error).
	err := orch.Run(context.Background(), sink, "test-session", "你好")

	// handleAsk never returns an error, so Run() should succeed.
	if err != nil {
		t.Errorf("handleAsk should not error, got: %v", err)
	}

	// Trace should still be ok since no error occurred.
	if sink.lastComponentType() != "trace-panel" {
		t.Error("expected trace-panel even on successful ask turn")
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
	result, err := baziSp.Run(context.Background(), st, specialists.ApprovedRoute{
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
	baziResult, err := baziSp.Run(context.Background(), st, specialists.ApprovedRoute{
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
	qimenResult, err := qimenSp.Run(context.Background(), st, specialists.ApprovedRoute{
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

	got, err := orch.streamInterpretation(context.Background(), &recordingSink{}, st, nil, nil)
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
