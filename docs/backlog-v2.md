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
| Sprint 2 | P0-07, P0-08, P0-09, P4-01…P4-07 | ⬜ Belum | Skema `user_roles` + media retire + sinkronisasi doc |
| Sprint 3 | P3-01…P3-10 | ⬜ Belum | Playwright + axe + schemathesis + test domain |
| Sprint 4 | P2-01…P2-15 | ⬜ Belum | Fitur PRD yang belum ada |
| Sprint 5 | P6-01…P6-09 | ⬜ Belum | Go-live readiness (Fase 4) |
| — | P1-02…P1-06, P5-01…P5-08 | ⬜ Belum | Keamanan & hygiene |

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

- [ ] **P0-07** `user_roles` PK `(user_id, role_id)` tak bisa multi-scope
  - Evidence: `apps/api/db/migrations/00004_iam.sql`
  - Dampak: 1 user = 1 baris/role → `admin_jurusan` di 2 jurusan tidak representable
  - Fix: PK → `(user_id, role_id, scope_major_id)` + migrasi baru + review `AssignUserRole`
  - Status: ⬜

- [ ] **P0-08** `AssignUserRole` `ON CONFLICT DO UPDATE` menimpa scope + expiry
  - Evidence: `apps/api/db/queries/fase1.sql`
  - Fix: tentukan semantik (append vs upsert eksplisit)
  - Status: ⬜

- [ ] **P0-09** Media retire tidak ada — objek lama tidak dihapus saat avatar/banner/thumbnail diganti
  - Evidence: `internal/http/media_upload.go` `handleConfirmMedia`; `db/queries/fase1.sql` `UpdateUserAvatar` (cuma update FK)
  - Dampak: storage membengkak; kuota 200MB ikut terinflasi. PRD §7.2 step 7 + Runbook "Media retire"
  - Fix: hapus objek + varian media lama (pola sudah ada di `handleDeleteMedia`)
  - Status: ⬜

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

- [ ] **P1-02** Guest reply feedback tanpa rate limit
  - Evidence: `internal/http/feedback_handler.go` — hanya `feedback:day:` (3/hari buat thread), tidak ada limiter reply
  - Spec: api-v2 §2.4 — "balasan guest 10/jam per thread+IP"
  - Status: ⬜

- [ ] **P1-03** Password policy lebih lemah dari PRD (tanpa zxcvbn)
  - Evidence: `internal/http/users_handler.go:29-48` — min 10 + upper/lower/digit
  - Spec: PRD §6.2 — "min 10 char + zxcvbn score ≥ 3"
  - Status: ⬜

- [ ] **P1-04** Bucket versioning OFF
  - Evidence: `internal/storage/s3.go` — tak ada `PutBucketVersioning`
  - Spec: PRD §7.1/§15.2 — "versioning on"
  - Status: ⬜

- [ ] **P1-05** Dragonfly tanpa snapshot periodik
  - Evidence: `docker-compose.yml:42-46` — `--dbfilename=dump` tanpa `--snapshot_cron`
  - Dampak: crash → counter rate-limit reset, blacklist JTI hilang, HLL view count turun
  - Status: ⬜

- [ ] **P1-06** Permission `system.backup` mati (0 route)
  - Evidence: 53 permission di `00004_iam.sql`; `system.backup` tidak dipakai route mana pun
  - Fix: implementasikan `POST /admin/backup` atau hapus dari katalog
  - Status: ⬜

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
- [ ] **P2-05** Object key pakai uuid4 + format generik, bukan uuid7 + path per-entity — PRD §7.1
  - Evidence: `internal/http/media_upload.go:233`
  - Status: ⬜
- [ ] **P2-06** `DeleteObjects` / `ListObjectsV2` tidak ada di storage wrapper — PRD §7.5
  - Status: ⬜
- [ ] **P2-07** Guard "hanya operasi S3 yang diizinkan" — PRD §7.5 (konvensi saja, tanpa enforcement)
  - Status: ⬜
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
  - Status: ⬜

---

## P3 — Tooling & Quality Gate (10)

- [ ] **P3-01** `golangci-lint` tidak ada (cuma gofmt + go vet) — PRD §2.2, §13.3
- [ ] **P3-02** `prettier` tidak ada — PRD §2.2
- [ ] **P3-03** Integration test `pgxmock`/`testcontainers` tidak ada — PRD §13.3
- [ ] **P3-04** E2E Playwright tidak dikonfigurasi (hanya transitif di lockfile) — PRD §14.3
- [ ] **P3-05** A11y test (axe, WCAG AA) tidak ada — PRD §14.3
- [ ] **P3-06** Contract test (schemathesis) tidak masuk CI — PRD §14.3
- [ ] **P3-07** Load test (k6) tidak masuk CI — PRD §14.3, §18
- [ ] **P3-08** Coverage test FE tipis (1 file: `block-form.test.ts`) — PRD §14.3
  - Prioritas: `lib/api.ts` (single-flight refresh), `block-logic.ts`, `deepLink`
- [ ] **P3-09** Test Go domain kritis belum ada (auth rotation, RBAC scope, promotion, outbox, upload)
  - Ada: `formula_test.go`, `contract_blocks_test.go`, `config_test.go`, `router_test.go`, `gen-openapi`, `migrate-old`
- [ ] **P3-10** `next-sitemap` tidak dipakai (sitemap manual — fungsional, boleh tetap)
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

- [ ] **P4-01** Jumlah endpoint: api-v2 "100 path/126 operasi" · PRD "114 path/142 operasi" → **117 path / 145 operasi**
- [ ] **P4-02** Rate limit global: PRD "100/mnt/IP + 5000/mnt global" → **300/menit** (api-v2 + kode benar)
- [ ] **P4-03** Lockout login: PRD "30 mnt" → **15 menit** (api-v2 + kode benar)
- [ ] **P4-04** Tipe notifikasi: PRD "8 tipe" → **11 enum** di kontrak
- [ ] **P4-05** Middleware chain: PRD sebut `RealIP` + `CORS` → kode pakai `clientIP()` custom (**lebih aman**) & tanpa CORS
- [ ] **P4-06** PRD sebut "LoadPermissions (cache)" sebagai middleware → kode memanggilnya di dalam `authRequired`
- [ ] **P4-07** `zod` + `react-hook-form` terpasang tapi 0 pemakaian (PRD §12.3 minta "validasi inline Zod")

---

## P5 — Hygiene / Dead Code (8)

- [ ] **P5-01** Direktori sampah `apps/api/{cmd/api,cmd/tools/{migrate,seed,verify-seed},internal/...}` (0 file, hasil brace expansion gagal)
- [ ] **P5-02** `/register`, `/forgot-password`, `/reset-password` ada di `PUBLIC_PATHS` tapi halamannya tak ada — `apps/web/middleware.ts:6`
- [ ] **P5-03** `--font-geist-mono` di `@theme inline` tapi tak terdefinisi di `:root`/`.dark` — `apps/web/app/globals.css`
- [ ] **P5-04** `next.config.ts` `remotePatterns` hanya `http://localhost`
- [ ] **P5-05** `RESERVED` di middleware memuat path yang tak pernah jadi username (`changelogs`, `register`, …)
- [ ] **P5-06** `apps/api/tmp/` ada di tree (sudah di-ignore ✅)
- [ ] **P5-07** Volume `/app/tmp` di-mount di compose (aman, catat saja)
- [ ] **P5-08** `json.Marshal` manual di 10 tempat handler — PRD §13.1 melarang ("via dto")

---

## P6 — Go-Live Readiness / Fase 4 (9)

- [ ] **P6-01** 🔴 Rollback deploy tidak rollback skema
  - Evidence: `.github/workflows/deploy.yml:130-137` — hanya `sed IMAGE_TAG` + `up -d`, tanpa `goose down`
- [ ] **P6-02** 🔴 Jendela backward-compat: `migrate up` jalan **sebelum** `up -d` → container lama jalan dengan skema baru
- [ ] **P6-03** Migrasi v1→v2 belum pernah di dry-run terhadap dump prod — PRD §16
- [ ] **P6-04** Storage mirror 5,1 GB belum dilakukan — PRD §16.3 (`tools/backup.sh` sudah ada `rclone sync`)
- [ ] **P6-05** k6 p95 belum diverifikasi di VPS 4GB — PRD §14.1, §18 gate
- [ ] **P6-06** Restore drill belum pernah — PRD §17, §18 gate
- [ ] **P6-07** 🔴 Disk prod 81% (gate: <70%) — PRD §0.1, §19, §18
- [ ] **P6-08** Maintenance mode (`NEXT_PUBLIC_MAINTENANCE_MODE`) belum ada — PRD §16.1
- [ ] **P6-09** Sunset plan "hapus v1 setelah 30 hari" belum ada — PRD §16.1, §18

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
