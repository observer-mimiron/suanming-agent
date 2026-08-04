// This file belongs to the bounded specialist layer.
// It owns runner behavior for this package.
// It runs bounded workers; final composition stays with Manager.
package specialists

import (
	"context"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// Request 是 manager 调用 specialist runner 时传入的执行上下文。
type Request struct {
	SessionID      string
	UserMessage    string
	Route          policy.ApprovedRoute
	ManagerContext state.ManagerContext
	DomainContext  state.DomainContext
	Session        *state.SessionState
}

// Result 是 specialist 返回给 manager 的结构化执行结果。
type Result struct {
	Domain             string
	Summary            string
	DirectAnswer       string
	KeyPoints          []string
	EvidenceSummary    string
	MissingSlots       []string
	ManagerBrief       string
	DomainContextPatch map[string]any
}

// NormalizedSummary returns the legacy summary when present and otherwise
// renders a stable text fallback from the structured fields. This lets the
// runtime adopt structured specialist outputs incrementally without breaking the
// existing manager/guard/SSE path that still expects text.
func (r Result) NormalizedSummary() string {
	if summary := strings.TrimSpace(r.Summary); summary != "" {
		return summary
	}

	parts := make([]string, 0, 2+len(r.KeyPoints))
	if answer := strings.TrimSpace(r.DirectAnswer); answer != "" {
		parts = append(parts, answer)
	}
	if len(r.KeyPoints) > 0 {
		for _, point := range r.KeyPoints {
			point = strings.TrimSpace(point)
			if point == "" {
				continue
			}
			parts = append(parts, "• "+point)
		}
	}
	if evidence := strings.TrimSpace(r.EvidenceSummary); evidence != "" {
		parts = append(parts, "依据："+evidence)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// Runner 定义 specialist 的有界执行接口。
type Runner interface {
	Run(ctx context.Context, req Request) (Result, error)
}
