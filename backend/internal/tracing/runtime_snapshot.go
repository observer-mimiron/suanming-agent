package tracing

import "github.com/observer-mimiron/suanming-agent/internal/contracts"

const executionSnapshotTraceKey = "runtime.execution_snapshot"

// SetLocalExecutionSnapshot stores the runtime execution snapshot on the local
// TurnTrace only. It intentionally avoids mirroring complex objects into OTel.
func SetLocalExecutionSnapshot(t *TurnTrace, snapshot contracts.ExecutionSnapshot) {
	if t == nil || !snapshot.HasSignal() {
		return
	}
	if t.Attributes == nil {
		t.Attributes = map[string]any{}
	}
	t.Attributes[executionSnapshotTraceKey] = snapshot
}

func executionSnapshotFromTrace(t *TurnTrace) (*contracts.ExecutionSnapshot, bool) {
	if t == nil || len(t.Attributes) == 0 {
		return nil, false
	}
	raw, ok := t.Attributes[executionSnapshotTraceKey]
	if !ok || raw == nil {
		return nil, false
	}
	switch v := raw.(type) {
	case contracts.ExecutionSnapshot:
		snapshot := v
		return &snapshot, true
	case *contracts.ExecutionSnapshot:
		if v == nil {
			return nil, false
		}
		return v, true
	default:
		return nil, false
	}
}
