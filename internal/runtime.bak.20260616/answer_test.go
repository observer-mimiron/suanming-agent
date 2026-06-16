package runtime

import (
	"context"
	"errors"
	"testing"

	einocallbacks "github.com/cloudwego/eino/callbacks"
	"github.com/wikiglobal/suanming-agent/internal/mcp"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tools"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

type answerTestTool struct {
	name      string
	executeFn func(ctx context.Context, params map[string]any) (any, error)
}

func (t *answerTestTool) Name() string        { return t.name }
func (t *answerTestTool) Description() string { return t.name }
func (t *answerTestTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	return t.executeFn(ctx, params)
}

type answerTestSink struct {
	events []Event
}

func (s *answerTestSink) Emit(_ context.Context, evt Event) error {
	s.events = append(s.events, evt)
	return nil
}

func TestRunKnowledgeSearch_EinoRetrieverTraceIncludesTopK(t *testing.T) {
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(tracing.NewEinoTraceCallbackHandler())

	reg := tools.NewRegistry()
	reg.Register(&answerTestTool{
		name: "knowledge_search",
		executeFn: func(_ context.Context, _ map[string]any) (any, error) {
			return map[string]any{
				"passages": []mcp.Passage{{Source: "滴天髓", Content: "财官印食"}},
			}, nil
		},
	})

	executor := NewExecutor(reg, nil, tracing.NewRealTracer(nil), "soft")
	st := state.NewSession("s1")
	st.LastUserQuestion = "看看事业"
	sink := &answerTestSink{}

	ctx, trace := tracing.NewRealTracer(nil).StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	passages := executor.runKnowledgeSearch(ctx, sink, st, "bazi")
	if len(passages) != 1 {
		t.Fatalf("passages len = %d, want 1", len(passages))
	}

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}

	var count int
	for _, span := range tr.Spans {
		if span.Name != "knowledge_search" || span.Kind != tracing.KindRetriever {
			continue
		}
		count++
		if got := span.Attributes["top_k"]; got != 5 {
			t.Fatalf("top_k attr = %v, want 5", got)
		}
	}
	if count != 1 {
		t.Fatalf("knowledge_search span count = %d, want 1", count)
	}
}

func TestRunKnowledgeSearch_MissingToolStillDegrades(t *testing.T) {
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(tracing.NewEinoTraceCallbackHandler())

	executor := NewExecutor(tools.NewRegistry(), nil, tracing.NewRealTracer(nil), "soft")
	st := state.NewSession("s2")
	st.LastUserQuestion = "看看事业"
	sink := &answerTestSink{}

	ctx, trace := tracing.NewRealTracer(nil).StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	passages := executor.runKnowledgeSearch(ctx, sink, st, "bazi")
	if len(passages) != 0 {
		t.Fatalf("passages len = %d, want 0", len(passages))
	}

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}

	var found bool
	for _, span := range tr.Spans {
		if span.Name != "knowledge_search" || span.Kind != tracing.KindRetriever {
			continue
		}
		found = true
		if span.Status != "degraded" {
			t.Fatalf("knowledge_search status = %s, want degraded", span.Status)
		}
	}
	if !found {
		t.Fatal("expected degraded knowledge_search span when tool is missing")
	}
}

func TestRunKnowledgeSearch_ToolErrorStillDegrades(t *testing.T) {
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(tracing.NewEinoTraceCallbackHandler())

	reg := tools.NewRegistry()
	reg.Register(&answerTestTool{
		name: "knowledge_search",
		executeFn: func(_ context.Context, _ map[string]any) (any, error) {
			return nil, errors.New("retriever unavailable")
		},
	})

	executor := NewExecutor(reg, nil, tracing.NewRealTracer(nil), "soft")
	st := state.NewSession("s3")
	st.LastUserQuestion = "看看事业"
	sink := &answerTestSink{}

	ctx, trace := tracing.NewRealTracer(nil).StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	passages := executor.runKnowledgeSearch(ctx, sink, st, "bazi")
	if len(passages) != 0 {
		t.Fatalf("passages len = %d, want 0", len(passages))
	}

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}

	var found bool
	for _, span := range tr.Spans {
		if span.Name != "knowledge_search" || span.Kind != tracing.KindRetriever {
			continue
		}
		found = true
		if span.Status != "degraded" {
			t.Fatalf("knowledge_search status = %s, want degraded", span.Status)
		}
	}
	if !found {
		t.Fatal("expected degraded knowledge_search span when tool execution fails")
	}
}
