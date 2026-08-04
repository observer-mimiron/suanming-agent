// This file belongs to the local observability layer.
// It owns cheap-gate summary aggregation for this package.
// It summarizes evidence for operators; it is not an acceptance source by itself.
package observability

import (
	"bufio"
	"encoding/json"
	"io"
	"sort"
	"strings"
	"time"
)

// CheapGateCountBucket stores one grouped count in the summary report.
type CheapGateCountBucket struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// CheapGateSummaryReport is the structured rollup view for local cheap gate
// hit samples. It mirrors eval report usage: machine-readable first, small
// enough for humans to scan.
type CheapGateSummaryReport struct {
	Dataset          string                 `json:"dataset"`
	Source           string                 `json:"source"`
	SourceExists     bool                   `json:"source_exists"`
	GeneratedAt      string                 `json:"generated_at"`
	TotalHits        int                    `json:"total_hits"`
	ByPrimaryDomain  []CheapGateCountBucket `json:"by_primary_domain"`
	ByTaskIntent     []CheapGateCountBucket `json:"by_task_intent"`
	ByGateReason     []CheapGateCountBucket `json:"by_gate_reason"`
	ByExecutionMode  []CheapGateCountBucket `json:"by_execution_mode"`
	ByDecisionSource []CheapGateCountBucket `json:"by_decision_source"`
	Preview          []CheapGateHit         `json:"preview"`
}

// SummarizeCheapGateHits rolls up append-only JSONL samples into a compact
// report. It never fails on malformed lines alone; bad lines are skipped so
// local analysis does not block on one broken sample.
func SummarizeCheapGateHits(r io.Reader, previewLimit int) (CheapGateSummaryReport, error) {
	report := CheapGateSummaryReport{
		Dataset:     "cheap-gate-summary",
		GeneratedAt: time.Now().Format(time.RFC3339),
		Preview:     make([]CheapGateHit, 0),
	}
	if previewLimit < 0 {
		previewLimit = 0
	}

	var hits []CheapGateHit
	byPrimaryDomain := map[string]int{}
	byTaskIntent := map[string]int{}
	byGateReason := map[string]int{}
	byExecutionMode := map[string]int{}
	byDecisionSource := map[string]int{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var hit CheapGateHit
		if err := json.Unmarshal([]byte(line), &hit); err != nil {
			continue
		}
		hits = append(hits, hit)
		byPrimaryDomain[normalizeBucketValue(hit.PrimaryDomain)]++
		byTaskIntent[normalizeBucketValue(hit.TaskIntent)]++
		byGateReason[normalizeBucketValue(hit.GateReason)]++
		byExecutionMode[normalizeBucketValue(hit.ExecutionMode)]++
		byDecisionSource[normalizeBucketValue(hit.DecisionSource)]++
	}
	if err := scanner.Err(); err != nil {
		return report, err
	}

	report.TotalHits = len(hits)
	report.ByPrimaryDomain = toSortedBuckets(byPrimaryDomain)
	report.ByTaskIntent = toSortedBuckets(byTaskIntent)
	report.ByGateReason = toSortedBuckets(byGateReason)
	report.ByExecutionMode = toSortedBuckets(byExecutionMode)
	report.ByDecisionSource = toSortedBuckets(byDecisionSource)

	if previewLimit > len(hits) {
		previewLimit = len(hits)
	}
	if previewLimit > 0 {
		report.Preview = append(report.Preview, hits[:previewLimit]...)
	}

	return report, nil
}

func normalizeBucketValue(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "(empty)"
	}
	return v
}

func toSortedBuckets(counts map[string]int) []CheapGateCountBucket {
	buckets := make([]CheapGateCountBucket, 0, len(counts))
	for value, count := range counts {
		buckets = append(buckets, CheapGateCountBucket{
			Value: value,
			Count: count,
		})
	}
	sort.Slice(buckets, func(i, j int) bool {
		if buckets[i].Count == buckets[j].Count {
			return buckets[i].Value < buckets[j].Value
		}
		return buckets[i].Count > buckets[j].Count
	})
	return buckets
}
