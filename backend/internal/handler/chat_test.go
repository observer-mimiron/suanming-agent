// This test file belongs to the HTTP and SSE adapter layer.
// It verifies chat request handling and protects the related contract from regressions.
// It adapts HTTP/SSE; route approval and domain execution stay below this layer.
package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/observer-mimiron/suanming-agent/internal/orchestrator"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

type sentEvent struct {
	typeName string
	data     any
}

type testSender struct {
	events []sentEvent
	err    error
}

func (s *testSender) Send(eventType string, data any) error {
	if s.err != nil {
		return s.err
	}
	s.events = append(s.events, sentEvent{typeName: eventType, data: data})
	return nil
}

type waitingLocker struct {
	waiting  chan struct{}
	release  chan struct{}
	unlocked chan struct{}
	started  sync.Once
	finished sync.Once
}

func newWaitingLocker() *waitingLocker {
	return &waitingLocker{
		waiting:  make(chan struct{}),
		release:  make(chan struct{}),
		unlocked: make(chan struct{}),
	}
}

func (l *waitingLocker) Lock(string) func() {
	l.started.Do(func() { close(l.waiting) })
	<-l.release
	return func() { l.finished.Do(func() { close(l.unlocked) }) }
}

func TestResolveSessionID_EmptyFallsBackToDefault(t *testing.T) {
	got, err := resolveSessionID("")
	if err != nil {
		t.Fatalf("resolveSessionID(\"\") error = %v", err)
	}
	if got != "default" {
		t.Fatalf("resolveSessionID(\"\") = %q, want default", got)
	}
}

func TestResolveSessionID_RejectsUnsafeValues(t *testing.T) {
	cases := []string{
		"../escape",
		`..\escape`,
		"bad/session",
		".",
		"..",
		"bad id",
	}

	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			if _, err := resolveSessionID(tc); err == nil {
				t.Fatalf("resolveSessionID(%q) error = nil, want invalid session id", tc)
			}
		})
	}
}

func TestSSEEventSink_CancelsExecutionAfterWriteFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wantErr := errors.New("client disconnected")
	sink := &sseEventSink{sw: &testSender{err: wantErr}, cancel: cancel}

	err := sink.Emit(ctx, orchestrator.Event{Type: "thinking", Data: map[string]any{"text": "处理中"}})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Emit() error = %v, want %v", err, wantErr)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("context error = %v, want context.Canceled", ctx.Err())
	}
}

func TestSSEEventSink_PreservesHealthyTextAndDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sender := &testSender{}
	sink := &sseEventSink{sw: sender, cancel: cancel}

	if err := sink.Emit(ctx, orchestrator.Event{Type: "text", Data: map[string]any{"text": "结果"}}); err != nil {
		t.Fatalf("text Emit() error = %v", err)
	}
	if err := sink.Emit(ctx, orchestrator.Event{Type: "done", Data: map[string]any{}}); err != nil {
		t.Fatalf("done Emit() error = %v", err)
	}
	if ctx.Err() != nil {
		t.Fatalf("healthy SSE must not cancel context: %v", ctx.Err())
	}
	if len(sender.events) != 2 || sender.events[0].typeName != "text" || sender.events[1].typeName != "done" {
		t.Fatalf("events = %#v, want text then done", sender.events)
	}
}

func TestHandleChat_RequestCancellationStopsSessionLockWait(t *testing.T) {
	gin.SetMode(gin.TestMode)
	locker := newWaitingLocker()
	orch := orchestrator.New(nil, nil, state.NewPersistentStore(""), locker, tracing.NewRealTracer(nil))
	handler := NewChatHandler(orch, false, "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"message":"测试","session_id":"same-session"}`)).WithContext(ctx)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)
	c.Request = req
	finished := make(chan struct{})

	go func() {
		handler.HandleChat(c)
		close(finished)
	}()

	select {
	case <-locker.waiting:
	case <-time.After(time.Second):
		t.Fatal("chat request did not begin waiting for the session lock")
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("canceled chat request remained blocked on the session lock")
	}
	close(locker.release)
	select {
	case <-locker.unlocked:
	case <-time.After(time.Second):
		t.Fatal("deferred lock waiter did not release the session lock")
	}
}
