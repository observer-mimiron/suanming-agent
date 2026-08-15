// Package domain 包含八字领域的不变事实和合同。
//
// 本文件负责 Graph 内可序列化的 repair 失败状态；
// 不持有 runtime Executor、会话、模型、检索或 SSE sink。
package domain

import "github.com/observer-mimiron/suanming-agent/internal/repair"

// RepairFailureState 是可穿过 Graph 状态的 repair 失败投影。
type RepairFailureState struct {
	Domain      string       `json:"domain,omitempty"`
	Stage       string       `json:"stage,omitempty"`
	Class       repair.Class `json:"class,omitempty"`
	Field       string       `json:"field,omitempty"`
	Code        string       `json:"code,omitempty"`
	Message     string       `json:"message,omitempty"`
	Excerpt     string       `json:"excerpt,omitempty"`
	MissingRefs []string     `json:"missing_refs,omitempty"`
	AllowedRefs []string     `json:"allowed_refs,omitempty"`
	Fallback    string       `json:"fallback,omitempty"`
	Retryable   bool         `json:"retryable,omitempty"`
	Repairable  bool         `json:"repairable,omitempty"`
}

// Runtime 将 Graph 状态投影回共享 repair 合同。
func (failure RepairFailureState) Runtime() repair.Failure {
	return repair.Failure{Domain: failure.Domain, Stage: failure.Stage, Class: failure.Class, Field: failure.Field, Code: failure.Code, Message: failure.Message, Excerpt: failure.Excerpt, MissingRefs: append([]string(nil), failure.MissingRefs...), AllowedRefs: append([]string(nil), failure.AllowedRefs...), Fallback: failure.Fallback, Retryable: failure.Retryable, Repairable: failure.Repairable}
}

// ContractFailure 是八字合同失败的可序列化分类状态。
type ContractFailure struct {
	Class          string
	FindingCode    string
	Field          string
	DetectedDomain string
	Excerpt        string
	Reason         string
	MissingRefs    []string
	AllowedRefs    []string
	RecoveryPolicy string
}
