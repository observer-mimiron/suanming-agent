package tools

import (
	"context"
	"strings"
	"testing"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type fakeLegacyTool struct {
	name        string
	description string
	executeFn   func(ctx context.Context, params map[string]any) (any, error)
	info        *schema.ToolInfo
}

func (f *fakeLegacyTool) Name() string        { return f.name }
func (f *fakeLegacyTool) Description() string { return f.description }
func (f *fakeLegacyTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	return f.executeFn(ctx, params)
}
func (f *fakeLegacyTool) EinoToolInfo() *schema.ToolInfo { return f.info }

var _ Tool = (*fakeLegacyTool)(nil)
var _ EinoDescriber = (*fakeLegacyTool)(nil)

func TestRegistryRegister_PreservesLegacyLookup(t *testing.T) {
	reg := NewRegistry()
	legacy := &fakeLegacyTool{
		name:        "demo",
		description: "demo tool",
		executeFn: func(_ context.Context, params map[string]any) (any, error) {
			return params, nil
		},
		info: testToolInfo("demo"),
	}

	reg.Register(legacy)

	got, ok := reg.Get("demo")
	if !ok {
		t.Fatal("expected legacy lookup to succeed")
	}
	if got != legacy {
		t.Fatalf("legacy lookup returned %T, want original tool", got)
	}
}

func TestRegistryEinoTools_ExposesWrappedInvokableTools(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&fakeLegacyTool{
		name:        "demo",
		description: "demo tool",
		executeFn: func(_ context.Context, params map[string]any) (any, error) {
			return params, nil
		},
		info: testToolInfo("demo"),
	})

	tools := reg.EinoTools()
	if len(tools) != 1 {
		t.Fatalf("len(EinoTools) = %d, want 1", len(tools))
	}

	info, err := tools[0].Info(context.Background())
	if err != nil {
		t.Fatalf("Info error = %v", err)
	}
	if info.Name != "demo" {
		t.Fatalf("info.Name = %q, want demo", info.Name)
	}
	if info.Desc != "demo tool" {
		t.Fatalf("info.Desc = %q, want demo tool", info.Desc)
	}
}

func TestLegacyToolAdapter_InvokableRun_ExecutesToolAndReturnsJSON(t *testing.T) {
	adapter := &legacyToolAdapter{
		tool: &fakeLegacyTool{
			name:        "demo",
			description: "demo tool",
			executeFn: func(_ context.Context, params map[string]any) (any, error) {
				return map[string]any{
					"echo": params["query"],
				}, nil
			},
			info: testToolInfo("demo"),
		},
		info: testToolInfo("demo"),
	}

	out, err := adapter.InvokableRun(context.Background(), `{"query":"hi"}`)
	if err != nil {
		t.Fatalf("InvokableRun error = %v", err)
	}
	if !strings.Contains(out, `"echo":"hi"`) {
		t.Fatalf("output = %s, want JSON echo payload", out)
	}
}

func TestLegacyToolAdapter_InvokableRun_PropagatesValidationError(t *testing.T) {
	adapter := &legacyToolAdapter{
		tool: &fakeLegacyTool{
			name:        "demo",
			description: "demo tool",
			executeFn: func(_ context.Context, params map[string]any) (any, error) {
				return nil, errMissingQuery
			},
			info: testToolInfo("demo"),
		},
		info: testToolInfo("demo"),
	}

	_, err := adapter.InvokableRun(context.Background(), `{"query":""}`)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("error = %v, want query is required", err)
	}
}

func testToolInfo(name string) *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: name,
		Desc: "demo tool",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "search query",
				Required: true,
			},
		}),
	}
}

var errMissingQuery = &toolError{msg: "query is required"}

type toolError struct {
	msg string
}

func (e *toolError) Error() string {
	return e.msg
}

var _ einotool.InvokableTool = (*legacyToolAdapter)(nil)
