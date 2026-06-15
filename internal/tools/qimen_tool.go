package tools

import (
	"context"

	"github.com/cloudwego/eino/schema"
	qmtool "github.com/wikiglobal/suanming-agent/internal/tools/qimen"
)

// QimenTool 奇门遁甲排盘工具（时家奇门）。委托给 internal 目录下的 qimen 包实现，返回九宫八门八神信息。
type QimenTool struct {
	inner qmtool.Tool
}

func (t *QimenTool) Name() string                                       { return t.inner.Name() }
func (t *QimenTool) Description() string                                { return t.inner.Description() }
func (t *QimenTool) EinoToolInfo() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: t.Name(),
		Desc: t.Description(),
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"year":   {Type: schema.Number, Desc: "起盘年，1900-2100", Required: true},
			"month":  {Type: schema.Number, Desc: "起盘月，1-12", Required: true},
			"day":    {Type: schema.Number, Desc: "起盘日，1-31", Required: true},
			"hour":   {Type: schema.Number, Desc: "起盘小时，0-23", Required: true},
			"minute": {Type: schema.Number, Desc: "起盘分钟，默认 0"},
		}),
	}
}
func (t *QimenTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	return t.inner.Execute(ctx, params)
}
