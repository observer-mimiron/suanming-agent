// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责证据规划、证据汇集与质量结果的纯数据合同；
// 不读取会话，不调用模型、检索、repair、追踪或输出传输。
package domain

// QueryPacket 是一次八字证据检索的受限查询描述。
type QueryPacket struct {
	Topic            string   `json:"topic"`
	Query            string   `json:"query"`
	PreferredSources []string `json:"preferred_sources"`
	SourceTier       string   `json:"source_tier"`
}

// EvidencePlan 是本轮 Graph 的模型规划检索范围；它不表达综合前置条件。
type EvidencePlan struct {
	NeedRetrieval     bool          `json:"need_retrieval"`
	AllowReflection   bool          `json:"allow_reflection"`
	Stage             string        `json:"stage"`
	EvidenceGaps      []string      `json:"evidence_gaps"`
	RecommendedSource []string      `json:"recommended_sources"`
	QueryPackets      []QueryPacket `json:"query_packets"`
}

// EvidenceBundle 是本轮检索产生的按主题聚合的古籍引用。
type EvidenceBundle struct {
	Stage                string                `json:"stage"`
	TopicBuckets         map[string][]Citation `json:"topic_buckets"`
	CriticalTopicBuckets map[string][]Citation `json:"critical_topic_buckets"`
	Citations            []Citation            `json:"citations"`
	Conflicts            []string              `json:"conflicts"`
	DegradedTopics       []string              `json:"degraded_topics"`
}

// EvidenceQuality 是证据覆盖检查的结果，不决定是否执行检索。
type EvidenceQuality struct {
	Enough         bool     `json:"enough"`
	FocusScore     string   `json:"focus_score"`
	ConflictScore  string   `json:"conflict_score"`
	Reason         string   `json:"reason"`
	RequiredTopics []string `json:"required_topics"`
	CoveredTopics  []string `json:"covered_topics"`
	MissingTopics  []string `json:"missing_topics"`
	DegradedTopics []string `json:"degraded_topics"`
}
