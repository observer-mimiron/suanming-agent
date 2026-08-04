// This file belongs to the manager-owned runtime layer.
// It owns follow-up policy selection for this package.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// resolveFollowupPolicy 由 manager 在构建 ExecutionPlan 时统一决定 follow-up 的处理方式。
// 当前先只保留两种真实可执行模式：
//   - direct: 直接答复，不进入 specialist 执行链
//   - rerun_specialist: 继续进入领域执行链
//
// 之所以暂不放 reuse_artifact，是因为仓库里还没有稳定的“结构化复用解读产物”合同；
// 先把决策权集中，再做真正的复用，避免名义上多模式、实际仍靠隐式分支。
func resolveFollowupPolicy(st *state.SessionState, route policy.ApprovedRoute, message string) (mode string, directAnswer string) {
	if route.TaskIntent != "fortune_followup" {
		return "", ""
	}
	if text, ok := maybeDirectBaziGlossaryFollowup(st, route, message); ok {
		return followupModeDirect, text
	}
	return followupModeRerunSpecialist, ""
}
