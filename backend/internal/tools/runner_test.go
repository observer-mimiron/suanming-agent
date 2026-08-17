// This test file belongs to the deterministic tool layer.
// It verifies runner behavior and protects the related contract from regressions.
// It executes governed tools; user-facing synthesis stays in runtime.
package tools

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

type runnerTool struct {
	name string
	fn   func(context.Context, map[string]any) (any, error)
}

func (t runnerTool) Name() string        { return t.name }
func (t runnerTool) Description() string { return "runner test tool" }
func (t runnerTool) Label() string       { return "Runner Test Tool" }
func (t runnerTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	return t.fn(ctx, params)
}

func TestToolRunner_RunSuccess(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterWithContract(runnerTool{
		name: "echo",
		fn: func(_ context.Context, params map[string]any) (any, error) {
			return map[string]any{"value": params["value"]}, nil
		},
	}, ToolContract{
		Name:       "echo",
		Version:    "v1",
		ReadOnly:   true,
		Idempotent: true,
		SideEffect: SideEffectRead,
		RiskLevel:  RiskLow,
		Params: []ParamSpec{
			{Name: "value", Type: "string", Required: true},
		},
		Retry: RetryPolicy{MaxAttempts: 1},
	})

	runner := NewToolRunner(reg)
	result := runner.Run(context.Background(), ToolRunRequest{
		ToolName:       "echo",
		Params:         map[string]any{"value": "ok"},
		DecisionSource: "prefill",
	})

	if result.Status != ToolRunStatusOK {
		t.Fatalf("Status = %q, want %q", result.Status, ToolRunStatusOK)
	}
	if result.ErrorClass != "" {
		t.Fatalf("ErrorClass = %q, want empty", result.ErrorClass)
	}
	if result.Attempts != 1 {
		t.Fatalf("Attempts = %d, want 1", result.Attempts)
	}
	data := result.Data.(map[string]any)
	if data["value"] != "ok" {
		t.Fatalf("Data[value] = %v, want ok", data["value"])
	}
}

func TestToolRunner_MissingTool(t *testing.T) {
	runner := NewToolRunner(NewRegistry())
	result := runner.Run(context.Background(), ToolRunRequest{
		ToolName:       "missing",
		DecisionSource: "prefill",
	})

	if result.Status != ToolRunStatusError {
		t.Fatalf("Status = %q, want %q", result.Status, ToolRunStatusError)
	}
	if result.ErrorClass != ToolErrorNotFound {
		t.Fatalf("ErrorClass = %q, want %q", result.ErrorClass, ToolErrorNotFound)
	}
}

func TestToolRunner_InvalidParamsDoNotExecuteTool(t *testing.T) {
	called := false
	reg := NewRegistry()
	reg.RegisterWithContract(runnerTool{
		name: "needs_query",
		fn: func(context.Context, map[string]any) (any, error) {
			called = true
			return map[string]any{"ok": true}, nil
		},
	}, ToolContract{
		Name:       "needs_query",
		Version:    "v1",
		ReadOnly:   true,
		Idempotent: true,
		SideEffect: SideEffectRead,
		RiskLevel:  RiskLow,
		Params: []ParamSpec{
			{Name: "query", Type: "string", Required: true},
		},
		Retry: RetryPolicy{MaxAttempts: 1},
	})

	result := NewToolRunner(reg).Run(context.Background(), ToolRunRequest{
		ToolName:       "needs_query",
		Params:         map[string]any{},
		DecisionSource: "prefill",
	})

	if called {
		t.Fatal("tool must not execute when required params are missing")
	}
	if result.ErrorClass != ToolErrorInvalidParams {
		t.Fatalf("ErrorClass = %q, want %q", result.ErrorClass, ToolErrorInvalidParams)
	}
}

func TestClassifyToolError(t *testing.T) {
	tests := []struct {
		err  error
		want ToolErrorClass
	}{
		{err: context.DeadlineExceeded, want: ToolErrorTransient},
		{err: context.Canceled, want: ToolErrorCanceled},
		{err: errors.New("query is required"), want: ToolErrorInvalidParams},
		{err: errors.New("permission denied"), want: ToolErrorPermissionDenied},
		{err: errors.New("business rejected"), want: ToolErrorBusinessRejected},
		{err: errors.New("something else"), want: ToolErrorInternal},
	}

	for _, tt := range tests {
		if got := ClassifyToolError(tt.err); got != tt.want {
			t.Fatalf("ClassifyToolError(%v) = %q, want %q", tt.err, got, tt.want)
		}
	}
}

func TestToolRunner_DoesNotRetryCanceledParent(t *testing.T) {
	called := 0
	reg := NewRegistry()
	reg.RegisterWithContract(runnerTool{
		name: "cancelled",
		fn: func(context.Context, map[string]any) (any, error) {
			called++
			return nil, errors.New("temporary failure")
		},
	}, ToolContract{
		Name:       "cancelled",
		ReadOnly:   true,
		Idempotent: true,
		SideEffect: SideEffectRead,
		Retry: RetryPolicy{
			MaxAttempts:       3,
			BackoffMillis:     100,
			RetryErrorClasses: []ToolErrorClass{ToolErrorTransient},
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := NewToolRunner(reg).Run(ctx, ToolRunRequest{ToolName: "cancelled"})
	if called != 0 || result.Attempts != 0 {
		t.Fatalf("called=%d attempts=%d, want no execution", called, result.Attempts)
	}
	if result.ErrorClass != ToolErrorCanceled {
		t.Fatalf("ErrorClass=%q, want %q", result.ErrorClass, ToolErrorCanceled)
	}
}

func TestToolRunner_RetriesTransientErrors(t *testing.T) {
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	attempts := 0
	reg := NewRegistry()
	reg.RegisterWithContract(runnerTool{
		name: "flaky",
		fn: func(context.Context, map[string]any) (any, error) {
			attempts++
			if attempts == 1 {
				return nil, errors.New("temporary network error")
			}
			return map[string]any{"ok": true}, nil
		},
	}, ToolContract{
		Name:       "flaky",
		Version:    "v1",
		ReadOnly:   true,
		Idempotent: true,
		SideEffect: SideEffectRead,
		RiskLevel:  RiskLow,
		Retry: RetryPolicy{
			MaxAttempts:       2,
			BackoffMillis:     0,
			RetryErrorClasses: []ToolErrorClass{ToolErrorTransient},
		},
	})

	result := NewToolRunner(reg).Run(context.Background(), ToolRunRequest{ToolName: "flaky"})
	if result.Status != ToolRunStatusOK {
		t.Fatalf("Status = %q, want %q", result.Status, ToolRunStatusOK)
	}
	if result.Attempts != 2 {
		t.Fatalf("Attempts = %d, want 2", result.Attempts)
	}
	entry := logs.String()
	if !strings.Contains(entry, "tool_name=flaky") || !strings.Contains(entry, "attempt=2") || !strings.Contains(entry, "error_class=transient") {
		t.Fatalf("retry log = %q, want structured tool retry fields", entry)
	}
	if strings.Contains(entry, "temporary network error") {
		t.Fatalf("retry log leaked raw error = %q", entry)
	}
}

func TestToolRunner_DoesNotRetryInvalidParams(t *testing.T) {
	attempts := 0
	reg := NewRegistry()
	reg.RegisterWithContract(runnerTool{
		name: "invalid_once",
		fn: func(context.Context, map[string]any) (any, error) {
			attempts++
			return nil, errors.New("query is required")
		},
	}, ToolContract{
		Name:       "invalid_once",
		Version:    "v1",
		ReadOnly:   true,
		Idempotent: true,
		SideEffect: SideEffectRead,
		RiskLevel:  RiskLow,
		Retry: RetryPolicy{
			MaxAttempts:       3,
			RetryErrorClasses: []ToolErrorClass{ToolErrorTransient},
		},
	})

	result := NewToolRunner(reg).Run(context.Background(), ToolRunRequest{ToolName: "invalid_once"})
	if result.ErrorClass != ToolErrorInvalidParams {
		t.Fatalf("ErrorClass = %q, want %q", result.ErrorClass, ToolErrorInvalidParams)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestToolRunner_BlocksApprovalRequiredTool(t *testing.T) {
	called := false
	reg := NewRegistry()
	reg.RegisterWithContract(runnerTool{
		name: "write_order",
		fn: func(context.Context, map[string]any) (any, error) {
			called = true
			return map[string]any{"ok": true}, nil
		},
	}, ToolContract{
		Name:             "write_order",
		Version:          "v1",
		ReadOnly:         false,
		Idempotent:       false,
		RequiresApproval: true,
		SideEffect:       SideEffectWrite,
		RiskLevel:        RiskHigh,
		Retry:            RetryPolicy{MaxAttempts: 1},
	})

	result := NewToolRunner(reg).Run(context.Background(), ToolRunRequest{ToolName: "write_order"})
	if called {
		t.Fatal("approval-required tool must not execute without approval")
	}
	if result.Status != ToolRunStatusBlocked {
		t.Fatalf("Status = %q, want %q", result.Status, ToolRunStatusBlocked)
	}
	if result.ErrorClass != ToolErrorApprovalRequired {
		t.Fatalf("ErrorClass = %q, want %q", result.ErrorClass, ToolErrorApprovalRequired)
	}
}

func TestToolRunner_ClassifiesTimeoutAsTransient(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterWithContract(runnerTool{
		name: "slow",
		fn: func(ctx context.Context, _ map[string]any) (any, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}, ToolContract{
		Name:          "slow",
		Version:       "v1",
		ReadOnly:      true,
		Idempotent:    true,
		SideEffect:    SideEffectRead,
		RiskLevel:     RiskLow,
		TimeoutMillis: 1,
		Retry:         RetryPolicy{MaxAttempts: 1},
	})

	result := NewToolRunner(reg).Run(context.Background(), ToolRunRequest{ToolName: "slow"})
	if result.ErrorClass != ToolErrorTransient {
		t.Fatalf("ErrorClass = %q, want %q", result.ErrorClass, ToolErrorTransient)
	}
}

func TestToolRunner_WritesGovernanceTraceAttributes(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterWithContract(runnerTool{
		name: "trace_tool",
		fn: func(context.Context, map[string]any) (any, error) {
			return map[string]any{"ok": true}, nil
		},
	}, ToolContract{
		Name:       "trace_tool",
		Version:    "v9",
		ReadOnly:   true,
		Idempotent: true,
		SideEffect: SideEffectRead,
		RiskLevel:  RiskMedium,
		Retry:      RetryPolicy{MaxAttempts: 1},
	})

	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	result := NewToolRunner(reg).Run(ctx, ToolRunRequest{
		ToolName:       "trace_tool",
		DecisionSource: "prefill",
	})
	trace.End()

	if result.Status != ToolRunStatusOK {
		t.Fatalf("Status = %q, want %q", result.Status, ToolRunStatusOK)
	}
	var found bool
	for _, span := range tracing.TraceFromContext(ctx).Spans {
		if span.Name != "trace_tool" {
			continue
		}
		found = true
		if span.Attributes["tool.version"] != "v9" {
			t.Fatalf("tool.version = %v, want v9", span.Attributes["tool.version"])
		}
		if span.Attributes["tool.risk_level"] != string(RiskMedium) {
			t.Fatalf("tool.risk_level = %v, want %s", span.Attributes["tool.risk_level"], RiskMedium)
		}
		if span.Attributes["tool.decision_source"] != "prefill" {
			t.Fatalf("tool.decision_source = %v, want prefill", span.Attributes["tool.decision_source"])
		}
	}
	if !found {
		t.Fatal("expected trace_tool span")
	}
}
