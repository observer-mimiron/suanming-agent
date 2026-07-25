// Package state 提供会话状态管理，包括会话数据的持久化存储、并发锁、以及会话内容的结构化表示。

package state

import "sync"

// Locker 提供按会话 ID 串行化执行的互斥锁接口。
type Locker interface {
	// Lock 获取指定会话 ID 的排他锁。
	// 返回一个解锁函数，必须在完成后调用。
	Lock(id string) func()
}

// MemoryLocker 是 Locker 的内存实现，使用 sync.Mutex 保证串行化。
type MemoryLocker struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// NewMemoryLocker 创建一个新的内存锁管理器。
func NewMemoryLocker() *MemoryLocker {
	return &MemoryLocker{
		locks: make(map[string]*sync.Mutex),
	}
}

func (l *MemoryLocker) Lock(id string) func() {
	l.mu.Lock()
	mu, ok := l.locks[id]
	if !ok {
		mu = &sync.Mutex{}
		l.locks[id] = mu
	}
	l.mu.Unlock()

	mu.Lock()
	return func() { mu.Unlock() }
}
