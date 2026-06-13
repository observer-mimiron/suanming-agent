package state

import "sync"

// Locker provides per-session serialized execution.
type Locker interface {
	// Lock acquires an exclusive lock for the given session ID.
	// Returns an unlock function that must be called when done.
	Lock(id string) func()
}

// MemoryLocker is an in-memory implementation of Locker.
type MemoryLocker struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// NewMemoryLocker creates a new MemoryLocker.
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
