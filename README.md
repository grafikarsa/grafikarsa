# Grafikarsa

Katalog portfolio + jaringan sosial warga **SMKN 4 Malang**. Monorepo: Next.js
(hanya ini yang diekspos) + API Go, Postgres, DragonflyDB, dan object storage
kompatibel-S3 (default RustFS).

## Stack

| Bagian | Teknologi |
|---|---|
| Web | Next.js (App Router) + Tailwind + shadcn/ui + TanStack Query + Inter |
| API | Go + chi + sqlc + pgx, migrasi goose, auth JWT + Argon2id |
| Data | PostgreSQL 16, DragonflyDB (cache/rate limit), storage S3-compatible |
| Edge | Caddy (TLS otomatis), hanya service `web` yang diekspos |
| Worker | Proses Go terpisah: outbox, stats, leaderboard, changelog, digest, retensi |

## Mulai cepat (dev)

```bash
make dev-init             # .env + secret acak (idempotent)
make dev                  # db + dragonfly + storage + api + worker
make dev-web              # terminal 2: Next.js HMR di :3020
make migrate-up && make seed
make verify-seed
```

`make dev-init` wajib jalan pertama kali: `.env.example` sengaja berisi
placeholder (`<rand: ...>`), dan kalau dipakai apa adanya maka DB, JWT, dan
object storage memakai password placeholder yang sama — yang bikin errornya
saja kriptik ("password authentication failed") kalau `.env` diganti
belakangan.

Kalau volume DB sudah ada dari run sebelumnya sementara `.env` sudah diganti,
passwordnya tidak cocok. Reset volumenya:

```bash
docker compose -f docker-compose.yml down -v && make dev
```

Port dev (bisa diubah lewat `.env`): web `3020`, api `8091`, db `5439`,
cache `6390`, storage `9020` (S3) / `9021` (console).

## Perintah penting

```bash
make help                 # daftar semua target
make migrate-up           # goose up (juga: migrate-down, migrate-status, migrate-new)
make seed                 # majors, tahun ajaran, 53 permission, admin
make sqlc                 # regenerate query type-safe
make openapi              # regenerate kontrak OpenAPI dari routes.go
make openapi-check        # pastikan kontrak tidak drift (dipakai CI)
make test                 # unit test backend
make lint                 # gofmt + go vet
make prod                 # simulasi prod lokal (image built)
make deploy-check         # validasi secret/domain sebelum deploy
make backup               # pg_dump dev ke ./backup
```

## Struktur

```
apps/api           API Go + worker (cmd/api, cmd/worker, cmd/tools/*)
apps/web           Next.js (BFF di app/api/[...path]/route.ts)
packages/contracts openapi.yaml (hasil generate)
docs/              prd-v2.md, api-v2.md, runbook.md
infra/             init DB, config pendukung
tools/             gk-backup.sh, verify-backup.sh, server-init.sh, deploy-check.sh
```

## Dokumentasi

| Dokumen | Isi |
|---|---|
| `docs/prd-v2.md` | PRD lengkap: role, fitur, skema, algoritma, roadmap |
| `docs/api-v2.md` | Ringkasan API, auth/CSRF/rate limit, media, job worker |
| `docs/runbook.md` | Operasi harian: start, deploy, backup, restore, observability |
| `packages/contracts/openapi.yaml` | Kontrak endpoint (generate dari kode) |

## Aturan kontribusi singkat

- Route baru di `apps/api/internal/http/routes.go` → jalankan `make openapi`
  (CI menolak kontrak yang stale).
- Perubahan skema → file baru di `apps/api/db/migrations` (JANGAN hapus/ubah
  migrasi yang sudah terpakai), lalu `make migrate-up`.
- Query baru → tulis di `apps/api/db/queries/*.sql`, jalankan `make sqlc`.
- Jangan commit `.env` (hanya `.env.example`).
