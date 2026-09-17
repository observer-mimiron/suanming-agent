// Package runtime 包含 Manager 所有的执行主链。
//
// 本文件只定义 Manager 这一运行时 owner 及其依赖；
// 路由计划、会话投影和最终合成分别位于同 package 的职责文件中。
package runtime

import "github.com/observer-mimiron/suanming-agent/internal/llm"

// Manager 是 runtime 中唯一的用户对话 owner。
// 它负责把批准路由落到当前对象、资产合同和最终回复；领域 worker 只产出受限结果，
// 不能绕过 Manager 直接拥有最终答复权。
type Manager struct {
	flash llm.Chat
}

// NewManager 创建 manager。
func NewManager(flash llm.Chat) *Manager {
	return &Manager{flash: flash}
}
