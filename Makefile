ifneq (,$(wildcard ./.env))
    include .env
    export
endif

dev:
	@make clean
	@docker compose up -d postgres minio
	@echo "Waiting for database..."
	@sleep 5
	@echo "Initializing database..."
	@docker compose exec -T postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) < docs/db/01_db.sql
	@docker compose exec -T postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) < docs/db/02_auth.sql
	@docker compose up -d api web
	@echo "Seeding admin..."
	@make seed-admin
	@docker compose logs -f

seed-admin:
	@docker compose exec api go run cmd/seeder/main.go $(if $(USERNAME),-username=$(USERNAME)) $(if $(EMAIL),-email=$(EMAIL)) $(if $(PASSWORD),-password=$(PASSWORD)) $(if $(NAME),-name=$(NAME))

stop:
	docker compose down

clean:
	docker compose down -v

seed:
	docker compose exec api go run cmd/seed/main.go

migrate:
	docker compose exec api go run cmd/migrate/main.go

test-api:
	@echo "Running Grafikarsa API Complete Test Suite..."
	@./scripts/test-api-complete.sh

