package runtime

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type captureSink struct {
	events []Event
}

func (s *captureSink) Emit(_ context.Context, evt Event) error {
	s.events = append(s.events, evt)
	return nil
}

func TestAgentEventBridge_AssistantToolCallTextBecomesThinking(t *testing.T) {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		gen.Send(&adk.AgentEvent{
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					Role: schema.Assistant,
					Message: schema.AssistantMessage("我先核对一下婚姻相关信息。", []schema.ToolCall{{
						ID: "call_1",
						Function: schema.FunctionCall{
							Name:      "knowledge_search",
							Arguments: `{"query":"婚姻"}`,
						},
					}}),
				},
			},
		})
		gen.Send(&adk.AgentEvent{
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					Role:     schema.Tool,
					ToolName: "knowledge_search",
					Message: schema.ToolMessage(`{"ok":true}`, "call_1", schema.WithToolName("knowledge_search")),
				},
			},
		})
		gen.Send(&adk.AgentEvent{
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					Role: schema.Assistant,
					Message: schema.AssistantMessage("这是最终回答。", nil),
				},
			},
		})
		gen.Close()
	}()

	sink := &captureSink{}
	finalText, err := agentEventBridge(context.Background(), sink, iter, nil, false)
	if err != nil {
		t.Fatalf("agentEventBridge returned error: %v", err)
	}
	if finalText != "这是最终回答。" {
		t.Fatalf("finalText = %q, want %q", finalText, "这是最终回答。")
	}
	if len(sink.events) != 3 {
		t.Fatalf("events = %d, want 3", len(sink.events))
	}
	if sink.events[0].Type != "thinking" {
		t.Fatalf("event 0 type = %q, want %q", sink.events[0].Type, "thinking")
	}
	thinkingData, ok := sink.events[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("event 0 data type = %T, want map[string]any", sink.events[0].Data)
	}
	if thinkingData["text"] != "我先核对一下婚姻相关信息。" {
		t.Fatalf("thinking text = %v", thinkingData["text"])
	}
	if sink.events[1].Type != "tool_call" {
		t.Fatalf("event 1 type = %q, want %q", sink.events[1].Type, "tool_call")
	}
	if sink.events[2].Type != "text" {
		t.Fatalf("event 2 type = %q, want %q", sink.events[2].Type, "text")
	}
}

func TestAgentEventBridge_BufferFinalWithXMLTags_NoToolCalls(t *testing.T) {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		gen.Send(&adk.AgentEvent{
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					Role: schema.Assistant,
					Message: schema.AssistantMessage(
						"<analysis>\n这是内部推理\n</analysis>\n<response>\n这是最终回答\n</response>",
						nil,
					),
				},
			},
		})
		gen.Close()
	}()

	sink := &captureSink{}
	finalText, err := agentEventBridge(context.Background(), sink, iter, nil, true)
	if err != nil {
		t.Fatalf("agentEventBridge returned error: %v", err)
	}
	if finalText != "这是最终回答" {
		t.Fatalf("finalText = %q, want %q", finalText, "这是最终回答")
	}
	if len(sink.events) != 1 {
		t.Fatalf("events = %d, want 1 (thinking only, text emitted by executor)", len(sink.events))
	}
	if sink.events[0].Type != "thinking" {
		t.Fatalf("event 0 type = %q, want %q", sink.events[0].Type, "thinking")
	}
	thinkingData, ok := sink.events[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("event 0 data type = %T, want map[string]any", sink.events[0].Data)
	}
	if thinkingData["text"] != "这是内部推理" {
		t.Fatalf("thinking text = %q, want %q", thinkingData["text"], "这是内部推理")
	}
}

func TestAgentEventBridge_BufferFinalFalse_WithXMLTags(t *testing.T) {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		gen.Send(&adk.AgentEvent{
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					Role: schema.Assistant,
					Message: schema.AssistantMessage(
						"<analysis>\n推理\n</analysis>\n<response>\n回答\n</response>",
						nil,
					),
				},
			},
		})
		gen.Close()
	}()

	sink := &captureSink{}
	finalText, err := agentEventBridge(context.Background(), sink, iter, nil, false)
	if err != nil {
		t.Fatalf("agentEventBridge returned error: %v", err)
	}
	if finalText != "回答" {
		t.Fatalf("finalText = %q, want %q", finalText, "回答")
	}
	// bufferFinal=false: text emitted inline (1 event), then XML parsing at end emits thinking (1 event)
	if len(sink.events) != 2 {
		t.Fatalf("events = %d, want 2", len(sink.events))
	}
	if sink.events[0].Type != "text" {
		t.Fatalf("event 0 type = %q, want text", sink.events[0].Type)
	}
	if sink.events[1].Type != "thinking" {
		t.Fatalf("event 1 type = %q, want thinking", sink.events[1].Type)
	}
}

func TestAgentEventBridge_NoXMLTags_Fallback(t *testing.T) {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		gen.Send(&adk.AgentEvent{
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					Role: schema.Assistant,
					Message: schema.AssistantMessage(
						"这是没有标记的纯文本回答。",
						nil,
					),
				},
			},
		})
		gen.Close()
	}()

	sink := &captureSink{}
	finalText, err := agentEventBridge(context.Background(), sink, iter, nil, true)
	if err != nil {
		t.Fatalf("agentEventBridge returned error: %v", err)
	}
	if finalText != "这是没有标记的纯文本回答。" {
		t.Fatalf("finalText = %q, want original text", finalText)
	}
	if len(sink.events) != 0 {
		t.Fatalf("events = %d, want 0 (no thinking for untagged text)", len(sink.events))
	}
}

func TestParseXMLSections(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantAnalysis string
		wantResponse string
		wantHasTags  bool
	}{
		{
			name:         "well-formed",
			input:        "<analysis>\n推理内容\n</analysis>\n<response>\n回答内容\n</response>",
			wantAnalysis: "推理内容",
			wantResponse: "回答内容",
			wantHasTags:  true,
		},
		{
			name:         "no tags",
			input:        "普通文本",
			wantAnalysis: "",
			wantResponse: "普通文本",
			wantHasTags:  false,
		},
		{
			name:         "empty analysis",
			input:        "<analysis>\n\n</analysis>\n<response>\n回答\n</response>",
			wantAnalysis: "",
			wantResponse: "回答",
			wantHasTags:  true,
		},
		{
			name:         "only analysis no response",
			input:        "<analysis>推理</analysis>",
			wantAnalysis: "",
			wantResponse: "<analysis>推理</analysis>",
			wantHasTags:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			analysis, response, hasTags := parseXMLSections(tc.input)
			if analysis != tc.wantAnalysis {
				t.Fatalf("analysis = %q, want %q", analysis, tc.wantAnalysis)
			}
			if response != tc.wantResponse {
				t.Fatalf("response = %q, want %q", response, tc.wantResponse)
			}
			if hasTags != tc.wantHasTags {
				t.Fatalf("hasTags = %v, want %v", hasTags, tc.wantHasTags)
			}
		})
	}
}
