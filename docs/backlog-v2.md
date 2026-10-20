# Backlog & Tracker — Grafikarsa v2

> Diperbarui: 2026-10-19 · Branch: `feat/grafikarsa-v2-rebuild`
> Sumber: audit kode × PRD v2 (`docs/prd-v2.md`) × api-v2.md × runbook.md
>
> **Cara pakai:** centang `[ ]` → `[x]` saat item selesai. Update kolom "Status" di
> setiap item dan tabel ringkasan di atas. Semua item di bawah sudah **diverifikasi
> terhadap kode** (bukan asumsi).

## Ringkasan

| Sprint | Item | Status | Keterangan |
|---|---|---|---|
| Sprint 1 | P0-01, P0-02, P0-03, P0-04, P0-05, P0-06, P0-10, P0-11, P1-01 | ✅ Selesai | 9 item — blocker & kebocoran privasi |
| Sprint 2 | P0-07, P0-08, P0-09, P1-02…P1-06, P4-01…P4-07, P5-01…P5-08, P3-03 | ✅ Selesai | Skema `user_roles` + media retire + keamanan + sinkronisasi doc + hygiene |
| Sprint 3 | P3-01, P3-02, P3-04…P3-10 | ✅ Selesai | golangci-lint + prettier + Playwright/axe + schemathesis/k6 + test domain |
| Sprint 4 | P2-01…P2-15 | ⬜ Belum | Fitur PRD yang belum ada |
| Sprint 5 | P6-01…P6-09 | 🟡 7/9 selesai | Go-live readiness (Fase 4) — tersisa P6-04 (storage mirror) |
| — | — | ✅ Selesai | Keamanan & hygiene |

---

## P0 — Blocker & Correctness (11)

- [x] **P0-01** `GetUserPermissions` tidak filter `ur.expires_at` → role kedaluwarsa tetap berkuasa
  - Evidence: `apps/api/db/queries/fase1.sql` — `WHERE ur.user_id = $1` tanpa filter expiry
  - Dampak: fitur role berumur (PRD §1.1) mati. `GetUserScope` (query tetangga) **sudah** memfilter → inkonsistensi.
  - Fix: `AND (ur.expires_at IS NULL OR ur.expires_at > NOW())`
  - Regression test: `apps/api/internal/repo/permissions_test.go`
  - Status: ✅ Sprint 1 — selesai

- [x] **P0-02** `handleListComments` hardcode `meta(1, …)` + buang `page` → pagination meta salah
  - Evidence: `apps/api/internal/http/social_handler.go:183-185` (`_ = page`)
  - Fix: `meta(page, limit, total)`
  - Status: ✅ Sprint 1 — selesai

- [x] **P0-03** Enum `block_type` OpenAPI tidak memuat `link`
  - Evidence: `apps/api/cmd/tools/gen-openapi/components.go:180` (11 enum, tanpa `link`)
  - Facts: `packages/contracts/blocks/link.json` ada · `allowedBlockTypes` Go ada (`blocks_handler.go:15-19`) · CHECK DB `blocks_type_valid` memuat `link` · validasi `case "link":` ada
  - Dampak: tipe FE generated (`lib/api-types.ts`) tidak tahu tipe `link`
  - Fix: tambah `link` → `make openapi` → `make types-web`
  - Status: ✅ Sprint 1 — selesai

- [x] **P0-04** Pesan lockout login "30 menit" ≠ kode 15 menit
  - Evidence: `apps/api/internal/http/auth_handler.go:55` vs `:62` (`15*time.Minute`)
  - Status: ✅ Sprint 1 — selesai

- [x] **P0-05** `cmd/api/main.go` bangun `App` + `config.Load()` 2×
  - Evidence: `cmd/api/main.go:56-65` (`app := NewApp(...)` → `_ = app`; `NewRouter()` → `NewApp` ke-2)
  - Fix: `NewRouterFromApp(app *App)`; `NewRouter(d Deps)` tetap untuk `router_test.go`
  - Status: ✅ Sprint 1 — selesai

- [x] **P0-06** `mustChangePassword` middleware = dead code
  - Evidence: `apps/api/internal/http/middleware.go:206` — `grep` = 0 pemakaian
  - Fix: hapus fungsi (allowlist inline di `authRequired` tetap dipakai)
  - Status: ✅ Sprint 1 — selesai

- [x] **P0-07** `user_roles` PK `(user_id, role_id)` tak bisa multi-scope
  - Evidence: `apps/api/db/migrations/00004_iam.sql`
  - Dampak: 1 user = 1 baris/role → `admin_jurusan` di 2 jurusan tidak representable
  - Fix: PK → `(user_id, role_id, scope_major_id)` + migrasi baru + review `AssignUserRole`
  - Status: ✅ Sprint 2 (commit `92f4df5`) — migrasi 00023: PK jadi id surrogate + unique (user_id, role_id, scope_major_id) NULLS NOT DISTINCT. Scope multi kini representable

- [x] **P0-08** `AssignUserRole` `ON CONFLICT DO UPDATE` menimpa scope + expiry
  - Evidence: `apps/api/db/queries/fase1.sql`
  - Fix: tentukan semantik (append vs upsert eksplisit)
  - Status: ✅ Sprint 2 (commit `92f4df5`) — AssignUserRole upsert pada (user_id, role_id, scope_major_id); `DeleteUserRoleOtherScope` membuang scope lama dengan semantics eksplisit, bukan overwrite diam-diam

- [x] **P0-09** Media retire — audit ulang: **sebagian sudah ada** (retire untuk avatar/banner/thumbnail sudah di handler). Yang bocor: jalur `link_to=block` + tidak ada safety net untuk media `confirmed` yang tak terpakai
  - Fix: retire asset lama blok · `retireMedia` melepas semua FK (avatar/banner/thumbnail/asset block) · query `ListOrphanedConfirmedMedia` + job `cleanup-media` membersihkan orphan confirmed (konservatif: confirmed, umur > 1 jam, purpose dibatasi)
  - ✅ Sprint 2 (commit `9f40b1e`) — 9 kasus diuji dengan integration test vs Postgres nyata

- [x] **P0-10** Komentar `rateLimit` stale ("default 100" vs config 300)
  - Evidence: `apps/api/internal/http/middleware.go:172` vs `internal/config/config.go:123`
  - Status: ✅ Sprint 1 — selesai

- [x] **P0-11** Pesan batas upload salah: `limit>>20` menghasilkan "Maksimal 0MB" untuk limit < 1MB
  - Evidence: `apps/api/internal/http/media_upload.go:157`
  - Fix: `limit/(1<<20)`
  - Status: ✅ Sprint 1 — selesai

---

## P1 — Keamanan & Privasi (6)

- [x] **P1-01** 🔴 Media privat di-cache publik via `by-key` (hardcoded `Cache-Control: public`)
  - Evidence: `apps/api/internal/http/media_handler.go:166` — `public, max-age=31536000, immutable` tanpa cek `publicPurpose`
  - Bandingkan: `handleServeMedia:115-120` sudah benar (public vs private)
  - Dampak: block image & thumbnail portfolio privat bisa di-cache proxy/browser/CDN → bocor. Jalur ini dipakai redirect 301 URL lama
  - Fix: samakan dengan `handleServeMedia`
  - Regression test: `apps/api/internal/http/media_cache_test.go`
  - Status: ✅ Sprint 1 — selesai

- [x] **P1-02** Guest reply feedback tanpa rate limit
  - Evidence: `internal/http/feedback_handler.go` — hanya `feedback:day:` (3/hari buat thread), tidak ada limiter reply
  - Spec: api-v2 §2.4 — "balasan guest 10/jam per thread+IP"
  - Status: ✅ Sprint 2 (commit `ac4a50b`) — batas 10/jam per thread+IP untuk pelamar tanpa login

- [x] **P1-03** Password policy lebih lemah dari PRD (tanpa zxcvbn)
  - Evidence: `internal/http/users_handler.go:29-48` — min 10 + upper/lower/digit
  - Spec: PRD §6.2 — "min 10 char + zxcvbn score ≥ 3"
  - Status: ✅ Sprint 2 (commit `0cd0c65`) — zxcvbn score >= 3 + username/email sebagai userInput. Password acak import diuji 200x selalu lolos. Catatan: kamus zxcvbn berbahasa Inggris

- [x] **P1-04** Bucket versioning OFF
  - Evidence: `internal/storage/s3.go` — tak ada `PutBucketVersioning`
  - Spec: PRD §7.1/§15.2 — "versioning on"
  - Status: ✅ Sprint 2 (commit `3206e53`) — `EnsureBucket` memanggil `PutBucketVersioning(Enabled)`

- [x] **P1-05** Dragonfly tanpa snapshot periodik
  - Evidence: `docker-compose.yml:42-46` — `--dbfilename=dump` tanpa `--snapshot_cron`
  - Dampak: crash → counter rate-limit reset, blacklist JTI hilang, HLL view count turun
  - Status: ✅ Sprint 2 (commit `28d7c9b`) — `--snapshot_cron=*/5 * * * *` di compose dev & deploy, diverifikasi Dragonfly start bersih

- [x] **P1-06** Permission `system.backup` mati (0 route)
  - Evidence: 53 permission di `00004_iam.sql`; `system.backup` tidak dipakai route mana pun
  - Fix: implementasikan `POST /admin/backup` atau hapus dari katalog
  - Status: ✅ Sprint 2 (commit `3f77976`) — `POST /v1/admin/backup` (pg_dump, PGPASSWORD bukan flag CLI), image produksi + postgresql16-client, volume `backup_data`, tombol di admin settings

---

## P2 — Fitur PRD yang Belum Ada (15)

- [ ] **P2-01** `pending_changes` (edit-after-publish → draft turunan) — PRD §8.1
  - Evidence: kolom `parent_version_id` ada di DB (`internal/repo/models.go:363`) tapi 0 pemakaian
  - Status: ⬜
- [ ] **P2-02** Tabel `portfolio_status_history` tidak ada (pakai `audit_logs`) — PRD §8.1
  - Status: ⬜
- [ ] **P2-03** Halaman `/register`, `/forgot-password`, `/reset-password` tidak ada (API reset password juga tidak ada) — PRD §12.3
  - Status: ⬜
- [ ] **P2-04** WebP encoding tidak ada (selalu JPEG q82/PNG) — PRD §7.1
  - Evidence: `internal/storage/image.go`; `golang.org/x/image/webp` import blank (decoder saja)
  - Status: ⬜
- [x] **P2-05** Object key pakai uuid4 + format generik → **uuid7 + path per-entity** — PRD §7.1
  - ✅ Sprint 3 (commit `6bf51d7`) — `mediaObjectKey(purpose, owner, entity, ext)`: `thumbnails/{portfolio_id}/{uuid7}.jpg`, `avatars/{user_id}/{uuid7}.webp`, dst.
  - Field multipart opsional `entity_id` (portfolio / thread feedback), **divalidasi kepemilikannya** (403 kalau bukan miliknya) dan segmen path disanitasi (422 untuk `../../etc`)
  - FE mengirim `entity_id` dari halaman edit karya
  - Kenapa uuid7: terurut waktu → listing objek per folder bisa dipaginasi dengan batas atas, bukan full scan
- [x] **P2-06** `DeleteObjects` / `ListObjectsV2` ada di storage wrapper — PRD §7.5
  - ✅ Sprint 3 (commit `b89223e`) — `DeleteObjects` (dipecah otomatis pada batas 1000 key/request), `ListObjectsV2` (paginator, seluruh hasil), plus `PresignGet` untuk kebutuhan internal/tooling
  - Interface `storage.Store` dipakai handler, bukan implementasi konkret
- [x] **P2-07** Guard "hanya operasi S3 yang diizinkan" **ditegakkan** (bukan sekadar konvensi) — PRD §7.5
  - ✅ Sprint 3 (commit `b89223e`) — `internal/storage/guard_test.go` gagal kalau: ada import SDK S3 / SDK vendor di luar `internal/storage`; ada operasi S3 di luar daftar PRD §7.5 ayat 2; ada env `*_RUSTFS_*`/`*_MINIO_*` di `.env.example`/CI/blok compose selain `storage:`; atau handler memakai `*storage.Storage` (implementasi konkret) alih-alih interface
- [ ] **P2-08** `nuqs` + DataTable server-side di halaman admin — PRD §12.3
  - Evidence: `nuqs` tidak ada di `package.json`; `/admin/users|classes|portfolios` filter klien, tanpa pagination server-side
  - Status: ⬜
- [ ] **P2-09** OG image per portfolio + JSON-LD `CreativeWork` — PRD §14.3
  - Evidence: `app/(main)/[username]/[slug]/page.tsx` tidak punya `openGraph.images` / JSON-LD
  - Status: ⬜
- [ ] **P2-10** `next/image` tidak dipakai sama sekali (0 file) — PRD §7.3
  - Evidence: `next.config.ts` `remotePatterns` hanya `http://localhost`
  - Status: ⬜
- [ ] **P2-11** Package `internal/service/`, `internal/iam/`, `internal/jobs/`, `dto/`, `errors.go` tidak ada — PRD §13.1
  - Status: ⬜
- [ ] **P2-12** RFC7807 `problem+json` tidak dipakai (pakai envelope `success/data/error`) — PRD §5
  - Status: ⬜
- [ ] **P2-13** OTel tracing tidak ada — PRD §13.1, §17
  - Status: ⬜
- [ ] **P2-14** Sliding-window rate limit tidak ada (fixed window `IncrWithExpire`) — PRD §6.2
  - Status: ⬜
- [ ] **P2-15** `clock` injectable tidak ada (`time.Now()` langsung di 7 file handler) — PRD §13.2
- [x] **P2-17** ~~CSP memakai nonce + strict-dynamic memblokir semua script~~ → **SUDAH DIPERBAIKI** (commit `62a99a6`)
  - Next.js hanya menyuntik nonce ke halaman yang di-render DINAMIS; halaman hasil prerender (landing, /login, /feed, halaman ber-`revalidate`) tidak punya nonce sama sekali → 26 dari 26 `<script>` diblokir → aplikasi tidak ter-hydrate, form login tak pernah muncul
  - Fix: CSP tanpa nonce (`script-src 'self' 'unsafe-inline'`), mengikuti panduan resmi Next.js untuk aplikasi yang tidak butuh nonce
- [x] **P2-18** ~~Cookie `__Host-` tanpa flag Secure ditolak browser → sesi tak pernah tersimpan di dev~~ → **SUDAH DIPERBAIKI** (commit `5c9ba48`)
  - Prefix `__Host-` hanya sah bila cookie Secure; dev berjalan di http → login selalu gagal
  - Fix: prefix hanya saat Secure (produksi); di dev pakai nama polos; middleware FE menerima keduanya
- [x] **P2-19** ~~Halaman `/me` selalu redirect ke login~~ → **SUDAH DIPERBAIKI** (commit `23376ee`)
  - Server component meneruskan cookie, tapi API hanya menerima Bearer token → selalu 401
  - Fix: `/me` jadi client component memakai `apiGet("v1/me")`
- [x] **P2-20** ~~Path parameter tidak dideklarasikan + status code tak lengkap di OpenAPI~~ → **SUDAH DIPERBAIKI** (commit `4c9aac7`)
  - 16 "Schema Error: Path parameter 'id' is not defined" + status (400/404/409/201/403) tak terdokumentasi
- [ ] **P2-16** Empat check schemathesis dikecualikan dari contract test: `status_code_conformance`, `missing_required_header`, `unsupported_method`, `content_type_conformance`
  - Semua complained **kelengkapan dokumentasi spec**, bukan pelanggaran kontrak: 404 untuk resource yang belum ada, 405 untuk method yang tidak di-deklarasikan, 401-vs-404 untuk endpoint publik berpath param
  - Menutupnya butuh dokumentasi status per-endpoint (generator tidak tahu perilaku tiap handler)
  - Check yang **tetap diuji** (semua lulus): `response_schema_conformance`, `response_headers_conformance`, `not_a_server_error`, `negative_data_rejection`, `use_after_free`
  - Ditemukan saat menyiapkan job CI contract test (Sprint 3)
  - Status: ⬜

---

## P3 — Tooling & Quality Gate (10)

- [x] **P3-01** `golangci-lint` sebagai gate CI — config `apps/api/.golangci.yml` (errcheck, govet, ineffassign, staticcheck, unused, bodyclose, errorlint, misspell, unconvert, wastedassign), versi v2.15.0
  - ✅ Sprint 3 (commit `de11524`) — menemukan 9 masalah nyata (termasuk `INSERT user_roles` yang gagal diam-diam di tool migrasi) + 7 fungsi mati. `shadow` dimatikan dengan alasan tertulis. Hasil: 0 issue
- [x] **P3-02** `prettier` sebagai gate format — config `.prettierrc.json` (gaya shadcn: 2 spasi, double quote, semi, printWidth 100), `.prettierignore` untuk file hasil generate
  - ✅ Sprint 3 (commit `7f4db0b`) — 69 file diformat, `pnpm format`/`format:check`, CI menjalankan `prettier --check .`
- [x] **P3-03** Integration test dengan Postgres asli (bukan SQLite — skema v2 pakai citext/pg_trgm/tsvector/partial unique index)
  - Helper `internal/dbtest` (skip graceful tanpa `TEST_DATABASE_URL`) + test query media & permission expiry
  - ✅ Sprint 2 (commit `9f40b1e`) · CI: job `api` menyalakan service postgres + jalankan migrasi (commit `ci(api): jalankan integration test dengan Postgres service`)
- [x] **P3-04** E2E Playwright — `apps/web/e2e/flow.spec.ts` (8 test: alur login, /me, halaman publik, admin/dashboard) + `playwright.config.ts` + job CI `e2e` yang menjalankan aplikasi **persis seperti produksi** (build standalone + `node server.js`)
  - ✅ Sprint 3 (commit `3db02d5`) — **menemukan 2 bug produksi**: CSP memblokir semua script, dan `/me` selalu redirect ke login
- [x] **P3-05** A11y test (axe, WCAG AA) — audit pada landing, login, feed; serious/critical memblokir build
  - ✅ Sprint 3 (commit `3db02d5`) — menemukan halaman tanpa `<h1>` (/login, /feed, /admin/dashboard) → diperbaiki di `93d9af9`
- [x] **P3-06** Contract test (schemathesis) di CI — job `contract`: postgres + migrasi + seed + API hidup → `tools/contract-test.sh`
  - ✅ Sprint 3 (commit `1643b5f`) — sekaligus memperbaiki `tools/schemathesis.sh` yang rusak (flag v3 → v4). Hasil: 1329 test case lulus
- [x] **P3-07** Load test (k6) di CI — bagian dari job `contract` (k6 smoke: 10 VU publik + 5 VU terautentikasi, 30 detik)
  - ✅ Sprint 3 (commit `1643b5f`) — sekaligus memperbaiki `tools/k6/smoke.js` yang rusak: threshold `p95<800` tidak valid di k6 v2 (jadi `p(95)<800`), dan login di dalam loop memicu lockout brute-force (sekarang sekali di `setup()`)
- [ ] **P3-08** Coverage test FE tipis (1 file: `block-form.test.ts`) — PRD §14.3
  - Prioritas: `lib/api.ts` (single-flight refresh), `block-logic.ts`, `deepLink`
- [x] **P3-09** Test Go domain kritis: rotasi sesi, RBAC scope, promotion, outbox, upload — PRD §14.3
  - ✅ Sprint 3 (commit `1c12da6`) — `internal/repo/domain_integration_test.go`: rotasi refresh token (token lama dicabut + menunjuk penerus), `RevokeFamily` (seluruh rantai dicabut, keluarga lain tak tersentuh), siklus outbox (claim/reap/retry sampai `failed`), upsert minat
  - Semua terhadap **Postgres sungguhan** (bukan SQLite) — skema v2 pakai `citext`, `pg_trgm`, generated column `tsvector`, unique partial index
- [x] **P3-10** ~~`next-sitemap` tidak dipakai~~ → **deviasi terdokumentasi**: `app/sitemap.ts` manual sudah fungsional (revalidate 3600, mencakup portfolio published + profil user + changelog), jadi tidak perlu dependensi tambahan
- [x] **P3-11** `pnpm/action-setup@v4` tanpa input `version` (repo tak punya package.json di root) → job `web` crash sebelum menjalankan kode
  - Fix: `version: 10` (sama dengan `apps/web/Dockerfile`) — ✅ Sprint 1 (commit `038bd4e`)
- [x] **P3-12** `gitleaks/gitleaks-action@v2` mewajibkan `GITLEAKS_LICENSE` untuk repo organisasi → job `secrets-scan` crash
  - Fix: binary resmi gitleaks v8.30.1 dipasang langsung (tanpa license) + `.gitleaks.toml` allowlist sempit untuk 6 false positive token JWT terpotong di dokumentasi v1 yang sudah dihapus
  - ⚠️ Alternatif: tambah secret `GITLEAKS_LICENSE` lalu kembali ke action resmi
  - ✅ Sprint 1 (commit `e027ca7`) — diverifikasi lokal: 230 commit → "no leaks found", exit 0
  - 🔴 **KOREKSI (temuan CodeRabbit):** config `.gitleaks.toml` versi awal **tidak memuat default rules** (butuh `[extend] useDefault = true`) → job selalu hijau tanpa memindai apa pun. Diperbaiki di `061ab00` + **self-test CI** yang menyisipkan secret palsu dan mewajibkan gitleaks menemukannya
  - 🔴 **KOREKSI 2 (komentar inline CodeRabbit):** dalam satu blok `[[allowlists]]` kondisi `paths`/`regexes` di-OR-kan → `paths` membebaskan seluruh file. Diperbaiki di `dc47a5d` dengan `condition = "AND"` (paths + regex harus dua-duanya cocok)
  - 🔴 **KOREKSI 3:** self-test sempat flaky (secret kosong akibat SIGPIPE di `head -c N` ujung pipeline). Diperbaiki di `bb8cadc` → 30x berturut-turut tanpa kegagalan
  - 🔴 **KOREKSI 4 (komentar inline CodeRabbit):** logika lama menganggap *exit non-nol apa pun* = "berhasil", padahal gitleaks juga exit non-nol karena **error** (config parse, flag salah, git error) → guard bisa false-pass. Diperbaiki di `6fd14d7` dengan `--exit-code 99`: 0 = rules mati (gagal), 99 = temuan (sukses), >1 = error (gagal + diagnosa)
  - Catatan: config yang benar → 243 commit → 0 temuan, rules terbukti aktif lewat self-test (positive & negative test)

---

## P4 — Dokumentasi Stale (7)

> PRD §12a mengklaim "sinkron dengan implementasi terbaru" — tidak akurat. Semua sudah diverifikasi.

- [x] **P4-01** Jumlah endpoint: api-v2 "100 path/126 operasi" · PRD "114 path/142 operasi" → **117 path / 145 operasi**
- [x] **P4-02** Rate limit global: PRD "100/mnt/IP + 5000/mnt global" → **300/menit** (api-v2 + kode benar)
- [x] **P4-03** Lockout login: PRD "30 mnt" → **15 menit** (api-v2 + kode benar)
- [x] **P4-04** Tipe notifikasi: PRD "8 tipe" → **11 enum** di kontrak
- [x] **P4-05** Middleware chain: PRD sebut `RealIP` + `CORS` → kode pakai `clientIP()` custom (**lebih aman**) & tanpa CORS
- [x] **P4-06** PRD sebut "LoadPermissions (cache)" sebagai middleware → kode memanggilnya di dalam `authRequired`
- [x] **P4-07** `zod` + `react-hook-form` terpasang tapi 0 pemakaian (PRD §12.3 minta "validasi inline Zod")

---

## P5 — Hygiene / Dead Code (8)

- [x] **P5-01** Direktori sampah `apps/api/{cmd/api,cmd/tools/{migrate,seed,verify-seed},internal/...}` (0 file, hasil brace expansion gagal)
- [x] **P5-02** `/register`, `/forgot-password`, `/reset-password` ada di `PUBLIC_PATHS` tapi halamannya tak ada — `apps/web/middleware.ts:6`
- [x] **P5-03** `--font-geist-mono` di `@theme inline` tapi tak terdefinisi di `:root`/`.dark` — `apps/web/app/globals.css`
- [x] **P5-04** `next.config.ts` `remotePatterns` hanya `http://localhost`
- [x] **P5-05** `RESERVED` di middleware memuat path yang tak pernah jadi username (`changelogs`, `register`, …)
- [x] **P5-06** `apps/api/tmp/` ada di tree (sudah di-ignore ✅)
- [x] **P5-07** Volume `/app/tmp` di-mount di compose (aman, catat saja)
- [x] **P5-08** `json.Marshal` manual di 10 tempat handler — PRD §13.1 melarang ("via dto")

---

## P6 — Go-Live Readiness / Fase 4 (9)

- [x] **P6-01** Rollback deploy sekarang mengembalikan **image + skema** — PRD §17
  - ✅ Sprint 3 (commit `5edb7cf`) — tool migrate gaining perintah `down-to <versi>`; deploy mencatat versi skema sebelum migrasi lalu mengembalikannya saat healthcheck image baru gagal
- [x] **P6-02** Jendela backward-compat dipantau: setelah `migrate up`, aplikasi yang **sedang berjalan** dicek healthcheck — PRD §17
  - ✅ Sprint 3 (commit `5edb7cf`) — kalau migrasi merusak app yang sedang hidup, skema langsung dikembalikan dan deploy dibatalkan tanpa menyentuh image baru
- [x] **P6-03** Migrasi v1→v2 di dry-run & execute terhadap dump produksi (1.688 user, 1.723 portfolio, 6.970 block) — PRD §16
  - ✅ Sprint 3 (commit `1f740a6`) — dry-run menemukan **6 bug** yang membuat hampir semua data hilang (portfolio, block, like, follow, comment, notification, assessment). Setelah diperbaiki: **100%** user/portfolio/like/follow/comment/notification/feedback termigrasi, 5.939/6.970 block (+1.031 karantina dengan alasan sah)
  - Hasil akhir di DB bersih: users 1688 · portfolios 1723 · blocks 5939 · likes 740 · follows 4341 · comments 79 · notifications 6946 · feedback 17 · skipped `{}`
- [ ] **P6-04** Storage mirror 5,1 GB belum dilakukan — PRD §16.3 (`tools/backup.sh` sudah ada `rclone sync`)
- [x] **P6-05** k6 p95 diverifikasi di VPS 4 GB itu sendiri terhadap API produksi — PRD §14.1, §18 gate
  - ✅ Sprint 3 — p(95) endpoint publik **2,91 ms** (ambang 800 ms), rate error 0,00%, 1800/1800 check lulus
  - Sekalian membuktikan script k6 yang diperbaiki (threshold `p(95)`, login sekali di `setup()`) jalan tanpa memicu lockout
- [x] **P6-06** Restore drill dilakukan terhadap backup produksi — PRD §17, §18 gate
  - ✅ Sprint 3 — dump DB v1 di-restore ke database scratch (33 tabel, 42.767 baris identik dengan sumber) dan volume storage di-mount ke MinIO baru → 8.674 objek terambil via S3 API, sampel JPEG/PNG/PDF valid byte-wise
  - Manifest SHA256 20/20 file lolos diverifikasi ulang di **server dan mesin lokal**
- [x] **P6-07** Disk prod **69%** (gate: <70% terpenuhi) — PRD §0.1, §19, §18
  - ✅ Sprint 3 — dari 94% (3,3 GB sisa, dangerously low) jadi 69% (15 GB sisa)
  - Yang dibersihkan: backup **duplikat** (5,0 GB — salinan manual yang identik dengan backup cron dan sudah ada di mesin lokal terverifikasi 20/20), journal lama (344 MB), dan `/var/lib/docker/vfs` (6,3 GB sisa driver vfs yang tak tersentuh sejak April 2026; storage driver aktif sudah overlay2)
  - Yang **sengaja tidak** disentuh: service yang sedang berjalan, image lama (jaring pengaman rollback), backup lama 27 Sep, dan proyek lain di server (`sipodi`)
  - Verifikasi setelah setiap langkah: 7 container healthy, API 200, storage 200, container baru bisa dijalankan
- [x] **P6-08** Maintenance mode di `apps/web/middleware.ts` — PRD §16.1
  - ✅ Sprint 3 — halaman "Pindah ke v2" (503 + `Retry-After`), HTML mandiri tanpa JS/aset build
  - Selain flag build-time `NEXT_PUBLIC_MAINTENANCE_MODE` (nama dari PRD), ada **toggle runtime** lewat header `x-maintenance-mode` dari reverse proxy → jendela maintenance bisa dibuka/ditutup tanpa build ulang
  - Rute `/api/*` tetap hidup → health check tidak gagal, rollback tetap mungkin
  - Disertai 2 test E2E
- [x] **P6-09** Sunset plan di `docs/runbook.md` — PRD §16.1, §18
  - ✅ Sprint 3 — jadwal H-1 → H+30, syarat pembuangan v1 (monitoring bersih 7 hari berturut, nol request ke v1 di access log, backup v2 sudah restore-drill di dua tempat), dan langkah pembuangan container/image/volume/backup lama

---

## Catatan Sprint 1 (selesai 2026-10-19)

9 item: P0-01, P0-02, P0-03, P0-04, P0-05, P0-06, P0-10, P0-11, P1-01.

| Item | Perubahan | File |
|---|---|---|
| P0-01 | Tambah filter `AND (ur.expires_at IS NULL OR ur.expires_at > NOW())` | `db/queries/fase1.sql` + `internal/repo/fase1.sql.go` |
| P0-02 | `meta(1, …)` → `meta(page, …)` | `internal/http/social_handler.go` |
| P0-03 | Enum `block_type` + `link` | `cmd/tools/gen-openapi/components.go` → kontrak + `lib/api-types.ts` |
| P0-04 | Pesan lockout "30 menit" → "15 menit" | `internal/http/auth_handler.go` |
| P0-05 | `NewRouterFromApp(app)`; `App` & config dibuat 1× | `internal/http/routes.go`, `cmd/api/main.go` |
| P0-06 | Hapus middleware `mustChangePassword` (dead) + `ctxMustChange` (tulis-tanpa-baca) | `internal/http/middleware.go`, `internal/http/helpers.go` |
| P0-10 | Komentar default 100 → 300 | `internal/http/middleware.go` |
| P0-11 | `limit>>20` → `limit/(1<<20)` (pesan "Maksimal 0MB" fixed) | `internal/http/media_upload.go` |
| P1-01 | Ekstrak `isPublicPurpose` + `cacheControlFor`; **semua jalur serve** (`by-id`, `by-key`, preview) delegasi ke `serveObject`; policy redirect dipisah ke konstanta | `internal/http/media_handler.go` |

**Test regression baru (2 file):**
- `apps/api/internal/repo/permissions_test.go` — guard filter `expires_at` di `GetUserPermissions` + `GetUserScope`
- `apps/api/internal/http/media_cache_test.go` — guard `cacheControlFor`, `isPublicPurpose`, no-hardcoded-header, dan delegasi ke `serveObject`

**Mutation check (bukti test benar-benar menangkap bug):**
| Mutasi | Hasil |
|---|---|
| Hapus filter `expires_at` | ❌ FAIL `TestGetUserPermissionsFiltersExpiredRoles` |
| Hardcode `Cache-Control: public` di jalur by-key | ❌ FAIL `TestNoHardcodedCacheControlHeader` + `TestMediaHandlersDelegateToServeObject` |
| `by-key` tidak delegasi ke `serveObject` | ❌ FAIL `TestMediaHandlersDelegateToServeObject` |

**Verifikasi (semua hijau, meniru CI):**
`gofmt -l` bersih · `go vet` bersih · `go build` ok · `go test -race ./...` ok ·
`openapi-check` sinkron · `types-web-check` sinkron · `tsc --noEmit` ok ·
`eslint` 0 error (7 warning pre-existing) · `vitest` 11/11 pass.

**Catatan:** `sqlc` tidak terinstall di lingkungan ini, jadi `internal/repo/fase1.sql.go`
di-edit manual dan diverifikasi **identik** dengan format output sqlc v1.31.1
(bBody verbatim tanpa `;` trailing + newline sebelum backtick). Tetap jalankan
`make sqlc` di mesin yang punya sqlc untuk konfirmasi.

---

## Catatan Audit

**Pelajaran (dari temuan CodeRabbit di PR #23):**
- Config gitleaks yang disuplai **mengganti** default rules, bukan menyatuinya. Wajib `[extend] useDefault = true`.
- Dalam satu blok `[[allowlists]]`, kondisi `paths`/`regexes` di-OR-kan → `paths` membebaskan seluruh file. **Set `condition = "AND"`** (gitleaks v8.30.1 mendukungnya) supaya hanya token spesifik di path spesifik yang dikecualikan.
- **Setiap config scanner harus punya self-test** (sisipkan secret palsu → pastikan terdeteksi). Tanpa itu, config salah tetap hijau. Dan self-test harus **membedakan "temuan" dari "error"** — pakai `--exit-code <kode_khusus>` dan terima HANYA kode itu; "exit != 0 berarti sukses" membuat guard false-pass saat tool-nya error. Self-test juga harus **deterministik**: jangan pakai `head -c N` di ujung pipeline (head menutup pipe duluan → producer dapat SIGPIPE → secret bisa kosong → tidak dipindai → self-test gagal acak). Tutup pipeline di ujung (`tr`), verifikasi panjang secret, dan pastikan isinya benar-benar tertulis. Guard yang flaky ≈ tidak ada guard.
- Gitleaks memindai **history**: string yang pernah masuk commit akan tetap terflag meski sudah dihapus dari working tree. Kalau string sintetis sempat masuk history branch yang belum ter-merge, rewrite history-nya (`--force-with-lease`).
- Kalau butuh string high-entropy di file CI, **generate saat runtime** — jangan tulis literal (self-referential).

**Yang sudah benar (jangan diubah):**
- Outbox `FOR UPDATE SKIP LOCKED` + reaper `processing` → event tidak hilang
- Normalisasi leaderboard/stats dihitung dari seluruh population
- Anti-gaming di level SQL (like akun <7 hari tidak dihitung)
- RBAC 2-lapis: `role_primary` (identitas) vs `roles`+`user_roles` (otorisasi)
- Scope fail-closed: `"__none__"` → query terfilter, bukan 403
- BFF benar-benar BFF: nol logika bisnis, streaming body, `getSetCookie()` multi-cookie aman
- `clientIP()` lebih aman dari chi `RealIP` (entri XFF terakhir + header BFF)
- `GetUserByID` sudah filter `deleted_at IS NULL` (komentar `userActive` akurat)
- Kontrak anti-drift berlapis (OpenAPI AST-generated + TS types + CI check)
- `clientIP` dipakai juga untuk audit → jejak IP konsisten
