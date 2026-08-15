// 本文件属于奇门 adapter 层。
// 本文件负责把 Qimen specialist 的提示词、工具白名单和会话注入配置接入公共 runner；
// 不负责排盘、领域规则、Session 写入、trace、SSE 或最终文本。
package adapter

import (
	"github.com/observer-mimiron/suanming-agent/internal/prompts"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
)

var cfg = specialists.Config{
	Domain:               "qimen",
	Name:                 "qimen_specialist",
	Description:          "奇门遁甲专家。分析当前时空的吉凶方位、门星神组合。",
	Instruction:          prompts.QimenInstruction,
	ToolNames:            []string{"knowledge_catalog", "knowledge_search"},
	InjectSessionContext: true,
}

// GetConfig 返回 Qimen specialist 的 adapter 配置，供 composition root 注册 runner。
func GetConfig() specialists.Config {
	return cfg
}
