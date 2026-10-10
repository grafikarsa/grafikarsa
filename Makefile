# ==============================================
# GRAFIKARSA v2 — Makefile (DX)
# ==============================================
# Semua target baca .env root. Jalankan dari root repo.
#   cp .env.example .env && make dev

-include .env
export

COMPOSE_DEV := docker compose -f docker-compose.yml
COMPOSE_PROD := docker compose -f docker-compose.prod.yml
API_DIR := apps/api
WEB_DIR := apps/web

DATABASE_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

.PHONY: help dev dev-web dev-down dev-logs \
	migrate-up migrate-down migrate-status migrate-new sqlc seed verify-seed \
	openapi openapi-check \
	backup restore test test-backend test-web lint lint-backend lint-web \
	prod prod-down deploy-check status

help: ## Tampilkan bantuan
	@grep -h -E '^[a-z-]+:.*?## ' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# ── DEV ───────────────────────────────────────
dev: ## Up db + dragonfly + storage + api + worker (web native terpisah)
	$(COMPOSE_DEV) up -d --build db dragonfly storage api worker
	@echo ""
	@echo "  API      http://localhost:$${API_PORT:-8091}/health"
	@echo "  DB       localhost:$${DB_PORT:-5439}"
	@echo "  Cache    localhost:$${DRAGONFLY_PORT:-6390}"
	@echo "  Storage  http://localhost:$${S3_PORT:-9020} (S3)  http://localhost:$${S3_CONSOLE_PORT:-9021} (console)"
	@echo "  Web      jalankan: make dev-web  ->  http://localhost:$${WEB_PORT:-3020}"
	@echo "  Worker   scheduler aktif (outbox 30s, stats 5m, leaderboard 02:05 WIB)"

dev-web: ## Jalankan Next.js native (HMR)
	WEB_PORT=$${WEB_PORT:-3020} pnpm --dir $(WEB_DIR) dev --port $${WEB_PORT:-3020}

dev-down: ## Matikan stack dev (volume dipertahankan)
	$(COMPOSE_DEV) down

dev-logs: ## Tail log semua service dev
	$(COMPOSE_DEV) logs -f --tail=100

# ── DB ────────────────────────────────────────
# Sumber migrasi = apps/api/db/migrations (ter-track git) dan di-embed
# langsung lewat db/migrations/embed.go, jadi tidak ada langkah salin.
migrate-up: ## goose up (via container api one-off)
	$(COMPOSE_DEV) run --rm api go run ./cmd/tools/migrate up

migrate-down: ## goose down 1 level (HATI-HATI di prod)
	$(COMPOSE_DEV) run --rm api go run ./cmd/tools/migrate down

migrate-status: ## Lihat versi migrasi saat ini
	$(COMPOSE_DEV) run --rm api go run ./cmd/tools/migrate status

migrate-new: ## Buat file migrasi baru: make migrate-new NAME=add_x
	@test -n "$(NAME)" || (echo "Usage: make migrate-new NAME=add_x" && exit 1)
	@mkdir -p $(API_DIR)/db/migrations
	@TS=$$(date +%Y%m%d%H%M%S); \
	touch $(API_DIR)/db/migrations/$${TS}_$(NAME).sql; \
	echo "-- +goose Up" > $(API_DIR)/db/migrations/$${TS}_$(NAME).sql; \
	echo "" >> $(API_DIR)/db/migrations/$${TS}_$(NAME).sql; \
	echo "-- +goose Down" >> $(API_DIR)/db/migrations/$${TS}_$(NAME).sql; \
	echo "created $(API_DIR)/db/migrations/$${TS}_$(NAME).sql"

openapi: ## Regenerate kontrak OpenAPI dari routes.go (wajib setelah ubah route)
	@cd $(API_DIR) && go run ./cmd/tools/gen-openapi > ../../packages/contracts/openapi.yaml
	@echo "packages/contracts/openapi.yaml diperbarui"

openapi-check: ## Cek drift kontrak OpenAPI (dipakai CI)
	@cd $(API_DIR) && go run ./cmd/tools/gen-openapi | diff -q ../../packages/contracts/openapi.yaml - >/dev/null \
	  && echo "openapi: sinkron" \
	  || (echo "FATAL: openapi.yaml stale — jalankan make openapi" && exit 1)

sqlc: ## Generate type-safe queries (butuh sqlc terinstall)
	sqlc -f $(API_DIR)/sqlc.yaml generate

seed: ## Seed majors/years/admin/roles (idempoten)
	$(COMPOSE_DEV) run --rm api go run ./cmd/tools/seed

verify-seed: ## Verifikasi seed (counts + admin)
	$(COMPOSE_DEV) run --rm api go run ./cmd/tools/verify-seed

# ── BACKUP (stub Fase 0: pg_dump saja) ─────────
backup: ## Backup DB dev ke ./backup/<date>.dump (+ storage bila rclone ada)
	@mkdir -p backup
	docker exec grafikarsa-db-dev pg_dump -Fc -U $(DB_USER) $(DB_NAME) > backup/db-$$(date +%F).dump
	@echo "backup/db-$$(date +%F).dump"

restore: ## Restore DB: make restore FILE=backup/2026-10-09.dump
	@test -n "$(FILE)" || (echo "Usage: make restore FILE=..." && exit 1)
	docker exec -i grafikarsa-db-dev pg_restore -c -U $(DB_USER) -d $(DB_NAME) < $(FILE)

# ── TEST & LINT ───────────────────────────────
test: test-backend ## Semua test (web menyusul Fase 1+)

test-backend:
	$(COMPOSE_DEV) run --rm api go test ./...

test-web:
	pnpm --dir $(WEB_DIR) test

test-e2e: ## E2E Playwright (jalankan bila sudah terpasang)
	@pnpm --dir $(WEB_DIR) exec playwright test || echo "SKIP: playwright belum dikonfigurasi"

lint: lint-backend ## Semua lint

lint-backend:
	$(COMPOSE_DEV) run --rm api sh -c "gofmt -l . && go vet ./..."

lint-web:
	pnpm --dir $(WEB_DIR) exec tsc --noEmit

# ── PROD SIM ──────────────────────────────────
prod: ## Simulasi prod lokal (api + web built)
	docker compose -f docker-compose.prod.yml up -d --build
	@echo "  Web  http://localhost:$${WEB_PORT:-3020}"

prod-down:
	docker compose -f docker-compose.prod.yml down

deploy-check: ## Validasi secret tidak default sebelum deploy
	@./tools/deploy-check.sh

status:
	$(COMPOSE_DEV) ps
