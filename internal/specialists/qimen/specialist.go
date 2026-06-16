// Package qimen 实现了奇门遁甲领域专家。
// 负责择时和方位分析的辅助判断，
// 作为八字排盘结果的补充，不独立主导咨询流程。
package qimen

import (
	"context"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// Specialist 是奇门遁甲领域专家。Phase 1 中作为薄边界层，提供择时辅助信息。
type Specialist struct{}

// New 创建并返回一个奇门专家实例。
func New() *Specialist {
	return &Specialist{}
}

// Name 返回专家标识符 "qimen"。
func (s *Specialist) Name() string {
	return "qimen"
}

// isTimingRelevant 判断路由是否包含择时相关问题。目前通过 PolicyHints.NeedsQimen 标记和 TaskIntent 判断。
func isTimingRelevant(route policy.ApprovedRoute) bool {
	switch route.PolicyHints.QimenMode {
	case "primary", "supplement":
		return true
	case "none":
		return false
	}
	if route.PolicyHints.NeedsQimen {
		return true
	}
	switch route.TaskIntent {
	case "timing_followup", "cross_domain_consult":
		return true
	}
	return false
}

// Run 执行奇门专家流程。Phase 1 中始终返回辅助结果（Final=false），
// 作为八字主线的补充，不会替代主线。非择时路由直接跳过（返回空 domain）。
func (s *Specialist) Run(ctx context.Context, st *state.SessionState, route policy.ApprovedRoute, sink specialists.EventSink) (schemas.DomainResult, error) {
	if sink != nil {
		sink(ctx, specialists.Event{Type: "specialist_qimen", Data: route.TaskIntent})
	}

	// 跳过非择时路由 — 奇门仅对择时问题激活。
	if !isTimingRelevant(route) {
		return schemas.DomainResult{
			Domain: "",
			Final:  false,
		}, nil
	}

	// 阶段一：奇门结果始终为补充性质。
	// 编排器负责实际的奇门工具调用和 SSE 推送。
	return schemas.DomainResult{
		Domain:  "qimen",
		Summary: "qimen timing supplement",
		StructuredData: map[string]any{
			"stage":      "supplemental",
			"time_scope": route.Slots.TimeScope,
		},
		Final: false,
	}, nil
}
