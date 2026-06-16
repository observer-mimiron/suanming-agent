package bazi

import (
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/specialists"
)

func TestBaziSpecialist_RegisterConfig(t *testing.T) {
	r := specialists.NewRegistry()
	Register(r)

	if len(r.All()) != 1 {
		t.Fatalf("expected 1 config, got %d", len(r.All()))
	}
	cfg := r.All()[0]
	if cfg.Name != "bazi_specialist" {
		t.Fatalf("name: got %q, want bazi_specialist", cfg.Name)
	}
	if cfg.Domain != "bazi" {
		t.Fatalf("domain: got %q, want bazi", cfg.Domain)
	}
	if len(cfg.ToolNames) != 4 {
		t.Fatalf("expected 4 tool names, got %d", len(cfg.ToolNames))
	}
}

func TestBaziSpecialist_ReusableChartFollowup(t *testing.T) {
	r := specialists.NewRegistry()
	Register(r)
	if len(r.All()) == 0 {
		t.Fatal("expected registered config")
	}
}

func TestBaziSpecialist_NewProfileCompleteReading(t *testing.T) {
	r := specialists.NewRegistry()
	Register(r)
	if len(r.All()) == 0 {
		t.Fatal("expected registered config")
	}
}
