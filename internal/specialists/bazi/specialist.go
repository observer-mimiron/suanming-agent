package bazi

import (
	"context"
	"fmt"

	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// Specialist is the bazi domain specialist. In phase 1 it acts as a thin boundary
// layer that validates profile readiness and delegates computation to the orchestrator.
type Specialist struct{}

// New creates a bazi specialist.
func New() *Specialist {
	return &Specialist{}
}

// Name returns the specialist identifier.
func (s *Specialist) Name() string {
	return "bazi"
}

// Run executes the bazi specialist flow. In phase 1:
// - Incomplete profile without reusable chart → clarification
// - Complete profile / reusable chart → returns a ready result tag
// The orchestrator handles actual LLM calls, tool execution, and SSE.
func (s *Specialist) Run(ctx context.Context, st *state.SessionState, route specialists.ApprovedRoute, sink specialists.EventSink) (schemas.DomainResult, error) {
	profileReady := st.IsProfileComplete()
	hasChart := st.HasBaziResult()

	// Emit a span marker for tracing.
	if sink != nil {
		sink(ctx, specialists.Event{Type: "specialist_bazi", Data: route.TaskIntent})
	}

	// Case 1: incomplete profile, no reusable chart → clarification
	if !profileReady && !hasChart && route.TaskIntent != "collect_profile" && route.TaskIntent != "amend_profile" && route.TaskIntent != "direct_bazi" {
		return schemas.DomainResult{
			Domain:  "bazi",
			Summary: "请提供您的出生信息（年份、月份、日期、时辰、性别），我来为您排盘分析。",
			Final:   true,
		}, nil
	}

	// Case 2: incomplete profile, but user IS providing data → accept and flag compute needed
	if !profileReady && !hasChart && (route.TaskIntent == "collect_profile" || route.TaskIntent == "amend_profile") {
		return schemas.DomainResult{
			Domain:         "bazi",
			Summary:        "已收到您的出生信息，正在为您排盘分析。",
			Final:          false,
			StructuredData: map[string]any{"stage": "collecting", "profile_complete": false},
		}, nil
	}

	// Case 3: profile ready but no chart yet → compute needed
	if profileReady && !hasChart {
		return schemas.DomainResult{
			Domain:         "bazi",
			Summary:        fmt.Sprintf("正在为您排出八字命盘并进行分析。"),
			Final:          false,
			StructuredData: map[string]any{"stage": "computing", "profile_complete": true, "chart_ready": false},
		}, nil
	}

	// Case 4: chart available → followup / interpretation
	return schemas.DomainResult{
		Domain:         "bazi",
		Summary:        fmt.Sprintf("基于您的命盘为您分析。"),
		Final:          false,
		StructuredData: map[string]any{"stage": "reading", "profile_complete": true, "chart_ready": true},
	}, nil
}
