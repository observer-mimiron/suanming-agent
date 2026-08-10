#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
input_path="${1:-logs/reports/cheap-gate/hits.jsonl}"
output_path="${2:-eval/reports/cheap-gate-summary.json}"
preview="${3:-5}"
cd "$repo_root/backend"
go run ./cmd/cheap_gate_report \
  -input "../$input_path" \
  -output "../$output_path" \
  -preview "$preview"
