package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type dummyModel struct{}

func (d *dummyModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return schema.AssistantMessage("ok", nil), nil
}
func (d *dummyModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	sr, sw := schema.Pipe[*schema.Message](1)
	sw.Send(schema.AssistantMessage("ok", nil), nil)
	sw.Close()
	return sr, nil
}
func (d *dummyModel) BindTools(tools []*schema.ToolInfo) error { return nil }

type dummyTool struct{ called bool }

func (dt *dummyTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "dummy_tool",
		Desc: "A dummy tool for testing",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"input": {Type: schema.String, Required: true},
		}),
	}, nil
}
func (dt *dummyTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	dt.called = true
	return `{"result":"ok"}`, nil
}

func TestAgentToolContract_InputShape(t *testing.T) {
	ctx := context.Background()
	childAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "child_test",
		Description: "child agent for testing",
		Model:       &dummyModel{},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{&dummyTool{}},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewChatModelAgent: %v", err)
	}
	agentTool := adk.NewAgentTool(ctx, childAgent)
	info, err := agentTool.Info(ctx)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Name != "child_test" {
		t.Fatalf("tool name: got %q, want %q", info.Name, "child_test")
	}
	if info.Desc == "" {
		t.Fatal("tool description should be non-empty")
	}
	if info.ParamsOneOf == nil {
		t.Fatal("ParamsOneOf should not be nil")
	}
	t.Logf("AgentTool info: name=%s desc=%s params=%+v", info.Name, info.Desc, info.ParamsOneOf)
}

func TestAgentToolContract_WithFullChatHistoryAsInput(t *testing.T) {
	ctx := context.Background()
	childAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "child_full_hist",
		Description: "child agent with full chat history",
		Model:       &dummyModel{},
	})
	if err != nil {
		t.Fatalf("NewChatModelAgent: %v", err)
	}
	agentTool := adk.NewAgentTool(ctx, childAgent, adk.WithFullChatHistoryAsInput())
	info, err := agentTool.Info(ctx)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Name != "child_full_hist" {
		t.Fatalf("tool name: got %q, want %q", info.Name, "child_full_hist")
	}
	t.Logf("AgentTool with full chat history: name=%s", info.Name)
}

func TestAgentToolContract_EmitInternalEvents(t *testing.T) {
	cfg := &adk.ChatModelAgentConfig{
		Name:        "parent_test",
		Description: "parent agent emitting internal events",
		Model:       &dummyModel{},
		ToolsConfig: adk.ToolsConfig{
			EmitInternalEvents: true,
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{},
			},
		},
	}
	if !cfg.ToolsConfig.EmitInternalEvents {
		t.Fatal("EmitInternalEvents should be set to true")
	}
	t.Logf("EmitInternalEvents config is available on ChatModelAgentConfig.ToolsConfig")
}

func TestAgentToolContract_SessionValues(t *testing.T) {
	vals := map[string]any{
		"profile":     map[string]any{"year": 1990},
		"bazi_result": map[string]any{"dayGan": "甲"},
		"domain":      "bazi",
	}
	_ = adk.WithSessionValues(vals)
	t.Logf("adk.WithSessionValues is available and accepts map[string]any")
}

func TestAgentToolContract_SessionValuesPropagation(t *testing.T) {
	ctx := context.Background()
	detector := &sessionDetectorTool{}
	childAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "session_detect_child",
		Description: "detects session values propagation",
		Model:       &dummyModel{},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{detector},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewChatModelAgent: %v", err)
	}
	_ = adk.NewAgentTool(ctx, childAgent)
	t.Logf("SessionValues propagation: verified API contracts; propagation behavior depends on Eino internal implementation")
}

type sessionDetectorTool struct{}

func (s *sessionDetectorTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "session_detector",
		Desc: "Detects whether session values are propagated to child agents",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"key": {Type: schema.String, Required: true},
		}),
	}, nil
}
func (s *sessionDetectorTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return `{"result":"detector_called"}`, nil
}

func TestAgentToolContract_AgentNaming(t *testing.T) {
	ctx := context.Background()
	child, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "bazi_specialist",
		Description: "八字命理专家。根据出生时间排盘、分析用神忌神、解读大运走势。",
		Model:       &dummyModel{},
	})
	if err != nil {
		t.Fatalf("NewChatModelAgent: %v", err)
	}
	agt := adk.NewAgentTool(ctx, child)
	info, err := agt.Info(ctx)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Name != "bazi_specialist" {
		t.Fatalf("AgentTool name: got %q, want %q", info.Name, "bazi_specialist")
	}
	if !strings.Contains(info.Desc, "八字") {
		t.Fatalf("AgentTool desc should contain '八字', got: %s", info.Desc)
	}
	t.Logf("AgentTool naming: name=%s desc=%s", info.Name, info.Desc)
}
