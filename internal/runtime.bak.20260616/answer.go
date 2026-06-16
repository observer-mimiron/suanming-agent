package runtime

import (
	"context"
	"errors"
	"strings"

	einocallbacks "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/mcp"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// answerWithKnowledge 执行统一的回答管线：知识检索 → LLM 流式解读。
func (e *Executor) answerWithKnowledge(ctx context.Context, sink EventSink, st *state.SessionState, primaryDomain string, errorPrefix string) (string, error) {
	passages := e.runKnowledgeSearch(ctx, sink, st, primaryDomain)
	fullText, err := e.StreamInterpretation(ctx, sink, st, passages, primaryDomain)
	if err != nil && errorPrefix != "" {
		sink.Emit(ctx, Event{Type: "error", Data: map[string]any{"message": errorPrefix + err.Error()}})
	}
	return fullText, err
}

// runKnowledgeSearch 执行知识库检索并推送相关 SSE 事件。
func (e *Executor) runKnowledgeSearch(ctx context.Context, sink EventSink, st *state.SessionState, primaryDomain string) []mcp.Passage {
	query := e.promptBuilder.BuildKnowledgeQuery(ctx, st, primaryDomain)
	cbCtx := einocallbacks.ReuseHandlers(ctx, &einocallbacks.RunInfo{
		Name:      "knowledge_search",
		Type:      "goRuntimeKnowledgeSearch",
		Component: components.ComponentOfRetriever,
	})
	cbCtx = einocallbacks.OnStart(cbCtx, &einoretriever.CallbackInput{
		Query: query,
		TopK:  5,
	})

	tool, ok := e.tools.Get("knowledge_search")
	if !ok {
		einocallbacks.OnError(cbCtx, errKnowledgeSearchToolNotRegistered)
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "知识检索未注册，跳过引用检索。",
		}})
		return []mcp.Passage{}
	}

	sink.Emit(ctx, Event{Type: "tool_call", Data: map[string]any{
		"tool":   "knowledge_search",
		"params": map[string]any{"query": query, "topK": 5},
	}})

	result, err := tool.Execute(ctx, map[string]any{"query": query, "topK": 5})
	if err != nil {
		einocallbacks.OnError(cbCtx, err)
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "知识检索失败，继续直接解读命盘。",
		}})
		return []mcp.Passage{}
	}

	payload, ok := result.(map[string]any)
	if !ok {
		einocallbacks.OnError(cbCtx, errKnowledgeSearchInvalidResult)
		return []mcp.Passage{}
	}
	passages, _ := payload["passages"].([]mcp.Passage)
	output := &einoretriever.CallbackOutput{
		Extra: map[string]any{"hits": len(passages)},
	}
	if len(passages) == 0 {
		output.Extra["status"] = "degraded"
		output.Extra["degrade_reason"] = "no_results"
	}
	einocallbacks.OnEnd(cbCtx, output)
	if len(passages) > 0 {
		sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
			"type":    "knowledge-sources",
			"payload": passages,
		}})
	}
	return passages
}

var (
	errKnowledgeSearchToolNotRegistered = errors.New("knowledge_search tool not registered")
	errKnowledgeSearchInvalidResult     = errors.New("knowledge_search result type invalid")
)

// streamInterpretation 构建解读提示词，调用 LLM 流式生成，通过 sink 推送文本分块。
func (e *Executor) StreamInterpretation(ctx context.Context, sink EventSink, st *state.SessionState, passages []mcp.Passage, primaryDomain string) (string, error) {
	systemPrompt := e.promptBuilder.BuildInterpretPrompt(st, passages, primaryDomain)
	messages := []llm.Message{
		{Role: "user", Content: CurrentQuestion(st)},
	}

	var tail strings.Builder
	var fullText strings.Builder
	blocked := false

	var llmSpan tracing.Span
	if llm.IsEinoChat(e.llm) {
		attrs := map[string]any{}
		if e.llmModel != "" {
			attrs["model"] = e.llmModel
		}
		ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
			Name:       "llm_generate",
			Kind:       tracing.KindLLM,
			Attributes: attrs,
		})
	} else {
		llmSpan = tracing.SpanFromContext(ctx, "llm_generate", tracing.KindLLM)
		if e.llmModel != "" {
			llmSpan.SetAttribute("model", e.llmModel)
		}
		llmSpan.SetAttribute("output_tokens", nil)
	}

	err := e.llm.Stream(ctx, systemPrompt, messages, func(chunk string) {
		if blocked {
			return
		}
		tail.WriteString(chunk)
		t := tail.String()
		if len(t) > 40 {
			t = t[len(t)-40:]
		}
		if strings.Contains(t, "仅供") || strings.Contains(t, "AI生成") || strings.Contains(t, "玄学算命") || strings.Contains(t, "以上内容由") {
			blocked = true
			return
		}
		fullText.WriteString(chunk)
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": chunk}})
	})

	if llmSpan != nil && err != nil {
		llmSpan.RecordError(err)
	}
	if llmSpan != nil {
		llmSpan.End()
	}
	return fullText.String(), err
}
