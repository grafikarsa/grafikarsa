# AGENTS.md — aturan kerja di repo Grafikarsa

Berlaku untuk semua agent/kontributor. Spesifikasi produk lengkap ada di `docs/revamp/`
(mulai dari `docs/revamp/README.md`). Jika instruksi tugas bertentangan dengan dokumen ini,
ikuti instruksi tugas, lalu sebutkan perbedaannya di laporan.

## 1. Prinsip umum

- Kerjakan **hanya** yang diminta. Jangan menambah fitur, refactor besar, atau dependensi yang tidak disebut.
- Sentuh hanya file yang diizinkan tugas. Orang lain mengerjakan bagian lain secara paralel.
- **Micro commit**: satu commit per langkah logis. Format: `type(scope): deskripsi singkat bahasa Indonesia`
  (`feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`). Contoh: `feat(api): tambah endpoint login`.
- Jangan pernah commit `.env`, secret, `node_modules`, `.next`, binary.
- Jangan merge PR. Buka PR ke `main`, laporkan URL-nya.
- Bahasa: kode, nama identifier, dan komentar kode **bahasa Inggris**. Teks yang dilihat user (UI, pesan error
  API `title`/`message`) **bahasa Indonesia**.

## 2. Struktur repo

```
apps/api/                 API Go (module github.com/grafikarsa/grafikarsa/apps/api)
  cmd/api/                server HTTP
  cmd/migrate/            migrasi goose (ter-embed) + River
  db/migrations/          migrasi goose (SQL) — sumber kebenaran skema
  db/queries/             query sqlc (*.sql)
  db/seeds/               seed idempoten
  internal/apperr/        error domain yang dipetakan ke problem+json
  internal/domain/<x>/    logika bisnis per domain (auth, user, academic, content, ...)
  internal/http/          router, middleware, problem, handler (api/), kode generate (apigen/)
  internal/platform/      config, db, cache (Valkey), storage (S3), logger
  internal/store/         kode sqlc (generate) + Store.WithTx
  internal/testutil/      helper test integrasi (Postgres sungguhan)
apps/web/                 Next.js (App Router), shadcn/ui, Inter
packages/contracts/       openapi.yaml — sumber kebenaran API
infra/                    compose dev/prod, Caddyfile
docs/revamp/              PRD, skema DB, API, algoritma, dll
```

## 3. Alur kerja API (wajib urut)

1. **Kontrak dulu**: tambah/ubah endpoint di `packages/contracts/openapi.yaml` (OpenAPI 3.0.3).
   - Field JSON **camelCase**. Semua error memakai `$ref: '#/components/responses/Problem'`.
   - Path relatif ke server `/api/v1`. `operationId` camelCase (mis. `postAuthLogin`).
2. `cd apps/api && go generate ./internal/http/apigen/` → commit `apigen.gen.go`.
3. Query SQL di `apps/api/db/queries/<domain>.sql` → `apps/api/scripts/sqlc.sh generate` → commit `internal/store/`.
4. Logika di `internal/domain/<domain>/` (service + test). Domain **tidak** mengimpor paket `http`.
5. Handler di `internal/http/api/<domain>.go` mengimplementasi method `StrictServerInterface`,
   tipis: validasi bentuk → panggil service → petakan hasil ke tipe `apigen`.
6. Web: `pnpm --filter web api:generate` untuk memperbarui klien TypeScript.

CI menolak PR jika kode hasil generate (apigen, sqlc, klien web) tidak sinkron.

## 4. Konvensi Go

- Go 1.26. Format `gofmt`. Wajib lulus `go vet ./...` dan `go test -race ./...`.
- Paket `internal/domain/media/process` (libvips, cgo) hanya ter-compile dengan **build tag `vips`**.
  Tanpa tag (laptop tanpa libvips) semua perintah standar tetap jalan dan worker memakai stub.
  Image worker & CI memakai `-tags vips`. Jangan impor paket itu dari API.
- Error yang **diharapkan** (validasi, tidak ditemukan, konflik, tidak berwenang) dikembalikan sebagai
  `*apperr.Error` (lihat `internal/apperr`). Handler cukup `return nil, err`; error handler strict server
  memetakannya ke problem+json. Error lain = 500 dan dicatat di log (tanpa detail ke klien).
- **Tidak berwenang melihat resource = 404**, bukan 403 (PRD §13.8). 403 hanya untuk kode khusus
  (`ACTIVATION_REQUIRED`, `PASSWORD_CHANGE_REQUIRED`, `CSRF_FAILED`, `CONTENT_BLOCKED`).
- Banyak perubahan data dalam satu aksi = satu transaksi (`store.WithTx`). Job River yang dipicu aksi itu
  di-enqueue **di transaksi yang sama** (`InsertTx`).
- SQL hanya lewat sqlc. Dilarang merangkai SQL dengan string dari input user.
- Filter scope jurusan dilakukan di SQL (`WHERE ... major_id = $n`), bukan memfilter slice di Go.
- Waktu: `time.Time` UTC. ID: `uuid.UUID` (v7 dari DB `uuidv7()`), jangan buat UUID v4.
- Konteks: setiap fungsi I/O menerima `ctx context.Context` sebagai argumen pertama.
- Log: `log/slog`, terstruktur (`logger.Info("msg", "key", val)`). Jangan log sandi, token, recovery code, cookie.

## 5. Test

- Unit test untuk logika murni (tanpa DB) di paket yang sama.
- Test integrasi yang butuh DB memakai `internal/testutil` (Postgres 18 sungguhan via testcontainers,
  migrasi diterapkan otomatis). Jangan mock database.
- Setiap aturan otorisasi wajib punya test "akses langsung ke API tanpa izin ditolak".
- Nama test deskriptif: `TestLogin_WrongPasswordReturnsInvalidCredentials`.

## 6. Konvensi web

- Semua komponen UI dari **shadcn/ui** (`pnpm dlx shadcn@latest add <komponen>`), font **Inter**, ikon lucide.
- Data dari API hanya lewat klien hasil generate (`apps/web/lib/api/gen`). Jangan tulis `fetch` manual ke `/api`.
- Teks UI bahasa Indonesia. Pesan error ramah, bukan teks teknis.
- Wajib lulus `pnpm --filter web lint`, `typecheck`, `build`.

## 7. Keamanan (tidak bisa ditawar)

- Sandi: Argon2id (`internal/platform/password`). Recovery code: HMAC-SHA256 dengan `GK_SECRET_KEY` (kode acak
  berentropi tinggi, bisa dicari lewat index). Token sesi: 32 byte acak, yang disimpan di DB hanya SHA-256-nya
  (`internal/platform/securetoken`).
- Cookie sesi `__Host-gk_session`: HttpOnly, Secure, SameSite=Lax, Path=/.
- Request yang mengubah data wajib lolos CSRF (header `X-CSRF-Token` = cookie `__Host-gk_csrf`).
- Data identitas (NISN/NIP, sandi, token) tidak pernah dikirim ke klien kecuali pemiliknya dan memang perlu.
- Batas panjang input divalidasi di service, DB punya CHECK sebagai pertahanan terakhir.

## 8. Perintah penting

```
make dev-init && make dev-up     # stack dev (Postgres 5439, Valkey 6391, RustFS 9020)
make migrate-up && make seed
make api-run                     # API :8091
make web-dev                     # web :3020
make test | make lint | make generate-check
make dev-reset                   # hapus stack + volume
```
