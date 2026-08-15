package adapter

import (
	"context"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/specialists"
)

func TestRunnerRejectsMissingRuntimeCapabilities(t *testing.T) {
	_, err := (&Runner{}).Run(context.Background(), specialists.Request{Session: &specialists.SessionView{}})
	if err == nil {
		t.Fatal("Run() error = nil, want missing runtime capabilities")
	}
}

func TestRunnerRejectsMissingSessionView(t *testing.T) {
	runner := &Runner{Port: RuntimePort{}}
	_, err := runner.Run(context.Background(), specialists.Request{})
	if err == nil {
		t.Fatal("Run() error = nil, want missing runtime capabilities or session view")
	}
}
