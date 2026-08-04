// This test file belongs to the session state layer.
// It verifies session ID store behavior and protects the related contract from regressions.
// It stores session truth; routing and interpretation decisions stay outside state structs.
package state

import "testing"

func TestPersistentStore_SaveRejectsUnsafeSessionID(t *testing.T) {
	store := NewPersistentStore(t.TempDir())
	st := NewSession("../escape")

	if err := store.Save(st); err == nil {
		t.Fatal("Save() error = nil, want invalid session id")
	}
}
