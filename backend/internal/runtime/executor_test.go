package runtime

import (
	"context"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
)

func TestExecutor_RouterField(t *testing.T) {
	e := &Executor{router: &intent.SemanticRouter{}}
	if e.router == nil {
		t.Fatal("router field not set")
	}
}

type executorGuardTool struct {
	called bool
}

func (t *executorGuardTool) Name() string        { return "needs_query" }
func (t *executorGuardTool) Description() string { return "test tool" }
func (t *executorGuardTool) Label() string       { return "Test Tool" }
func (t *executorGuardTool) Execute(context.Context, map[string]any) (any, error) {
	t.called = true
	return map[string]any{"ok": true}, nil
}

func TestExecutorCallTool_UsesToolRunnerInvalidParamGuard(t *testing.T) {
	tool := &executorGuardTool{}
	reg := tools.NewRegistry()
	reg.RegisterWithContract(tool, tools.ToolContract{
		Name:       "needs_query",
		Version:    "v1",
		ReadOnly:   true,
		Idempotent: true,
		SideEffect: tools.SideEffectRead,
		RiskLevel:  tools.RiskLow,
		Params: []tools.ParamSpec{
			{Name: "query", Type: "string", Required: true},
		},
		Retry: tools.RetryPolicy{MaxAttempts: 1},
	})

	e := &Executor{reg: reg}
	result := e.callTool(context.Background(), "needs_query", map[string]any{})
	if result != nil {
		t.Fatalf("result = %v, want nil when params are invalid", result)
	}
	if tool.called {
		t.Fatal("tool must not execute when ToolRunner rejects invalid params")
	}
}
