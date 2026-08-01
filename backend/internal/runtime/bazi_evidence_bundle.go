// Package runtime contains the manager-owned BaZi evidence contracts.
//
// This file evaluates retrieval coverage and never adjudicates chart-specific
// methodology or rewrites synthesis conclusions.
package runtime

import (
	"fmt"
	"strings"
)

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
	Stage                string                    `json:"stage"`
	TopicBuckets         map[string][]baziCitation `json:"topic_buckets"`
	CriticalTopicBuckets map[string][]baziCitation `json:"critical_topic_buckets"`
	Citations            []baziCitation            `json:"citations"`
	Conflicts            []string                  `json:"conflicts"`
	DegradedTopics       []string                  `json:"degraded_topics"`
}

type baziEvidenceQuality struct {
	Enough         bool     `json:"enough"`
	FocusScore     string   `json:"focus_score"`
	ConflictScore  string   `json:"conflict_score"`
	Reason         string   `json:"reason"`
	RequiredTopics []string `json:"required_topics"`
	CoveredTopics  []string `json:"covered_topics"`
	MissingTopics  []string `json:"missing_topics"`
	DegradedTopics []string `json:"degraded_topics"`
}

// evaluateEvidenceBundleQuality checks authority evidence against every A-tier
// topic requested by the active plan. B-tier examples cannot satisfy a missing
// critical topic, which keeps retrieval breadth from masquerading as coverage.
func evaluateEvidenceBundleQuality(plan baziEvidencePlan, bundle baziEvidenceBundle) baziEvidenceQuality {
	sources := stageAuthoritySources(bundle.Stage)
	required := requiredEvidenceTopics(plan)
	covered := make([]string, 0, len(required))
	missing := make([]string, 0, len(required))
	for _, topic := range required {
		if topicHasAuthorityEvidence(bundle.CriticalTopicBuckets[topic], sources) {
			covered = append(covered, topic)
		} else {
			missing = append(missing, topic)
		}
	}
	quality := baziEvidenceQuality{
		Enough:         len(missing) == 0,
		FocusScore:     "high",
		ConflictScore:  conflictScore(bundle.Conflicts),
		RequiredTopics: required,
		CoveredTopics:  covered,
		MissingTopics:  missing,
		DegradedTopics: append([]string{}, bundle.DegradedTopics...),
	}
	if len(required) == 0 {
		quality.Enough = false
		quality.FocusScore = "low"
		quality.Reason = "missing A-tier evidence plan"
		return quality
	}
	if len(missing) > 0 {
		quality.FocusScore = "low"
		quality.Reason = fmt.Sprintf("missing authority evidence for critical topics: %s", strings.Join(missing, ", "))
		return quality
	}
	if len(bundle.DegradedTopics) > 0 {
		return baziEvidenceQuality{
			Enough:         quality.Enough,
			FocusScore:     quality.FocusScore,
			ConflictScore:  quality.ConflictScore,
			Reason:         "critical topics covered; some retrieval queries degraded",
			RequiredTopics: quality.RequiredTopics,
			CoveredTopics:  quality.CoveredTopics,
			MissingTopics:  quality.MissingTopics,
			DegradedTopics: quality.DegradedTopics,
		}
	}
	quality.Reason = "all critical topics contain authority-ranked evidence"
	return quality
}

// requiredEvidenceTopics returns unique A-tier topics from the active plan.
func requiredEvidenceTopics(plan baziEvidencePlan) []string {
	var topics []string
	for _, packet := range plan.QueryPackets {
		topic := strings.TrimSpace(packet.Topic)
		if !strings.EqualFold(strings.TrimSpace(packet.SourceTier), "A") || topic == "" || containsString(topics, topic) {
			continue
		}
		topics = append(topics, topic)
	}
	return topics
}

// topicHasAuthorityEvidence requires a citation whose source is recognized for
// the current stage; a merely non-empty search hit is not sufficient.
func topicHasAuthorityEvidence(citations []baziCitation, sources authoritySourceSet) bool {
	for _, citation := range citations {
		if containsString(sources.Primary, citation.Classic) || containsString(sources.Secondary, citation.Classic) {
			return true
		}
	}
	return false
}

// shouldReflectOnEvidence requests a bounded retry only for incomplete or
// conflicting evidence. The retry planner decides whether retrieval can help.
func shouldReflectOnEvidence(q baziEvidenceQuality) bool {
	if !q.Enough {
		return true
	}
	return q.ConflictScore == "high"
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

// containsString reports exact membership in a string slice.
func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
