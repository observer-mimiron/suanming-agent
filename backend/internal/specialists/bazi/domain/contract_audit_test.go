package domain

import (
	"encoding/json"
	"testing"
)

func TestContractAuditKeepsWireShape(t *testing.T) {
	encoded, err := json.Marshal(ContractAudit{Compliant: false, Findings: []ContractAuditFinding{{Code: "fact_conflict", Field: "static.main_axis", Reason: "conflict"}}})
	if err != nil {
		t.Fatalf("marshal audit: %v", err)
	}
	if got, want := string(encoded), `{"compliant":false,"findings":[{"code":"fact_conflict","field":"static.main_axis","reason":"conflict"}]}`; got != want {
		t.Fatalf("audit JSON = %s, want %s", got, want)
	}
}
