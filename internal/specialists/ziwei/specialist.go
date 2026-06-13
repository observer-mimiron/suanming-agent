package ziwei

import (
	"context"
	"fmt"

	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// Specialist 紫微斗数领域专家，负责资料完备性检查和路由分发
type Specialist struct{}

func New() *Specialist { return &Specialist{} }

func (s *Specialist) Name() string { return "ziwei" }

func (s *Specialist) Run(ctx context.Context, st *state.SessionState, route specialists.ApprovedRoute, sink specialists.EventSink) (schemas.DomainResult, error) {
	profileComplete := st.IsProfileComplete()

	// 无资料、未见命盘，且非资料收集意图 → 提示澄清
	if !profileComplete && route.TaskIntent != "collect_profile" && route.TaskIntent != "amend_profile" {
		return schemas.DomainResult{
			Domain:  "ziwei",
			Summary: fmt.Sprintf("需要出生信息才能排紫微斗数命盘。请提供出生年月日时和性别。"),
			Final:   true,
		}, nil
	}

	// 资料齐全但没有命盘 → 路由到排盘
	if profileComplete && !st.HasZiWeiResult() {
		return schemas.DomainResult{
			Domain: "ziwei",
			Summary: "资料齐全，开始紫微斗数排盘",
			Final:   false,
		}, nil
	}

	// 已有命盘 → 复用
	if st.HasZiWeiResult() {
		return schemas.DomainResult{
			Domain: "ziwei",
			Summary: "复用已有紫微斗数命盘",
			Final:   false,
		}, nil
	}

	return schemas.DomainResult{Domain: "ziwei", Final: false}, nil
}
