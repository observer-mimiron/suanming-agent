// This test file belongs to the dependency wiring layer.
// It verifies dependency assembly behavior and protects the related contract from regressions.
// It assembles dependencies; business rules stay in orchestrator, runtime, and tools.
package container

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	appRuntime "github.com/observer-mimiron/suanming-agent/internal/runtime"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/specialists/bazi"
	baziAdapter "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/adapter"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
)

func testProjectRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func TestResolveProjectPathFrom_UsesWorkspaceRoot(t *testing.T) {
	root := testProjectRoot(t)

	gotFromRoot, err := resolveProjectPathFrom(root, "data/sessions")
	if err != nil {
		t.Fatalf("resolveProjectPathFrom(root): %v", err)
	}
	wantSessions := filepath.Join(root, "data", "sessions")
	if gotFromRoot != wantSessions {
		t.Fatalf("resolveProjectPathFrom(root) = %q, want %q", gotFromRoot, wantSessions)
	}

	gotFromBackend, err := resolveProjectPathFrom(filepath.Join(root, "backend"), "logs/debug")
	if err != nil {
		t.Fatalf("resolveProjectPathFrom(backend): %v", err)
	}
	wantDebug := filepath.Join(root, "logs", "debug")
	if gotFromBackend != wantDebug {
		t.Fatalf("resolveProjectPathFrom(backend) = %q, want %q", gotFromBackend, wantDebug)
	}

	gotFromContainer, err := resolveProjectPathFrom(filepath.Join(root, "backend", "internal", "container"), "logs/traces")
	if err != nil {
		t.Fatalf("resolveProjectPathFrom(container): %v", err)
	}
	wantTraces := filepath.Join(root, "logs", "traces")
	if gotFromContainer != wantTraces {
		t.Fatalf("resolveProjectPathFrom(container) = %q, want %q", gotFromContainer, wantTraces)
	}
}

func TestBuildContainer_WiresSupervisorAndSpecialists(t *testing.T) {
	t.Setenv("LLM_API_KEY", "test-key")
	c := BuildContainer()
	assertContainerWiring(t, c)
}

func TestBuildContainer_WiresADKSupervisorEngine(t *testing.T) {
	t.Setenv("LLM_API_KEY", "test-key")

	c := BuildContainer()

	handlerValue := reflect.ValueOf(c.Handler).Elem()
	orchField := handlerValue.FieldByName("orch")
	orchValue := reflect.NewAt(orchField.Type(), unsafe.Pointer(orchField.UnsafeAddr())).Elem().Elem()

	supervisorField := reflect.NewAt(orchValue.FieldByName("supervisor").Type(), unsafe.Pointer(orchValue.FieldByName("supervisor").UnsafeAddr())).Elem()
	if supervisorField.IsNil() {
		t.Fatal("expected supervisor client to be wired")
	}

	supervisorValue := supervisorField.Elem()
	if supervisorValue.Kind() == reflect.Ptr {
		supervisorValue = supervisorValue.Elem()
	}
	routeEngineField := reflect.NewAt(supervisorValue.FieldByName("routeEngine").Type(), unsafe.Pointer(supervisorValue.FieldByName("routeEngine").UnsafeAddr())).Elem()
	if routeEngineField.IsNil() {
		t.Fatal("expected adk route engine to be wired")
	}
}

func TestBuildContainer_RegistersBaziCompositeAndOtherDirectRunners(t *testing.T) {
	t.Setenv("LLM_API_KEY", "test-key")

	c := BuildContainer()
	registry := extractSpecialistRegistry(t, c)

	baziRunner, ok := registry.RunnerFor("bazi")
	if !ok {
		t.Fatal("expected bazi runner")
	}
	composite, ok := baziRunner.(*bazi.Runner)
	if !ok {
		t.Fatalf("expected bazi composite runner, got %T", baziRunner)
	}
	if _, ok := composite.Primary.(*baziAdapter.Runner); !ok {
		t.Fatalf("expected bazi primary graph runner, got %T", composite.Primary)
	}
	if _, ok := composite.Support.(*appRuntime.ADKSpecialistRunner); !ok {
		t.Fatalf("expected bazi support ADK runner, got %T", composite.Support)
	}

	for _, domain := range []string{"qimen", "ziwei"} {
		runner, ok := registry.RunnerFor(domain)
		if !ok {
			t.Fatalf("expected non-nil specialist runner for %s", domain)
		}
		if !strings.HasSuffix(reflect.TypeOf(runner).String(), ".ADKSpecialistRunner") {
			t.Fatalf("expected %s runner to use direct ADK specialist path, got %T", domain, runner)
		}
	}
}

func TestBuildContainer_RegistersToolContracts(t *testing.T) {
	t.Setenv("LLM_API_KEY", "test-key")

	c := BuildContainer()
	if c.Tools == nil {
		t.Fatal("expected container to expose tool registry")
	}

	tests := []struct {
		name       string
		sideEffect tools.SideEffectLevel
		readOnly   bool
	}{
		{name: "bazi_calc", sideEffect: tools.SideEffectNone, readOnly: true},
		{name: "yongshen", sideEffect: tools.SideEffectNone, readOnly: true},
		{name: "dayun_analyzer", sideEffect: tools.SideEffectNone, readOnly: true},
		{name: "bazi_liunian", sideEffect: tools.SideEffectNone, readOnly: true},
		{name: "qimen_dunjia", sideEffect: tools.SideEffectNone, readOnly: true},
		{name: "ziwei_calc", sideEffect: tools.SideEffectNone, readOnly: true},
		{name: "ziwei_liunian", sideEffect: tools.SideEffectNone, readOnly: true},
		{name: "knowledge_search", sideEffect: tools.SideEffectRead, readOnly: true},
		{name: "knowledge_catalog", sideEffect: tools.SideEffectRead, readOnly: true},
	}

	for _, tt := range tests {
		if _, ok := c.Tools.Get(tt.name); !ok {
			t.Fatalf("expected tool %s to be registered", tt.name)
		}
		contract, ok := c.Tools.Contract(tt.name)
		if !ok {
			t.Fatalf("expected contract for tool %s", tt.name)
		}
		if contract.SideEffect != tt.sideEffect {
			t.Fatalf("%s SideEffect = %q, want %q", tt.name, contract.SideEffect, tt.sideEffect)
		}
		if contract.ReadOnly != tt.readOnly {
			t.Fatalf("%s ReadOnly = %v, want %v", tt.name, contract.ReadOnly, tt.readOnly)
		}
		if contract.Version == "" {
			t.Fatalf("%s contract version must not be empty", tt.name)
		}
	}
}

func assertContainerWiring(t *testing.T, c *Container) {
	t.Helper()

	handlerValue := reflect.ValueOf(c.Handler).Elem()
	orchField := handlerValue.FieldByName("orch")
	if !orchField.IsValid() {
		t.Fatal("chat handler missing orch field")
	}

	orchValue := reflect.NewAt(orchField.Type(), unsafe.Pointer(orchField.UnsafeAddr())).Elem().Elem()

	supervisorField := orchValue.FieldByName("supervisor")
	if supervisorField.IsNil() {
		t.Fatal("expected supervisor client to be wired")
	}

	runtimeField := orchValue.FieldByName("runtime")
	if runtimeField.IsNil() {
		t.Fatal("expected runtime executor to be wired")
	}

	// 验证 specialist registry 已注册配置
	runtimeValue := reflect.NewAt(runtimeField.Type(), unsafe.Pointer(runtimeField.UnsafeAddr())).Elem().Elem()
	srField := runtimeValue.FieldByName("specialistRegistry")
	if srField.IsNil() {
		t.Fatal("expected specialist registry to be wired")
	}
}

func extractSpecialistRegistry(t *testing.T, c *Container) *specialists.Registry {
	t.Helper()

	handlerValue := reflect.ValueOf(c.Handler).Elem()
	orchField := handlerValue.FieldByName("orch")
	if !orchField.IsValid() {
		t.Fatal("chat handler missing orch field")
	}

	orchValue := reflect.NewAt(orchField.Type(), unsafe.Pointer(orchField.UnsafeAddr())).Elem().Elem()
	runtimeField := orchValue.FieldByName("runtime")
	if runtimeField.IsNil() {
		t.Fatal("expected runtime executor to be wired")
	}

	runtimeValue := reflect.NewAt(runtimeField.Type(), unsafe.Pointer(runtimeField.UnsafeAddr())).Elem().Elem()
	srField := runtimeValue.FieldByName("specialistRegistry")
	if srField.IsNil() {
		t.Fatal("expected specialist registry to be wired")
	}

	registry, ok := reflect.NewAt(srField.Type(), unsafe.Pointer(srField.UnsafeAddr())).Elem().Interface().(*specialists.Registry)
	if !ok || registry == nil {
		t.Fatal("failed to extract specialist registry")
	}
	return registry
}

func TestBuildContainer_RouterModeOff(t *testing.T) {
	t.Setenv("LLM_API_KEY", "test-key")
	t.Setenv("ROUTER_MODE", "off")
	t.Setenv("EMBEDDING_API_KEY", "")

	c := BuildContainer()
	if c == nil {
		t.Fatal("container is nil")
	}
	if c.Config.RouterMode != "off" {
		t.Fatalf("RouterMode = %q, want off", c.Config.RouterMode)
	}
}

func TestBuildContainer_UsesWorkspaceRootPaths(t *testing.T) {
	t.Setenv("LLM_API_KEY", "test-key")
	c := BuildContainer()

	wantDebugDir := filepath.Join(testProjectRoot(t), "logs", "debug")
	if c.DebugDir != wantDebugDir {
		t.Fatalf("DebugDir = %q, want %q", c.DebugDir, wantDebugDir)
	}

	wantTraceDir := filepath.Join(testProjectRoot(t), "logs", "traces")
	if c.TraceDir != wantTraceDir {
		t.Fatalf("TraceDir = %q, want %q", c.TraceDir, wantTraceDir)
	}
}

func TestBuildContainer_ExposesTraceStatus(t *testing.T) {
	t.Setenv("LLM_API_KEY", "test-key")
	t.Setenv("DEBUG_TRACE", "1")
	t.Setenv("OTEL_ENABLED", "0")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "0")

	c := BuildContainer()

	if !c.TracePersistenceEnabled {
		t.Fatal("expected TracePersistenceEnabled to be true when DEBUG_TRACE=1")
	}
	if c.OTelEnabled {
		t.Fatal("expected OTelEnabled to remain false when OTEL is not configured")
	}
	if c.OTelEndpoint != "" {
		t.Fatalf("OTelEndpoint = %q, want empty when OTEL is not configured", c.OTelEndpoint)
	}
}

func TestContainer_TraceStartupLines(t *testing.T) {
	c := &Container{
		TraceDir:                "D:/trace-dir",
		TracePersistenceEnabled: true,
		OTelEnabled:             true,
		OTelEndpoint:            "https://example.com/v1/traces",
	}

	lines := c.TraceStartupLines()
	if len(lines) != 2 {
		t.Fatalf("len(lines) = %d, want 2", len(lines))
	}
	if got, want := lines[0], "[tracing] local TurnTrace persistence: enabled -> D:/trace-dir"; got != want {
		t.Fatalf("lines[0] = %q, want %q", got, want)
	}
	if got, want := lines[1], "[tracing] OTel export mirror: enabled -> https://example.com/v1/traces"; got != want {
		t.Fatalf("lines[1] = %q, want %q", got, want)
	}
}

func TestContainer_TraceStartupLines_Disabled(t *testing.T) {
	c := &Container{TraceDir: "D:/trace-dir"}

	lines := c.TraceStartupLines()
	if len(lines) != 2 {
		t.Fatalf("len(lines) = %d, want 2", len(lines))
	}
	if got, want := lines[0], "[tracing] local TurnTrace persistence: disabled (set DEBUG_TRACE=1 to write JSON files) -> D:/trace-dir"; got != want {
		t.Fatalf("lines[0] = %q, want %q", got, want)
	}
	if got, want := lines[1], "[tracing] OTel export mirror: disabled"; got != want {
		t.Fatalf("lines[1] = %q, want %q", got, want)
	}
}
