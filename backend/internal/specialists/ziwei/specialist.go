// This file belongs to the Zi Wei specialist layer.
// It owns domain specialist configuration for this package.
// It configures Zi Wei worker behavior; final answer ownership stays with Manager.
package ziwei

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

// GetConfig 返回 ziwei specialist 的当前配置，供测试使用。
func GetConfig() specialists.Config {
	return cfg
}
