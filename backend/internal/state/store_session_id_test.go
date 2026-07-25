package state

import "testing"

func TestPersistentStore_SaveRejectsUnsafeSessionID(t *testing.T) {
	store := NewPersistentStore(t.TempDir())
	st := NewSession("../escape")

	if err := store.Save(st); err == nil {
		t.Fatal("Save() error = nil, want invalid session id")
	}
}
