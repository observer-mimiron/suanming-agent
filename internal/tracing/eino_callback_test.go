package tracing

import (
	"context"
	"testing"

	einocallbacks "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	einoretriever "github.com/cloudwego/eino/components/retriever"
)

func TestEinoRetrieverCallback_EmitsTraceSpan(t *testing.T) {
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(NewEinoTraceCallbackHandler())

	rt := NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	ctx = einocallbacks.ReuseHandlers(ctx, &einocallbacks.RunInfo{
		Name:      "knowledge_search",
		Type:      "goRuntimeRetriever",
		Component: components.ComponentOfRetriever,
	})
	ctx = einocallbacks.OnStart(ctx, &einoretriever.CallbackInput{
		Query: "事业 运势",
		TopK:  5,
	})
	einocallbacks.OnEnd(ctx, &einoretriever.CallbackOutput{
		Extra: map[string]any{"hits": 2},
	})

	tr := TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}

	var found bool
	for _, span := range tr.Spans {
		if span.Name != "knowledge_search" || span.Kind != KindRetriever {
			continue
		}
		found = true
		if got := span.Attributes["query"]; got != "事业 运势" {
			t.Fatalf("query attr = %v, want 事业 运势", got)
		}
		if got := span.Attributes["top_k"]; got != 5 {
			t.Fatalf("top_k attr = %v, want 5", got)
		}
		if got := span.Attributes["hits"]; got != 2 {
			t.Fatalf("hits attr = %v, want 2", got)
		}
	}

	if !found {
		t.Fatal("expected knowledge_search retriever span emitted from Eino callback")
	}
}
