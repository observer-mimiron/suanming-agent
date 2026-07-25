package runtime

import "github.com/observer-mimiron/suanming-agent/internal/specialists"

// FinalReplyComposer 定义 manager 组合最终回复的最小契约。
type FinalReplyComposer interface {
	ComposeFinalReply(userMessage string, result specialists.Result) string
}
