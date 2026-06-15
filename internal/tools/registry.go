// Package tools 提供命理分析工具的统一注册与调用接口。
// 包含八字排盘、大运分析、用神推算、奇门遁甲、紫微斗数、知识库检索等工具，
// 支持通过 Eino 框架的 Tool 接口集成到 LLM Agent 工作流中。
package tools

import (
	"context"
	"sort"
	"sync"

	einotool "github.com/cloudwego/eino/components/tool"
)

// Tool 命理工具接口，每个工具需提供名称、描述和执行方法。
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]any) (any, error)
}

// Registry 工具注册中心，管理所有命理工具的注册与查询。
type Registry struct {
	mu        sync.RWMutex
	tools     map[string]Tool
	einoTools map[string]einotool.InvokableTool
}

// NewRegistry 创建并初始化一个新的工具注册中心。
func NewRegistry() *Registry {
	return &Registry{
		tools:     make(map[string]Tool),
		einoTools: make(map[string]einotool.InvokableTool),
	}
}

// Register 注册一个命理工具。如果工具实现了 EinoDescriber 接口，会自动创建对应的 Eino Tool 适配。
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
	if describer, ok := t.(EinoDescriber); ok {
		r.einoTools[t.Name()] = &legacyToolAdapter{
			tool: t,
			info: describer.EinoToolInfo(),
		}
	}
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

// EinoTools 返回按名称排序的 Eino Tool 列表，供 LLM Agent 使用。
func (r *Registry) EinoTools() []einotool.InvokableTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.einoTools))
	for name := range r.einoTools {
		names = append(names, name)
	}
	sort.Strings(names)

	list := make([]einotool.InvokableTool, 0, len(names))
	for _, name := range names {
		list = append(list, r.einoTools[name])
	}
	return list
}
