// Package bazi 实现了八字（四柱命理）领域专家。
// 负责用户出生信息校验、命盘状态判断和路由分发，
// 将具体排盘计算委托给编排器处理。
package bazi

import (
	"context"
	"fmt"

	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// Specialist 是八字（四柱命理）领域专家。Phase 1 中作为薄边界层，验证资料完备性并将计算委托给编排器。
type Specialist struct{}

// New 创建并返回一个八字专家实例。
func New() *Specialist {
	return &Specialist{}
}

// Name 返回专家标识符 "bazi"。
func (s *Specialist) Name() string {
	return "bazi"
}

// Run 执行八字专家流程，根据资料完备性和命盘状态决定后续操作。
// Phase 1 中：资料不完整且无命盘则提示澄清，用户正在提供资料则接受并标记需要计算，
// 资料齐备但无命盘则标记需要排盘，已有命盘则进入解读阶段。
// 实际 LLM 调用、工具执行和 SSE 推送由编排器处理。
func (s *Specialist) Run(ctx context.Context, st *state.SessionState, route specialists.ApprovedRoute, sink specialists.EventSink) (schemas.DomainResult, error) {
	profileReady := st.IsProfileComplete()
	hasChart := st.HasBaziResult()

	// 发送追踪跨度标记。
	if sink != nil {
		sink(ctx, specialists.Event{Type: "specialist_bazi", Data: route.TaskIntent})
	}

	// 情况 1：资料不完整且无可用命盘 → 提示澄清
	if !profileReady && !hasChart && route.TaskIntent != "collect_profile" && route.TaskIntent != "amend_profile" && route.TaskIntent != "direct_bazi" {
		return schemas.DomainResult{
			Domain:  "bazi",
			Summary: "请提供您的出生信息（年份、月份、日期、时辰、性别），我来为您排盘分析。",
			Final:   true,
		}, nil
	}

	// 情况 2：资料不完整但用户正在提供资料 → 接受并标记需要计算
	if !profileReady && !hasChart && (route.TaskIntent == "collect_profile" || route.TaskIntent == "amend_profile") {
		return schemas.DomainResult{
			Domain:         "bazi",
			Summary:        "已收到您的出生信息，正在为您排盘分析。",
			Final:          false,
			StructuredData: map[string]any{"stage": "collecting", "profile_complete": false},
		}, nil
	}

	// 情况 3：资料齐全但尚无命盘 → 需要计算
	if profileReady && !hasChart {
		return schemas.DomainResult{
			Domain:         "bazi",
			Summary:        fmt.Sprintf("正在为您排出八字命盘并进行分析。"),
			Final:          false,
			StructuredData: map[string]any{"stage": "computing", "profile_complete": true, "chart_ready": false},
		}, nil
	}

	// 情况 4：已有命盘 → 后续解读
	return schemas.DomainResult{
		Domain:         "bazi",
		Summary:        fmt.Sprintf("基于您的命盘为您分析。"),
		Final:          false,
		StructuredData: map[string]any{"stage": "reading", "profile_complete": true, "chart_ready": true},
	}, nil
}
