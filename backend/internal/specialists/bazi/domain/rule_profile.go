// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责八字规则画像及其可引用的古籍资料值对象；
// 不读取会话，不调用模型、检索、repair、追踪或输出传输。
package domain

// Citation 是一条可由八字规则或证据引用的古籍原文。
type Citation struct {
	Classic string   `json:"classic"`
	Quotes  []string `json:"quotes"`
}

// RuleProfile 表示单个命盘允许使用的、已版本化的规则族。
// 事实可复用，但判定范围必须由该画像显式授权。
type RuleProfile struct {
	ID             string               `json:"id"`
	Status         string               `json:"status"`
	Scope          string               `json:"scope"`
	NotImplemented []string             `json:"not_implemented"`
	Overlays       []RuleProfileOverlay `json:"overlays,omitempty"`
	Verdicts       []ProfileRuleVerdict `json:"verdicts,omitempty"`
	Claims         []ProfileClaim       `json:"claims,omitempty"`
}

// ProfileClaim 是规则画像明确授权的结论，不是可被渲染层强化的完整裁断。
type ProfileClaim struct {
	ID           string     `json:"id"`
	Category     string     `json:"category"`
	Verdict      string     `json:"verdict"`
	Confidence   string     `json:"confidence"`
	RuleID       string     `json:"rule_id"`
	SupportFacts []string   `json:"support_facts"`
	CounterFacts []string   `json:"counter_facts,omitempty"`
	Boundary     string     `json:"boundary"`
	Citations    []Citation `json:"citations,omitempty"`
}

// RuleProfileOverlay 表示仅在声明范围内生效的附加规则族。
type RuleProfileOverlay struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Scope  string `json:"scope"`
}

// ProfileRuleVerdict 是规则画像能根据命盘事实确定给出的受限结论。
// 它与模型的整盘综合分离，避免窄规则被扩写为宽泛预测。
type ProfileRuleVerdict struct {
	RuleID    string     `json:"rule_id"`
	Status    string     `json:"status"`
	Summary   string     `json:"summary"`
	Facts     []string   `json:"facts"`
	Boundary  string     `json:"boundary"`
	Citations []Citation `json:"citations"`
}
