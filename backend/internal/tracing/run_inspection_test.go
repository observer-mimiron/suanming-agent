// This test file belongs to the trace projection layer.
// It verifies run-inspection projection and protects the related contract from regressions.
// It projects runtime evidence; it must not change execution decisions.
package tracing

import (
	"strings"
	"testing"
	"time"
)

func TestBuildRunInspection_ProjectsSafeSpansAndFiltersSensitiveAttrs(t *testing.T) {
	now := time.Now()
	tr := &TurnTrace{
		TraceID:   "trc_inspect",
		SessionID: "sess_inspect",
		TurnType:  "agent_reading",
		StartedAt: now,
		EndedAt:   now.Add(2 * time.Second),
		Status:    "ok",
		Attributes: map[string]any{
			"decision_source":         "cheap_followup_reuse",
			"gate.reason":             "cheap_followup_reuse",
			"model":                   "qwen-plus",
			"output_tokens":           256,
			"input.value":             "1994年1月21日20点30分 女 南京",
			"output.value":            "完整回答",
			"input.messages.preview":  "system prompt",
			"bazi.final.audit_result": "clean",
		},
		Spans: []TraceSpan{
			{
				SpanID:     "root",
				Name:       "chat.turn",
				Kind:       KindAgent,
				Status:     "ok",
				DurationMs: 2000,
			},
			{
				SpanID:       "spn_llm",
				ParentSpanID: "root",
				Name:         "llm_generate",
				Kind:         KindLLM,
				Status:       "ok",
				DurationMs:   500,
				Attributes: map[string]any{
					"gen_ai.request.model":      "qwen-plus",
					"gen_ai.usage.input_tokens": 128,
					"input.message_count":       4,
					"input.message_roles":       []string{"system", "user"},
					"tool.status":               "ok",
					"input.messages.preview":    "should not leak",
				},
			},
		},
	}

	inspection := tr.BuildRunInspection()
	if inspection.TraceID != "trc_inspect" || inspection.SessionID != "sess_inspect" {
		t.Fatalf("unexpected ids: %+v", inspection)
	}
	if inspection.TotalMs != 2000 {
		t.Fatalf("total_ms = %d, want 2000", inspection.TotalMs)
	}
	if inspection.Summary.DecisionSource != "cheap_followup_reuse" {
		t.Fatalf("decision source = %q", inspection.Summary.DecisionSource)
	}
	if !hasDiagnostic(inspection.Diagnostics, "route.cheap_gate_reuse") {
		t.Fatalf("missing cheap gate diagnostic: %+v", inspection.Diagnostics)
	}
	if len(inspection.Spans) != 2 {
		t.Fatalf("spans = %d, want 2", len(inspection.Spans))
	}
	rootAttrs := inspection.Spans[0].Attributes
	if rootAttrs["decision_source"] != "cheap_followup_reuse" {
		t.Fatalf("root decision_source = %v", rootAttrs["decision_source"])
	}
	if rootAttrs["model"] != "qwen-plus" || rootAttrs["output_tokens"] != 256 {
		t.Fatalf("safe root model attrs missing: %+v", rootAttrs)
	}
	childAttrs := inspection.Spans[1].Attributes
	if childAttrs["gen_ai.request.model"] != "qwen-plus" || childAttrs["gen_ai.usage.input_tokens"] != 128 {
		t.Fatalf("safe child llm attrs missing: %+v", childAttrs)
	}
	for _, forbidden := range []string{"input.value", "output.value", "input.messages.preview", "user_message"} {
		if _, ok := rootAttrs[forbidden]; ok {
			t.Fatalf("forbidden root attr leaked: %s", forbidden)
		}
		if _, ok := childAttrs[forbidden]; ok {
			t.Fatalf("forbidden child attr leaked: %s", forbidden)
		}
	}
}

func TestBuildRunInspection_GeneratesDeterministicDiagnostics(t *testing.T) {
	tr := &TurnTrace{
		TraceID:   "trc_diag",
		SessionID: "sess_diag",
		TurnType:  "agent_reading",
		StartedAt: time.Now(),
		Status:    "error",
		Attributes: map[string]any{
			"failure.class":                      "artifact_missing",
			"failure.stage":                      "final_guard",
			"failure.code":                       "FINAL_ARTIFACT_MISSING",
			"failure.retryable":                  true,
			"failure.degraded":                   false,
			"bazi.final.audit_result":            "repaired",
			"bazi.contract.finding_code":         "outcome_domain_mismatch",
			"bazi.contract.recovery_policy":      "dynamic_facts_only",
			"bazi.internal_graph.recovery_state": "dynamic_recovered",
			"bazi.static.source":                 "model",
			"bazi.dynamic.source":                "facts_only_degraded",
		},
		Spans: []TraceSpan{
			{
				SpanID:     "root",
				Name:       "chat.turn",
				Kind:       KindAgent,
				Status:     "error",
				DurationMs: 1000,
			},
			{
				SpanID:       "spn_retrieval",
				ParentSpanID: "root",
				Name:         "knowledge_search",
				Kind:         KindRetriever,
				Status:       "ok",
				DurationMs:   20,
				Attributes: map[string]any{
					"query": "亥月甲木 调候",
					"hits":  0,
				},
			},
			{
				SpanID:       "spn_guard",
				ParentSpanID: "root",
				Name:         "contract_gate",
				Kind:         KindChain,
				Status:       "error",
				Error:        "blocked",
				DurationMs:   5,
				Attributes: map[string]any{
					"guardrail_result": "blocked",
				},
			},
		},
	}

	inspection := tr.BuildRunInspection()
	for _, code := range []string{
		"runtime.failure",
		"artifact.missing",
		"guard.blocked",
		"contract.repaired",
		"contract.facts_only_or_partial",
		"retrieval.no_hits",
		"span.error",
	} {
		if !hasDiagnostic(inspection.Diagnostics, code) {
			t.Fatalf("missing diagnostic %s in %+v", code, inspection.Diagnostics)
		}
	}
	if inspection.Summary.InspectionText == "" || strings.Contains(inspection.Summary.InspectionText, "未发现") {
		t.Fatalf("summary did not pick actionable diagnostic: %+v", inspection.Summary)
	}
	if got := spanByID(inspection.Spans, "spn_guard").Category; got != "guard" {
		t.Fatalf("guard category = %q", got)
	}
	if got := spanByID(inspection.Spans, "spn_retrieval").Category; got != "retriever" {
		t.Fatalf("retriever category = %q", got)
	}
}

func TestBuildRunInspection_ReportsRetrievalTimeoutInsteadOfNoHits(t *testing.T) {
	tr := &TurnTrace{
		TraceID:   "trc_retrieval_timeout",
		SessionID: "sess_retrieval_timeout",
		StartedAt: time.Now(),
		Status:    "ok",
		Spans: []TraceSpan{{
			SpanID: "spn_retrieval_timeout",
			Name:   "knowledge_search",
			Kind:   KindRetriever,
			Status: "degraded",
			Attributes: map[string]any{
				"query":          "子平真诠 破格 败格",
				"hits":           0,
				"degrade_reason": "timeout",
			},
		}},
	}

	inspection := tr.BuildRunInspection()
	if !hasDiagnostic(inspection.Diagnostics, "retrieval.service_timeout") {
		t.Fatalf("missing retrieval timeout diagnostic: %+v", inspection.Diagnostics)
	}
	if hasDiagnostic(inspection.Diagnostics, "retrieval.no_hits") {
		t.Fatalf("timeout must not be reported as no hits: %+v", inspection.Diagnostics)
	}
}

func hasDiagnostic(items []RunDiagnostic, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}

func spanByID(spans []RunSpan, id string) RunSpan {
	for _, span := range spans {
		if span.SpanID == id {
			return span
		}
	}
	return RunSpan{}
}
