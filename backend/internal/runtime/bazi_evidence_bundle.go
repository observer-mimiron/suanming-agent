package runtime

type baziQueryPacket struct {
	Topic            string   `json:"topic"`
	Query            string   `json:"query"`
	PreferredSources []string `json:"preferred_sources"`
	SourceTier       string   `json:"source_tier"`
}

type baziEvidencePlan struct {
	NeedRetrieval     bool              `json:"need_retrieval"`
	Stage             string            `json:"stage"`
	EvidenceGaps      []string          `json:"evidence_gaps"`
	RecommendedSource []string          `json:"recommended_sources"`
	QueryPackets      []baziQueryPacket `json:"query_packets"`
	AllowReflection   bool              `json:"allow_reflection"`
}

type baziEvidenceBundle struct {
	Stage        string                    `json:"stage"`
	TopicBuckets map[string][]baziCitation `json:"topic_buckets"`
	Citations    []baziCitation            `json:"citations"`
	Conflicts    []string                  `json:"conflicts"`
}

type baziEvidenceQuality struct {
	Enough        bool   `json:"enough"`
	FocusScore    string `json:"focus_score"`
	ConflictScore string `json:"conflict_score"`
	Reason        string `json:"reason"`
}

func evaluateEvidenceBundleQuality(bundle baziEvidenceBundle) baziEvidenceQuality {
	sources := stageAuthoritySources(bundle.Stage)
	authorityCount := 0
	for _, citation := range bundle.Citations {
		if containsString(sources.Primary, citation.Classic) || containsString(sources.Secondary, citation.Classic) {
			authorityCount++
		}
	}
	if authorityCount == 0 {
		return baziEvidenceQuality{
			Enough:        false,
			FocusScore:    "low",
			ConflictScore: "unknown",
			Reason:        "missing primary or secondary authority evidence",
		}
	}
	return baziEvidenceQuality{
		Enough:        true,
		FocusScore:    "high",
		ConflictScore: conflictScore(bundle.Conflicts),
		Reason:        "contains authority-ranked evidence",
	}
}

func shouldReflectOnEvidence(q baziEvidenceQuality) bool {
	if !q.Enough {
		return true
	}
	return q.ConflictScore == "high"
}

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

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
