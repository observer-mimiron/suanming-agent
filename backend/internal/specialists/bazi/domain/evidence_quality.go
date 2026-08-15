// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责证据覆盖和冲突程度的确定性判断；
// 不读取会话，不调用模型、检索、repair、追踪或输出传输。
package domain

// EvaluateEvidenceBundleQuality 只记录可选材料的命中情况。它不再把古籍缺失转成
// 综合合同门槛，确定性命盘事实和模型既有领域知识始终可以完成本轮解读。
func EvaluateEvidenceBundleQuality(plan EvidencePlan, bundle EvidenceBundle) EvidenceQuality {
	_ = plan
	covered := make([]string, 0, len(bundle.TopicBuckets))
	for topic := range bundle.TopicBuckets {
		covered = append(covered, topic)
	}
	quality := EvidenceQuality{Enough: true, FocusScore: "optional", ConflictScore: evidenceConflictScore(bundle.Conflicts), CoveredTopics: covered, DegradedTopics: append([]string(nil), bundle.DegradedTopics...)}
	if len(bundle.Citations) == 0 {
		quality.Reason = "no optional classical passages"
	} else {
		quality.Reason = "optional classical passages available"
	}
	return quality
}

// evidenceConflictScore 将冲突数量投影为固定等级。
func evidenceConflictScore(conflicts []string) string {
	if len(conflicts) >= 2 {
		return "high"
	}
	if len(conflicts) == 1 {
		return "medium"
	}
	return "low"
}
