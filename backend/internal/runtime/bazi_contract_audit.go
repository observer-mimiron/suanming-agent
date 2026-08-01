// Package runtime contains the manager-owned BaZi synthesis contract audit.
//
// This file runs an independent semantic reviewer after deterministic checks.
// It may reject and retry a synthesis, but it never edits user-visible text.
package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// runBaziContractAudit asks an isolated fast-model agent to compare one
// synthesis with facts, evidence coverage and age-domain authorization.
func (e *Executor) runBaziContractAudit(ctx context.Context, st *state.SessionState, stage string, chartState baziCharterState, candidate any) (baziContractAudit, error) {
	payload := buildBaziContractAuditPayload(stage, chartState, candidate)
	return runBaziInnerAgentJSON[baziContractAudit](ctx, e.builder, baziContractAuditConfig(), st, buildBaziCharterPrompt("合同审计", chartState.Input.UserQuestion, payload))
}

// buildBaziContractAuditPayload exposes only the facts and authorization needed
// to evaluate the requested stage, keeping the reviewer independent.
func buildBaziContractAuditPayload(stage string, chartState baziCharterState, candidate any) map[string]any {
	payload := map[string]any{
		"stage":            strings.TrimSpace(stage),
		"candidate":        candidate,
		"evidence_quality": chartState.EvidenceQuality,
	}
	input := map[string]any{"core_chart": buildCoreChartView(chartState.Input)}
	if stage == "dynamic" {
		input["dynamic_facts"] = buildDynamicFactsView(chartState.Input)
		input["subject_context"] = buildBaziSubjectContext(chartState.Input)
	}
	payload["input"] = input
	return payload
}

// validateBaziContractAudit converts a failed binary review into a structured
// violation so the existing feedback and facts-only recovery paths can act.
func validateBaziContractAudit(stage string, audit baziContractAudit) error {
	if audit.Compliant && len(audit.Findings) == 0 {
		return nil
	}
	message := "independent synthesis contract audit failed"
	field := strings.TrimSpace(stage)
	if len(audit.Findings) > 0 {
		finding := audit.Findings[0]
		field = firstNonEmptyTrim(finding.Field, field)
		message = firstNonEmptyTrim(finding.Reason, finding.Code, message)
	}
	return baziViolationError(baziViolationSemanticContract, field, "", message, nil, nil)
}

// baziContractAuditSummary returns a compact trace value without retaining the
// complete model-authored audit explanation.
func baziContractAuditSummary(audit baziContractAudit) string {
	if audit.Compliant && len(audit.Findings) == 0 {
		return "clean"
	}
	if len(audit.Findings) == 0 {
		return "not_run"
	}
	return fmt.Sprintf("failed:%s", strings.TrimSpace(audit.Findings[0].Code))
}
