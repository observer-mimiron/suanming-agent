// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责八字合同审计结果的稳定数据形状；
// 不执行审计，不调用模型、检索、repair、追踪或输出传输。
package domain

// ContractAuditFinding 保存一次八字合同 finding 的稳定元数据。
type ContractAuditFinding struct {
	Code           string `json:"code"`
	Field          string `json:"field"`
	Excerpt        string `json:"excerpt,omitempty"`
	DetectedDomain string `json:"detected_domain,omitempty"`
	Reason         string `json:"reason"`
}

// ContractAudit 是一次八字合同校验的结果摘要，不包含模型调用或恢复决策。
type ContractAudit struct {
	Compliant bool                   `json:"compliant"`
	Findings  []ContractAuditFinding `json:"findings"`
}
