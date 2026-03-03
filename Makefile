# ==============================================
# Grafikarsa Monorepo - Makefile
# ==============================================
# Usage: make help

.PHONY: help dev dev-down dev-web dev-logs prod prod-down prod-logs \
        db-import db-backup db-shell db-reset \
        build push deploy \
        test-backend test-web \
        clean restart status

.DEFAULT_GOAL := help

# Load .env if exists (suppress error if not)
-include .env
export

# ========================================
# Help
# ========================================
help: ## Show this help message
	@echo ""
	@echo "  Grafikarsa Monorepo"
	@echo "  ==================="
	@echo ""
	@echo "  Development:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "    \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# ========================================
# Development (recommended workflow)
# ========================================
dev: ## Start dev: backend (Docker + hot reload) + web (npm run dev)
	@echo ""
	@echo "🚀 Starting development environment..."
	@echo ""
	@echo "📦 Starting backend services (db + minio + backend)..."
	docker compose up -d --build
	@echo ""
	@echo "✅ Backend services started!"
	@echo ""
	@echo "📍 Services:"
	@echo "   Backend API:   http://localhost:8080"
	@echo "   MinIO Console: http://localhost:9001"
	@echo "   Database:      localhost:5432"
	@echo ""
	@echo "🌐 Now start the web frontend in a NEW terminal:"
	@echo ""
	@echo "   cd apps/web && npm install && npm run dev"
	@echo ""
	@echo "   Frontend:      http://localhost:3000"
	@echo ""

dev-web: ## Start web frontend (run in separate terminal)
	@if not exist apps\web\.env.local copy .env apps\web\.env.local
	cd apps/web && npm install && npm run dev

dev-down: ## Stop dev backend services
	@echo "🛑 Stopping development services..."
	docker compose down
	@echo "✅ Development services stopped!"

dev-logs: ## View dev backend logs
	docker compose logs -f

logs-backend: ## View backend container logs only
	docker logs grafikarsa-backend-dev -f

logs-db: ## View database logs only
	docker logs grafikarsa-db-dev -f

# ========================================
# Production simulation (local)
# ========================================
prod: ## Simulate production locally (builds & runs all containers)
	@echo ""
	@echo "🏗️  Building and starting production simulation..."
	@echo ""
	docker compose -f docker-compose.prod.yml up -d --build
	@echo ""
	@echo "✅ Production simulation started!"
	@echo ""
	@echo "📍 Services:"
	@echo "   Frontend:      http://localhost:3000"
	@echo "   Backend API:   http://localhost:8080"
	@echo "   MinIO Console: http://localhost:9001"
	@echo ""

prod-down: ## Stop production simulation
	@echo "🛑 Stopping production simulation..."
	docker compose -f docker-compose.prod.yml down
	@echo "✅ Production simulation stopped!"

prod-logs: ## View production simulation logs
	docker compose -f docker-compose.prod.yml logs -f

prod-status: ## Show production simulation status
	docker compose -f docker-compose.prod.yml ps

# ========================================
# Database
# ========================================
db-import: ## Import database schema from db/db.sql
	@echo "📥 Importing database schema..."
	docker exec -i grafikarsa-db-dev psql -U $(DB_USER) -d $(DB_NAME) < db/db.sql
	@echo "✅ Database schema imported!"

db-backup: ## Backup database to backup_YYYYMMDD.sql
	@echo "💾 Backing up database..."
	docker exec grafikarsa-db-dev pg_dump -U $(DB_USER) $(DB_NAME) > backup_$$(date +%Y%m%d_%H%M%S).sql
	@echo "✅ Database backed up!"

db-shell: ## Open database shell (psql)
	docker exec -it grafikarsa-db-dev psql -U $(DB_USER) -d $(DB_NAME)

db-reset: ## Reset database (WARNING: deletes all data)
	@echo "⚠️  Resetting database..."
	docker exec -it grafikarsa-db-dev psql -U $(DB_USER) -d postgres -c "DROP DATABASE IF EXISTS $(DB_NAME);"
	docker exec -it grafikarsa-db-dev psql -U $(DB_USER) -d postgres -c "CREATE DATABASE $(DB_NAME);"
	@make db-import
	@echo "✅ Database reset complete!"

# ========================================
# Build & Deploy (CI/CD)
# ========================================
build: ## Build production Docker images
	@echo "🏗️  Building production images..."
	docker build -t $(DOCKERHUB_USERNAME)/grafikarsa-backend:latest --target production ./apps/backend
	docker build -t $(DOCKERHUB_USERNAME)/grafikarsa-web:latest --target production \
		--build-arg NEXT_PUBLIC_API_URL=$(NEXT_PUBLIC_API_URL) \
		--build-arg NEXT_PUBLIC_APP_URL=$(NEXT_PUBLIC_APP_URL) \
		./apps/web
	@echo "✅ Images built!"

push: ## Push images to Docker Hub
	@echo "📤 Pushing images..."
	docker push $(DOCKERHUB_USERNAME)/grafikarsa-backend:latest
	docker push $(DOCKERHUB_USERNAME)/grafikarsa-web:latest
	@echo "✅ Images pushed!"

# ========================================
# Testing
# ========================================
test-backend: ## Run backend tests (inside container)
	docker exec grafikarsa-backend-dev go test ./...

test-backend-cover: ## Run backend tests with coverage
	docker exec grafikarsa-backend-dev go test -cover ./...

# ========================================
# Maintenance
# ========================================
clean: ## Clean up ALL Docker resources (volumes included!)
	@echo "🧹 Cleaning up..."
	docker compose down -v 2>/dev/null || true
	docker compose -f docker-compose.prod.yml down -v 2>/dev/null || true
	docker system prune -f
	@echo "✅ Cleanup complete!"

restart: ## Restart dev backend services
	@echo "🔄 Restarting services..."
	docker compose restart
	@echo "✅ Services restarted!"

restart-backend: ## Restart only the backend container
	docker compose restart backend

status: ## Show status of dev services
	@echo "📊 Service Status:"
	@docker compose ps

shell-backend: ## Open shell in backend container
	docker exec -it grafikarsa-backend-dev sh

ps: status ## Alias for status
