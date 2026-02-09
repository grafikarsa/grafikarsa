dev:
	docker compose up

stop:
	docker compose down

seed:
	docker compose exec api go run cmd/seed/main.go

migrate:
	docker compose exec api go run cmd/migrate/main.go
