// 本文件属于奇门 application 层。
// 本文件负责把问事盘工具输入和已持久化盘面与本轮问事合同对齐；
// 不负责调用工具、写入 Session、trace/SSE 或生成最终文本。
package application

import (
	"strings"
	"time"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
)

// QuestionTimeParams 构造奇门问事盘的最小工具参数。
// 只返回 question_time，避免出生资料或其他 profile 字段泄漏到问事盘。
func QuestionTimeParams(questionTime time.Time) map[string]any {
	return map[string]any{
		"question_time": questionTime.Format(time.RFC3339),
	}
}

// MatchesStoredCaseChart 判断已有盘面是否严格绑定当前 Case 和问事时间。
// 旧 payload 缺少 owner、purpose、time_source 或盘式字段时必须视为过期，不能复用。
func MatchesStoredCaseChart(chart map[string]any, turn contracts.TurnContext) bool {
	caseID := strings.TrimSpace(turn.CaseID)
	if len(chart) == 0 || stringValue(chart["case_id"]) != caseID {
		return false
	}
	owner, ok := chart["owner_ref"].(map[string]any)
	if !ok || stringValue(owner["kind"]) != "case" || stringValue(owner["id"]) != caseID {
		return false
	}
	if stringValue(chart["purpose"]) != "event_question" || stringValue(chart["time_source"]) != "question_time" {
		return false
	}
	return stringValue(chart["question_time"]) == strings.TrimSpace(turn.QuestionTime) &&
		stringValue(chart["pan_schema"]) == "rotating_8" &&
		stringValue(chart["symbol_system"]) == "eight_gate_eight_god"
}

// stringValue 只接受原始字符串，避免兼容载荷被隐式格式化后误通过合同。
func stringValue(raw any) string {
	value, _ := raw.(string)
	return value
}
