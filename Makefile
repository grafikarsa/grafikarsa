.DEFAULT_GOAL := help

COMPOSE_DEV := docker compose --env-file .env -f infra/compose/dev.yml -p grafikarsa-dev
COMPOSE_PROD_LOCAL := docker compose --env-file .env -f infra/compose/prod.yml -p grafikarsa-prodlocal
API := cd apps/api &&

.PHONY: help dev-init dev-up dev-down dev-reset dev-ps dev-logs db-shell valkey-cli \
	migrate-up migrate-down migrate-status seed api-run web-dev sqlc openapi generate-check \
	test test-integration lint prod-local-up prod-local-down

help: ## Tampilkan daftar target
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

dev-init: ## Buat .env dengan secret acak
	./scripts/dev-init.sh

dev-up: ## Jalankan postgres, valkey, rustfs
	@test -f .env || (echo "Jalankan 'make dev-init' dulu" && exit 1)
	$(COMPOSE_DEV) up -d --wait

dev-down: ## Hentikan container dev
	$(COMPOSE_DEV) down

dev-reset: ## Hapus container dan volume dev (data hilang)
	@echo "PERINGATAN: menghapus container dan volume dev"
	$(COMPOSE_DEV) down -v

dev-ps: ## Tampilkan status container dev
	$(COMPOSE_DEV) ps

dev-logs: ## Ikuti log container dev
	$(COMPOSE_DEV) logs -f --tail=100

db-shell: ## Buka psql di container postgres
	$(COMPOSE_DEV) exec postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

valkey-cli: ## Buka valkey-cli di container valkey
	$(COMPOSE_DEV) exec valkey sh -c 'valkey-cli -a "$$VALKEY_PASSWORD" --no-auth-warning'

migrate-up: ## Jalankan migrasi goose + River
	$(API) go run ./cmd/migrate up

migrate-down: ## Mundurkan satu migrasi goose
	$(API) go run ./cmd/migrate down

migrate-status: ## Status migrasi goose + River
	$(API) go run ./cmd/migrate status

seed: ## Jalankan seed idempoten (butuh migrate-up)
	@for f in apps/api/db/seeds/*.sql; do \
		echo "seed $$f"; \
		$(COMPOSE_DEV) exec -T postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -v ON_ERROR_STOP=1 -q' < $$f || exit 1; \
	done

api-run: ## Jalankan API di :8091 (baca .env root)
	$(API) go run ./cmd/api

web-dev: ## Jalankan Next.js dev di :3020
	pnpm --filter web dev

sqlc: ## Generate kode sqlc
	apps/api/scripts/sqlc.sh generate

openapi: ## Generate server Go dari packages/contracts/openapi.yaml
	$(API) go generate ./internal/http/apigen/

generate-check: ## Pastikan kode hasil generate tidak drift
	apps/api/scripts/sqlc.sh generate && git diff --exit-code -- apps/api/internal/store
	@if [ -x apps/api/scripts/check-generated.sh ]; then apps/api/scripts/check-generated.sh; fi

test: ## Unit test API + lint/typecheck web
	$(API) go test -race -count=1 ./...
	pnpm --filter web typecheck

test-integration: ## Test integrasi API (butuh dev-up + migrate-up)
	$(API) set -a && . ../../.env && set +a && go test -tags integration -count=1 ./...

lint: ## gofmt + go vet + eslint
	$(API) test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)
	$(API) go vet ./...
	pnpm --filter web lint

prod-local-up: ## Simulasi produksi lokal di https://localhost:8443
	GK_ENV=staging HTTP_PORT=8080 HTTPS_PORT=8443 $(COMPOSE_PROD_LOCAL) up -d --build --wait

prod-local-down: ## Hentikan simulasi produksi lokal (hapus volume)
	$(COMPOSE_PROD_LOCAL) down -v
