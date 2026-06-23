.PHONY: build dev dev-backend dev-frontend \
        knowledge-start knowledge-stop knowledge-status knowledge-restart \
        backend-start backend-stop backend-restart \
        frontend-start frontend-stop frontend-status frontend-restart restart status

SERVER_BIN := /tmp/suanming-server
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# ===== 构建 =====
build:
	@cd $(CURDIR) && go build -ldflags="-X github.com/wikiglobal/suanming-agent/internal/container.BuildCommit=$(COMMIT)" -o $(SERVER_BIN) ./cmd/server/
	@echo "Built: $(COMMIT) -> $(SERVER_BIN)"

# ===== 全栈 =====
dev:
	@bash start.sh

status:
	@echo "=== suanming-server ==="
	@curl -s http://localhost:18080/api/health 2>/dev/null | python3 -m json.tool 2>/dev/null || echo "❌ 未运行"
	@echo "=== knowledge MCP ==="
	@$(MAKE) knowledge-status

restart: backend-stop knowledge-stop backend-start knowledge-start
	@echo "=== 全部重启完成 ==="

# ===== 后端 =====
dev-backend:
	@LLM_API_KEY=$$(grep LLM_API_KEY .env | cut -d '=' -f2) go run ./cmd/server/

backend-start: build
	@$(MAKE) backend-stop >/dev/null 2>&1 || true
	@set -a; source $(CURDIR)/.env; set +a; LISTEN_ADDR=:18080 $(SERVER_BIN) &
	@sleep 3
	@curl -s http://localhost:18080/api/health | python3 -c "import sys,json; d=json.load(sys.stdin); print('后端 ✅', d.get('commit',''))"

backend-stop:
	@lsof -ti :18080 | xargs kill 2>/dev/null; echo "后端已停止"

backend-restart: backend-stop backend-start

# ===== 知识库 =====
knowledge-start:
	@$(MAKE) knowledge-stop >/dev/null 2>&1 || true
	@cd knowledge && set -a; source .env.local; set +a; NODE_OPTIONS="--max-old-space-size=4096" npx next dev -p 3100 &
	@sleep 6
	@$(MAKE) knowledge-status

knowledge-stop:
	@lsof -ti :3100 | xargs kill 2>/dev/null; echo "知识库已停止"

knowledge-status:
	@curl -s http://localhost:3100/api/status 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print('知识库 ✅' if d.get('configured') else '❌ 异常')" 2>/dev/null || echo "❌ 未运行"

knowledge-restart: knowledge-stop knowledge-start

knowledge-import:
	@bash scripts/import-smart.py

# ===== 前端 =====
dev-frontend:
	@$(MAKE) frontend-stop >/dev/null 2>&1 || true
	@cd web && npm run dev &
	@sleep 3
	@$(MAKE) frontend-status

frontend-stop:
	@lsof -ti :5173 | xargs kill 2>/dev/null; echo "前端已停止"

frontend-status:
	@lsof -ti :5173 | xargs -I{} echo "前端 ✅ PID:" {} 2>/dev/null || echo "前端 ❌ 未运行"

frontend-restart: frontend-stop dev-frontend
frontend-build:
	@cd web && npm run build

frontend-preview:
	@$(MAKE) frontend-stop-preview >/dev/null 2>&1 || true
	@cd web && npm run preview &
	@sleep 2
	@lsof -ti :4173 | xargs -I{} echo "前端预览 ✅ PID:" {} 2>/dev/null || echo "前端预览 ❌ 未启动"

frontend-stop-preview:
	@lsof -ti :4173 | xargs kill 2>/dev/null; echo "前端预览已停止"

frontend-status-preview:
	@lsof -ti :4173 | xargs -I{} echo "前端预览 ✅ PID:" {} 2>/dev/null || echo "前端预览 ❌ 未运行"
