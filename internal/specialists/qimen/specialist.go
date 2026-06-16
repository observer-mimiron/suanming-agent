// Package qimen 实现了奇门遁甲领域专家 AgentTool 配置。
// Config 由 Register() 注册到 Registry，由 runtime.AgentBuilder 构建 ChatModelAgent。
package qimen

import "github.com/wikiglobal/suanming-agent/internal/specialists"

// Register 向 Registry 注册奇门遁甲领域专家 AgentTool 配置。
func Register(r *specialists.Registry) {
	r.Register(specialists.Config{
		Domain:      "qimen",
		Name:        "qimen_specialist",
		Description: "奇门遁甲专家。分析当前时空的吉凶方位、门星神组合。",
		Instruction: `你是奇门遁甲专家。

## 可调用工具
- qimen_dunjia：排奇门遁甲盘（需要时间信息）
- knowledge_search：检索古籍原文

## 执行规则
1. 调 qimen_dunjia 排盘获取九宫信息
2. 调 knowledge_search 查相关古籍
3. 分析宫、门、星、神组合，给出吉凶判断

## 注意
- 如果 SessionValues 中已有用户出生时间，排盘时可使用该时间
- 如果 SessionValues 中无时间信息，用当前时间排盘`,
		ToolNames: []string{"qimen_dunjia", "knowledge_search"},
	})
}
