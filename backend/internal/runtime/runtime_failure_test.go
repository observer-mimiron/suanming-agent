// This test file belongs to the manager-owned runtime layer.
// It verifies structured runtime failure handling and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

func TestValidatePlanArtifacts_ReturnsArtifactMissingFailure(t *testing.T) {
	st := state.NewSession("artifact-missing")
	plan := ExecutionPlan{
		Route: policy.ApprovedRoute{
			PrimaryDomain: "qimen",
		},
		Domains:      []string{"qimen"},
		Requirements: selectArtifactRequirements(st, []string{"qimen"}),
	}

	err := validatePlanArtifacts(st, plan)
	if err == nil {
		t.Fatal("expected error")
	}

	var rf *RuntimeFailure
	if !errors.As(err, &rf) {
		t.Fatalf("expected RuntimeFailure, got %T", err)
	}
	if rf.Class != "artifact_missing" {
		t.Fatalf("Class = %q, want artifact_missing", rf.Class)
	}
	if rf.Stage != "prefill" {
		t.Fatalf("Stage = %q, want prefill", rf.Stage)
	}
	if rf.Domain != "qimen" {
		t.Fatalf("Domain = %q, want qimen", rf.Domain)
	}
	if !rf.UserVisible {
		t.Fatal("UserVisible = false, want true")
	}
}

func TestGuardFinalAnswerWithTrace_AnnotatesRuntimeFailureMetadata(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	st := state.NewSession("guard-failure")
	route := policy.ApprovedRoute{PrimaryDomain: "qimen"}

	turnType, _ := guardFinalAnswerWithTrace(ctx, route, st, "final")
	if turnType != "guardrail_blocked" {
		t.Fatalf("turnType = %q, want guardrail_blocked", turnType)
	}

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	if got := tr.Attributes["failure.class"]; got != "specialist_contract_violation" {
		t.Fatalf("failure.class = %v, want specialist_contract_violation", got)
	}
	if got := tr.Attributes["failure.stage"]; got != "final_guard" {
		t.Fatalf("failure.stage = %v, want final_guard", got)
	}
	if got := tr.Attributes["failure.domain"]; got != "qimen" {
		t.Fatalf("failure.domain = %v, want qimen", got)
	}
	if got := tr.Attributes["failure.user_visible"]; got != true {
		t.Fatalf("failure.user_visible = %v, want true", got)
	}
}

func TestRuntimeFailureEventDataUsesSpecificBaziContractMessage(t *testing.T) {
	cause := baziContractAuditError("static", baziContractAuditFinding{
		Code:   "evidence_topic_overclaim",
		Field:  "static.tier",
		Reason: "tier overclaim",
	})
	err := baziSynthesisRuntimeFailure("static_synthesis", "BAZI_STATIC_SYNTHESIS_CONTRACT_FAILED", cause)

	data := RuntimeFailureEventData(context.Background(), err, "agent")

	if got := data["message"]; got != "证据主题不足，已停止展示过度裁断。请稍后重试。" {
		t.Fatalf("message = %v", got)
	}
}

func TestAnnotateBaziGraphErrorProjectsContractFindingTraceAttrs(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()
	cause := baziContractAuditError("dynamic", baziContractAuditFinding{
		Code:           "outcome_domain_mismatch",
		Field:          "dynamic.dayun_judgments[0].interpretation",
		DetectedDomain: "finance",
		Reason:         "未授权财务领域",
	})

	annotateBaziGraphError(ctx, "dynamic_synthesis", cause)

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	if got := tr.Attributes["bazi.contract.finding_code"]; got != "outcome_domain_mismatch" {
		t.Fatalf("finding_code = %v", got)
	}
	if got := tr.Attributes["bazi.contract.failure_class"]; got != baziContractFailureDomainUnauthorized {
		t.Fatalf("failure_class = %v", got)
	}
	if got := tr.Attributes["bazi.contract.recovery_policy"]; got != baziRecoveryPolicyDynamicFactsOnly {
		t.Fatalf("recovery_policy = %v", got)
	}
}
