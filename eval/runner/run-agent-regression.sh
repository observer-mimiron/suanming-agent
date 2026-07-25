#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

if [[ -z "${AGENT_REGRESSION_SERVER:-}" ]]; then
  server_url="http://localhost:8080"
else
  server_url="$AGENT_REGRESSION_SERVER"
fi
langfuse_url="${LANGFUSE_URL:-http://localhost:3001}"
export GOCACHE="${GOCACHE:-/tmp/suanming-go-build-cache}"
mkdir -p "$GOCACHE"

if [[ -f backend/.env ]]; then
  while IFS='=' read -r key value; do
    [[ -z "${key:-}" || "${key:0:1}" == "#" || -z "${value:-}" ]] && continue
    value="${value%$'\r'}"
    value="${value%\"}"; value="${value#\"}"
    value="${value%\'}"; value="${value#\'}"
    export "$key=$value"
  done < backend/.env
fi

export LISTEN_ADDR=:8080
backend_log="$repo_root/backend-regression.log"
backend_err="$repo_root/backend-regression.err.log"
rm -f "$backend_log" "$backend_err"

if ! curl -fsS "$server_url/api/health" >/dev/null 2>&1; then
  go build -o "$repo_root/backend-regression" ./backend/cmd/server/
  "$repo_root/backend-regression" >"$backend_log" 2>"$backend_err" &
  backend_pid=$!
  trap 'kill "$backend_pid" >/dev/null 2>&1 || true' EXIT
  backend_ready=false
  for _ in {1..30}; do
    if curl -fsS "$server_url/api/health" >/dev/null 2>&1; then
      backend_ready=true
      break
    fi
    sleep 1
  done
  if [[ "$backend_ready" != "true" ]]; then
    echo "backend did not become healthy: $server_url/api/health" >&2
    tail -n 80 "$backend_err" >&2 || true
    tail -n 80 "$backend_log" >&2 || true
    exit 1
  fi
fi

go test ./backend/internal/runtime -run 'ExecutionPlan|Manager|GuardFinalAnswer|Prefill|RuntimeFailure' -v
go test ./backend/internal/supervisor ./backend/internal/policy -v
./eval/runner/run-langfuse-eval.sh --dataset-path eval/datasets/runtime-smoke-v1.json --server-url "$server_url" --langfuse-url "$langfuse_url"
