// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责模型可引用事实、规则和关系 ID 的目录值对象；
// 不读取会话，不调用模型、检索、repair、追踪或输出传输。
package domain

import (
	"sort"
	"strings"
)

// ReferenceCatalog 是单轮模型引用的 allow-list。
type ReferenceCatalog struct {
	Facts     map[string]struct{}
	Claims    map[string]struct{}
	Relations map[string]struct{}
}

// ValidateReferenceCatalog 拒绝断言引用未在本轮目录声明的事实、规则或关系 ID。
func ValidateReferenceCatalog(assertions []Assertion, catalog ReferenceCatalog) error {
	for _, assertion := range assertions {
		for _, ref := range assertion.FactRefs {
			id := strings.TrimSpace(string(ref))
			if _, ok := catalog.Facts[id]; !ok {
				return NewValidationError(ViolationUndeclaredFactClaim, "assertions.fact_refs", assertion.ID, "fact_ref is not declared in this reference catalog", []string{id}, sortedReferenceIDs(catalog.Facts))
			}
		}
		for _, ref := range assertion.ClaimRefs {
			id := strings.TrimSpace(string(ref))
			if _, ok := catalog.Claims[id]; !ok {
				return NewValidationError(ViolationUndeclaredFactClaim, "assertions.claim_refs", assertion.ID, "claim_ref is not declared in this reference catalog", []string{id}, sortedReferenceIDs(catalog.Claims))
			}
		}
		for _, ref := range assertion.RelationRefs {
			id := strings.TrimSpace(string(ref))
			if _, ok := catalog.Relations[id]; !ok {
				return NewValidationError(ViolationUndeclaredFactClaim, "assertions.relation_refs", assertion.ID, "relation_ref is not declared in this reference catalog", []string{id}, sortedReferenceIDs(catalog.Relations))
			}
		}
	}
	return nil
}

// sortedReferenceIDs 返回稳定顺序的目录 ID，供错误合同展示允许集合。
func sortedReferenceIDs(values map[string]struct{}) []string {
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
