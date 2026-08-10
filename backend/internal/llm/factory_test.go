// This test file belongs to the LLM adapter layer.
// It verifies provider factory behavior and protects the related contract from regressions.
// It wraps model providers; domain prompts and contracts stay outside this package.
package llm

import (
	"context"
	"testing"

	deepseekmodel "github.com/cloudwego/eino-ext/components/model/deepseek"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func TestNewChatClient_BuildsEinoChat(t *testing.T) {
	oldEino := newEinoToolCallingChatModel
	defer func() {
		newEinoToolCallingChatModel = oldEino
	}()

	einoCalled := 0
	einoModel := &fakeToolCallingChatModel{}
	einoModel.generateFn = func(context.Context, []*schema.Message, ...einomodel.Option) (*schema.Message, error) {
		return schema.AssistantMessage("ok", nil), nil
	}
	einoModel.streamFn = func(context.Context, []*schema.Message, ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
		return nil, nil
	}
	einoModel.withToolsFn = func(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
		return einoModel, nil
	}

	newEinoToolCallingChatModel = func(_ context.Context, cfg FactoryConfig) (einomodel.ToolCallingChatModel, error) {
		einoCalled++
		return einoModel, nil
	}

	client, err := NewChatClient(context.Background(), FactoryConfig{Model: "x"})
	if err != nil {
		t.Fatalf("NewChatClient error = %v", err)
	}
	if _, ok := client.(*EinoChat); !ok {
		t.Fatalf("client type = %T, want *EinoChat", client)
	}

	if einoCalled != 1 {
		t.Fatalf("einoCalled = %d, want 1", einoCalled)
	}
}

func TestNewChatClient_ForwardsDisableThinkingToEinoFactory(t *testing.T) {
	oldEino := newEinoToolCallingChatModel
	defer func() {
		newEinoToolCallingChatModel = oldEino
	}()

	einoModel := &fakeToolCallingChatModel{}
	einoModel.generateFn = func(context.Context, []*schema.Message, ...einomodel.Option) (*schema.Message, error) {
		return schema.AssistantMessage("ok", nil), nil
	}
	einoModel.streamFn = func(context.Context, []*schema.Message, ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
		return nil, nil
	}
	einoModel.withToolsFn = func(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
		return einoModel, nil
	}

	var captured FactoryConfig
	newEinoToolCallingChatModel = func(_ context.Context, cfg FactoryConfig) (einomodel.ToolCallingChatModel, error) {
		captured = cfg
		return einoModel, nil
	}

	_, err := NewChatClient(context.Background(), FactoryConfig{
		Model:           "deepseek-v4-flash",
		DisableThinking: true,
	})
	if err != nil {
		t.Fatalf("NewChatClient error = %v", err)
	}

	if !captured.DisableThinking {
		t.Fatal("expected DisableThinking to be forwarded to Eino factory")
	}
}

func TestNewToolCallingModelWithJSON_UsesJSONObjectResponseFormat(t *testing.T) {
	old := newDeepSeekChatModel
	defer func() { newDeepSeekChatModel = old }()

	var captured *deepseekmodel.ChatModelConfig
	newDeepSeekChatModel = func(ctx context.Context, cfg *deepseekmodel.ChatModelConfig) (*deepseekmodel.ChatModel, error) {
		captured = cfg
		return old(ctx, cfg)
	}

	if _, err := NewToolCallingModelWithJSON(context.Background(), FactoryConfig{Model: "deepseek-v4-flash"}); err != nil {
		t.Fatalf("NewToolCallingModelWithJSON error = %v", err)
	}
	if captured == nil || captured.ResponseFormatType != deepseekmodel.ResponseFormatTypeJSONObject {
		t.Fatalf("response format = %#v, want json_object", captured)
	}
}
