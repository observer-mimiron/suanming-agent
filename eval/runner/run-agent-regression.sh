#!/usr/bin/env bash
# 评测入口负责本地 Go 合同测试，并执行一次当前构建的 runtime smoke。
# 不运行全量在线数据集，避免普通改动隐式触发额外模型调用。
# 本脚本不定义评测断言，断言仍由 runtime-smoke 数据集和 runner 负责。
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

# A smoke run drives real LLM calls against shared local services. Concurrent
# invocations can mix sessions and make a failed run look like a runtime fault.
run_dir="/tmp/suanming-agent"
mkdir -p "$run_dir"
exec 9>"$run_dir/agent-regression.lock"
if ! flock -n 9; then
  echo "agent regression is already running; wait for it to finish before starting another run" >&2
  exit 1
fi

if [[ -n "${AGENT_REGRESSION_SERVER:-}" ]]; then
  server_url="$AGENT_REGRESSION_SERVER"
  owns_backend=false
else
  regression_port="${AGENT_REGRESSION_PORT:-18080}"
  server_url="http://127.0.0.1:${regression_port}"
  owns_backend=true
fi
langfuse_url="${LANGFUSE_URL:-http://localhost:3001}"
if [[ -f backend/.env ]]; then
  while IFS='=' read -r key value; do
    [[ -z "${key:-}" || "${key:0:1}" == "#" || -z "${value:-}" ]] && continue
    value="${value%$'\r'}"
    value="${value%\"}"; value="${value#\"}"
    value="${value%\'}"; value="${value#\'}"
    export "$key=$value"
  done < backend/.env
fi

backend_bin="$run_dir/backend-regression-${BASHPID}"
backend_log="$run_dir/backend-regression-${BASHPID}.log"
backend_err="$run_dir/backend-regression-${BASHPID}.err.log"
backend_pid=""

cleanup() {
  if [[ -n "$backend_pid" ]]; then
    kill "$backend_pid" >/dev/null 2>&1 || true
    wait "$backend_pid" >/dev/null 2>&1 || true
  fi
  rm -f "$backend_bin" "$backend_log" "$backend_err"
}
trap cleanup EXIT

if [[ "$owns_backend" == "true" ]]; then
  export LISTEN_ADDR=":${regression_port}"
  go build -ldflags="-X github.com/observer-mimiron/suanming-agent/internal/container.BuildCommit=$(git rev-parse --short HEAD)" -o "$backend_bin" ./backend/cmd/server/
  "$backend_bin" >"$backend_log" 2>"$backend_err" &
  backend_pid=$!
else
  if ! curl -fsS "$server_url/api/health" >/dev/null 2>&1; then
    echo "configured regression server is unhealthy: $server_url/api/health" >&2
    exit 1
  fi
fi

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

report_path="${AGENT_REGRESSION_REPORT:-$run_dir/runtime-smoke-report.json}"
go test ./backend/internal/specialists/bazi/presentation -count=1
go test ./backend/internal/runtime -run 'ExecutionPlan|Manager|GuardFinalAnswer|Prefill|RuntimeFailure' -v
go test ./backend/internal/state ./backend/internal/tools/bazi -run 'SessionAssets|ExportsDeterministicDayunContract|FindCurrentDayunAt' -v
go test ./backend/internal/supervisor ./backend/internal/policy -v
if ! curl -fsS "$langfuse_url/api/public/health" >/dev/null 2>&1; then
  echo "Langfuse is unavailable: $langfuse_url/api/public/health" >&2
  exit 1
fi
./eval/runner/run-langfuse-eval.sh --dataset-path eval/datasets/runtime-smoke-v2.json --server-url "$server_url" --langfuse-url "$langfuse_url" --report-path "$report_path" --include-response
