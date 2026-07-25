package specialists

import (
	"context"
	"testing"
)

type stubRunner struct {
	result Result
	err    error
}

func (r stubRunner) Run(_ context.Context, _ Request) (Result, error) {
	return r.result, r.err
}

func TestRegistry_RegisterRunnerAndLookupByDomain(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Config{Domain: "bazi", Name: "bazi_specialist"}, stubRunner{
		result: Result{Domain: "bazi"},
	})

	runner, ok := reg.RunnerFor("bazi")
	if !ok {
		t.Fatal("RunnerFor should return the registered runner")
	}

	result, err := runner.Run(context.Background(), Request{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Domain != "bazi" {
		t.Fatalf("result.Domain = %q, want bazi", result.Domain)
	}
}
