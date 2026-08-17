package adapter

import (
	"context"
	"testing"
)

func TestBaziRecoverFactsNodeMarksMissingCurrentPeriod(t *testing.T) {
	in := &baziInternalGraphState{Phase: baziPhaseDynamic}

	if _, err := (&Executor{}).baziRecoverFactsNode(context.Background(), in); err != nil {
		t.Fatalf("baziRecoverFactsNode() error = %v", err)
	}
	if in.TerminationReason != "current_period_unavailable" {
		t.Fatalf("termination reason = %q, want current_period_unavailable", in.TerminationReason)
	}
}
