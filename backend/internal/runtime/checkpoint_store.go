package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// fileCheckPointStore 是 compose.CheckPointStore 的文件系统实现。
// 内存 map + 文件持久化（仿 state.MemoryStore 模式），不引入 Redis 等中间件。
// 用于 prefill 后、agent 前的中断-恢复交互（C1 能力）。
//
// Checkpoint 是 Eino serializer 序列化后的二进制数据，以 .bin 文件存储。
type fileCheckPointStore struct {
	mu  sync.RWMutex
	cps map[string][]byte
	dir string
}

func newFileCheckPointStore(dir string) (*fileCheckPointStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create checkpoint dir: %w", err)
	}
	s := &fileCheckPointStore{cps: make(map[string][]byte), dir: dir}
	files, _ := os.ReadDir(dir)
	for _, f := range files {
		if filepath.Ext(f.Name()) != ".bin" {
			continue
		}
		path := filepath.Join(dir, f.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		s.cps[f.Name()[:len(f.Name())-len(".bin")]] = data
	}
	return s, nil
}

// Get 返回 (checkpoint bytes, found, error)。
func (s *fileCheckPointStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.cps[checkPointID]
	if !ok {
		return nil, false, nil
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, true, nil
}

func (s *fileCheckPointStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	s.mu.Lock()
	cp := make([]byte, len(checkPoint))
	copy(cp, checkPoint)
	s.cps[checkPointID] = cp
	s.mu.Unlock()

	if s.dir == "" {
		return nil
	}
	return os.WriteFile(filepath.Join(s.dir, checkPointID+".bin"), checkPoint, 0644)
}

// Delete 实现 compose.CheckPointDeleter（可选接口，用于显式清理 stale checkpoint）。
func (s *fileCheckPointStore) Delete(ctx context.Context, checkPointID string) error {
	s.mu.Lock()
	delete(s.cps, checkPointID)
	s.mu.Unlock()

	if s.dir == "" {
		return nil
	}
	err := os.Remove(filepath.Join(s.dir, checkPointID+".bin"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
