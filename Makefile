.PHONY: dev dev-backend dev-frontend knowledge-start knowledge-stop knowledge-status knowledge-import

# ===== 服务启动 =====
dev:
	@bash start.sh

dev-backend:
	@LLM_API_KEY=$$(grep LLM_API_KEY .env | cut -d '=' -f2) go run ./cmd/server/

dev-frontend:
	cd web && npm run dev

# ===== 知识库 =====
knowledge-start:
	docker run -d --name suanming-knowledge \
		-p 3100:3100 \
		-v $$(pwd)/knowledge/wiki:/data/wiki \
		wikiglobal/yopedia:latest

knowledge-stop:
	docker stop suanming-knowledge && docker rm suanming-knowledge

knowledge-status:
	@curl -s http://localhost:3100/health 2>/dev/null && echo "✅ 运行中" || echo "❌ 未运行"

knowledge-import:
	@bash scripts/import-smart.py
