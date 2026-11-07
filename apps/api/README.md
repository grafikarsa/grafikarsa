# API Grafikarsa

HTTP API Grafikarsa: Go 1.26, chi, pgx, go-redis, dan S3-compatible storage.

## Menjalankan secara lokal

Siapkan dependensi dev dari root repo:

```bash
make dev-init
make dev-up
```

Lalu jalankan API:

```bash
cd apps/api
go run ./cmd/api
```

Loader konfigurasi membaca `.env` di root repo (dicari sampai tiga direktori ke atas) selama `GK_ENV` bukan `production`. Variabel environment yang sudah diset selalu menang.

## Migrasi

Migrasi goose ter-embed di binary dan dijalankan lewat perintah `migrate`. Perintah `up` juga menerapkan migrasi River.

```bash
cd apps/api && go run ./cmd/migrate up
cd apps/api && go run ./cmd/migrate down
cd apps/api && go run ./cmd/migrate down-to 0
cd apps/api && go run ./cmd/migrate status
cd apps/api && go run ./cmd/migrate version
```

## Test

```bash
cd apps/api
go test ./...
```

## Endpoint

| Method | Path | Keterangan |
|---|---|---|
| GET | `/healthz` | Liveness, selalu `200` jika proses hidup |
| GET | `/readyz` | Readiness PostgreSQL, Valkey, dan storage |
| GET | `/api/v1/ping` | Smoke test API |

Error memakai format `application/problem+json` (RFC 9457).
