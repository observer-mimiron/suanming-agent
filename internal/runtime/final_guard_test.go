package runtime

import (
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

func TestShouldBufferFinalAnswer(t *testing.T) {
	tests := []struct {
		name   string
		route  policy.ApprovedRoute
		buffer bool
	}{
		{name: "bazi streams", route: policy.ApprovedRoute{PrimaryDomain: "bazi"}, buffer: false},
		{name: "qimen buffered", route: policy.ApprovedRoute{PrimaryDomain: "qimen"}, buffer: true},
		{name: "ziwei buffered", route: policy.ApprovedRoute{PrimaryDomain: "ziwei"}, buffer: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldBufferFinalAnswer(tc.route); got != tc.buffer {
				t.Fatalf("shouldBufferFinalAnswer() = %v, want %v", got, tc.buffer)
			}
		})
	}
}

func TestGuardFinalAnswer_BlocksMissingPrimaryArtifacts(t *testing.T) {
	tests := []struct {
		name  string
		route policy.ApprovedRoute
		state *state.SessionState
		turn  string
	}{
		{
			name:  "qimen missing result",
			route: policy.ApprovedRoute{PrimaryDomain: "qimen"},
			state: state.NewSession("s1"),
			turn:  "guardrail_blocked",
		},
		{
			name:  "ziwei missing result",
			route: policy.ApprovedRoute{PrimaryDomain: "ziwei"},
			state: state.NewSession("s2"),
			turn:  "guardrail_blocked",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			turnType, text := guardFinalAnswer(tc.route, tc.state, "final")
			if turnType != tc.turn {
				t.Fatalf("turnType = %q, want %q", turnType, tc.turn)
			}
			if text == "final" {
				t.Fatal("guardFinalAnswer should not pass through original text when artifact is missing")
			}
		})
	}
}
