// Package state 暂与 locker.go 共享包注释，本文件提供 Store 接口和 MemoryStore 实现。

package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Store 是会话状态持久化的统一接口。
type Store interface {
	LoadOrCreate(id string) *SessionState
	Save(st *SessionState) error
}

// MemoryStore 是 Store 的内存实现，可选地支持 JSON 文件持久化。
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*SessionState
	dir      string // if non-empty, persist sessions to this directory
}

// NewMemoryStore 创建一个不带持久化的纯内存会话存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sessions: make(map[string]*SessionState)}
}

// NewPersistentStore 创建一个将会话持久化为 JSON 文件的 MemoryStore。
func NewPersistentStore(dir string) *MemoryStore {
	os.MkdirAll(dir, 0755)
	s := &MemoryStore{sessions: make(map[string]*SessionState), dir: dir}
	// 从磁盘加载已有会话
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

// LoadOrCreate 根据会话 ID 加载已有状态，若不存在则创建一个新会话。
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

// Save 将会话状态持久化到磁盘（仅在创建时指定了持久化目录时生效）。
func (s *MemoryStore) Save(st *SessionState) error {
	if st == nil {
		return nil
	}

	snapshot := st.Clone()
	s.mu.Lock()
	if existing, ok := s.sessions[st.SessionID]; ok && existing != nil {
		*existing = *snapshot
	} else {
		s.sessions[st.SessionID] = snapshot
	}
	s.mu.Unlock()

	if s.dir == "" {
		return nil
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, st.SessionID+".json"), data, 0644)
}
