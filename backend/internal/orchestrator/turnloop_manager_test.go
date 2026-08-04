// This test file belongs to the session orchestration layer.
// It verifies legacy turn-loop manager coverage and protects the related contract from regressions.
// It owns turn lifecycle; domain reasoning stays in runtime and tools.
package orchestrator

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/observer-mimiron/suanming-agent/internal/runtime"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// echoAgent 是最小的 TypedAgent 实现，用于测试 TurnLoopSessionManager。
type echoAgent struct{}

func (a *echoAgent) Name(_ context.Context) string        { return "echo" }
func (a *echoAgent) Description(_ context.Context) string { return "echo agent for test" }

func (a *echoAgent) Run(_ context.Context, input *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		defer gen.Close()
		_ = input
		gen.Send(&adk.AgentEvent{Output: &adk.AgentOutput{}})
	}()
	return iter
}

// mockSink 记录所有 Emit 的事件，用于测试。
type mockSink struct {
	mu     sync.Mutex
	events []runtime.Event
	closed bool
}

func (s *mockSink) Emit(_ context.Context, evt runtime.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.events = append(s.events, evt)
	return nil
}

func (s *mockSink) Events() []runtime.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]runtime.Event{}, s.events...)
}

// mockCfgFactory 构造最小可运行的 TurnLoopConfig。
// OnAgentEvents 不写 sink（A2.1b 真集成时实现），只 drain events。
func mockCfgFactory(_ string, _ func() EventSink, _ func()) adk.TurnLoopConfig[string, *schema.Message] {
	return adk.TurnLoopConfig[string, *schema.Message]{
		GenInput: func(_ context.Context, _ *adk.TurnLoop[string, *schema.Message], items []string) (*adk.GenInputResult[string, *schema.Message], error) {
			return &adk.GenInputResult[string, *schema.Message]{
				Input:    &adk.AgentInput{},
				Consumed: items,
			}, nil
		},
		PrepareAgent: func(_ context.Context, _ *adk.TurnLoop[string, *schema.Message], _ []string) (adk.TypedAgent[*schema.Message], error) {
			return &echoAgent{}, nil
		},
	}
}

func newTestManager(t *testing.T) *TurnLoopSessionManager {
	t.Helper()
	store := state.NewPersistentStore("")
	return NewTurnLoopSessionManager(store, mockCfgFactory)
}

func sinkForSession(t *testing.T, m *TurnLoopSessionManager, sessionID string) EventSink {
	t.Helper()
	return m.loadSink(sessionID)
}

func loopCount(m *TurnLoopSessionManager) int {
	count := 0
	m.loops.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

func TestTurnLoopSessionManager_GetOrCreateIdempotent(t *testing.T) {
	m := newTestManager(t)
	defer m.StopAll(false)

	loop1 := m.getOrCreate("session-A")
	loop2 := m.getOrCreate("session-A")
	if loop1 != loop2 {
		t.Fatal("同 sessionID 应返回同一实例")
	}
}

func TestTurnLoopSessionManager_DistinctSessions(t *testing.T) {
	m := newTestManager(t)
	defer m.StopAll(false)

	loopA := m.getOrCreate("session-A")
	loopB := m.getOrCreate("session-B")
	if loopA == loopB {
		t.Fatal("不同 sessionID 应返回不同实例")
	}
}

func TestTurnLoopSessionManager_ConcurrentGetOrCreate(t *testing.T) {
	m := newTestManager(t)
	defer m.StopAll(false)

	const n = 20
	var wg sync.WaitGroup
	loops := make([]*adk.TurnLoop[string, *schema.Message], n)
	loops[0] = m.getOrCreate("session-concurrent")

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			loops[idx] = m.getOrCreate("session-concurrent")
		}(i)
	}
	wg.Wait()

	first := loops[0]
	for i := 1; i < n; i++ {
		if loops[i] != first {
			t.Fatalf("goroutine %d 返回了不同实例", i)
		}
	}
}

func TestTurnLoopSessionManager_SinkMapping(t *testing.T) {
	m := newTestManager(t)
	defer m.StopAll(false)

	if s := sinkForSession(t, m, "session-X"); s != nil {
		t.Fatal("未 Push 前应返回 nil")
	}

	sink1 := &mockSink{}
	m.Push(context.Background(), "session-X", "msg1", sink1)
	if got := sinkForSession(t, m, "session-X"); got != sink1 {
		t.Fatal("Push 后 GetSink 应返回 sink1")
	}

	sink2 := &mockSink{}
	m.Push(context.Background(), "session-X", "msg2", sink2)
	if got := sinkForSession(t, m, "session-X"); got != sink2 {
		t.Fatal("第二次 Push 后应返回 sink2（覆盖）")
	}

	m.sinks.Delete("session-X")
	if got := sinkForSession(t, m, "session-X"); got != nil {
		t.Fatal("ClearSink 后应返回 nil")
	}
}

func TestTurnLoopSessionManager_StopAll(t *testing.T) {
	m := newTestManager(t)

	_ = m.getOrCreate("session-1")
	_ = m.getOrCreate("session-2")

	count := loopCount(m)
	if count != 2 {
		t.Fatalf("StopAll 前应有 2 个实例，实际 %d", count)
	}

	m.StopAll(true)
	time.Sleep(200 * time.Millisecond)
}

func TestTurnLoopSessionManager_PushEndToEnd(t *testing.T) {
	m := newTestManager(t)
	defer m.StopAll(false)

	sink := &mockSink{}
	m.Push(context.Background(), "session-push", "hello", sink)
	// 给 TurnLoop 时间处理
	time.Sleep(300 * time.Millisecond)

	if s := sinkForSession(t, m, "session-push"); s != sink {
		t.Fatal("Push 后 sink 应被绑定")
	}
}
