// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责一次八字分析的输入数据合同；
// 不读取会话，不调用模型、检索、repair、追踪或输出传输。
package domain

// CharterInput 是八字 Graph 计算所需的窄输入，不包含完整会话状态。
type CharterInput struct {
	UserQuestion string         `json:"user_question"`
	BaziResult   map[string]any `json:"bazi_result"`
	Yongshen     map[string]any `json:"yongshen"`
	Dayun        map[string]any `json:"dayun"`
	Liunian      map[string]any `json:"liunian"`
	RuleProfile  RuleProfile    `json:"selected_rule_profile"`
}
