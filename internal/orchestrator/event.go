package orchestrator

import "context"

// Event is a single event emitted during orchestration.
type Event struct {
	Type string
	Data any
}

// EventSink receives events from the orchestrator.
// The handler layer adapts SSE/HTTP to this interface.
type EventSink interface {
	Emit(ctx context.Context, evt Event) error
}
