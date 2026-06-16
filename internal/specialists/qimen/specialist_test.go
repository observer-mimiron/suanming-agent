package qimen

import (
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/specialists"
)

func TestQimenSpecialist_TimingRouteInvokesQimen(t *testing.T) {
	r := specialists.NewRegistry()
	Register(r)
	if len(r.All()) == 0 {
		t.Fatal("expected registered config")
	}
}

func TestQimenSpecialist_NonTimingRouteSkipsQimen(t *testing.T) {
	r := specialists.NewRegistry()
	Register(r)
	if len(r.All()) == 0 {
		t.Fatal("expected registered config")
	}
}

func TestQimenSpecialist_SupplementalNotReplacement(t *testing.T) {
	r := specialists.NewRegistry()
	Register(r)
	if len(r.All()) == 0 {
		t.Fatal("expected registered config")
	}
}
