package tracing

import (
	"context"
	"strings"
	"testing"

	einocallbacks "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	einomodel "github.com/cloudwego/eino/components/model"
	einoretriever "github.com/cloudwego/eino/components/retriever"
	einotool "github.com/cloudwego/eino/components/tool"
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
		if got := span.Attributes["input.value"]; got == nil {
			t.Fatal("input.value should be recorded for Langfuse-readable LLM input")
		}
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

func TestEinoChatModelCallback_RecordsOutputValue(t *testing.T) {
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
			schema.UserMessage("分析这个命盘"),
		},
	})
	einocallbacks.OnEnd(ctx, &einomodel.CallbackOutput{
		Message: schema.AssistantMessage("这是最终解读。", nil),
	})

	tr := TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	for _, span := range tr.Spans {
		if span.Name != "bazi_specialist" || span.Kind != KindLLM {
			continue
		}
		if got := span.Attributes["output.value"]; got != "这是最终解读。" {
			t.Fatalf("output.value = %v, want 这是最终解读。", got)
		}
		return
	}
	t.Fatal("expected bazi_specialist LLM span with output.value")
}

func TestEinoChatModelCallback_UsesRunInfoNameNotCtxCfgName(t *testing.T) {
	// Q5: specialist 的 LLM 调用应使用 RunInfo.Name（如 bazi_specialist），
	// 而不是继承 supervisor ctx 里 cfg.Name（如 adk_supervisor_agent）。
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(NewEinoTraceCallbackHandler())

	rt := NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	// 模拟 supervisor ctx 里设置的 span config（name=supervisor）
	ctx = WithEinoCallbackSpan(ctx, EinoCallbackSpanConfig{
		Name: "adk_supervisor_agent",
		Kind: KindLLM,
	})
	// 但 RunInfo.Name 是 specialist 自己的名字
	ctx = einocallbacks.ReuseHandlers(ctx, &einocallbacks.RunInfo{
		Name:      "bazi_specialist",
		Type:      "ChatModel",
		Component: components.ComponentOfChatModel,
	})
	ctx = einocallbacks.OnStart(ctx, &einomodel.CallbackInput{
		Messages: []*schema.Message{schema.UserMessage("test")},
	})
	einocallbacks.OnEnd(ctx, &einomodel.CallbackOutput{})

	tr := TraceFromContext(ctx)
	for _, span := range tr.Spans {
		if span.Kind != KindLLM {
			continue
		}
		if span.Name != "bazi_specialist" {
			t.Fatalf("span.Name = %q, want %q (RunInfo.Name should override cfg.Name)", span.Name, "bazi_specialist")
		}
		return
	}
	t.Fatal("no LLM span found")
}

func TestEinoChatModelCallback_FallsBackToCfgNameWhenRunInfoIsComponentDefault(t *testing.T) {
	// ADK Runner 内部 ChatModel 不是 graph 节点，RunInfo.Name 退化成 eino 默认组件名 "ChatModel"
	//（= string(info.Component)）。此时 span name 应 fallback 到 cfg.Name（语义化名字），
	// 而不是用无意义的 "ChatModel"。
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(NewEinoTraceCallbackHandler())

	rt := NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	ctx = WithEinoCallbackSpan(ctx, EinoCallbackSpanConfig{
		Name: "supervisor_model",
		Kind: KindLLM,
	})
	// RunInfo.Name 是 eino 默认组件名 "ChatModel"（ADK Runner 内部场景）
	ctx = einocallbacks.ReuseHandlers(ctx, &einocallbacks.RunInfo{
		Name:      "ChatModel",
		Type:      "ChatModel",
		Component: components.ComponentOfChatModel,
	})
	ctx = einocallbacks.OnStart(ctx, &einomodel.CallbackInput{
		Messages: []*schema.Message{schema.UserMessage("test")},
	})
	einocallbacks.OnEnd(ctx, &einomodel.CallbackOutput{})

	tr := TraceFromContext(ctx)
	for _, span := range tr.Spans {
		if span.Kind != KindLLM {
			continue
		}
		if span.Name != "supervisor_model" {
			t.Fatalf("span.Name = %q, want %q (cfg.Name should win when RunInfo.Name is component default)", span.Name, "supervisor_model")
		}
		return
	}
	t.Fatal("no LLM span found")
}

func TestEinoToolCallback_NormalizesSupervisorOutputSpanName(t *testing.T) {
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(NewEinoTraceCallbackHandler())

	rt := NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	ctx = einocallbacks.ReuseHandlers(ctx, &einocallbacks.RunInfo{
		Name:      "output",
		Type:      "Tool",
		Component: components.ComponentOfTool,
	})
	ctx = einocallbacks.OnStart(ctx, &einotool.CallbackInput{
		ArgumentsInJSON: `{"conversation_intent":"consult","primary_domain":"bazi","task_intent":"collect_profile"}`,
	})
	einocallbacks.OnEnd(ctx, &einotool.CallbackOutput{
		Response: `{"primary_domain":"bazi","task_intent":"collect_profile"}`,
	})

	tr := TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	for _, span := range tr.Spans {
		if span.Kind != KindTool {
			continue
		}
		if span.Name != "supervisor_output" {
			t.Fatalf("tool span name = %q, want supervisor_output", span.Name)
		}
		if got := span.Attributes["args"]; got == nil {
			t.Fatal("expected args attribute on normalized supervisor_output span")
		}
		if got := span.Attributes["input.value"]; got != `{"conversation_intent":"consult","primary_domain":"bazi","task_intent":"collect_profile"}` {
			t.Fatalf("input.value = %v, want serialized tool arguments", got)
		}
		if got := span.Attributes["output.value"]; got != `{"primary_domain":"bazi","task_intent":"collect_profile"}` {
			t.Fatalf("output.value = %v, want serialized tool response", got)
		}
		return
	}
	t.Fatal("expected normalized supervisor_output tool span")
}
