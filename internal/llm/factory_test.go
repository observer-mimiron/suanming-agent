package llm

import (
	"context"
	"testing"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func TestNewChatClient_SelectsBackendFromConfig(t *testing.T) {
	oldNative := newNativeChatClient
	oldEino := newEinoToolCallingChatModel
	defer func() {
		newNativeChatClient = oldNative
		newEinoToolCallingChatModel = oldEino
	}()

	nativeCalled := 0
	einoCalled := 0
	nativeClient := &NoopClient{}
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

	newNativeChatClient = func(cfg FactoryConfig) Chat {
		nativeCalled++
		return nativeClient
	}
	newEinoToolCallingChatModel = func(_ context.Context, cfg FactoryConfig) (einomodel.ToolCallingChatModel, error) {
		einoCalled++
		return einoModel, nil
	}

	client, err := NewChatClient(context.Background(), FactoryConfig{Backend: "native", Model: "x"})
	if err != nil {
		t.Fatalf("native error = %v", err)
	}
	if client != nativeClient {
		t.Fatalf("native client mismatch")
	}

	client, err = NewChatClient(context.Background(), FactoryConfig{Backend: "eino", Model: "x"})
	if err != nil {
		t.Fatalf("eino error = %v", err)
	}
	if _, ok := client.(*EinoChat); !ok {
		t.Fatalf("client type = %T, want *EinoChat", client)
	}

	if nativeCalled != 1 {
		t.Fatalf("nativeCalled = %d, want 1", nativeCalled)
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
		Backend:         "eino",
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
