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

## Worker

Proses `worker` menjalankan job River dari antrean `default`, `media`, dan `bulk`. Worker membaca
environment yang sama dengan API dan menyediakan liveness check di
`GK_WORKER_HEALTH_ADDR` (default `:8092`).

```bash
cd apps/api && go run ./cmd/worker
# atau dari root repo
make worker-run

curl -s localhost:8092/healthz
```

Job di-enqueue di transaksi yang sama dengan perubahan data lewat klien insert-only
(`internal/jobs.NewInsertClient`), sehingga perubahan yang di-commit selalu punya job dan perubahan
yang di-rollback tidak meninggalkan job. Tipe argumen job didefinisikan di domain pemiliknya,
misalnya `internal/domain/users/jobs.go`, dengan `Kind()` berformat `<domain>.<action>` dan
`InsertOpts()` yang menentukan antrean serta `MaxAttempts`.

## Admin pertama

Belum ada endpoint untuk membuat admin, jadi akun admin pertama dibuat lewat CLI. CLI membaca
`DATABASE_URL` dari environment (dan `.env` di root repo selama `GK_ENV` bukan `production`).

```bash
cd apps/api
go run ./cmd/admin create --name "Nama Admin" --username admin.sekolah
go run ./cmd/admin create --name "Nama Admin" --username admin.sekolah --staff-number 1987654321
```

Dari root repo:

```bash
make admin-create NAME="Nama Admin" USERNAME=admin.sekolah
```

Perintah ini mencetak sandi awal yang berlaku 7 hari. Admin wajib menyelesaikan aktivasi
(username, sandi baru, dan recovery code) saat pertama kali masuk. Username yang sudah dipakai
atau dicadangkan ditolak dengan exit code `1`; kesalahan pemakaian argumen memakai exit code `2`.

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
