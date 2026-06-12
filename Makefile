.PHONY: knowledge-start knowledge-stop knowledge-status knowledge-import

KNOWLEDGE_DIR  := /Users/wikiglobal/workSapce/suanming-agent/knowledge
KNOWLEDGE_PORT := 3100
KNOWLEDGE_PID  := /tmp/knowledge-suanming.pid
KNOWLEDGE_LOG  := /tmp/knowledge-suanming.log

# ─── 知识库服务 ────────────────────────────────────────

knowledge-start:
	@if [ -f $(KNOWLEDGE_PID) ] && kill -0 $$(cat $(KNOWLEDGE_PID)) 2>/dev/null; then \
		echo "✅ 知识库已运行 (pid $$(cat $(KNOWLEDGE_PID)))"; \
	else \
		echo "🚀 启动知识库..."; \
		cd $(KNOWLEDGE_DIR) && PORT=$(KNOWLEDGE_PORT) pnpm dev > $(KNOWLEDGE_LOG) 2>&1 & \
		echo $$! > $(KNOWLEDGE_PID); \
		sleep 3; \
		if curl -s -o /dev/null http://localhost:$(KNOWLEDGE_PORT); then \
			echo "✅ 知识库已启动 http://localhost:$(KNOWLEDGE_PORT)"; \
		else \
			echo "⚠️ 启动中，查看日志: tail -f $(KNOWLEDGE_LOG)"; \
		fi \
	fi

knowledge-stop:
	@if [ -f $(KNOWLEDGE_PID) ]; then \
		kill $$(cat $(KNOWLEDGE_PID)) 2>/dev/null && echo "🛑 知识库已停止" || true; \
		rm -f $(KNOWLEDGE_PID); \
	else \
		echo "⚠️ 知识库未在运行"; \
	fi

knowledge-status:
	@if [ -f $(KNOWLEDGE_PID) ] && kill -0 $$(cat $(KNOWLEDGE_PID)) 2>/dev/null; then \
		echo "✅ 知识库运行中 http://localhost:$(KNOWLEDGE_PORT)"; \
	else \
		echo "❌ 知识库未运行"; \
	fi

knowledge-import:
	@if ! curl -s -o /dev/null http://localhost:$(KNOWLEDGE_PORT); then \
		echo "❌ 知识库未启动，先执行 make knowledge-start"; \
		exit 1; \
	fi
	@python3 scripts/import-smart.py
