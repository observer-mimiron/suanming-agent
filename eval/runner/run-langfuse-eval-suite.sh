#!/usr/bin/env bash
set -euo pipefail
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
report_dir="eval/reports/suite"
server_url="http://localhost:8080"
langfuse_url="http://localhost:3001"
write_scores=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --server-url)
      server_url="$2"
      shift 2
      ;;
    --langfuse-url)
      langfuse_url="$2"
      shift 2
      ;;
    --report-dir)
      report_dir="$2"
      shift 2
      ;;
    --write-scores)
      write_scores="--write-scores"
      shift
      ;;
    *)
      echo "unknown arg: $1" >&2
      exit 2
      ;;
  esac
done

mkdir -p "$repo_root/$report_dir"
for dataset in "$repo_root/eval/datasets/runtime-smoke-v2.json" "$repo_root/eval/datasets/retrieval-benchmark-v1.json" "$repo_root/eval/datasets/bazi-quality-v2.json"; do
  dataset_name="$(basename "$dataset" .json)"
  report_path="$repo_root/$report_dir/$dataset_name.json"
  python3 "$script_dir/run_langfuse_eval.py" \
    --dataset-path "$dataset" \
    --server-url "$server_url" \
    --langfuse-url "$langfuse_url" \
    --report-path "$report_path" \
    $write_scores
done
