package tools

import (
	"context"
	"encoding/json"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// EinoDescriber 描述器接口，工具实现此接口可提供 Eino ToolInfo 用于 LLM 函数调用。
type EinoDescriber interface {
	EinoToolInfo() *schema.ToolInfo
}

// legacyToolAdapter 将传统 Tool 接口适配为 Eino 的 InvokableTool 接口。
// 通过 JSON 序列化完成参数解析和结果返回。
type legacyToolAdapter struct {
	tool Tool
	info *schema.ToolInfo
}

// Info 返回工具的元数据描述（名称、参数、说明）。
func (a *legacyToolAdapter) Info(_ context.Context) (*schema.ToolInfo, error) {
	return a.info, nil
}

// InvokableRun 执行工具调用。入参为 JSON 字符串，反序列化后委托给内部 Tool 执行，结果序列化为 JSON 返回。
func (a *legacyToolAdapter) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	var params map[string]any
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return "", err
	}

	result, err := a.tool.Execute(ctx, params)
	if err != nil {
		return "", err
	}

	b, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
