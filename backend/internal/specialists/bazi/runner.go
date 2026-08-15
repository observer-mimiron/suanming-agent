// 本文件属于八字 specialist 层。
// 本文件负责按 Manager 已批准的角色选择八字主线或辅助执行委托；
// 不负责 Graph、会话、事件、模型、检索或最终答复。
package bazi

import (
	"context"
	"fmt"

	"github.com/observer-mimiron/suanming-agent/internal/specialists"
)

// Runner 把八字的主线和辅助执行收敛为 runtime 唯一调用的领域入口。
// Primary 与 Support 均由 composition root 提供，Runner 不拥有它们的运行时能力。
type Runner struct {
	Primary specialists.Runner
	Support specialists.Runner
}

// Run 按 Request.Role 委托八字主线或辅助 runner；角色未知或委托缺失时返回错误。
func (r *Runner) Run(ctx context.Context, req specialists.Request) (specialists.Result, error) {
	if r == nil {
		return specialists.Result{}, fmt.Errorf("bazi runner is nil")
	}

	var delegate specialists.Runner
	switch req.Role {
	case specialists.RolePrimary:
		delegate = r.Primary
	case specialists.RoleSupport:
		delegate = r.Support
	default:
		return specialists.Result{}, fmt.Errorf("bazi runner requires known role, got %q", req.Role)
	}
	if delegate == nil {
		return specialists.Result{}, fmt.Errorf("bazi runner missing %s delegate", req.Role)
	}
	return delegate.Run(ctx, req)
}
