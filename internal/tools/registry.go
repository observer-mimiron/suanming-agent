// Package tools 提供命理分析工具的统一注册与调用接口。
// 包含八字排盘、大运分析、用神推算、奇门遁甲、紫微斗数、知识库检索等工具，
// 支持通过 Eino 框架的 Tool 接口集成到 LLM Agent 工作流中。
package tools

import (
	"context"
	"sync"
)

// Tool 命理工具接口，每个工具需提供名称、描述、展示标签和执行方法。
type Tool interface {
	Name() string
	Description() string
	Label() string
	Execute(ctx context.Context, params map[string]any) (any, error)
}

// Registry 工具注册中心，管理所有命理工具的注册与查询。
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry 创建并初始化一个新的工具注册中心。
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register 注册一个命理工具。
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Get 根据工具名称查询已注册的工具。
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List 返回所有已注册工具的列表。
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}

// DisplayName 返回工具的展示标签。若工具已注册则返回其 Label()，
// 否则降级为内部标识名。
func (r *Registry) DisplayName(name string) string {
	t, ok := r.Get(name)
	if !ok {
		return name
	}
	return t.Label()
}
