package bazi

import (
	"context"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/specialists"
)

type recordingRunner struct {
	calls  int
	result specialists.Result
	err    error
}

func (r *recordingRunner) Run(_ context.Context, _ specialists.Request) (specialists.Result, error) {
	r.calls++
	return r.result, r.err
}

func TestRunner_DelegatesByRole(t *testing.T) {
	primary := &recordingRunner{result: specialists.Result{Domain: "bazi", Summary: "primary"}}
	support := &recordingRunner{result: specialists.Result{Domain: "bazi", Summary: "support"}}
	runner := &Runner{Primary: primary, Support: support}

	result, err := runner.Run(context.Background(), specialists.Request{Role: specialists.RolePrimary})
	if err != nil {
		t.Fatalf("primary Run() error = %v", err)
	}
	if result.Summary != "primary" || primary.calls != 1 || support.calls != 0 {
		t.Fatalf("primary delegation result=%#v calls=(%d,%d)", result, primary.calls, support.calls)
	}

	result, err = runner.Run(context.Background(), specialists.Request{Role: specialists.RoleSupport})
	if err != nil {
		t.Fatalf("support Run() error = %v", err)
	}
	if result.Summary != "support" || primary.calls != 1 || support.calls != 1 {
		t.Fatalf("support delegation result=%#v calls=(%d,%d)", result, primary.calls, support.calls)
	}
}

func TestRunner_RejectsInvalidRoleAndMissingDelegate(t *testing.T) {
	tests := []struct {
		name   string
		runner *Runner
		role   string
	}{
		{name: "nil runner", runner: nil, role: specialists.RolePrimary},
		{name: "empty role", runner: &Runner{}, role: ""},
		{name: "unknown role", runner: &Runner{}, role: "other"},
		{name: "missing primary", runner: &Runner{}, role: specialists.RolePrimary},
		{name: "missing support", runner: &Runner{}, role: specialists.RoleSupport},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.runner.Run(context.Background(), specialists.Request{Role: tt.role})
			if err == nil {
				t.Fatal("Run() error = nil, want error")
			}
		})
	}
}
