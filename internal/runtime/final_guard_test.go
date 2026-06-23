package runtime

import (
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/policy"
)

func TestShouldBufferFinalAnswer(t *testing.T) {
	tests := []struct {
		name   string
		route  policy.ApprovedRoute
		buffer bool
	}{
		{name: "bazi buffered", route: policy.ApprovedRoute{PrimaryDomain: "bazi"}, buffer: true},
		{name: "qimen buffered", route: policy.ApprovedRoute{PrimaryDomain: "qimen"}, buffer: true},
		{name: "ziwei buffered", route: policy.ApprovedRoute{PrimaryDomain: "ziwei"}, buffer: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldBufferFinalAnswer(); got != tc.buffer {
				t.Fatalf("shouldBufferFinalAnswer() = %v, want %v", got, tc.buffer)
			}
		})
	}
}

