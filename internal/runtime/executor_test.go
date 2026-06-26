package runtime

import (
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/intent"
)

func TestExecutor_RouterField(t *testing.T) {
	e := &Executor{router: &intent.SemanticRouter{}}
	if e.router == nil {
		t.Fatal("router field not set")
	}
}

func TestExecutor_SetRouter(t *testing.T) {
	e := &Executor{}
	r := &intent.SemanticRouter{}
	e.SetRouter(r)
	if e.router != r {
		t.Fatal("SetRouter did not set router field")
	}
}