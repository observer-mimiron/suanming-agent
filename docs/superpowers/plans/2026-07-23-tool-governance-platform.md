# Tool Governance Platform Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a production-shaped tool governance layer so every runtime tool call has a contract, policy decision, structured result, trace metadata, retry behavior, and an upgrade path for future side-effect tools.

**Architecture:** Keep the existing `RouteAdvisor -> Policy Gate -> Manager -> ExecutionPlan -> Prefill -> specialist runner(s) -> manager compose -> final guard -> SSE` chain. Insert a `ToolRunner` between runtime code and concrete tools, backed by `ToolContract` metadata stored in the existing `tools.Registry`. Start with read-only and deterministic tools; add approval/idempotency hooks as enforced no-op policy for future write tools.

**Tech Stack:** Go, existing `backend/internal/tools` registry, existing `backend/internal/runtime.Executor`, existing `backend/internal/tracing` spans, `go test`.

## Execution Status

- **Status:** Implemented in the main workspace on 2026-07-23.
- **Verification:** `go test ./backend/internal/tools ./backend/internal/container ./backend/internal/runtime -v` passed.
- **Commit:** Not created. The repository already contains unrelated dirty changes, so this plan stops at verified workspace changes.
- **Boundary:** This implements the production-shaped governance core for runtime-owned deterministic tool calls. Specialist ADK tool adapter migration remains a separate follow-up.

---

## Scope

This plan covers the production-shaped tool governance core:

- Tool contracts and metadata.
- Unified tool execution through `ToolRunner`.
- Parameter validation, error classification, retry policy, timeout policy.
- Trace metadata for tool execution.
- Future side-effect controls: risk levels, approval requirement, idempotency key fields.
- Runtime integration for deterministic prefill tool calls.
- Documentation and progress tracking.

This plan does not convert every specialist ADK tool call in one pass. After `ToolRunner` is stable for deterministic runtime-owned tools, specialist tool invocation can be migrated in a separate plan because it crosses Eino ADK tool adapter boundaries.

## File Structure

- Create `backend/internal/tools/contract.go`
  Defines `ToolContract`, risk levels, side-effect levels, retry policies, parameter schema, and default contracts for existing tools.
- Create `backend/internal/tools/runner.go`
  Defines `ToolRunner`, `ToolRunRequest`, `ToolRunResult`, error classes, parameter validation, timeout, retry, trace attributes, and future approval/idempotency gates.
- Create `backend/internal/tools/runner_test.go`
  Unit tests for success, missing tool, invalid params, transient retry, non-retryable params, timeout, and blocked high-risk write tools.
- Modify `backend/internal/tools/registry.go`
  Store and retrieve contracts next to tools. Keep `Register(t Tool)` backward-compatible by assigning `DefaultContractFor(t.Name())`.
- Modify `backend/internal/container/container.go`
  Register explicit contracts for all existing tools and expose the registry as `Container.Tools` for runtime inspection and tests.
- Modify `backend/internal/runtime/executor.go`
  Add `toolRunner *tools.ToolRunner` to `Executor` and replace `callTool` direct execution with `ToolRunner.Run`.
- Modify `backend/internal/tracing/turn_trace.go`
  Add user-facing labels for new tool governance spans if a dedicated `tool_runner` span is used.
- Modify `docs/architecture.md`
  Add the new tool governance layer to runtime architecture.
- Modify `PROGRESS.md`
  Record the architecture decision and implementation status once implementation is complete.

---

### Task 1: Add Tool Contracts to the Registry

**Files:**
- Create: `backend/internal/tools/contract.go`
- Modify: `backend/internal/tools/registry.go`
- Test: `backend/internal/tools/registry_test.go`

- [ ] **Step 1: Write failing registry contract tests**

Create `backend/internal/tools/registry_test.go` with:

```go
package tools

import (
	"context"
	"testing"
)

type testTool struct {
	name string
}

func (t testTool) Name() string { return t.name }
func (t testTool) Description() string { return "test tool" }
func (t testTool) Label() string { return "Test Tool" }
func (t testTool) Execute(context.Context, map[string]any) (any, error) {
	return map[string]any{"ok": true}, nil
}

func TestRegistry_Register_AssignsDefaultContract(t *testing.T) {
	reg := NewRegistry()
	reg.Register(testTool{name: "custom_read_tool"})

	contract, ok := reg.Contract("custom_read_tool")
	if !ok {
		t.Fatal("expected contract for registered tool")
	}
	if contract.Name != "custom_read_tool" {
		t.Fatalf("contract.Name = %q, want custom_read_tool", contract.Name)
	}
	if contract.Version != "v1" {
		t.Fatalf("contract.Version = %q, want v1", contract.Version)
	}
	if contract.SideEffect != SideEffectRead {
		t.Fatalf("contract.SideEffect = %q, want %q", contract.SideEffect, SideEffectRead)
	}
	if !contract.ReadOnly {
		t.Fatal("default contract must be read-only")
	}
}

func TestRegistry_RegisterWithContract_PreservesExplicitContract(t *testing.T) {
	reg := NewRegistry()
	contract := ToolContract{
		Name:       "write_order",
		Version:    "v2",
		ReadOnly:   false,
		SideEffect: SideEffectWrite,
		RiskLevel:  RiskHigh,
		Retry: RetryPolicy{
			MaxAttempts: 1,
		},
	}
	reg.RegisterWithContract(testTool{name: "write_order"}, contract)

	got, ok := reg.Contract("write_order")
	if !ok {
		t.Fatal("expected explicit contract")
	}
	if got.Version != "v2" {
		t.Fatalf("Version = %q, want v2", got.Version)
	}
	if got.RiskLevel != RiskHigh {
		t.Fatalf("RiskLevel = %q, want %q", got.RiskLevel, RiskHigh)
	}
	if got.ReadOnly {
		t.Fatal("explicit write contract must not be read-only")
	}
}

func TestDefaultContractFor_KnownTools(t *testing.T) {
	tests := []struct {
		name       string
		sideEffect SideEffectLevel
		readOnly   bool
	}{
		{name: "knowledge_search", sideEffect: SideEffectRead, readOnly: true},
		{name: "knowledge_catalog", sideEffect: SideEffectRead, readOnly: true},
		{name: "bazi_calc", sideEffect: SideEffectNone, readOnly: true},
		{name: "qimen_dunjia", sideEffect: SideEffectNone, readOnly: true},
		{name: "ziwei_calc", sideEffect: SideEffectNone, readOnly: true},
	}

	for _, tt := range tests {
		contract := DefaultContractFor(tt.name)
		if contract.Name != tt.name {
			t.Fatalf("%s: contract.Name = %q", tt.name, contract.Name)
		}
		if contract.SideEffect != tt.sideEffect {
			t.Fatalf("%s: SideEffect = %q, want %q", tt.name, contract.SideEffect, tt.sideEffect)
		}
		if contract.ReadOnly != tt.readOnly {
			t.Fatalf("%s: ReadOnly = %v, want %v", tt.name, contract.ReadOnly, tt.readOnly)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./backend/internal/tools -run 'TestRegistry_Register_AssignsDefaultContract|TestRegistry_RegisterWithContract_PreservesExplicitContract|TestDefaultContractFor_KnownTools' -v
```

Expected: FAIL with undefined identifiers such as `ToolContract`, `SideEffectRead`, `RegisterWithContract`, or `Contract`.

- [ ] **Step 3: Create tool contract types**

Create `backend/internal/tools/contract.go`:

```go
package tools

// RiskLevel describes how dangerous a tool is if selected or repeated incorrectly.
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// SideEffectLevel describes whether a tool changes external or durable state.
type SideEffectLevel string

const (
	SideEffectNone        SideEffectLevel = "none"
	SideEffectRead        SideEffectLevel = "read"
	SideEffectWrite       SideEffectLevel = "write"
	SideEffectDestructive SideEffectLevel = "destructive"
)

// ToolErrorClass describes why a tool call failed.
type ToolErrorClass string

const (
	ToolErrorInvalidParams    ToolErrorClass = "invalid_params"
	ToolErrorTransient        ToolErrorClass = "transient"
	ToolErrorPermissionDenied ToolErrorClass = "permission_denied"
	ToolErrorBusinessRejected ToolErrorClass = "business_rejected"
	ToolErrorInternal         ToolErrorClass = "internal_error"
	ToolErrorNotFound         ToolErrorClass = "not_found"
	ToolErrorApprovalRequired ToolErrorClass = "approval_required"
)

// RetryPolicy declares which tool failures can be retried by ToolRunner.
type RetryPolicy struct {
	MaxAttempts     int
	BackoffMillis   int
	RetryErrorClasses []ToolErrorClass
}

// ParamSpec is a small runtime schema for parameters that should be checked before tool execution.
type ParamSpec struct {
	Name     string
	Type     string
	Required bool
}

// ToolContract is the runtime contract for a tool. It is the source of truth for
// visibility, risk, retry, trace, and future write-operation controls.
type ToolContract struct {
	Name                  string
	Version               string
	Description           string
	ReadOnly              bool
	Idempotent            bool
	RequiresApproval      bool
	RequiresIdempotencyKey bool
	SideEffect            SideEffectLevel
	RiskLevel             RiskLevel
	TimeoutMillis         int
	Retry                 RetryPolicy
	Params                []ParamSpec
}

// DefaultContractFor returns a conservative contract for existing tools.
func DefaultContractFor(name string) ToolContract {
	contract := ToolContract{
		Name:          name,
		Version:       "v1",
		ReadOnly:      true,
		Idempotent:    true,
		SideEffect:    SideEffectRead,
		RiskLevel:     RiskLow,
		TimeoutMillis: 10_000,
		Retry: RetryPolicy{
			MaxAttempts:   1,
			BackoffMillis: 0,
		},
	}

	switch name {
	case "bazi_calc", "yongshen", "dayun_analyzer", "bazi_liunian", "qimen_dunjia", "ziwei_calc", "ziwei_liunian":
		contract.SideEffect = SideEffectNone
	case "knowledge_search":
		contract.Params = []ParamSpec{
			{Name: "query", Type: "string", Required: true},
			{Name: "top_k", Type: "number", Required: false},
		}
		contract.Retry = RetryPolicy{
			MaxAttempts:       2,
			BackoffMillis:     100,
			RetryErrorClasses: []ToolErrorClass{ToolErrorTransient, ToolErrorInternal},
		}
	case "knowledge_catalog":
		contract.Params = []ParamSpec{
			{Name: "prefix", Type: "string", Required: false},
		}
		contract.Retry = RetryPolicy{
			MaxAttempts:       2,
			BackoffMillis:     100,
			RetryErrorClasses: []ToolErrorClass{ToolErrorTransient, ToolErrorInternal},
		}
	}

	return contract
}
```

- [ ] **Step 4: Modify registry to store contracts**

Update `backend/internal/tools/registry.go`:

```go
type Registry struct {
	mu        sync.RWMutex
	tools     map[string]Tool
	contracts map[string]ToolContract
}

func NewRegistry() *Registry {
	return &Registry{
		tools:     make(map[string]Tool),
		contracts: make(map[string]ToolContract),
	}
}

func (r *Registry) Register(t Tool) {
	r.RegisterWithContract(t, DefaultContractFor(t.Name()))
}

// RegisterWithContract 注册工具及其运行合同。
func (r *Registry) RegisterWithContract(t Tool, contract ToolContract) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if contract.Name == "" {
		contract.Name = t.Name()
	}
	if contract.Version == "" {
		contract.Version = "v1"
	}
	r.tools[t.Name()] = t
	r.contracts[t.Name()] = contract
}

// Contract 根据工具名称查询运行合同。
func (r *Registry) Contract(name string) (ToolContract, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	contract, ok := r.contracts[name]
	return contract, ok
}
```

Keep existing `Get`, `List`, and `DisplayName` unchanged.

- [ ] **Step 5: Run registry tests**

Run:

```bash
go test ./backend/internal/tools -run 'TestRegistry_Register_AssignsDefaultContract|TestRegistry_RegisterWithContract_PreservesExplicitContract|TestDefaultContractFor_KnownTools' -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/tools/contract.go backend/internal/tools/registry.go backend/internal/tools/registry_test.go
git commit -m "feat: add tool contracts to registry"
```

---

### Task 2: Add ToolRunner Result Envelope and Error Classification

**Files:**
- Create: `backend/internal/tools/runner.go`
- Test: `backend/internal/tools/runner_test.go`

- [ ] **Step 1: Write failing ToolRunner tests**

Create `backend/internal/tools/runner_test.go` with:

```go
package tools

import (
	"context"
	"errors"
	"testing"
)

type runnerTool struct {
	name string
	fn   func(context.Context, map[string]any) (any, error)
}

func (t runnerTool) Name() string { return t.name }
func (t runnerTool) Description() string { return "runner test tool" }
func (t runnerTool) Label() string { return "Runner Test Tool" }
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
		SideEffect: SideEffectNone,
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./backend/internal/tools -run 'TestToolRunner|TestClassifyToolError' -v
```

Expected: FAIL with undefined identifiers such as `NewToolRunner`, `ToolRunRequest`, and `ToolErrorClass`.

- [ ] **Step 3: Implement runner envelope and classifier**

Create `backend/internal/tools/runner.go`:

```go
package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// ToolRunStatus is the normalized execution result status for a tool call.
type ToolRunStatus string

const (
	ToolRunStatusOK       ToolRunStatus = "ok"
	ToolRunStatusFallback ToolRunStatus = "fallback"
	ToolRunStatusBlocked  ToolRunStatus = "blocked"
	ToolRunStatusError    ToolRunStatus = "error"
)

// ToolRunRequest is a normalized request to execute a registered tool.
type ToolRunRequest struct {
	ToolName       string
	Params         map[string]any
	DecisionSource string
	IdempotencyKey string
	Approved       bool
}

// ToolRunResult is a structured envelope for every tool execution.
type ToolRunResult struct {
	ToolName       string
	Version        string
	Status         ToolRunStatus
	Data           any
	Error          error
	ErrorClass     ToolErrorClass
	Retryable      bool
	Fallback       bool
	Attempts       int
	DecisionSource string
	DurationMs     int64
}

// ToolRunner executes tools through contracts, policy, retry, and tracing.
type ToolRunner struct {
	reg *Registry
}

// NewToolRunner creates a runner backed by the shared tool registry.
func NewToolRunner(reg *Registry) *ToolRunner {
	return &ToolRunner{reg: reg}
}

// Run executes one tool and always returns a structured result envelope.
func (r *ToolRunner) Run(ctx context.Context, req ToolRunRequest) ToolRunResult {
	start := time.Now()
	result := ToolRunResult{
		ToolName:       req.ToolName,
		Status:         ToolRunStatusError,
		DecisionSource: req.DecisionSource,
	}

	if r == nil || r.reg == nil {
		result.ErrorClass = ToolErrorNotFound
		result.Error = fmt.Errorf("tool registry is nil")
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	tool, ok := r.reg.Get(req.ToolName)
	if !ok {
		result.ErrorClass = ToolErrorNotFound
		result.Error = fmt.Errorf("tool %s not found", req.ToolName)
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	contract, ok := r.reg.Contract(req.ToolName)
	if !ok {
		contract = DefaultContractFor(req.ToolName)
	}
	result.Version = contract.Version

	span := tracing.SpanFromContext(ctx, req.ToolName, tracing.KindTool)
	span.SetAttribute("tool.name", req.ToolName)
	span.SetAttribute("tool.version", contract.Version)
	span.SetAttribute("tool.risk_level", string(contract.RiskLevel))
	span.SetAttribute("tool.side_effect", string(contract.SideEffect))
	span.SetAttribute("tool.read_only", contract.ReadOnly)
	span.SetAttribute("tool.idempotent", contract.Idempotent)
	span.SetAttribute("tool.decision_source", req.DecisionSource)
	defer func() {
		span.SetAttribute("tool.attempts", result.Attempts)
		span.SetAttribute("tool.status", string(result.Status))
		if result.ErrorClass != "" {
			span.SetAttribute("tool.error_class", string(result.ErrorClass))
		}
		span.End()
	}()

	if err := validateParams(contract, req.Params); err != nil {
		result.Error = err
		result.ErrorClass = ToolErrorInvalidParams
		result.DurationMs = time.Since(start).Milliseconds()
		span.RecordError(err)
		span.SetStatus("error")
		return result
	}

	if contract.RequiresApproval && !req.Approved {
		err := fmt.Errorf("tool %s requires approval", req.ToolName)
		result.Status = ToolRunStatusBlocked
		result.Error = err
		result.ErrorClass = ToolErrorApprovalRequired
		result.DurationMs = time.Since(start).Milliseconds()
		span.RecordError(err)
		span.SetStatus("error")
		return result
	}

	attempts := normalizedAttempts(contract.Retry.MaxAttempts)
	for attempt := 1; attempt <= attempts; attempt++ {
		result.Attempts = attempt
		data, err := tool.Execute(ctx, req.Params)
		if err == nil && data != nil {
			result.Status = ToolRunStatusOK
			result.Data = data
			result.DurationMs = time.Since(start).Milliseconds()
			return result
		}

		if err == nil {
			err = fmt.Errorf("tool %s returned nil result", req.ToolName)
		}
		result.Error = err
		result.ErrorClass = ClassifyToolError(err)
		result.Retryable = canRetry(result.ErrorClass, contract.Retry)
		if !result.Retryable || attempt == attempts {
			break
		}
		if contract.Retry.BackoffMillis > 0 {
			time.Sleep(time.Duration(contract.Retry.BackoffMillis) * time.Millisecond)
		}
	}

	result.DurationMs = time.Since(start).Milliseconds()
	span.RecordError(result.Error)
	span.SetStatus("error")
	return result
}

// ClassifyToolError maps raw errors into governance-level classes.
func ClassifyToolError(err error) ToolErrorClass {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return ToolErrorTransient
	}
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "required"), strings.Contains(text, "invalid"), strings.Contains(text, "out of range"):
		return ToolErrorInvalidParams
	case strings.Contains(text, "permission"), strings.Contains(text, "unauthorized"), strings.Contains(text, "forbidden"):
		return ToolErrorPermissionDenied
	case strings.Contains(text, "business rejected"), strings.Contains(text, "rejected"):
		return ToolErrorBusinessRejected
	case strings.Contains(text, "timeout"), strings.Contains(text, "temporary"), strings.Contains(text, "rate limit"):
		return ToolErrorTransient
	default:
		return ToolErrorInternal
	}
}

func validateParams(contract ToolContract, params map[string]any) error {
	for _, spec := range contract.Params {
		value, ok := params[spec.Name]
		if spec.Required && (!ok || value == nil || value == "") {
			return fmt.Errorf("%s is required", spec.Name)
		}
		if !ok || value == nil {
			continue
		}
		switch spec.Type {
		case "string":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("%s must be string", spec.Name)
			}
		case "number":
			switch value.(type) {
			case int, int64, float64, float32:
			default:
				return fmt.Errorf("%s must be number", spec.Name)
			}
		case "object":
			if _, ok := value.(map[string]any); !ok {
				return fmt.Errorf("%s must be object", spec.Name)
			}
		}
	}
	return nil
}

func normalizedAttempts(maxAttempts int) int {
	if maxAttempts <= 0 {
		return 1
	}
	return maxAttempts
}

func canRetry(class ToolErrorClass, policy RetryPolicy) bool {
	for _, allowed := range policy.RetryErrorClasses {
		if allowed == class {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run ToolRunner tests**

Run:

```bash
go test ./backend/internal/tools -run 'TestToolRunner|TestClassifyToolError' -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/tools/runner.go backend/internal/tools/runner_test.go
git commit -m "feat: add governed tool runner"
```

---

### Task 3: Add Retry, Timeout, and Approval Tests

**Files:**
- Modify: `backend/internal/tools/runner.go`
- Modify: `backend/internal/tools/runner_test.go`

- [ ] **Step 1: Add failing retry and approval tests**

Append to `backend/internal/tools/runner_test.go`:

```go
func TestToolRunner_RetriesTransientErrors(t *testing.T) {
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
```

- [ ] **Step 2: Run tests to verify retry test fails before policy is complete**

Run:

```bash
go test ./backend/internal/tools -run 'TestToolRunner_RetriesTransientErrors|TestToolRunner_DoesNotRetryInvalidParams|TestToolRunner_BlocksApprovalRequiredTool' -v
```

Expected: If Task 2 was implemented exactly, these may already PASS. If any fail, fix `Run`, `ClassifyToolError`, or `canRetry` to match the tests.

- [ ] **Step 3: Add timeout enforcement**

Modify the execution block inside `ToolRunner.Run` so calls use a timeout when `contract.TimeoutMillis > 0`:

```go
execCtx := ctx
cancel := func() {}
if contract.TimeoutMillis > 0 {
	execCtx, cancel = context.WithTimeout(ctx, time.Duration(contract.TimeoutMillis)*time.Millisecond)
}
data, err := tool.Execute(execCtx, req.Params)
cancel()
```

Replace the existing `tool.Execute(ctx, req.Params)` line inside the attempt loop with this snippet.

- [ ] **Step 4: Add timeout test**

Append to `backend/internal/tools/runner_test.go`:

```go
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
```

- [ ] **Step 5: Run full tools tests**

Run:

```bash
go test ./backend/internal/tools -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/tools/runner.go backend/internal/tools/runner_test.go
git commit -m "feat: enforce tool retry timeout and approval policy"
```

---

### Task 4: Wire ToolRunner into Runtime Prefill

**Files:**
- Modify: `backend/internal/runtime/executor.go`
- Test: `backend/internal/runtime/executor_test.go`

- [ ] **Step 1: Add failing runtime test for governed tool execution**

Add this test to `backend/internal/runtime/executor_test.go`:

```go
func TestExecutorCallTool_UsesToolRunnerInvalidParamGuard(t *testing.T) {
	reg := tools.NewRegistry()
	reg.RegisterWithContract(fakeRuntimeTool{
		name: "needs_query",
		run: func(context.Context, map[string]any) (any, error) {
			t.Fatal("tool must not execute when ToolRunner rejects params")
			return nil, nil
		},
	}, tools.ToolContract{
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

	ex := &Executor{
		reg:        reg,
		toolRunner: tools.NewToolRunner(reg),
	}

	got := ex.callTool(context.Background(), "needs_query", map[string]any{})
	if got != nil {
		t.Fatalf("callTool = %#v, want nil", got)
	}
}
```

If `fakeRuntimeTool` does not exist in `executor_test.go`, add:

```go
type fakeRuntimeTool struct {
	name string
	run  func(context.Context, map[string]any) (any, error)
}

func (t fakeRuntimeTool) Name() string { return t.name }
func (t fakeRuntimeTool) Description() string { return "fake runtime tool" }
func (t fakeRuntimeTool) Label() string { return "Fake Runtime Tool" }
func (t fakeRuntimeTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	return t.run(ctx, params)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./backend/internal/runtime -run TestExecutorCallTool_UsesToolRunnerInvalidParamGuard -v
```

Expected: FAIL because `Executor` has no `toolRunner` field or `callTool` still directly executes tools.

- [ ] **Step 3: Add ToolRunner to Executor**

Modify `backend/internal/runtime/executor.go` `Executor` struct:

```go
type Executor struct {
	reg                *tools.Registry
	toolRunner         *tools.ToolRunner
	flashChat          llm.Chat
	summarizerModel    einomodel.ToolCallingChatModel
	specialistRegistry *specialists.Registry
	builder            *AgentBuilder
	manager            *Manager
	llmModel           string
	historyLimit       int
	orchestrationGraph compose.Runnable[string, string]
	cpStore            compose.CheckPointStore
	router             intent.Router
}
```

Modify `NewExecutor` return value:

```go
	return &Executor{
		reg:                reg,
		toolRunner:         tools.NewToolRunner(reg),
		flashChat:          flashChat,
		summarizerModel:    summarizerModel,
		specialistRegistry: sr,
		builder:            NewAgentBuilder(model, reg, flashChat, summarizerModel, cfg.Builder),
		llmModel:           cfg.LLMModel,
		historyLimit:       cfg.HistoryLimit,
		manager:            NewManager(flashChat),
		orchestrationGraph: graph,
		router:             cfg.Router,
	}, nil
```

- [ ] **Step 4: Replace direct tool execution in `callTool`**

Replace `callTool` in `backend/internal/runtime/executor.go` with:

```go
func (e *Executor) callTool(ctx context.Context, name string, params map[string]any) map[string]any {
	if e == nil || e.toolRunner == nil {
		return nil
	}
	result := e.toolRunner.Run(ctx, tools.ToolRunRequest{
		ToolName:       name,
		Params:         params,
		DecisionSource: "prefill",
	})
	if result.Status != tools.ToolRunStatusOK || result.Data == nil {
		if result.Error != nil {
			log.Printf("prefill: tool %s failed: %v", name, result.Error)
		}
		return nil
	}
	m, _ := result.Data.(map[string]any)
	return m
}
```

- [ ] **Step 5: Run focused runtime test**

Run:

```bash
go test ./backend/internal/runtime -run TestExecutorCallTool_UsesToolRunnerInvalidParamGuard -v
```

Expected: PASS.

- [ ] **Step 6: Run runtime package tests**

Run:

```bash
go test ./backend/internal/runtime -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/runtime/executor.go backend/internal/runtime/executor_test.go
git commit -m "feat: route runtime tool calls through tool runner"
```

---

### Task 5: Register Explicit Contracts for Existing Tools

**Files:**
- Modify: `backend/internal/container/container.go`
- Test: `backend/internal/container/container_test.go`

- [ ] **Step 1: Add container test for explicit contracts**

Add to `backend/internal/container/container_test.go`:

```go
func TestBuildContainer_RegistersToolContracts(t *testing.T) {
	c := BuildContainer()
	required := map[string]tools.SideEffectLevel{
		"bazi_calc":         tools.SideEffectNone,
		"yongshen":          tools.SideEffectNone,
		"dayun_analyzer":    tools.SideEffectNone,
		"bazi_liunian":      tools.SideEffectNone,
		"qimen_dunjia":      tools.SideEffectNone,
		"ziwei_calc":        tools.SideEffectNone,
		"ziwei_liunian":     tools.SideEffectNone,
		"knowledge_search":  tools.SideEffectRead,
		"knowledge_catalog": tools.SideEffectRead,
	}

	for name, sideEffect := range required {
		contract, ok := c.Tools.Contract(name)
		if !ok {
			t.Fatalf("missing contract for %s", name)
		}
		if contract.SideEffect != sideEffect {
			t.Fatalf("%s SideEffect = %q, want %q", name, contract.SideEffect, sideEffect)
		}
		if !contract.ReadOnly {
			t.Fatalf("%s must be read-only in current runtime", name)
		}
	}
}
```

Add this field to the `Container` struct:

```go
Tools *tools.Registry
```

Set it in `BuildContainer` result:

```go
Tools: reg,
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./backend/internal/container -run TestBuildContainer_RegistersToolContracts -v
```

Expected: FAIL until `Container` exposes the registry or explicit contracts are registered.

- [ ] **Step 3: Replace simple registrations with explicit contracts**

Modify `backend/internal/container/container.go` tool registration block:

```go
	reg := tools.NewRegistry()
	reg.RegisterWithContract(&baziCalc.CalcTool{}, tools.DefaultContractFor("bazi_calc"))
	reg.RegisterWithContract(&baziCalc.YongShenTool{}, tools.DefaultContractFor("yongshen"))
	reg.RegisterWithContract(&baziCalc.DayunAnalyzer{}, tools.DefaultContractFor("dayun_analyzer"))
	reg.RegisterWithContract(&baziCalc.BaziLiuNianTool{}, tools.DefaultContractFor("bazi_liunian"))
	reg.RegisterWithContract(&qimenTools.Tool{}, tools.DefaultContractFor("qimen_dunjia"))
	reg.RegisterWithContract(&ziweiTools.ZiWeiCalcTool{}, tools.DefaultContractFor("ziwei_calc"))
	reg.RegisterWithContract(&ziweiTools.ZiWeiLiuNianTool{}, tools.DefaultContractFor("ziwei_liunian"))
	reg.RegisterWithContract(tools.NewKnowledgeSearchTool(mcpClient), tools.DefaultContractFor("knowledge_search"))
	reg.RegisterWithContract(tools.NewKnowledgeCatalogTool(mcpClient), tools.DefaultContractFor("knowledge_catalog"))
```

- [ ] **Step 4: Run container test**

Run:

```bash
go test ./backend/internal/container -run TestBuildContainer_RegistersToolContracts -v
```

Expected: PASS.

- [ ] **Step 5: Run container package tests**

Run:

```bash
go test ./backend/internal/container -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/container/container.go backend/internal/container/container_test.go
git commit -m "feat: register explicit tool contracts"
```

---

### Task 6: Add Trace Coverage for Tool Governance Metadata

**Files:**
- Modify: `backend/internal/tracing/turn_trace.go`
- Test: `backend/internal/tools/runner_test.go`

- [ ] **Step 1: Add trace metadata test**

Append to `backend/internal/tools/runner_test.go`:

```go
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
```

Add this import to `runner_test.go`:

```go
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
```

- [ ] **Step 2: Run trace test**

Run:

```bash
go test ./backend/internal/tools -run TestToolRunner_WritesGovernanceTraceAttributes -v
```

Expected: PASS if Task 2 trace attributes were implemented. If it fails, add the missing `span.SetAttribute` calls in `runner.go`.

- [ ] **Step 3: Add user-facing label for governance spans**

Modify `backend/internal/tracing/turn_trace.go` `stepLabel` map if a generic span name is introduced:

```go
		"tool_runner":          "工具治理",
```

If each span keeps the concrete tool name, no new label is required for existing known tools.

- [ ] **Step 4: Run tracing and tools tests**

Run:

```bash
go test ./backend/internal/tools ./backend/internal/tracing -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/tools/runner_test.go backend/internal/tools/runner.go backend/internal/tracing/turn_trace.go
git commit -m "test: cover tool governance trace metadata"
```

---

### Task 7: Document Architecture and Operational Boundaries

**Files:**
- Modify: `docs/architecture.md`
- Modify: `PROGRESS.md`
- Create: `docs/tool-governance.md`

- [ ] **Step 1: Create tool governance document**

Create `docs/tool-governance.md`:

```markdown
# Tool Governance

本文说明当前工具治理层的真实边界。

## 目标

工具治理层把工具从“业务代码里的函数调用”升级为“运行时受治理的执行资源”。

它负责：

- 工具合同：名称、版本、风险级别、只读/写入、副作用、幂等、参数 schema。
- 执行控制：参数校验、超时、错误分类、是否重试、是否阻断。
- 观测审计：工具名、版本、参数摘要、耗时、错误类别、重试次数、决策来源。
- 副作用预留：写操作必须通过审批、幂等键和服务端去重。

## 当前实现

当前生产形态先覆盖 runtime-owned deterministic tools：

- `bazi_calc`
- `yongshen`
- `dayun_analyzer`
- `bazi_liunian`
- `qimen_dunjia`
- `ziwei_calc`
- `ziwei_liunian`
- `knowledge_search`
- `knowledge_catalog`

这些工具都通过 `ToolContract` 声明风险和副作用，并通过 `ToolRunner` 执行。

## 当前不做

- 不把所有工具改成 MCP。
- 不把只读工具强行接审批。
- 不把 final guard 当成工具治理替代品。
- 不在没有真实写操作前实现重型补偿事务。

## 副作用工具接入规则

未来新增写操作工具时，必须满足：

- `ReadOnly=false`
- `SideEffect=write` 或 `SideEffect=destructive`
- `RequiresApproval=true`
- `RequiresIdempotencyKey=true`
- trace 中必须出现审批 ID、幂等键摘要、工具版本和结果状态

未满足这些合同字段的写操作不得注册到 runtime 工具表。
```

- [ ] **Step 2: Update architecture doc**

In `docs/architecture.md`, under “确定性执行层”, add:

```markdown
### Tool Governance

`ToolRunner` 位于 runtime 与具体工具之间。它统一执行工具合同、参数校验、错误分类、超时、重试和 trace 打点。

当前工具治理只覆盖 runtime-owned deterministic tools；specialist ADK 内部工具调用保持现状，后续单独迁移。

关键文件：

- `backend/internal/tools/contract.go`
- `backend/internal/tools/runner.go`
- `backend/internal/runtime/executor.go`
```

- [ ] **Step 3: Update progress**

In `PROGRESS.md`, add a stable capability entry:

```markdown
- 已新增工具治理第一版：`ToolContract` 统一描述工具版本、风险、副作用、幂等、参数 schema 和重试策略；runtime-owned deterministic tools 已通过 `ToolRunner` 执行，trace 中可见工具版本、风险等级、错误类别、重试次数和决策来源。
```

Add a decision record:

```markdown
- 工具治理先覆盖 runtime-owned deterministic tools，不一次性迁移 specialist ADK 内部工具调用；未来接入写操作工具时，必须先通过 `ToolContract` 声明审批、幂等和副作用等级。
```

- [ ] **Step 4: Run docs grep check**

Run:

```bash
rg -n "ToolRunner|ToolContract|工具治理|tool-governance" docs PROGRESS.md
```

Expected: output includes `docs/architecture.md`, `docs/tool-governance.md`, and `PROGRESS.md`.

- [ ] **Step 5: Commit**

```bash
git add docs/architecture.md docs/tool-governance.md PROGRESS.md
git commit -m "docs: document tool governance runtime"
```

---

### Task 8: Full Verification and Regression Gate

**Files:**
- Verify: `backend/internal/tools/...`
- Verify: `backend/internal/runtime/...`
- Verify: `backend/internal/container/...`
- Verify: `docs/tool-governance.md`

- [ ] **Step 1: Run focused backend tests**

Run:

```bash
go test ./backend/internal/tools ./backend/internal/runtime ./backend/internal/container ./backend/internal/tracing -v
```

Expected: PASS.

- [ ] **Step 2: Run full backend tests**

Run:

```bash
go test ./backend/... -v
```

Expected: PASS.

- [ ] **Step 3: Run frontend type and build checks only if backend touched SSE payloads**

This plan does not change SSE payload contracts. If an implementation changes `Event` data or frontend trace panels, run:

```bash
cd web && npx vue-tsc --noEmit
cd web && npm run build
```

Expected: PASS.

- [ ] **Step 4: Inspect git diff**

Run:

```bash
git diff -- backend/internal/tools backend/internal/runtime backend/internal/container backend/internal/tracing docs PROGRESS.md
```

Expected:

- `tools.Registry` remains backward-compatible with `Register(t Tool)`.
- `Executor.callTool` no longer calls concrete tools directly.
- Tool governance tests cover success, invalid params, retry, timeout, approval blocking, and trace attributes.
- Docs describe current coverage and explicitly state specialist ADK tool migration is separate.

- [ ] **Step 5: Commit verification-only fixes**

If verification required small fixes:

```bash
git add backend/internal/tools backend/internal/runtime backend/internal/container backend/internal/tracing docs PROGRESS.md
git commit -m "test: verify tool governance integration"
```

If no changes were required, skip this commit.

---

## Follow-up Plan Required After This One

Create a separate plan for specialist ADK tool migration after this plan lands.

That plan should cover:

- How Eino ADK tool adapters receive `ToolContract`.
- How specialist tool-call events enter `ToolRunner`.
- How `knowledge_search` inside authority-first BaZi graph should preserve its current fallback behavior while adding governance metadata.
- Whether MCP `retrieve_passages` should remain behind `knowledge_search` or become directly visible as a governed MCP tool.

## Self-Review

Spec coverage:

- Tool contract: Task 1 and Task 5.
- Unified execution: Task 2 and Task 4.
- Error classification and retry: Task 2 and Task 3.
- Timeout and approval hooks: Task 3.
- Trace metadata and replay foundation: Task 6.
- Documentation and progress: Task 7.
- Regression gate: Task 8.

Placeholder scan:

- No forbidden placeholder markers.
- No unnamed validation steps.
- Every code-changing task includes exact code snippets or exact replacement snippets.

Type consistency:

- `ToolContract`, `RetryPolicy`, `ParamSpec`, `ToolRunner`, `ToolRunRequest`, `ToolRunResult`, `ToolErrorClass`, `ToolRunStatus`, `SideEffectLevel`, and `RiskLevel` are introduced before use.
- Runtime integration uses `tools.NewToolRunner(reg)` and `tools.ToolRunRequest`.
- Tests refer to the same exported names as the implementation tasks.
