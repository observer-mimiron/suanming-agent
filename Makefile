SHELL := /bin/bash

.PHONY: build dev dev-core dev-backend dev-frontend \
        langfuse-start langfuse-stop langfuse-status langfuse-restart \
        knowledge-start knowledge-stop knowledge-status knowledge-restart \
        backend-start backend-stop backend-restart \
        regression eval-smoke eval-suite cheap-gate-report \
        frontend-start frontend-stop frontend-status frontend-restart restart restart-core status \
        clean clean-logs clean-sessions

SERVER_BIN := /tmp/suanming-server
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
RUN_DIR    := /tmp/suanming-agent
GOCACHE    ?= /tmp/suanming-go-build-cache
export GOCACHE
BACKEND_ENV := $(CURDIR)/backend/.env
BACKEND_PID := $(RUN_DIR)/backend.pid
BACKEND_LOG := $(RUN_DIR)/backend.log
BACKEND_SESSION := suanming-backend
KNOWLEDGE_LOG := $(RUN_DIR)/knowledge.log
KNOWLEDGE_SESSION := suanming-knowledge
FRONTEND_LOG := $(RUN_DIR)/frontend.log
FRONTEND_PORT_FILE := $(RUN_DIR)/frontend.port
FRONTEND_SESSION := suanming-frontend
WSL_NODE_HOME ?= $(HOME)/.local/node-linux
WSL_NODE_BIN := $(WSL_NODE_HOME)/bin
LANGFUSE_COMPOSE := deploy/langfuse/docker-compose.yml

# ===== 构建 =====
build:
	@cd $(CURDIR) && go build -ldflags="-X github.com/observer-mimiron/suanming-agent/internal/container.BuildCommit=$(COMMIT)" -o $(SERVER_BIN) ./backend/cmd/server/
	@echo "Built: $(COMMIT) -> $(SERVER_BIN)"

# ===== 全栈 =====
dev: langfuse-start knowledge-start backend-start frontend-start
	@echo "=== 本地开发栈已启动（含 Langfuse 观测） ==="
	@$(MAKE) status

dev-core: knowledge-start backend-start frontend-start
	@echo "=== 核心开发栈已启动（不启动 Langfuse） ==="
	@$(MAKE) status

status:
	@echo "=== Langfuse (:3001) ==="
	@$(MAKE) langfuse-status
	@echo "=== suanming-server (:8080) ==="
	@curl -s http://localhost:8080/api/health 2>/dev/null | python3 -m json.tool 2>/dev/null || echo "❌ :8080 未运行"
	@echo "=== frontend ==="
	@$(MAKE) frontend-status
	@echo "=== knowledge MCP ==="
	@$(MAKE) knowledge-status

restart:
	@set -e; \
	$(MAKE) frontend-stop; \
	$(MAKE) backend-stop; \
	$(MAKE) knowledge-stop; \
	$(MAKE) langfuse-start; \
	$(MAKE) knowledge-start; \
	$(MAKE) backend-start; \
	$(MAKE) frontend-start; \
	echo "=== 全部重启完成 ==="; \
	$(MAKE) status

restart-core:
	@set -e; \
	$(MAKE) frontend-stop; \
	$(MAKE) backend-stop; \
	$(MAKE) knowledge-stop; \
	$(MAKE) knowledge-start; \
	$(MAKE) backend-start; \
	$(MAKE) frontend-start; \
	echo "=== 核心开发栈重启完成 ==="; \
	$(MAKE) status

# ===== Langfuse =====
langfuse-start:
	@docker compose -f $(LANGFUSE_COMPOSE) up -d >/dev/null
	@set -e; \
	for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55 56 57 58 59 60; do \
		if curl -fsS http://localhost:3001/api/public/health >/dev/null 2>&1; then \
			curl -fsS http://localhost:3001/api/public/health | python3 -c "import sys,json; d=json.load(sys.stdin); print('Langfuse ✅', d.get('version',''))"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "Langfuse ❌ 启动失败"; \
	docker compose -f $(LANGFUSE_COMPOSE) ps; \
	exit 1

langfuse-stop:
	@docker compose -f $(LANGFUSE_COMPOSE) stop >/dev/null
	@echo "Langfuse 已停止"

langfuse-status:
	@curl -s http://localhost:3001/api/public/health 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print('Langfuse ✅', d.get('version',''))" 2>/dev/null || echo "Langfuse ❌ 未运行"

langfuse-restart: langfuse-stop langfuse-start

# ===== 评测 / 回归 =====
regression:
	@bash eval/runner/run-agent-regression.sh

eval-smoke:
	@bash eval/runner/run-langfuse-eval.sh --dataset-path eval/datasets/runtime-smoke-v1.json --server-url http://localhost:8080 --langfuse-url $${LANGFUSE_URL:-http://localhost:3001}

eval-suite:
	@bash eval/runner/run-langfuse-eval-suite.sh --server-url http://localhost:8080 --langfuse-url $${LANGFUSE_URL:-http://localhost:3001}

cheap-gate-report:
	@bash eval/runner/build-cheap-gate-report.sh

# ===== 后端 =====
dev-backend:
	@set -a; [ -f $(BACKEND_ENV) ] && source $(BACKEND_ENV); set +a; DEBUG_TRACE=$${DEBUG_TRACE:-1} go run ./backend/cmd/server/

backend-start: build
	@mkdir -p $(RUN_DIR)
	@$(MAKE) backend-stop >/dev/null 2>&1 || true
	@rm -f $(BACKEND_LOG) $(BACKEND_PID)
	@tmux new-session -d -s $(BACKEND_SESSION) "cd $(CURDIR) && set -a; [ -f $(BACKEND_ENV) ] && source $(BACKEND_ENV); set +a; DEBUG_TRACE=\$${DEBUG_TRACE:-1} LISTEN_ADDR=:8080 $(SERVER_BIN)"
	@set -e; \
	for _ in 1 2 3 4 5 6 7 8 9 10; do \
		tmux capture-pane -pt $(BACKEND_SESSION) -S -200 > $(BACKEND_LOG) 2>/dev/null || true; \
		if curl -fsS http://localhost:8080/api/health >/dev/null 2>&1; then \
			tmux list-panes -t $(BACKEND_SESSION) -F '#{pane_pid}' | head -n 1 > $(BACKEND_PID) 2>/dev/null || true; \
			curl -fsS http://localhost:8080/api/health | python3 -c "import sys,json; d=json.load(sys.stdin); print('后端 ✅', d.get('commit',''))"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "后端 ❌ 启动失败"; \
	tail -n 80 $(BACKEND_LOG) 2>/dev/null || true; \
	exit 1

backend-stop:
	@tmux kill-session -t $(BACKEND_SESSION) 2>/dev/null || true
	@if [ -f $(BACKEND_PID) ]; then kill -9 "$$(cat $(BACKEND_PID))" 2>/dev/null || true; rm -f $(BACKEND_PID); fi
	@lsof -ti :8080 | xargs kill -9 2>/dev/null || true
	@echo "后端已停止"

backend-restart: backend-stop backend-start

# ===== 知识库 =====
knowledge-start:
	@$(MAKE) knowledge-stop >/dev/null 2>&1 || true
	@mkdir -p $(RUN_DIR)
	@rm -f $(KNOWLEDGE_LOG)
	@tmux new-session -d -s $(KNOWLEDGE_SESSION) "cd $(CURDIR)/knowledge && set -a; source .env.local; set +a; if [ -x \"$(WSL_NODE_BIN)/node\" ]; then export PATH=\"$(WSL_NODE_BIN):\$$PATH\"; fi; NODE_OPTIONS=\"--max-old-space-size=4096\" npm run dev -- --hostname 0.0.0.0"
	@set -e; \
	for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55 56 57 58 59 60; do \
		tmux capture-pane -pt $(KNOWLEDGE_SESSION) -S -200 > $(KNOWLEDGE_LOG) 2>/dev/null || true; \
		if curl -fsS http://localhost:3100/api/status >/dev/null 2>&1; then \
			curl -fsS http://localhost:3100/api/status | python3 -c "import sys,json; d=json.load(sys.stdin); print('知识库 ✅' if d.get('configured') else '知识库 ⚠️ 未配置')"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "知识库 ❌ 启动失败"; \
	tail -n 80 $(KNOWLEDGE_LOG) 2>/dev/null || true; \
	exit 1

knowledge-stop:
	@tmux kill-session -t $(KNOWLEDGE_SESSION) 2>/dev/null || true
	@lsof -ti :3100 | xargs kill -9 2>/dev/null || true
	@echo "知识库已停止"

knowledge-status:
	@curl -s http://localhost:3100/api/status 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print('知识库 ✅' if d.get('configured') else '❌ 异常')" 2>/dev/null || echo "❌ 未运行"

knowledge-restart: knowledge-stop knowledge-start

# ===== 前端 =====
dev-frontend:
	@mkdir -p $(RUN_DIR)
	@$(MAKE) frontend-stop >/dev/null 2>&1 || true
	@rm -f $(FRONTEND_PORT_FILE) $(FRONTEND_LOG)
	@tmux new-session -d -s $(FRONTEND_SESSION) "cd $(CURDIR)/web && npm run dev -- --host 0.0.0.0"
	@set -e; \
	for _ in 1 2 3 4 5 6 7 8 9 10; do \
		tmux capture-pane -pt $(FRONTEND_SESSION) -S -200 > $(FRONTEND_LOG) 2>/dev/null || true; \
		port="$$(grep -Eo 'http://localhost:[0-9]+' $(FRONTEND_LOG) 2>/dev/null | tail -n 1 | sed 's#.*:##')"; \
		if [ -n "$$port" ] && curl -fsS "http://localhost:$$port" >/dev/null 2>&1; then \
			echo "$$port" > $(FRONTEND_PORT_FILE); \
			echo "前端 ✅ http://localhost:$$port/ (tmux: $(FRONTEND_SESSION))"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "前端 ❌ 未启动"; \
	tmux capture-pane -pt $(FRONTEND_SESSION) -S -200 > $(FRONTEND_LOG) 2>/dev/null || true; \
	tail -n 60 $(FRONTEND_LOG) 2>/dev/null || true; \
	exit 1

frontend-start: dev-frontend

frontend-stop:
	@tmux kill-session -t $(FRONTEND_SESSION) 2>/dev/null || true
	@rm -f $(FRONTEND_PORT_FILE)
	@echo "前端已停止"

frontend-status:
	@if tmux has-session -t $(FRONTEND_SESSION) 2>/dev/null; then \
		port="$$(cat $(FRONTEND_PORT_FILE) 2>/dev/null || true)"; \
		if [ -n "$$port" ] && curl -fsS "http://localhost:$$port" >/dev/null 2>&1; then \
			echo "前端 ✅ http://localhost:$$port/ (tmux: $(FRONTEND_SESSION))"; \
		else \
			echo "前端 ⚠️ tmux 会话仍在，但端口未就绪"; \
		fi; \
	else \
		echo "前端 ❌ 未运行"; \
	fi

frontend-restart: frontend-stop dev-frontend
frontend-build:
	@cd web && npm run build

frontend-preview:
	@$(MAKE) frontend-stop-preview >/dev/null 2>&1 || true
	@cd web && npm run preview &
	@sleep 2
	@lsof -ti :4173 | xargs -I{} echo "前端预览 ✅ PID:" {} 2>/dev/null || echo "前端预览 ❌ 未启动"

frontend-stop-preview:
	@lsof -ti :4173 | xargs kill -9 2>/dev/null; echo "前端预览已停止"

frontend-status-preview:
	@lsof -ti :4173 | xargs -I{} echo "前端预览 ✅ PID:" {} 2>/dev/null || echo "前端预览 ❌ 未运行"

# ===== 清理 =====
clean: clean-logs clean-sessions
	@echo "=== 清理完成 ==="

clean-logs:
	@rm -rf logs/debug/*.jsonl
	@echo "已清理 logs/debug/ 下的 session 日志"

clean-sessions:
	@rm -rf /tmp/suanming-sessions/*.json 2>/dev/null || true
	@rm -rf /tmp/suanming-cache/* 2>/dev/null || true
	@echo "已清理临时 session 和缓存文件"
