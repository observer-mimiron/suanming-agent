package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/observer-mimiron/suanming-agent/internal/observability"
)

func main() {
	var (
		inputPath  = flag.String("input", "../logs/reports/cheap-gate/hits.jsonl", "path to cheap gate JSONL samples")
		outputPath = flag.String("output", "../eval/reports/cheap-gate-summary.json", "path to write summary JSON")
		preview    = flag.Int("preview", 5, "number of raw samples to keep in preview")
	)
	flag.Parse()

	report, err := buildReport(*inputPath, *preview)
	if err != nil {
		log.Fatalf("build cheap gate report: %v", err)
	}
	report.Source = *inputPath

	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("marshal report: %v", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(*outputPath, raw, 0o644); err != nil {
		log.Fatalf("write report: %v", err)
	}
}

func buildReport(inputPath string, preview int) (observability.CheapGateSummaryReport, error) {
	report := observability.CheapGateSummaryReport{
		Dataset:          "cheap-gate-summary",
		Source:           inputPath,
		SourceExists:     false,
		GeneratedAt:      time.Now().Format(time.RFC3339),
		ByPrimaryDomain:  make([]observability.CheapGateCountBucket, 0),
		ByTaskIntent:     make([]observability.CheapGateCountBucket, 0),
		ByGateReason:     make([]observability.CheapGateCountBucket, 0),
		ByExecutionMode:  make([]observability.CheapGateCountBucket, 0),
		ByDecisionSource: make([]observability.CheapGateCountBucket, 0),
		Preview:          make([]observability.CheapGateHit, 0),
	}

	f, err := os.Open(inputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return report, err
	}
	defer f.Close()

	report, err = observability.SummarizeCheapGateHits(f, preview)
	if err != nil {
		return report, err
	}
	report.Source = inputPath
	report.SourceExists = true
	return report, nil
}
