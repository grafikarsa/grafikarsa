# Grafikarsa v2 — Runbook (Fase 0)

## Perintah harian

| Perlu | Jalankan |
|---|---|
| Start dev | `cp .env.example .env && make dev` + `make dev-web` |
| Migrasi + seed | `make migrate-up && make seed && make verify-seed` |
| Lihat log | `make dev-logs` |
| Matikan | `make dev-down` |
| Simulasi prod | `make prod` / `make prod-down` |

## Restart order (server)

`db` → `dragonfly` → `storage` → `api` → `web` → `proxy`

## Lokasi log

`docker compose logs -f <service>` (dev) / `docker compose -f docker-compose.deploy.yml logs -f` (server).
