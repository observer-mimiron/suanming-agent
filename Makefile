.PHONY: yopedia-start yopedia-stop yopedia-status yopedia-import

YOPEDIA_DIR  := /Users/wikiglobal/workSapce/suanming-agent/yopedia
YOPEDIA_PORT := 3100
YOPEDIA_PID  := /tmp/yopedia-suanming.pid
YOPEDIA_LOG  := /tmp/yopedia-suanming.log

# ─── 知识库服务 ────────────────────────────────────────

yopedia-start:
	@if [ -f $(YOPEDIA_PID) ] && kill -0 $$(cat $(YOPEDIA_PID)) 2>/dev/null; then \
		echo "✅ yopedia 已运行 (pid $$(cat $(YOPEDIA_PID)))"; \
	else \
		echo "🚀 启动 yopedia 知识库..."; \
		cd $(YOPEDIA_DIR) && PORT=$(YOPEDIA_PORT) pnpm dev > $(YOPEDIA_LOG) 2>&1 & \
		echo $$! > $(YOPEDIA_PID); \
		sleep 3; \
		if curl -s -o /dev/null http://localhost:$(YOPEDIA_PORT); then \
			echo "✅ yopedia 已启动 http://localhost:$(YOPEDIA_PORT)"; \
		else \
			echo "⚠️ 启动中，查看日志: tail -f $(YOPEDIA_LOG)"; \
		fi \
	fi

yopedia-stop:
	@if [ -f $(YOPEDIA_PID) ]; then \
		kill $$(cat $(YOPEDIA_PID)) 2>/dev/null && echo "🛑 yopedia 已停止" || true; \
		rm -f $(YOPEDIA_PID); \
	else \
		echo "⚠️ yopedia 未在运行"; \
	fi

yopedia-status:
	@if [ -f $(YOPEDIA_PID) ] && kill -0 $$(cat $(YOPEDIA_PID)) 2>/dev/null; then \
		echo "✅ yopedia 运行中 http://localhost:$(YOPEDIA_PORT)"; \
	else \
		echo "❌ yopedia 未运行"; \
	fi

yopedia-import:
	@if ! curl -s -o /dev/null http://localhost:$(YOPEDIA_PORT); then \
		echo "❌ yopedia 未启动，先执行 make yopedia-start"; \
		exit 1; \
	fi
	@python3 scripts/import-to-yopedia.py
