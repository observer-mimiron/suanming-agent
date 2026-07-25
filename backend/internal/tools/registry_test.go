package tools

import (
	"context"
	"testing"
)

type testTool struct {
	name string
}

func (t testTool) Name() string        { return t.name }
func (t testTool) Description() string { return "test tool" }
func (t testTool) Label() string       { return "Test Tool" }
func (t testTool) Execute(context.Context, map[string]any) (any, error) {
	return map[string]any{"ok": true}, nil
}

func TestRegistry_Register_AssignsDefaultContract(t *testing.T) {
	reg := NewRegistry()
	reg.Register(testTool{name: "custom_read_tool"})

	contract, ok := reg.Contract("custom_read_tool")
	if !ok {
		t.Fatal("expected contract for registered tool")
	}
	if contract.Name != "custom_read_tool" {
		t.Fatalf("contract.Name = %q, want custom_read_tool", contract.Name)
	}
	if contract.Version != "v1" {
		t.Fatalf("contract.Version = %q, want v1", contract.Version)
	}
	if contract.SideEffect != SideEffectRead {
		t.Fatalf("contract.SideEffect = %q, want %q", contract.SideEffect, SideEffectRead)
	}
	if !contract.ReadOnly {
		t.Fatal("default contract must be read-only")
	}
}

func TestRegistry_RegisterWithContract_PreservesExplicitContract(t *testing.T) {
	reg := NewRegistry()
	contract := ToolContract{
		Name:       "write_order",
		Version:    "v2",
		ReadOnly:   false,
		SideEffect: SideEffectWrite,
		RiskLevel:  RiskHigh,
		Retry: RetryPolicy{
			MaxAttempts: 1,
		},
	}
	reg.RegisterWithContract(testTool{name: "write_order"}, contract)

	got, ok := reg.Contract("write_order")
	if !ok {
		t.Fatal("expected explicit contract")
	}
	if got.Version != "v2" {
		t.Fatalf("Version = %q, want v2", got.Version)
	}
	if got.RiskLevel != RiskHigh {
		t.Fatalf("RiskLevel = %q, want %q", got.RiskLevel, RiskHigh)
	}
	if got.ReadOnly {
		t.Fatal("explicit write contract must not be read-only")
	}
}

func TestDefaultContractFor_KnownTools(t *testing.T) {
	tests := []struct {
		name       string
		sideEffect SideEffectLevel
		readOnly   bool
	}{
		{name: "knowledge_search", sideEffect: SideEffectRead, readOnly: true},
		{name: "knowledge_catalog", sideEffect: SideEffectRead, readOnly: true},
		{name: "bazi_calc", sideEffect: SideEffectNone, readOnly: true},
		{name: "qimen_dunjia", sideEffect: SideEffectNone, readOnly: true},
		{name: "ziwei_calc", sideEffect: SideEffectNone, readOnly: true},
	}

	for _, tt := range tests {
		contract := DefaultContractFor(tt.name)
		if contract.Name != tt.name {
			t.Fatalf("%s: contract.Name = %q", tt.name, contract.Name)
		}
		if contract.SideEffect != tt.sideEffect {
			t.Fatalf("%s: SideEffect = %q, want %q", tt.name, contract.SideEffect, tt.sideEffect)
		}
		if contract.ReadOnly != tt.readOnly {
			t.Fatalf("%s: ReadOnly = %v, want %v", tt.name, contract.ReadOnly, tt.readOnly)
		}
	}
}
