package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Store manages session state persistence.
type Store interface {
	LoadOrCreate(id string) *SessionState
	Save(st *SessionState) error
}

// MemoryStore is an in-memory implementation of Store with optional file persistence.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*SessionState
	dir      string // if non-empty, persist sessions to this directory
}

// NewMemoryStore creates a new MemoryStore without persistence.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sessions: make(map[string]*SessionState)}
}

// NewPersistentStore creates a MemoryStore that persists sessions as JSON files.
func NewPersistentStore(dir string) *MemoryStore {
	os.MkdirAll(dir, 0755)
	s := &MemoryStore{sessions: make(map[string]*SessionState), dir: dir}
	// Load existing sessions from disk
	files, _ := os.ReadDir(dir)
	for _, f := range files {
		if filepath.Ext(f.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, f.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var st SessionState
		if json.Unmarshal(data, &st) == nil && st.SessionID != "" {
			s.sessions[st.SessionID] = &st
		}
	}
	return s
}

func (s *MemoryStore) LoadOrCreate(id string) *SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.sessions[id]; ok {
		return existing
	}
	st := NewSession(id)
	s.sessions[id] = st
	return st
}

func (s *MemoryStore) Save(st *SessionState) error {
	if s.dir == "" {
		return nil
	}
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, st.SessionID+".json"), data, 0644)
}
