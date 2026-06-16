package container

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestBuildContainer_WiresSupervisorAndSpecialists(t *testing.T) {
	t.Setenv("LLM_API_KEY", "test-key")
	assertContainerWiring(t, BuildContainer())
}

func TestBuildContainer_UsesDedicatedFlashClient(t *testing.T) {
	t.Setenv("LLM_API_KEY", "test-key")
	t.Setenv("LLM_FLASH_MODEL", "")

	c := BuildContainer()

	handlerValue := reflect.ValueOf(c.Handler).Elem()
	orchField := handlerValue.FieldByName("orch")
	orchValue := reflect.NewAt(orchField.Type(), unsafe.Pointer(orchField.UnsafeAddr())).Elem().Elem()

	runtimeField := reflect.NewAt(orchValue.FieldByName("runtime").Type(), unsafe.Pointer(orchValue.FieldByName("runtime").UnsafeAddr())).Elem()
	if runtimeField.IsNil() {
		t.Fatal("expected runtime executor to be present")
	}
	runtimeValue := runtimeField.Elem()
	llmField := reflect.NewAt(runtimeValue.FieldByName("llm").Type(), unsafe.Pointer(runtimeValue.FieldByName("llm").UnsafeAddr())).Elem()
	flashField := reflect.NewAt(orchValue.FieldByName("flash").Type(), unsafe.Pointer(orchValue.FieldByName("flash").UnsafeAddr())).Elem()

	if llmField.IsNil() || flashField.IsNil() {
		t.Fatal("expected llm and flash clients to be present")
	}
	if llmField.Interface() == flashField.Interface() {
		t.Fatal("expected flash client to be a dedicated instance, not the same as main llm client")
	}
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
	runtimeValue := reflect.NewAt(runtimeField.Type(), unsafe.Pointer(runtimeField.UnsafeAddr())).Elem().Elem()

	baziField := runtimeValue.FieldByName("baziSp")
	if baziField.IsNil() {
		t.Fatal("expected bazi specialist to be wired")
	}

	qimenField := runtimeValue.FieldByName("qimenSp")
	if qimenField.IsNil() {
		t.Fatal("expected qimen specialist to be wired")
	}
}
