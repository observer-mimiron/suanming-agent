package tracing

import (
	"context"
	"strings"
	"testing"

	einocallbacks "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	einomodel "github.com/cloudwego/eino/components/model"
	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
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

func TestEinoChatModelCallback_RecordsInputMessageSummary(t *testing.T) {
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(NewEinoTraceCallbackHandler())

	rt := NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	ctx = WithEinoCallbackSpan(ctx, EinoCallbackSpanConfig{
		Name: "bazi_specialist",
		Kind: KindLLM,
	})
	ctx = einocallbacks.ReuseHandlers(ctx, &einocallbacks.RunInfo{
		Name:      "bazi_specialist",
		Type:      "ChatModel",
		Component: components.ComponentOfChatModel,
	})
	ctx = einocallbacks.OnStart(ctx, &einomodel.CallbackInput{
		Messages: []*schema.Message{
			schema.SystemMessage("你是八字专家"),
			schema.UserMessage("1992年12月1日 男 北京"),
		},
	})
	einocallbacks.OnEnd(ctx, &einomodel.CallbackOutput{})

	tr := TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	var found bool
	for _, span := range tr.Spans {
		if span.Name != "bazi_specialist" || span.Kind != KindLLM {
			continue
		}
		found = true
		if got := span.Attributes["input.message_count"]; got != 2 {
			t.Fatalf("input.message_count = %v, want 2", got)
		}
		if got := span.Attributes["input.message_roles"]; got != "system,user" {
			t.Fatalf("input.message_roles = %v, want system,user", got)
		}
		if _, ok := span.Attributes["input.messages.preview"]; ok {
			t.Fatal("input.messages.preview should not be recorded by default")
		}
	}
	if !found {
		t.Fatal("expected bazi_specialist LLM span with input message summary")
	}
}

func TestEinoChatModelCallback_RecordsInputMessagePreviewWhenEnabled(t *testing.T) {
	t.Setenv("TRACE_LLM_INPUT_MESSAGES", "1")
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(NewEinoTraceCallbackHandler())

	rt := NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	ctx = WithEinoCallbackSpan(ctx, EinoCallbackSpanConfig{
		Name: "bazi_specialist",
		Kind: KindLLM,
	})
	ctx = einocallbacks.ReuseHandlers(ctx, &einocallbacks.RunInfo{
		Name:      "bazi_specialist",
		Type:      "ChatModel",
		Component: components.ComponentOfChatModel,
	})
	ctx = einocallbacks.OnStart(ctx, &einomodel.CallbackInput{
		Messages: []*schema.Message{
			schema.SystemMessage("你是八字专家，Authorization: Bearer sk-test"),
			schema.UserMessage("1992年12月1日12点 男 北京"),
		},
	})
	einocallbacks.OnEnd(ctx, &einomodel.CallbackOutput{})

	tr := TraceFromContext(ctx)
	for _, span := range tr.Spans {
		if span.Name != "bazi_specialist" || span.Kind != KindLLM {
			continue
		}
		got, ok := span.Attributes["input.messages.preview"]
		if !ok {
			t.Fatal("input.messages.preview attribute not set on span")
		}
		s, ok := got.(string)
		if !ok {
			t.Fatalf("input.messages.preview type = %T, want string", got)
		}
		if !strings.Contains(s, "1992") {
			t.Fatalf("input.messages.preview = %q, want contains 1992", s)
		}
		if strings.Contains(s, "sk-test") {
			t.Fatalf("input.messages.preview leaked secret: %q", s)
		}
		return
	}
	t.Fatal("expected bazi_specialist LLM span with input.messages.preview")
}
