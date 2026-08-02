package state

import "testing"

func TestSessionState_PreservesLayeredContexts(t *testing.T) {
	st := NewSession("s1")
	st.ManagerContext = ManagerContext{
		ActiveDomain:   "bazi",
		CurrentTopic:   "career",
		LastReplyOwner: "manager",
	}
	st.DomainContexts.Bazi = DomainContext{
		Version:        1,
		CheckpointID:   "cp-bazi-1",
		WorkingSummary: "已完成排盘与用神分析",
		RuntimeValues: map[string]any{
			"phase": "interpret",
		},
	}

	cloned := st.Clone()

	if cloned.ManagerContext.ActiveDomain != "bazi" {
		t.Fatalf("ActiveDomain = %q, want bazi", cloned.ManagerContext.ActiveDomain)
	}
	if cloned.DomainContexts.Bazi.CheckpointID != "cp-bazi-1" {
		t.Fatalf("CheckpointID = %q, want cp-bazi-1", cloned.DomainContexts.Bazi.CheckpointID)
	}

	cloned.DomainContexts.Bazi.RuntimeValues["phase"] = "followup"
	if st.DomainContexts.Bazi.RuntimeValues["phase"] != "interpret" {
		t.Fatal("RuntimeValues should be deep-cloned")
	}
}
