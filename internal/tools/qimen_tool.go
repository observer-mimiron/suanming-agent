package tools

import (
	"context"

	qmtool "github.com/wikiglobal/suanming-agent/internal/tools/qimen"
)

// QimenTool delegates to the internal qimen package.
type QimenTool struct {
	inner qmtool.Tool
}

func (t *QimenTool) Name() string                                       { return t.inner.Name() }
func (t *QimenTool) Description() string                                { return t.inner.Description() }
func (t *QimenTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	return t.inner.Execute(ctx, params)
}
