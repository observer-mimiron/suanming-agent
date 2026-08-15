// 本文件属于有界 specialist 层。
// 本文件负责定义领域 runner 的最小输入与结果合同；
// 不负责最终合成、会话持久化或传输层事件输出。
package specialists

import (
	"context"
	"strings"
)

// SessionTurn 是 specialist 构建对话消息所需的只读消息投影。
type SessionTurn struct {
	Role    string
	Content string
}

// SessionView 是 runtime 从完整会话生成的 specialist 只读输入投影。
// 它只表达提示构建和会话消息需要的资料，不拥有会话、资产或状态写回权。
type SessionView struct {
	Subject        string
	Profile        map[string]any
	BaziResult     map[string]any
	QimenResult    map[string]any
	ZiWeiResult    map[string]any
	RecentTurns    []SessionTurn
	RunningSummary string
}

// Request 是 runtime 调用 specialist runner 时传入的最小执行上下文。
// 它只携带当前 runner 实际需要的问题、领域、只读会话投影和可选回写回调，
// 不传播完整 SessionState、Manager 或其他领域的持久化上下文。
type Request struct {
	UserMessage    string
	Domain         string
	Role           string
	Session        *SessionView
	SaveToolResult func(toolName, resultJSON string)
}

const (
	// RolePrimary 表示本轮执行计划指定的主领域步骤。
	RolePrimary = "primary"
	// RoleSupport 表示为主领域提供补充的领域步骤。
	RoleSupport = "support"
)

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

// Failure 是领域 runner 返回给 runtime 的可分类失败合同。
// 它只传递错误语义和根因，不决定重试、最终答复或 SSE 输出。
type Failure struct {
	Class       string
	Stage       string
	Domain      string
	Code        string
	Retryable   bool
	Degraded    bool
	UserVisible bool
	Message     string
	Cause       error
}

// Error 返回保留给 runtime 映射的失败说明。
func (e *Failure) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Unwrap 返回根因，供 runtime 保留 errors.Is 和 errors.As 语义。
func (e *Failure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
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
