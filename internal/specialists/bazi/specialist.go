// Package bazi 实现了八字（四柱命理）领域专家。
// 负责用户出生信息校验、命盘状态判断和路由分发，
// 将具体排盘计算委托给编排器处理。
package bazi

import (
	"context"
	"fmt"

	"github.com/wikiglobal/suanming-agent/internal/policy"
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
func (s *Specialist) Run(ctx context.Context, st *state.SessionState, route policy.ApprovedRoute, sink specialists.EventSink) (schemas.DomainResult, error) {
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

// Register 向 Registry 注册八字领域专家 AgentTool 配置。
// 与现有的 Specialist/DomainHandler 并行存在，Task 8 时才删除旧路径。
func Register(r *specialists.Registry) {
	r.Register(specialists.Config{
		Domain:      "bazi",
		Name:        "bazi_specialist",
		Description: "八字命理专家。根据出生时间排盘、分析用神忌神、解读大运走势。",
		Instruction: `你是八字命理专家。

## 可调用工具
- bazi_calc：排八字四柱命盘（需要年/月/日/时/性别）
- yongshen：分析日主强弱、取用神忌神（需要先有排盘结果）
- dayun_analyzer：分析大运走势、各步大运起止时间（需要先有排盘结果）
- knowledge_search：检索古籍原文（《渊海子平》《滴天髓》等）

## 执行规则
1. 用户提供了出生信息 → 先调 bazi_calc 排盘
2. 排盘后 → 根据需要调 yongshen 或 dayun_analyzer
3. 关键论断前 → 调 knowledge_search 获取古籍原文
4. 综合输出中文解读，引用古籍时标注出处

## 禁止
- 不得自行推算四柱（以系统排盘结果为准）
- 不得跳过排盘直接分析（除非 session 中已有命盘）`,
		ToolNames: []string{"bazi_calc", "yongshen", "dayun_analyzer", "knowledge_search"},
	})
}
