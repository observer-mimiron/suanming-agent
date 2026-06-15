package container

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestBuildContainer_WiresSupervisorAndSpecialists(t *testing.T) {
	for _, backend := range []string{"native", "eino"} {
		t.Run(backend, func(t *testing.T) {
			t.Setenv("LLM_BACKEND", backend)
			t.Setenv("LLM_API_KEY", "test-key")
			assertContainerWiring(t, BuildContainer())
		})
	}
}

func TestBuildContainer_UsesDedicatedFlashClient(t *testing.T) {
	for _, backend := range []string{"native", "eino"} {
		t.Run(backend, func(t *testing.T) {
			t.Setenv("LLM_BACKEND", backend)
			t.Setenv("LLM_API_KEY", "test-key")
			t.Setenv("LLM_FLASH_MODEL", "")

			c := BuildContainer()

			handlerValue := reflect.ValueOf(c.Handler).Elem()
			orchField := handlerValue.FieldByName("orch")
			orchValue := reflect.NewAt(orchField.Type(), unsafe.Pointer(orchField.UnsafeAddr())).Elem().Elem()

			llmField := reflect.NewAt(orchValue.FieldByName("llm").Type(), unsafe.Pointer(orchValue.FieldByName("llm").UnsafeAddr())).Elem()
			flashField := reflect.NewAt(orchValue.FieldByName("flash").Type(), unsafe.Pointer(orchValue.FieldByName("flash").UnsafeAddr())).Elem()

			if llmField.IsNil() || flashField.IsNil() {
				t.Fatal("expected llm and flash clients to be present")
			}
			if llmField.Interface() == flashField.Interface() {
				t.Fatal("expected flash client to be a dedicated instance, not the same as main llm client")
			}
		})
	}
}

func TestBuildContainer_WiresADKSupervisorEngineForEino(t *testing.T) {
	t.Setenv("LLM_BACKEND", "eino")
	t.Setenv("SUPERVISOR_ENGINE", "adk")
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
		t.Fatal("expected adk route engine to be wired for eino supervisor")
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

	baziField := orchValue.FieldByName("baziSp")
	if baziField.IsNil() {
		t.Fatal("expected bazi specialist to be wired")
	}

	qimenField := orchValue.FieldByName("qimenSp")
	if qimenField.IsNil() {
		t.Fatal("expected qimen specialist to be wired")
	}
}
