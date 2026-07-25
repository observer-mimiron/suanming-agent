package state

import "testing"

func TestClone_PreservesGuidanceState(t *testing.T) {
	s := NewSession("guidance-clone")
	s.Guidance = &GuidanceState{
		DirectiveKind: "collect_slot",
		ChosenTopic:   "事业",
		PendingSlot:   "birth_time",
		RetryCount:    2,
	}

	clone := s.Clone()
	if clone.Guidance == nil {
		t.Fatal("clone.Guidance = nil, want guidance state")
	}
	if clone.Guidance.DirectiveKind != "collect_slot" {
		t.Fatalf("clone.Guidance.DirectiveKind = %q, want collect_slot", clone.Guidance.DirectiveKind)
	}
	if clone.Guidance.ChosenTopic != "事业" {
		t.Fatalf("clone.Guidance.ChosenTopic = %q, want 事业", clone.Guidance.ChosenTopic)
	}
	if clone.Guidance.PendingSlot != "birth_time" {
		t.Fatalf("clone.Guidance.PendingSlot = %q, want birth_time", clone.Guidance.PendingSlot)
	}
	if clone.Guidance.RetryCount != 2 {
		t.Fatalf("clone.Guidance.RetryCount = %d, want 2", clone.Guidance.RetryCount)
	}

	clone.Guidance.PendingSlot = "gender"
	if s.Guidance.PendingSlot != "birth_time" {
		t.Fatalf("original Guidance.PendingSlot = %q, want birth_time", s.Guidance.PendingSlot)
	}
}

func TestPersistentStore_RoundTripPreservesGuidanceState(t *testing.T) {
	dir := t.TempDir()
	store := NewPersistentStore(dir)

	st := NewSession("guidance-store")
	st.Guidance = &GuidanceState{
		DirectiveKind: "collect_slot",
		ChosenTopic:   "感情",
		PendingSlot:   "birthplace",
		RetryCount:    3,
	}
	if err := store.Save(st); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reloaded := NewPersistentStore(dir).LoadOrCreate("guidance-store")
	if reloaded.Guidance == nil {
		t.Fatal("reloaded.Guidance = nil, want guidance state")
	}
	if reloaded.Guidance.RetryCount != 3 {
		t.Fatalf("reloaded.Guidance.RetryCount = %d, want 3", reloaded.Guidance.RetryCount)
	}
	if reloaded.Guidance.ChosenTopic != "感情" {
		t.Fatalf("reloaded.Guidance.ChosenTopic = %q, want 感情", reloaded.Guidance.ChosenTopic)
	}
	if reloaded.Guidance.PendingSlot != "birthplace" {
		t.Fatalf("reloaded.Guidance.PendingSlot = %q, want birthplace", reloaded.Guidance.PendingSlot)
	}
}

func TestPersistentStore_SameStoreRoundTripPreservesGuidanceState(t *testing.T) {
	dir := t.TempDir()
	store := NewPersistentStore(dir)

	st := NewSession("guidance-same-store")
	st.Guidance = &GuidanceState{
		DirectiveKind: "collect_slot",
		ChosenTopic:   "财运",
		PendingSlot:   "gender",
		RetryCount:    1,
	}
	if err := store.Save(st); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reloaded := store.LoadOrCreate("guidance-same-store")
	if reloaded.Guidance == nil {
		t.Fatal("reloaded.Guidance = nil, want guidance state")
	}
	if reloaded.Guidance.RetryCount != 1 {
		t.Fatalf("reloaded.Guidance.RetryCount = %d, want 1", reloaded.Guidance.RetryCount)
	}
	if reloaded.Guidance.ChosenTopic != "财运" {
		t.Fatalf("reloaded.Guidance.ChosenTopic = %q, want 财运", reloaded.Guidance.ChosenTopic)
	}
	if reloaded.Guidance.PendingSlot != "gender" {
		t.Fatalf("reloaded.Guidance.PendingSlot = %q, want gender", reloaded.Guidance.PendingSlot)
	}
}

func TestPersistentStore_SavePreservesExistingSessionIdentity(t *testing.T) {
	dir := t.TempDir()
	store := NewPersistentStore(dir)

	owned := store.LoadOrCreate("guidance-owned")
	owned.Guidance = &GuidanceState{
		DirectiveKind: "collect_slot",
		ChosenTopic:   "事业",
		PendingSlot:   "birth_time",
		RetryCount:    1,
	}

	clone := owned.Clone()
	clone.Guidance.PendingSlot = "gender"
	clone.Guidance.RetryCount = 3

	if err := store.Save(clone); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reloaded := store.LoadOrCreate("guidance-owned")
	if reloaded != owned {
		t.Fatal("LoadOrCreate() returned a different session pointer, want store-owned identity preserved")
	}
	if owned.Guidance == nil {
		t.Fatal("owned.Guidance = nil, want merged guidance state")
	}
	if owned.Guidance.PendingSlot != "gender" {
		t.Fatalf("owned.Guidance.PendingSlot = %q, want gender", owned.Guidance.PendingSlot)
	}
	if owned.Guidance.RetryCount != 3 {
		t.Fatalf("owned.Guidance.RetryCount = %d, want 3", owned.Guidance.RetryCount)
	}
}
