package llm

import (
	"context"
	"io"
	"reflect"
	"strings"
	"testing"

	einocallbacks "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

type fakeToolCallingChatModel struct {
	emitCallbacks bool
	generateFn    func(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error)
	streamFn      func(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.StreamReader[*schema.Message], error)
	withToolsFn   func(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error)
}

func (f *fakeToolCallingChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error) {
	if !f.emitCallbacks {
		return f.generateFn(ctx, input, opts...)
	}

	ctx = einocallbacks.EnsureRunInfo(ctx, "fakeToolCallingChatModel", components.ComponentOfChatModel)
	ctx = einocallbacks.OnStart(ctx, &einomodel.CallbackInput{Messages: input})
	msg, err := f.generateFn(ctx, input, opts...)
	if err != nil {
		einocallbacks.OnError(ctx, err)
		return nil, err
	}
	einocallbacks.OnEnd(ctx, &einomodel.CallbackOutput{
		Message:    msg,
		TokenUsage: toCallbackTokenUsage(msg),
	})
	return msg, nil
}

func (f *fakeToolCallingChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
	if !f.emitCallbacks {
		return f.streamFn(ctx, input, opts...)
	}

	ctx = einocallbacks.EnsureRunInfo(ctx, "fakeToolCallingChatModel", components.ComponentOfChatModel)
	ctx = einocallbacks.OnStart(ctx, &einomodel.CallbackInput{Messages: input})
	sr, err := f.streamFn(ctx, input, opts...)
	if err != nil {
		einocallbacks.OnError(ctx, err)
		return nil, err
	}
	_, sr = einocallbacks.OnEndWithStreamOutput(ctx, sr)
	return sr, nil
}

func (f *fakeToolCallingChatModel) WithTools(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
	return f.withToolsFn(tools)
}

func toCallbackTokenUsage(msg *schema.Message) *einomodel.TokenUsage {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return nil
	}
	return &einomodel.TokenUsage{
		PromptTokens:     msg.ResponseMeta.Usage.PromptTokens,
		CompletionTokens: msg.ResponseMeta.Usage.CompletionTokens,
		TotalTokens:      msg.ResponseMeta.Usage.TotalTokens,
	}
}

func TestEinoChatGenerate_ReturnsTextAndUsage(t *testing.T) {
	model := &fakeToolCallingChatModel{}
	model.generateFn = func(_ context.Context, input []*schema.Message, _ ...einomodel.Option) (*schema.Message, error) {
		if got, want := len(input), 2; got != want {
			t.Fatalf("input len = %d, want %d", got, want)
		}
		if input[0].Role != schema.System {
			t.Fatalf("first role = %s, want system", input[0].Role)
		}
		return &schema.Message{
			Role:    schema.Assistant,
			Content: "ok",
			ResponseMeta: &schema.ResponseMeta{
				Usage: &schema.TokenUsage{
					PromptTokens:     11,
					CompletionTokens: 7,
				},
			},
		}, nil
	}
	model.streamFn = func(context.Context, []*schema.Message, ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
		return nil, nil
	}
	model.withToolsFn = func(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
		return model, nil
	}

	chat := NewEinoChat(model)
	text, usage, err := chat.Generate(context.Background(), "system", []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	if text != "ok" {
		t.Fatalf("text = %q, want ok", text)
	}
	if usage != (TokenUsage{Input: 11, Output: 7}) {
		t.Fatalf("usage = %+v, want %+v", usage, TokenUsage{Input: 11, Output: 7})
	}
}

func TestEinoChatStream_FiltersReasoningChunks(t *testing.T) {
	model := &fakeToolCallingChatModel{}
	model.generateFn = func(context.Context, []*schema.Message, ...einomodel.Option) (*schema.Message, error) {
		return nil, nil
	}
	model.streamFn = func(_ context.Context, _ []*schema.Message, _ ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
		sr, sw := schema.Pipe[*schema.Message](2)
		go func() {
			defer sw.Close()
			sw.Send(&schema.Message{Role: schema.Assistant, ReasoningContent: "thinking"}, nil)
			sw.Send(&schema.Message{Role: schema.Assistant, Content: "visible"}, nil)
		}()
		return sr, nil
	}
	model.withToolsFn = func(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
		return model, nil
	}

	var chunks []string
	chat := NewEinoChat(model)
	err := chat.Stream(context.Background(), "system", []Message{{Role: "user", Content: "hi"}}, func(s string) {
		chunks = append(chunks, s)
	})
	if err != nil && err != io.EOF {
		t.Fatalf("Stream error = %v", err)
	}
	if got, want := strings.Join(chunks, ""), "visible"; got != want {
		t.Fatalf("stream output = %q, want %q", got, want)
	}
}

func TestEinoChatGenerateWithTool_ParsesFirstToolCallArguments(t *testing.T) {
	var toolNames []string
	model := &fakeToolCallingChatModel{}
	model.emitCallbacks = true
	model.generateFn = func(_ context.Context, _ []*schema.Message, _ ...einomodel.Option) (*schema.Message, error) {
		return &schema.Message{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "output",
						Arguments: `{"answer":"yes","score":0.9}`,
					},
				},
			},
			ResponseMeta: &schema.ResponseMeta{
				Usage: &schema.TokenUsage{
					PromptTokens:     5,
					CompletionTokens: 3,
				},
			},
		}, nil
	}
	model.streamFn = func(context.Context, []*schema.Message, ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
		return nil, nil
	}
	model.withToolsFn = func(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
		for _, ti := range tools {
			toolNames = append(toolNames, ti.Name)
		}
		return model, nil
	}

	chat := NewEinoChat(model)
	got, usage, err := chat.GenerateWithTool(context.Background(), "system", []Message{{Role: "user", Content: "hi"}}, ToolDef{
		Name:        "output",
		Description: "structured result",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"answer": map[string]any{"type": "string"},
			},
		},
	})
	if err != nil {
		t.Fatalf("GenerateWithTool error = %v", err)
	}
	if !reflect.DeepEqual(toolNames, []string{"output"}) {
		t.Fatalf("toolNames = %v, want [output]", toolNames)
	}
	if got["answer"] != "yes" {
		t.Fatalf("answer = %v, want yes", got["answer"])
	}
	if usage != (TokenUsage{Input: 5, Output: 3}) {
		t.Fatalf("usage = %+v, want %+v", usage, TokenUsage{Input: 5, Output: 3})
	}
}

func TestEinoChatGenerate_EmitsCallbackTraceSpan(t *testing.T) {
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(tracing.NewEinoTraceCallbackHandler())

	model := &fakeToolCallingChatModel{}
	model.emitCallbacks = true
	model.generateFn = func(_ context.Context, _ []*schema.Message, _ ...einomodel.Option) (*schema.Message, error) {
		return &schema.Message{
			Role:    schema.Assistant,
			Content: "ok",
			ResponseMeta: &schema.ResponseMeta{
				Usage: &schema.TokenUsage{
					PromptTokens:     11,
					CompletionTokens: 7,
				},
			},
		}, nil
	}
	model.streamFn = func(context.Context, []*schema.Message, ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
		return nil, nil
	}
	model.withToolsFn = func(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
		return model, nil
	}

	rt := tracing.NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")
	defer trace.End()
	ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
		Name: "llm_generate",
		Kind: tracing.KindLLM,
		Attributes: map[string]any{
			"model": "test-model",
		},
	})

	chat := NewEinoChat(model)
	_, _, err := chat.Generate(ctx, "system", []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	var found bool
	for _, span := range tr.Spans {
		if span.Name == "llm_generate" && span.Kind == tracing.KindLLM {
			found = true
			if span.Attributes["model"] != "test-model" {
				t.Fatalf("model attr = %v, want test-model", span.Attributes["model"])
			}
			if span.Attributes["output_tokens"] != 7 {
				t.Fatalf("output_tokens = %v, want 7", span.Attributes["output_tokens"])
			}
		}
	}
	if !found {
		t.Fatal("expected llm_generate span emitted from Eino callback")
	}
}

func TestEinoChatStream_EmitsCallbackTraceSpan(t *testing.T) {
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(tracing.NewEinoTraceCallbackHandler())

	model := &fakeToolCallingChatModel{}
	model.emitCallbacks = true
	model.generateFn = func(context.Context, []*schema.Message, ...einomodel.Option) (*schema.Message, error) {
		return nil, nil
	}
	model.streamFn = func(_ context.Context, _ []*schema.Message, _ ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
		sr, sw := schema.Pipe[*schema.Message](2)
		go func() {
			defer sw.Close()
			sw.Send(&schema.Message{Role: schema.Assistant, Content: "visible"}, nil)
		}()
		return sr, nil
	}
	model.withToolsFn = func(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
		return model, nil
	}

	rt := tracing.NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")
	defer trace.End()
	ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
		Name: "llm_generate",
		Kind: tracing.KindLLM,
	})

	chat := NewEinoChat(model)
	err := chat.Stream(ctx, "system", []Message{{Role: "user", Content: "hi"}}, func(string) {})
	if err != nil && err != io.EOF {
		t.Fatalf("Stream error = %v", err)
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
