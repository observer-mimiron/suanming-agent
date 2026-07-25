package state

import (
	"fmt"
	"sync"
	"testing"
)

func TestMemoryStore_ConcurrentLoadOrCreateReturnsOneSessionPerID(t *testing.T) {
	store := NewPersistentStore(t.TempDir())
	const workers = 32

	var wg sync.WaitGroup
	results := make(chan *SessionState, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- store.LoadOrCreate("same-session")
		}()
	}
	wg.Wait()
	close(results)

	var first *SessionState
	for st := range results {
		if st == nil {
			t.Fatal("LoadOrCreate returned nil")
		}
		if first == nil {
			first = st
			continue
		}
		if st != first {
			t.Fatal("LoadOrCreate returned different session pointers for the same session_id")
		}
	}
}

func TestMemoryStore_ConcurrentSaveKeepsSessionsIsolated(t *testing.T) {
	store := NewPersistentStore(t.TempDir())
	const sessions = 24

	var wg sync.WaitGroup
	for i := 0; i < sessions; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			sessionID := fmt.Sprintf("session-%02d", i)
			st := NewSession(sessionID)
			st.Execution.PrimaryDomain = "bazi"
			st.Execution.TaskIntent = fmt.Sprintf("intent-%02d", i)
			st.RecordTurn("user", fmt.Sprintf("question-%02d", i))
			st.BaziResult = map[string]any{"session": sessionID}
			if err := store.Save(st); err != nil {
				t.Errorf("Save(%s) error = %v", sessionID, err)
			}
		}()
	}
	wg.Wait()

	for i := 0; i < sessions; i++ {
		sessionID := fmt.Sprintf("session-%02d", i)
		st, ok := store.Peek(sessionID)
		if !ok || st == nil {
			t.Fatalf("Peek(%s) missing session", sessionID)
		}
		if st.SessionID != sessionID {
			t.Fatalf("Peek(%s).SessionID = %q", sessionID, st.SessionID)
		}
		if got := st.Execution.TaskIntent; got != fmt.Sprintf("intent-%02d", i) {
			t.Fatalf("Peek(%s).Execution.TaskIntent = %q", sessionID, got)
		}
		if len(st.RecentTurns) != 1 || st.RecentTurns[0].Content != fmt.Sprintf("question-%02d", i) {
			t.Fatalf("Peek(%s).RecentTurns = %+v", sessionID, st.RecentTurns)
		}
		if got := st.BaziResult["session"]; got != sessionID {
			t.Fatalf("Peek(%s).BaziResult.session = %v", sessionID, got)
		}
	}
}
