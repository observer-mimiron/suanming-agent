package domain

import "testing"

func TestNormalizeStaticClaimsOrdersNamedSlots(t *testing.T) {
	claims, err := NormalizeStaticClaims([]StructuredStaticClaim{
		{Slot: "pattern_usage"},
		{Slot: "tiaohou"},
		{Slot: "main_axis"},
		{Slot: "strength"},
	})
	if err != nil {
		t.Fatalf("NormalizeStaticClaims() error = %v", err)
	}
	for index, want := range []string{"main_axis", "strength", "tiaohou", "pattern_usage"} {
		if got := claims[index].Slot; got != want {
			t.Fatalf("claims[%d].Slot = %q, want %q", index, got, want)
		}
	}
}

func TestNormalizeStaticClaimsRejectsDuplicateSlot(t *testing.T) {
	_, err := NormalizeStaticClaims([]StructuredStaticClaim{
		{Slot: "main_axis"},
		{Slot: "strength"},
		{Slot: "tiaohou"},
		{Slot: "tiaohou"},
	})
	if err == nil {
		t.Fatal("NormalizeStaticClaims() error = nil, want duplicate-slot rejection")
	}
}
