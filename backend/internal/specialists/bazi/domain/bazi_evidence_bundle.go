// Package domain 包含不需要 runtime 服务的八字证据合同。
//
// 本文件适配可选检索状态，不裁断命盘方法、要求证据覆盖或改写综合结论。
package domain

type baziQueryPacket = QueryPacket

type baziEvidencePlan = EvidencePlan

type baziEvidenceBundle = EvidenceBundle

type baziEvidenceQuality = EvidenceQuality

// evaluateEvidenceBundleQuality projects optional evidence for trace and prompts.
// It must not turn a missing citation into a synthesis precondition.
func evaluateEvidenceBundleQuality(plan baziEvidencePlan, bundle baziEvidenceBundle) baziEvidenceQuality {
	return EvaluateEvidenceBundleQuality(plan, bundle)
}

// conflictScore maps declared evidence conflicts to a stable severity.
func conflictScore(conflicts []string) string {
	switch {
	case len(conflicts) >= 2:
		return "high"
	case len(conflicts) == 1:
		return "medium"
	default:
		return "low"
	}
}
