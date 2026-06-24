#!/bin/bash
set -e

# 加载 .env
if [ -f .env ]; then
    set -a; source .env; set +a
fi

# 清理旧端口占用
lsof -ti :3100 | xargs kill -9 2>/dev/null || true
lsof -ti :8080 | xargs kill -9 2>/dev/null || true
lsof -ti :5173 | xargs kill -9 2>/dev/null || true

if [ -z "$LLM_API_KEY" ]; then
    echo "错误: 请设置 LLM_API_KEY"
    echo "  方式1: cp .env.example .env 并填入真实 key"
    echo "  方式2: export LLM_API_KEY=sk-xxx"
    exit 1
fi

echo "=== 命理大师 v1.1 ==="

echo "[1/3] 启动知识库..."
cd knowledge && set -a; source .env.local; set +a; NODE_OPTIONS="--max-old-space-size=4096" npx next dev -p 3100 --turbo &
KNPID=$!
cd ..

echo "[2/3] 启动 Go 后端..."
go run ./cmd/server/ &
GOPID=$!

echo "[3/3] 启动前端..."
cd web && npm run dev &
VUEPID=$!

echo ""
echo "命理大师: http://localhost:5173"
echo "Ctrl+C 停止"
trap "kill $KNPID $GOPID $VUEPID 2>/dev/null" EXIT
wait
