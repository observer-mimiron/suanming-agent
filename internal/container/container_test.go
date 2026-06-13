package container

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestBuildContainer_WiresSupervisorAndSpecialists(t *testing.T) {
	c := BuildContainer()

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
