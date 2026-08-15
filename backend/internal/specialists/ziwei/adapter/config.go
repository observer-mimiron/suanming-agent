// 本文件属于紫微 adapter 层。
// 本文件负责把 Ziwei specialist 的提示词、工具白名单和会话注入配置接入公共 runner；
// 不负责排盘、领域规则、Session 写入、trace、SSE 或最终文本。
package adapter

import (
	"github.com/observer-mimiron/suanming-agent/internal/prompts"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
)

var cfg = specialists.Config{
	Domain:               "ziwei",
	Name:                 "ziwei_specialist",
	Description:          "紫微斗数专家。读取 runtime 预执行的命盘和动态事实，分析十二宫星曜布局、四化飞星。",
	Instruction:          prompts.ZiWeiInstruction,
	ToolNames:            []string{"knowledge_catalog", "knowledge_search"},
	InjectSessionContext: true,
}

// GetConfig 返回 Ziwei specialist 的 adapter 配置，供 composition root 注册 runner。
func GetConfig() specialists.Config {
	return cfg
}
