.DEFAULT_GOAL := help

COMPOSE_DEV := docker compose --env-file .env -f infra/compose/dev.yml -p grafikarsa-dev

.PHONY: help dev-init dev-up dev-down dev-reset dev-ps dev-logs db-shell valkey-cli

help: ## Tampilkan daftar target
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

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
