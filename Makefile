.PHONY: build test start stop restart dev clean

ROOT := $(shell pwd)
.DEFAULT_GOAL := help

# 从 .env 加载配置
-include .env
export

GO_PORT  := 8080
VUE_PORT := 5173

help: ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-12s\033[0m %s\n", $$1, $$2}'

build: ## 编译前后端
	go build ./cmd/server/
	cd web && npm run build

test: ## 运行全部测试
	go test ./... -v -count=1
	cd web && npx vue-tsc --noEmit

start: ## 启动全部服务（后台）
	@if [ ! -f .env ]; then echo "请先创建 .env 文件: cp .env.example .env"; exit 1; fi
	@if [ -z "$$LLM_API_KEY" ]; then echo "错误: .env 中 LLM_API_KEY 未设置"; exit 1; fi
	@echo "[1/2] 启动 Go 后端 (:$(GO_PORT))..."
	@cd $(ROOT) && go run ./cmd/server/ > /tmp/suanming-go.log 2>&1 & echo $$! > /tmp/suanming-go.pid
	@sleep 1
	@if ! kill -0 $$(cat /tmp/suanming-go.pid) 2>/dev/null; then \
		echo "Go 后端启动失败，查看 /tmp/suanming-go.log"; exit 1; \
	fi
	@echo "[2/2] 启动 Vue 前端 (:$(VUE_PORT))..."
	@cd $(ROOT)/web && npm run dev > /tmp/suanming-vue.log 2>&1 & echo $$! > /tmp/suanming-vue.pid
	@sleep 2
	@echo ""
	@echo "命理大师: http://localhost:$(VUE_PORT)"
	@echo "make stop — 停止  make status — 状态  make logs — 日志"

stop: ## 停止全部服务
	@-kill $$(cat /tmp/suanming-go.pid 2>/dev/null) 2>/dev/null && echo "Go 后端已停止" || true
	@-kill $$(cat /tmp/suanming-vue.pid 2>/dev/null) 2>/dev/null && echo "Vue 前端已停止" || true
	@-lsof -ti:$(GO_PORT) | xargs kill -9 2>/dev/null || true
	@-lsof -ti:$(VUE_PORT) | xargs kill -9 2>/dev/null || true
	@rm -f /tmp/suanming-go.pid /tmp/suanming-vue.pid

restart: stop start ## 重启全部服务

status: ## 查看服务状态
	@echo "Go 后端 (:$(GO_PORT)): $$(if [ -f /tmp/suanming-go.pid ] && kill -0 $$(cat /tmp/suanming-go.pid) 2>/dev/null; then echo '运行中 (PID '$$(cat /tmp/suanming-go.pid)')'; else echo '未运行'; fi)"
	@echo "Vue 前端 (:$(VUE_PORT)): $$(if [ -f /tmp/suanming-vue.pid ] && kill -0 $$(cat /tmp/suanming-vue.pid) 2>/dev/null; then echo '运行中 (PID '$$(cat /tmp/suanming-vue.pid)')'; else echo '未运行'; fi)"

logs: ## 查看日志
	@echo "=== Go 后端 ===" && tail -20 /tmp/suanming-go.log 2>/dev/null || echo "无日志"
	@echo "=== Vue 前端 ===" && tail -20 /tmp/suanming-vue.log 2>/dev/null || echo "无日志"

dev: ## 前台启动（日志直接输出终端）
	@if [ ! -f .env ]; then echo "请先创建 .env 文件: cp .env.example .env"; exit 1; fi
	@if [ -z "$$LLM_API_KEY" ]; then echo "错误: .env 中 LLM_API_KEY 未设置"; exit 1; fi
	@echo "启动后端..."
	@go run ./cmd/server/ &
	@sleep 1
	@echo "启动前端..."
	@cd web && npm run dev &
	@echo "命理大师: http://localhost:$(VUE_PORT)"
	@trap "make stop" EXIT; wait

clean: ## 清理
	rm -rf $(ROOT)/dist $(ROOT)/web/dist /tmp/suanming-go.pid /tmp/suanming-vue.pid
