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
