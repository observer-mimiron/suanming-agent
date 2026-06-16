package tools

import (
	"context"

	qmtool "github.com/wikiglobal/suanming-agent/internal/tools/qimen"
)

// QimenTool 奇门遁甲排盘工具（时家奇门）。委托给 internal 目录下的 qimen 包实现，返回九宫八门八神信息。
type QimenTool struct {
	inner qmtool.Tool
}

func (t *QimenTool) Name() string        { return t.inner.Name() }
func (t *QimenTool) Description() string { return t.inner.Description() }
func (t *QimenTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	return t.inner.Execute(ctx, params)
}
