# Grafikarsa v2 — PRD (Product Requirements Document)

> Status: **Draft untuk build ulang dari nol — siap implementasi**
> Versi: 2.0.0 | Tanggal: 2026-10-09 | Penulis: PM + Solution Architect + Fullstack + Performance + System Designer + QA
> Bahasa: Indonesia. Semua istilah teknis dalam Bahasa Inggris dipertahankan apa adanya.
> Cakupan: monorepo baru, bukan patch v1. Data + file lama **wajib bisa migrasi**.

---

## 0. Ringkasan Eksekutif

### 0.1 Masalah yang diselesaikan v2

Hasil bedah codebase v1 + inspeksi server prod (`103.134.78.58`, Ubuntu 22.04, 4 GB RAM, disk 50 GB terpakai 81%):

| Temuan v1 (fakta prod) | Dampak | Keputusan v2 |
|---|---|---|
| DB 35 MB, MinIO 5,1 GB, 1.687 user (1.686 student + 1 admin), portfolio 1.705 (861 published, 502 draft, 252 pending_review, 74 rejected, 16 archived), blocks 6.903 (text 3.603, image 2.251, youtube 931, sisanya pdf/button/figma/doc/table/canva) | Beban moderasi besar (252 pending), banyak draft mangkrak | Workflow + moderasi + autosave + batas draft + notifikasi yang proper |
| `pendingUploads` in-memory (hilang saat restart), bucket MinIO public-read, URL absolut `https://grafikarsa.com/storage/...` tersebar di `users.avatar_url`, `portfolios.thumbnail_url`, `content_blocks.payload` | Restart = upload rusak, URL rapuh saat ganti domain/storage, file privat tidak ada | Tabel `media_assets` + bucket RustFS **private** + semua akses via backend/BFF, URL DB hanya simpan `object_key`, migrasi URL lama via mapping |
| Auth: access di `localStorage`, refresh rotation ada tapi tanpa `__Host-` prefix, tanpa CSRF double-submit, tanpa OAuth/OIDC, captcha matematika in-house | XSS bisa curi token, CSRF di cookie refresh, tidak ada SSO sekolah | JWT best-practice industri: access 15 mnt di memori, refresh opaque rotation + reuse-detection + HttpOnly `__Host-` Secure SameSite=Lax, CSRF token, Argon2id, Turnstile, session management |
| Role: `student/alumni/admin/teacher` + `special_roles` text[] tanpa scope, tanpa row-level enforcement, `GetBySlug` scan in-memory 100 row | Privilege escalation rawan, query tidak scalable | IAM RBAC + scope (`global` / `jurusan`) + permission catalog + enforcement di middleware + SQL |
| Feed dihitung on-the-fly per request + sort in-memory, `maxStats` di Redis, tidak ada materialisasi | Lambat saat portfolio >10rb, tidak konsisten untuk landing | `portfolio_stats` materialized + worker incremental + Dragonfly cache + leaderboard precompute harian |
| Kenaikan kelas: hanya flag `is_active` + `promotion_month/day`, tanpa job, tanpa dry-run, tanpa histori imutabel | Kenaikan manual, rawan salah rombel/angkatan | Tabel `enrollments` imutabel + `promotion_jobs` dry-run → approve → execute transaksional + audit |
| Search hanya `ILIKE %q%`, index trigram cuma di `users.nama` | Typo tidak ketemu, katalog lambat | `pg_trgm` + `tsvector` Indonesia + ranking + suggest, index GIN di judul/nama/tags |
| Infra: Nginx host (443/80) → Caddy (8081/8443) → web/backend/minio; backend + storage masih bisa dijangkau via path publik `/api/*`, `/storage/*` | Dua lapis proxy, permukaan serang lebih luas, CORS rumit | **Hanya FE yang diekspos**: Internet → Caddy 80/443 → `web:3000` saja. Backend, RustFS, Dragonfly, Postgres **tanpa port host**, komunikasi via network internal + BFF Route Handlers |
| CI/CD pakai password SSH root di prompt, image tag `latest` + sha panjang, tanpa migrasi otomatis, tanpa backup pre-deploy | Deploy rapuh, rollback sulit | GH Actions OIDC/key SSH ed25519 + `latest` + `sha-xxxx` + `goose up` + backup otomatis + healthcheck + rollback |
| Frontend: Next 16 + Tailwind v4 + shadcn + lucide + Geist | Masih oke, tapi ikon + font mau diseragamkan | Next latest + shadcn latest + **Hugeicons free only** + **Inter only**, semua komponen dari shadcn |

### 0.2 Tujuan v2 (SMART)

1. **Proper & prod-ready**: satu repo, satu `.env` root, satu `.gitignore` root, `make dev` < 5 menit di laptop baru, `make deploy` deterministik.
2. **Secure**: lulus OWASP ASVS L2 untuk auth/session/access-control/input-validation/logging (dibuktikan via checklist QA §14).
3. **Performant**: p95 feed < 300 ms, profil < 150 ms, upload 10 MB < 8 s di VPS 4 GB; Lighthouse landing ≥ 90.
4. **Scalable**: 10× data prod sekarang (17rb user, 17rb portfolio, 70rb blocks, 50 GB storage) tanpa refactor skema.
5. **Migratable**: 100% user + portfolio + blocks + likes/comments/follows + file 5,1 GB pindah tanpa URL mati.
6. **Maintainable**: backend tanpa ORM (sqlc + pgx), migrasi versioned (goose), FE 100% shadcn, kontrak API OpenAPI.

### 0.3 Non-goals v2.0

- Bukan multi-sekolah (tetap SMKN 4 Malang, tapi skema siap `schools` di masa depan via `scope`).
- Bukan mobile native (mobile web responsive + PWA ringan saja).
- Bukan real-time chat (komentar + notifikasi async saja, tanpa WebSocket).

---

## 1. Personas & Roles (IAM)

### 1.1 Empat role utama + custom

| Role sistem (`is_system=true`, tidak bisa dihapus) | Scope default | Deskripsi |
|---|---|---|
| `siswa` | `global` (diri sendiri) | Buat portfolio, follow/like/komen, feed, feedback, baca changelog. |
| `guru` | `jurusan` (diampu) atau `global` jika pembina | Semua milik siswa + nilai (assessment), beri komentar terbina, lihat analitik kelas/jurusan yang diampu, kelola series template pembelajaran. |
| `admin_jurusan` | `jurusan_id` wajib | Kelola user/guru/siswa di jurusannya, moderasi portfolio jurusannya, kelola kelas tahun berjalan, lihat dashboard jurusan. Tidak bisa kelola jurusan lain / settings global. |
| `admin` | `global` | Akses penuh: user/roles, semua moderasi, akademik, storage, changelog/feedback, import, promotion, settings. |

Custom role (IAM):

- Admin bisa buat role misal `moderator_konten_rpl` (scope jurusan RPL, permissions `portfolios.moderate + comments.moderate`), `kurikulum` (read analitik + manage series), `tata_usaha` (users.create tanpa delete).
- `admin_jurusan` bisa buat custom role **hanya dalam jurusannya** dan **hanya subset permission miliknya** (anti privilege-escalation).
- Setiap assignment tercatat `assigned_by`, `expires_at` opsional (misal OSIS 1 tahun), dan masuk audit log.

Status `alumni` di v1 **bukan role** di v2 — melainkan `status` di profil (`siswa` → `alumni` otomatis saat lulus via promotion job). Ini memperbaiki kebingungan v1 di mana `role` dicampur dengan status kelulusan.

### 1.2 Permission catalog (final, dikunci di kode + seed)

Format `domain.action`. 53 keys:

```
# Overview & analitik
dashboard.view_global, dashboard.view_jurusan
# Portfolio & moderasi
portfolios.create, portfolios.edit_own, portfolios.submit, portfolios.delete_own,
portfolios.view_all, portfolios.moderate, portfolios.moderate_jurusan, portfolios.archive_any
# Blocks & media
blocks.manage_own, media.upload, media.delete_own, media.delete_any
# Social
social.follow, social.comment, social.like
# Users
users.view, users.create, users.edit, users.reset_password, users.activate, users.delete
users.view_jurusan, users.create_jurusan, users.edit_jurusan
# Roles
roles.view, roles.create, roles.assign, roles.assign_jurusan
# Akademik
academic.view, academic.manage_classes, academic.manage_majors, academic.manage_years,
academic.promote_dryrun, academic.promote_execute, academic.import_students
# Assessment
assessments.view, assessments.score, assessments.manage_metrics
# Series & tags
series.view, series.manage, tags.manage
# Feedback & changelog
feedback.create, feedback.view_all, feedback.triage, feedback.reply,
changelog.view, changelog.manage, changelog.publish
# System
system.settings, system.backup, system.jobs
```

Mapping default:

| Permission | siswa | guru | admin_jurusan | admin |
|---|---|---|---|---|
| portfolios.create/edit_own/submit/delete_own, blocks.manage_own, media.upload/delete_own, social.*, feedback.create, changelog.view | ✅ | ✅ | ✅ | ✅ |
| assessments.score/view, series.view | ❌ | ✅ | ✅ | ✅ |
| portfolios.moderate_jurusan, users.view/create/edit_jurusan, dashboard.view_jurusan, academic.view/manage_classes (jurusan sendiri), feedback.triage/reply (jurusan) | ❌ | ❌ | ✅ | ✅ |
| Semua `*_any`, `roles.*`, `academic.promote_execute`, `system.*`, `changelog.manage/publish`, `tags/series.manage` | ❌ | ❌ | ❌ (kecuali assign_jurusan subset) | ✅ |

Enforcement 3 lapis (wajib, bukan opsional):

1. **Middleware HTTP** (`RequirePermission("portfolios.moderate")`): cek JWT → load `user_roles + permissions` dari Dragonfly (5 mnt) fallback DB → 403 dengan `code: FORBIDDEN_SCOPE` jika scope tidak cocok.
2. **SQL row-level**: setiap query yang scope-jurusan wajib `WHERE jurusan_id = $scope` (dilarang filter di Go saja). Untuk admin global, scope = NULL (bypass).
3. **UI gating**: sidebar/menu disembunyikan, tapi **bukan** pengganti enforcement server (QA wajib test direct-API bypass).

---

## 2. Arsitektur Sistem

### 2.1 Topologi deploy (VPS Ubuntu tunggal, single-domain, hanya FE diekspos)

```
Internet (80/443, IPv4/IPv6)
  │
  ▼
Caddy (proxy, :80/:443 host, auto HTTPS LE, HSTS, CSP, gzip/zstd, max_body 55MB)
  │  Caddyfile HANYA:
  │    reverse_proxy web:3000
  ▼
Next.js web:3000 (SSR + BFF Route Handlers /api/*)
  │  INTERNAL_API_URL=http://api:8080/v1 (docker DNS, tanpa keluar host)
  ▼
Go API api:8080 (Chi, tanpa port host)
  ├── Postgres db:5432 (tanpa port host, volume postgres_data)
  ├── Dragonfly cache:6379 (tanpa port host, volume dragonfly_data, RESP compatible)
  └── Storage S3 `storage:9000` (tanpa port host, volume storage_data, S3 API, bucket private; default RustFS, bisa diganti vendor via `S3_*`)
```

Tidak ada `nginx` host. Tidak ada `/api/* → backend` di Caddy. Tidak ada path publik ke storage. Browser **tidak pernah** kontak backend/storage langsung — semua via `https://DOMAIN` (Next) yang meneruskan server-side.

Kenapa BFF, bukan Caddy path-routing seperti v1:

- Backend benar-benar internal (tidak ada CORS, tidak ada origin terpisah, tidak ada presigned host publik yang rapuh).
- Cookie `HttpOnly Secure` + CSRF bisa di-set/di-validasi di satu origin.
- Next bisa caching (`fetch` + `revalidate`), image optimization, dan rate-limit edge sebelum ke backend.
- Audit + request-id satu jalur (`x-request-id` dari Caddy → Next → API → worker).

Trade-off yang diterima: upload melewati 2 hop (browser → Next → API → storage S3). Di VPS 4 GB ini oke untuk 5–50 GB; jika kelak >200 GB, aktifkan mode **presigned via BFF-issued redirect** (backend terbitkan presigned 60 dtk dengan host `https://DOMAIN/media/*` yang di-proxy streaming oleh Next, bukan direct storage). Pola ini sudah disiapkan (§7.3) tanpa mengubah kontrak FE.

### 2.2 Monorepo layout (scalable, tidy)

```
grafikarsa-v2/
  .env                    # SATU-SATUNYA env (compose + api + web + tools baca ini)
  .env.example            # contoh lengkap + komentar
  .gitignore              # root only (node_modules, .next, bin, volumes, .env)
  .dockerignore
  Makefile                # dx: dev/prod/migrate/seed/backup/test/deploy
  Caddyfile               # prod (hanya web) + Caddyfile.dev (opsional)
  docker-compose.yml      # dev: db+dragonfly+storage+api (web native)
  docker-compose.prod.yml # prod lokal: +web built
  docker-compose.deploy.yml # server: proxy+web+api+db+dragonfly+storage (tanpa port host kecuali proxy)
  apps/
    web/                  # Next.js (App Router, TS strict)
      app/
        (auth)/login/
        (main)/ page(portfolio feed) portfolios/ users/ settings/ notifications/ changelog/ [username]/ [username]/[slug]/
        admin/(dashboard)/ page moderation/ portfolios/ users/ roles/ majors/ classes/ years/ assessments/ series/ tags/ feedback/ changelogs/ import/ promotion/ settings/
        api/[...path]/route.ts   # BFF proxy → INTERNAL_API_URL (GET/POST/PUT/PATCH/DELETE)
        api/uploads/route.ts     # BFF upload streaming (multipart, progress)
        api/media/[...key]/route.ts # BFF media streaming (Range, Cache-Control, auth check via backend)
      components/ui/      # 100% shadcn (button, card, dialog, table, tabs, dsb — tanpa custom aneh)
      components/{layout,portfolio,feed,admin,user,search,notifications,changelog,feedback}/
      lib/{api-client,query,stores,hooks,types,utils}/
      public/  next.config.ts  middleware.ts (auth gate via cookie, bukan localStorage)
    api/                  # Go (Chi + pgx + sqlc + goose)
      cmd/api/main.go
      cmd/tools/migrate-old/  # CLI migrasi v1→v2 (db + storage)
      internal/
        config/  db/ (pgxpool + goose embed)  cache/ (dragonfly client)
        auth/ (jwt, argon2, session, csrf, turnstile)
        iam/ (roles, permissions, scope middleware)
        http/ (router chi, handlers, dto, validators, problem+json errors)
        service/ (portfolio, media, feed, leaderboard, promotion, notification, feedback, changelog, search)
        repo/ (sqlc generated + queries/*.sql)
        storage/ (s3 client aws-sdk-go-v2 → storage S3-agnostik, image pipeline)
        jobs/ (scheduler + workers: leaderboard, promotion, notifications fanout, thumbnails, cleanup)
        observability/ (slog, otel, metrics)
      db/migrations/*.sql  # goose versioned (00001_init …)
      db/queries/*.sql     # sqlc source of truth
      sqlc.yaml
      Dockerfile (multi-stage dev/prod)
  packages/
    contracts/openapi.yaml # kontrak FE↔BE (generate types TS + Go validation)
    ui-preset/ (opsional tailwind preset shadcn)
  infra/
    caddy/Caddyfile*  postgres/init/  storage/init/  dragonfly/dragonfly.conf
  tools/
    backup.sh  restore.sh  server-init.sh
  docs/prd-v2.md (file ini)  docs/api-v2.md  docs/runbook.md
  .github/workflows/ci.yml  .github/workflows/deploy.yml
```

Aturan kerapian (lint-enforced):

- FE: `pnpm`, `eslint`, `tsc --noEmit`, `prettier`, import alias `@/*`, tidak ada `any` tanpa `// TODO` + issue.
- BE: `gofmt`, `go vet`, `golangci-lint`, error envelope tunggal `problem+json`, tidak ada query SQL di handler (wajib via `repo/` sqlc), tidak ada `time.Now()` langsung di handler (via `clock` injectable untuk test).
- Migrasi: hanya via `db/migrations`, dilarang `AutoMigrate`/SQL di kode. Seed via `db/seeds/*.sql` + `make seed`.

### 2.3 Request lifecycle (contoh: buka portfolio)

```
1. Browser GET https://DOMAIN/rafli/proyek-iot → Caddy → Next SSR
2. Next server fetch INTERNAL: GET http://api:8080/v1/portfolios/by-slug?username=rafli&slug=proyek-iot
   Header: Cookie (__Host-rt), x-request-id, x-forwarded-for (dari Caddy)
3. API: auth optional → iam scope check → repo sqlc (1 query + preload tags/blocks/stats) → Dragonfly incr view (HyperLogLog + counter)
   → return JSON + ETag
4. Next render + cache (revalidate 60s, stale-while-revalidate) → HTML + JSON-LD SEO
5. Browser load images via /api/media/<key> → Next Route Handler → API GET /v1/media/<id>/stream (cek visibility)
   → API stream dari storage S3 (Range support) dengan Cache-Control: public, max-age=31536000, immutable (karena key content-hash)
```

Semua API di bawah `/api/*` di Next adalah proxy tipis: meneruskan method/body/query + `Authorization` dari cookie→header, mengembalikan status/body apa adanya, menambah `x-request-id`. Tidak ada logika bisnis di BFF kecuali: auth cookie↔header mapping, upload multipart streaming, media Range/Cache, dan rate-limit edge ringan.

---

## 3. Tech Stack Final (dikunci)

| Layer | Pilihan | Versi / catatan | Alasan |
|---|---|---|---|
| FE framework | Next.js App Router | latest 15.x (pin di `package.json`, upgrade ke 16 setelah GA stabil + e2e hijau) | SSR/SEO + BFF Route Handlers + Image Optimization tanpa server tambahan |
| UI | shadcn/ui latest (`new-york`, neutral, CSS vars) | Semua komponen dari `components/ui` (button, card, dialog, drawer, table, tabs, form, sonner, skeleton, dsb). Dilarang bikin `CustomButton` di luar shadcn. | Konsistensi + a11y + theming |
| Icons | Hugeicons Free only | `hugeicons-react` (free pack). Dilarang lucide/emojis sebagai ikon. | Sesuai permintaan + lisensi jelas |
| Font | Inter only | `next/font/google` Inter, latin subset, `display:swap`, preload | Satu font, cepat,ahu |
| Styling | Tailwind CSS v4 + `tailwind-merge` + `cva` | Token via CSS vars OKLCH (light/dark) | Konsisten dengan v1 tapi dibersihkan |
| Data FE | TanStack Query v5 + Zustand v5 (persist minimal) + `nuqs` untuk filter URL | `staleTime 60s`, `gcTime 5m`, infinite + `IntersectionObserver` | Server-state di RQ, client-state di Zustand/URL |
| Forms | React Hook Form + Zod v4 + `@hookform/resolvers` | Skema Zod digenerate dari `openapi.yaml` bila memungkinkan | Validasi konsisten FE↔BE |
| BE framework | Go + Chi v5 | Go 1.24+, `chi/middleware` (RequestID, RealIP, Recoverer, Timeout 30s, Throttle) | Ringan, stdlib-friendly, router eksplisit (pengganti Fiber) |
| DB driver + query | `pgx/v5` + `sqlc` | Pool `min 4 max 25`, `statement_cache`, `query_exec_mode=cache_statement` | Tanpa ORM, type-safe, cepat, mudah tuning |
| Migrasi | `goose` (SQL) | `db/migrations/*.sql` + `goose embed` | Versioned, reviewable, rollback jelas |
| Auth | JWT access (HS256, 15 mnt, `kid` rotation) + refresh opaque (32B, sha256 di DB, family + reuse-detection) | `golang-jwt/v5`, `argon2id` (atau bcrypt cost 12 untuk kompat hash lama), `go-redis` client ke Dragonfly | Best-practice industri (§6) |
| Cache/queue | DragonflyDB | Image `docker.dragonflydb.com/dragonflydb/dragonfly`, `--maxmemory 512mb --cache-mode --eviction allkeys-lru`, AOF on, tanpa port host | Drop-in RESP, 1 container gantikan Redis, throughput lebih tinggi di 4 GB |
| Storage | S3-compatible, default RustFS self-host | Image `rustfs/rustfs:latest`, SNSD `/data`, non-root `10001:10001`, bucket **private**, versioning on, tanpa port host; env + SDK agnostik (`S3_*`, `aws-sdk-go-v2` — lihat §7.5) | Ganti ke R2/S3/MinIO tanpa ubah kode |
| S3 client (BE) | `aws-sdk-go-v2` (S3, path-style configurable) | `BaseEndpoint=S3_ENDPOINT`, region `S3_REGION`; operasi hanya subset §7.5 | Satu-satunya dependensi storage; vendor-agnostik |
| Gateway | Caddy 2-alpine | Auto HTTPS, `encode zstd gzip`, `request_body max_size 55MB`, security headers | Tanpa nginx, config 20 baris |
| Observability | `slog` JSON + OpenTelemetry + `/health` + `/metrics` | Loki/Prom opsional di fase 2, fase 1 cukup log file + `docker logs` + uptime check | Cukup untuk VPS tunggal |
| CI/CD | GitHub Actions + DockerHub + SSH ed25519 | `buildx`, `trivy` scan, `goose up`, backup pre-deploy | Proper (§11) |
| Env | Single `.env` root | Compose + API + Web + tools baca file yang sama; `INTERNAL_*` untuk antar-container | DX seperti v1, tapi dibersihkan |

Bukti riset stack (diringkas dari dokumentasi resmi, Okt 2026):

- RustFS S3-compatible, Apache-2.0, Docker SNSD `rustfs/rustfs:latest` dengan user `10001:10001`, port 9000/9001, support IAM/policies/OIDC/konsol — dipakai sebagai **implementasi default**, dipilih via `S3_ENDPOINT` (bukan dependensi kode). Ganti vendor = ganti env. [GitHub rustfs/rustfs](https://github.com/rustfs/rustfs) · [Docs RustFS](https://docs.rustfs.com/en) · [Docker install](https://rustfs.org/installation/docker/) · [Announcing 1.0 GA](https://rustfs.com/blog/announcing-rustfs-1-0-0-ga/) · [Docker Hub](https://hub.docker.com/r/rustfs/rustfs)
- Chi + pgx + sqlc + goose adalah template produksi standar (clean architecture, migrasi + codegen). [go-chi-postgres-starter](https://github.com/justyn-clark/go-chi-postgres-starter) · [go-chi-pg-sqlc-goose-template](https://github.com/high-horse/go-chi-pg-sqlc-goose-template) · [Go HTTP Service spec](https://specdriven.sh/specs/go-service) · [sqlc+goose guide](https://pressly.github.io/goose/blog/2024/goose-sqlc/) · [pgx+sqlc Ubuntu](https://www.facsiaginsa.com/golang/connect-go-to-postgresql-with-pgx-and-sqlc-on-ubuntu)
- BFF via Next Route Handlers adalah pola resmi Next untuk backend privat. [BFF guide](https://nextjs.org/docs/app/guides/backend-for-frontend) · [Route Handlers](https://nextjs.org/docs/app/getting-started/route-handlers) · [BFF proxy pattern](https://medium.com/digigeek/bff-backend-for-frontend-pattern-with-next-js-api-routes-secure-and-scalable-architecture-d6e088a39855)
- JWT best-practice 2026: access pendek di memori, refresh di HttpOnly Secure SameSite cookie + rotation + reuse-detection. [Refresh rotation](https://codecondo.com/jwt-refresh-token-rotation/) · [Access+Refresh architecture](https://dev.to/devanshu_patil/jwt-authentication-architecture-access-tokens-refresh-tokens-and-secure-httponly-cookies-270m) · [Best practices](https://skycloak.io/blog/jwt-best-practices-developers/) · [HttpOnly rotation](https://blogdeveloperspot.blogspot.com/2026/03/secure-jwt-authentication-refresh-token.html)

---

## 4. Data Model v2 (Postgres, sqlc, goose)

### 4.1 Prinsip

- PK `uuid` (`gen_random_uuid()`), `created_at/updated_at` `timestamptz`, soft-delete hanya di `users/portfolios/comments` (lainnya hard-delete + audit log).
- `username`/`email` `citext` unique, `slug` unique per user (bukan global).
- Enum via `CHECK` atau Postgres `TYPE` (dipilih `TYPE` untuk `portfolio_status`, `role_scope`, `media_status`, dsb agar type-safe di sqlc).
- Semua tabel hot punya `updated_at` trigger + `COMMENT`.
- Tidak ada `payload JSONB` liar: setiap block type punya JSON Schema di `packages/contracts/blocks/*.json` + validasi di BE (go-jsonschema) + FE (zod). JSONB tetap dipakai tapi **tervalidasi**.
- Uang/waktu: tidak ada uang; waktu selalu `timestamptz`, durasi `interval`.

### 4.2 ERD tekstual (inti)

```
majors (id, code unique lower, name)
academic_years (id, start_year unique, is_active bool unique-where-true, promotion_month/day, starts_at/ends_at)
classes (id, academic_year_id→years, major_id→majors, grade 10/11/12, group A-Z, name auto XII-RPL-A, homeroom_teacher_id→users, UNIQUE(year,major,grade,group))
users (id, username citext unique, email citext unique, password_hash, name, role_primary enum siswa/guru/admin_jurusan/admin, status enum aktif/lulus/nonaktif, avatar_media_id→media, banner_media_id, bio, class_id nullable (kelas aktif), major_id denormal untuk query cepat, graduated_year, is_active, last_login_at)
enrollments (id, user_id, class_id, academic_year_id, status enrolled/promoted/graduated/repeated/transferred, is_current bool, UNIQUE(user_id, academic_year_id) where is_current)
roles (id, slug unique, name, scope_type global/jurusan, is_system, created_by)
permissions (id, key unique, group, description)
role_permissions (role_id, permission_id PK komposit)
user_roles (user_id, role_id, scope_major_id nullable, assigned_by, expires_at, PK user+role+scope)
sessions (id, user_id, refresh_hash unique, family_id, device_label, ip, user_agent, expires_at, revoked_at, rotated_to nullable, created_at)
  + auth_audit (id, user_id, event login_success/login_fail/refresh/reuse_detected/logout, ip, ua, created_at) BRIN
portfolios (id, owner_id→users, title, slug, status draft/pending_review/changes_requested/published/rejected/archived,
  thumbnail_media_id→media, series_id nullable, published_at, submitted_at, reviewed_by/at, review_note, view_count_cached, like_count_cached, comment_count_cached, quality_score_cached, trending_score_cached, search_tsv tsvector)
portfolio_tags (portfolio_id, tag_id PK)
tags (id, slug unique, name unique)
portfolio_blocks (id, portfolio_id, type text/image/table/youtube/button/embed/figma/canva/ppt/pdf/doc/link, position int, payload JSONB validated, asset_media_id nullable→media, UNIQUE(portfolio_id, position) DEFERRABLE)
series (id, slug unique, name, description, is_active) + series_blocks (id, series_id, type, position, instruction)
media_assets (id, owner_id, purpose avatar/banner/thumbnail/block_image/document/feedback/changelog, object_key unique (ex: thumbnails/<pid>/<uuid>.webp), mime, size_bytes, width/height nullable, checksum_sha256, status pending/confirmed/quarantined/deleted, created_at, confirmed_at)
portfolio_likes (user_id, portfolio_id PK, created_at)
portfolio_views (id, portfolio_id, viewer_id nullable, session_id, viewed_date date, UNIQUE(portfolio_id, viewer_id, viewed_date) + UNIQUE(portfolio_id, session_id, viewed_date)) — agregat harian, bukan per-view row abadi
comments (id, portfolio_id, author_id, parent_id self-ref nullable, body_md (max 2000), body_html sanitized, is_edited, deleted_at, created_at)
follows (follower_id, following_id PK, created_at, CHECK follower!=following)
notifications (id, user_id, type 8 tipe, title, body, data JSONB {actor_id, portfolio_id, comment_id, ...}, is_read, read_at, created_at) + index unread
user_interests (user_id PK, tag_weights JSONB {tag_id: count}, major_weights JSONB, total_likes, updated_at)
feed_preferences (user_id PK, algorithm smart/recent/following)
portfolio_stats (portfolio_id PK, likes, views_7d, views_30d, comments, assessment_avg, quality, engagement, trending, computed_at) — materialized oleh worker
leaderboards (id, period daily/weekly/monthly, scope global/jurusan:<code>, kind best_students/best_projects, computed_for date, UNIQUE(period, scope, kind, computed_for))
leaderboard_entries (leaderboard_id, rank, subject_user_id nullable, subject_portfolio_id nullable, score, breakdown JSONB, delta_rank)
assessments: assessment_metrics (id, name, sort, is_active) + portfolio_assessments (portfolio_id unique, assessor_id, total_score numeric(4,2), comment) + assessment_scores (assessment_id, metric_id, score 1-10, comment)
feedback: feedback_threads (id, reporter_id nullable (guest boleh), category bug/idea/question/praise, title, body_md, status open/triaged/in_progress/resolved/closed/duplicate, priority low/medium/high/critical, attachment_media_id, upvotes_cached, assignee_id, resolved_at, created_at) + feedback_messages (id, thread_id, author_id nullable (admin/user), body_md, is_internal bool, created_at) + feedback_votes (thread_id, user_id PK)
changelog: changelogs (id, version semver unique, title, summary, channel stable/beta, audience all/siswa/guru/admin, status draft/scheduled/published/archived, publish_at, created_by) + changelog_sections (id, changelog_id, kind added/improved/fixed/removed, title, body_md, sort) + changelog_reads (changelog_id, user_id PK, read_at)
promotion_jobs (id, from_year_id, to_year_id, status draft/previewed/approved/executing/done/failed, dryrun_result JSONB, executed_by, executed_at, UNIQUE(from_year_id, to_year_id))
audit_logs (id, actor_id, action, entity, entity_id, diff JSONB, ip, created_at) — untuk semua mutasi admin/IAM/promotion/moderasi
outbox (id, topic, payload JSONB, status pending/sent/failed, attempts, next_retry_at) — transactional outbox untuk worker (notif, thumbnails, stats, leaderboard)
```

Perbedaan kunci vs v1:

- `users.role` tunggal → `role_primary` + `user_roles` multi + scope (mendukung guru wali + admin_jurusan RPL saja).
- `alumni` → `users.status=lulus`, bukan role.
- `content_blocks.payload` bebas → `portfolio_blocks` + JSON Schema per type + `asset_media_id` FK (bisa join + cleanup orphan).
- `pendingUploads` map → `media_assets` persisten + lifecycle `pending(15 mnt)→confirmed/quarantined/deleted`.
- `portfolio_views` per-view abadi → agregat harian + HLL di Dragonfly (hemat 90% row).
- `user_interests` counter mentah → bobot + decay di worker (tidak meledak).
- Tambahan `portfolio_stats`, `leaderboards`, `promotion_jobs`, `audit_logs`, `outbox` yang tidak ada di v1.

### 4.3 Index & search (proper)

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

-- trigram untuk typo-tolerant
CREATE INDEX idx_users_name_trgm ON users USING gin (name gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX idx_portfolios_title_trgm ON portfolios USING gin (title gin_trgm_ops) WHERE status='published';

-- full-text Indonesia (simple, tanpa dictionary khusus + stopwords default)
ALTER TABLE portfolios ADD COLUMN search_tsv tsvector
  GENERATED ALWAYS AS (to_tsvector('simple', coalesce(title,'') || ' ' || coalesce(review_note,''))) STORED;
CREATE INDEX idx_portfolios_tsv ON portfolios USING gin (search_tsv) WHERE status='published';

-- feed & moderasi (partial + covering)
CREATE INDEX idx_portfolios_feed ON portfolios (status, published_at DESC, owner_id)
  INCLUDE (id, title, slug, thumbnail_media_id) WHERE status='published';
CREATE INDEX idx_portfolios_pending ON portfolios (submitted_at) WHERE status='pending_review';
CREATE INDEX idx_portfolio_stats_trending ON portfolio_stats (trending DESC);
CREATE INDEX idx_notifications_unread ON notifications (user_id, created_at DESC) WHERE is_read=false;
CREATE INDEX idx_outbox_pending ON outbox (next_retry_at) WHERE status='pending';
```

Pencarian v2 = 2 tahap: `ts_rank(search_tsv, query) * 0.6 + similarity(title, query) * 0.4`, threshold `similarity > 0.2` untuk typo, limit 20, suggest via `pg_trgm` saat 0 hasil.

### 4.4 Contoh migrasi goose (nama file)

```
db/migrations/00001_extensions.sql
db/migrations/00002_majors_years_classes.sql
db/migrations/00003_users_auth.sql
db/migrations/00004_iam.sql
db/migrations/00005_media.sql
db/migrations/00006_portfolios.sql
db/migrations/00007_social.sql
db/migrations/00008_stats_leaderboard.sql
db/migrations/00009_assessment.sql
db/migrations/00010_feedback_changelog.sql
db/migrations/00011_promotion_outbox_audit.sql
db/migrations/00012_indexes_search.sql
```

---

## 5. API Contract (OpenAPI, versioned `/v1`)

Base via BFF: `https://DOMAIN/api/v1/*` → `http://api:8080/v1/*`. Tidak ada URL backend publik.

Format sukses: `{ success:true, data, meta:{page, per_page, total_pages, total}, request_id }`.
Format gagal (RFC7807 problem+json): `{ success:false, error:{ code, message, details:[{field,message}], request_id } }`.

Grup (ringkas, detail di `packages/contracts/openapi.yaml`):

```
GET /health, GET /ready (db+dragonfly+storage ping)
POST /auth/login {username,password,captcha_token?} → 200 set cookies + {user}
POST /auth/refresh (cookie) → rotation
POST /auth/logout, POST /auth/logout-all, GET /auth/sessions, DELETE /auth/sessions/:id
GET /me, PATCH /me, PATCH /me/password, PUT /me/social-links, GET /me/check-username
GET /users?search&major&class&role&status&page, GET /users/:username, GET /users/:username/followers|following
POST|DELETE /users/:username/follow
GET /portfolios?search&tags&major&class&series&sort=relevance|latest|top&page (hanya published)
POST /portfolios {title, tag_ids, series_id} → draft
GET /portfolios/by-slug?username&slug, GET /portfolios/:id (owner/admin/mod)
PATCH /portfolios/:id, DELETE /portfolios/:id, POST /:id/submit|archive|unarchive|like, DELETE /:id/like
POST /portfolios/:id/blocks, PATCH /blocks/:blockId, DELETE /blocks/:blockId, PUT /portfolios/:id/blocks/reorder
POST /portfolios/:id/comments, GET /portfolios/:id/comments, DELETE /comments/:id
POST /media/presign {purpose, filename, mime, size} → {upload_id, method: POST_BFF} (v2 tidak presign direct, tapi issue upload ticket)
POST /media/upload (multipart, via BFF) → {media_id, object_key}
POST /media/:id/confirm, DELETE /media/:id, GET /media/:id/stream (Range, auth-aware)
GET /feed?algorithm=smart|recent|following&page, GET|PUT /feed/preferences
GET /leaderboards?period=weekly&scope=global|jurusan:rpl&kind=best_students|best_projects
GET|POST /search (unified: users+portfolios+tags)
GET /majors|classes|series|tags|top-* (publik, cache 10 mnt)
GET /notifications, GET /notifications/count, PATCH /:id/read, POST /read-all, DELETE /:id
POST /feedback/threads (+guest), GET /feedback/threads (own/all by perm), POST /threads/:id/messages|votes
GET /changelogs, GET /changelogs/latest, POST /:id/read, admin CRUD + publish/unpublish/schedule
ADMIN (RequirePermission per domain):
  /admin/users CRUD + activate + reset-password, /admin/roles CRUD + assign,
  /admin/portfolios + /pending + :id/approve|:id/request-changes|:id/reject,
  /admin/majors|classes|years CRUD, /admin/promotion/jobs (dryrun/approve/execute),
  /admin/series CRUD + move + export, /admin/tags CRUD,
  /admin/assessments + metrics, /admin/feedback triage, /admin/changelogs,
  /admin/import/students (dryrun/execute), /admin/dashboard/stats, /admin/audit-logs, /admin/jobs
```

Semua list: `page, per_page (max 50), sort, order`, validasi via `go-playground/validator` + pesan Indonesia.

---

## 6. Auth & Session (best-practice industri, secure by default)

### 6.1 Token design

- **Access JWT**: HS256, `iss: grafikarsa`, `aud: grafikarsa-web`, `sub: user_id`, `jti`, `iat/exp 15 mnt`, `kid` (key id untuk rotasi secret tanpa logout massal), claims `role_primary + scope_major_id` (denormal untuk auth cepat, tapi otoritas tetap DB). Disimpan **di memori FE** (variabel JS + RQ header), **bukan localStorage**. Dikirim sebagai `Authorization: Bearer` dari BFF ke API (BFF membaca dari cookie `__Host-at` HttpOnly? — dipilih pola **split**: access di memori, refresh di cookie. BFF proxy membaca refresh cookie untuk `/auth/refresh`, tapi untuk request biasa BFF meneruskan `Authorization` dari memori FE. SSR membaca cookie refresh → tukar ke access server-side sekali, lalu render).
- **Refresh opaque**: 32 byte `crypto/rand` → base64url, simpan `sha256` di `sessions.refresh_hash`, `family_id` UUID, `expires_at` 30 hari absolut + `idle 7 hari` (diperbarui tiap rotation), `device_label` + `ip/ua` untuk daftar sesi. Cookie: `__Host-rt=...; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age=2592000`. Prefix `__Host-` memaksa `Secure + Path=/ + tanpa Domain` (anti subdomain-fixation).
- **Rotation + reuse-detection**: tiap `/refresh` → revoke token lama (`revoked_at`, `rotated_to=new_id`) + terbitkan pasangan baru dalam **satu transaksi**. Jika token yang sudah di-revoke dipakai lagi → **revoke seluruh family** + audit `reuse_detected` + paksa logout + notifikasi email (fase 2) / notifikasi in-app. Ini standar OWASP + yang dipakai v1, dipertahankan + ditambah `auth_audit`.
- **Logout**: revoke per-sesi / per-family / semua (`logout-all`), blacklist `jti` access di Dragonfly 15 mnt (sisa umur) agar access yang dicuri tidak bisa dipakai walau belum expired.

### 6.2 Password & brute-force

- Hash baru: **Argon2id** (`memory 64MB, iterations 3, parallelism 2`). Hash lama bcrypt (v1) tetap diverifikasi saat login pertama, lalu **di-rehash otomatis** ke Argon2id dalam transaksi yang sama (migrasi tanpa reset password massal).
- Login rate-limit: 5 gagal / 10 mnt per `username+IP` (Dragonfly sliding-window Lua), lalu wajib **Cloudflare Turnstile** (ganti captcha matematika v1 yang bisa di-OCR). Setelah 10 gagal, kunci 30 mnt + audit.
- Validasi: username `^[a-z0-9._-]{3,30}$`, password min 10 char + zxcvbn score ≥ 3 untuk akun baru (siswa import pakai password acak 12 char + wajib ganti saat login pertama via `must_change_password`).

### 6.3 CSRF & XSS & headers

- Karena refresh via cookie + API via same-origin BFF, CSRF dilindungi 3 lapis: `SameSite=Lax` + cek `Origin/Referer` di BFF untuk `POST/PUT/PATCH/DELETE` + **double-submit token** `X-CSRF-Token` (cookie `__Host-csrf` non-HttpOnly + header, diverifikasi di API).
- XSS: semua render komentar/deskripsi via `body_html` yang sudah disanitasi server (`bluemonday` UGC policy + link `rel=noopener`), FE tidak pakai `dangerouslySetInnerHTML` kecuali untuk HTML tersanitasi + `sandbox` iframe untuk embed. CSP via Caddy/Next: `default-src 'self'; img-src 'self' data: https:; media-src 'self'; frame-src youtube-nocookie figma canva docs.google drive.google; script-src 'self' 'nonce-...'; object-src 'none'`.
- Headers (Caddy + Next): `Strict-Transport-Security max-age=63072000`, `X-Content-Type-Options nosniff`, `Referrer-Policy strict-origin-when-cross-origin`, `Permissions-Policy camera=(), mic=()`, `Cross-Origin-Opener-Policy same-origin`.

### 6.4 Sessions UX

Halaman `/settings/sessions`: daftar device (label, IP, terakhir aktif, current), tombol revoke per-device + logout-all. Admin bisa revoke sesi user (misal akun diretas) + audit.

---

## 7. Media, Upload, Embed, Object Storage (S3-compatible, default RustFS)

> **Keputusan arsitektur**: seluruh storage memakai **API S3 standar** (`aws-sdk-go-v2`) dan env `S3_*` yang agnostik. **RustFS adalah implementasi default** (self-host di VPS, service `storage:` di compose), tapi bisa diganti ke MinIO / AWS S3 / Cloudflare R2 / SeaweedFS / Garage **tanpa ubah kode** — cukup ganti env + restart. Tidak ada import SDK RustFS/MinIO di kode, tidak ada env `RUSTFS_*`/`MINIO_*`, tidak ada asumsi path-style/host yang spesifik vendor.

### 7.1 Keputusan inti

- Bucket `S3_BUCKET` **private** (tanpa policy public-read). Tidak ada URL publik langsung ke storage.
- DB hanya simpan `object_key` + `media_id`, **bukan URL absolut**. URL publik dibangun saat serve: `https://DOMAIN/api/media/<media_id>` (BFF) yang 302/stream dari backend dengan `Cache-Control` + `ETag` + `Range`.
- Semua upload via **BFF streaming**: browser `POST /api/uploads (multipart)` → Next Route Handler (validasi awal size/MIME) → `POST http://api:8080/v1/media/upload` (stream, tanpa buffer file penuh, pakai `io.Pipe` + `multipart`) → API validasi lanjutan (magic bytes via `h2non/filetype`, dimensi via `imaging`, EXIF strip, virus-scan hook) → `PutObject` ke storage S3 → insert `media_assets(status=pending)` → worker buat varian thumbnail → `confirm` kaitkan ke profil/portfolio/block.
- Batas (dapat diubah via `system.settings` tanpa deploy):

```
avatar: 2MB, image/*, 800x800 square crop FE + server re-encode webp q80
banner: 10MB, image/*, 1500x500
thumbnail: 5MB, image/*, 1200x900 4:3
block_image: 10MB, image/*
document: 20MB, pdf/doc/ppt + office mime, disimpan apa adanya + preview via Google Docs viewer (embed, bukan file langsung)
feedback_attachment: 5MB, image/pdf
changelog_cover: 5MB
```

- Penamaan key deterministik + anti-collision + cache-friendly:

```
avatars/{user_id}/{uuid7}.webp
banners/{user_id}/{uuid7}.webp
thumbnails/{portfolio_id}/{uuid7}.webp (+ varian _320/_800)
blocks/{portfolio_id}/{block_id}/{uuid7}.webp
documents/{portfolio_id}/{uuid7}.pdf
feedback/{thread_id}/{uuid7}.bin
```

Gunakan `uuid7` (time-ordered) agar listing cepat + `sha256` untuk dedup (jika file sama sudah ada dan milik user yang sama, reuse tanpa upload ulang).

### 7.2 Alur upload (sequence, happy path + gagal)

```
1. FE crop (react-easy-crop) → POST /api/uploads {purpose, portfolio_id?} multipart
2. BFF cek session + size + MIME allowlist → stream ke API (progress via XHR onUploadProgress)
3. API: auth + RequirePermission(media.upload) + quota (max 200MB/user, dihitung dari SUM media confirmed) + magic-byte check
   → jika gagal: 422 {code: INVALID_CONTENT_TYPE|FILE_TOO_LARGE|QUOTA_EXCEEDED} (tanpa simpan apa pun)
4. API `PutObject` storage (internal) + insert media pending (tx) → return {media_id, preview_url (/api/media/<id>)}
5. Worker thumbnails: buat _320/_800 webp, update width/height, status confirmed
6. FE panggil confirm: POST /media/:id/confirm {link_to: avatar|thumbnail|block} → API cek ownership + update FK dalam tx + hapus media lama (mark deleted, hapus objek async 7 hari kemudian via lifecycle)
7. Cleanup: cron tiap jam hapus media pending >24 jam (DB + objek), quarantine file mencurigakan (mime mismatch) tanpa di-serve
```

Keamanan upload:

- Validasi 3 lapis: ekstensi → MIME header → magic bytes + dimensi. File `svg` **dilarang** (XSS via script), `html` dilarang sebagai upload (hanya via embed sanitized).
- EXIF GPS di-strip (privasi siswa), ICC dipertahankan.
- Rate-limit upload: 20 file/jam/user, 100/hari (Dragonfly counter).
- Audit: setiap `media.delete_any` + `quarantine` masuk `audit_logs`.

### 7.3 Serve & cache (performant)

- `GET /api/media/:id` → API cek visibility: avatar/banner publik, thumbnail/block publik jika portfolio `published`, selain itu hanya owner/guru pengampu/admin (403 jika tidak). Lalu:
  - Jika `Range` tidak ada dan file < 5MB: `302` ke URL streaming internal? Tidak — karena storage internal, API **stream** langsung dengan `io.Copy` + `http.ServeContent` (support Range untuk PDF/video) + `ETag: "<sha256>"` + `Cache-Control: public, max-age=31536000, immutable` (karena key content-addressed, aman immutable).
  - Next Image loader memakai `/api/media/:id?w=800&q=75` → BFF teruskan `w` ke API yang serve varian `_800` jika ada (tanpa on-the-fly resize di request panas; resize hanya di worker).
- Dragonfly cache: hot media metadata (`media:<id>` 10 mnt) + leaderboard/feed (lihat §9), bukan bytes file (bytes di-cache di browser/CDN + Next).
- Backup storage: `mc mirror`/`rclone sync` harian storage S3 → disk `/backup/storage` + `pg_dump` (lihat §11.4). Tool backup memakai env `S3_*` yang sama (agnostik vendor).

### 7.5 Kontrak S3-agnostic (aturan kode + config)

Aturan keras agar ganti vendor tanpa ubah kode:

1. **SDK tunggal**: hanya `aws-sdk-go-v2` (`service/s3` + `feature/s3/manager`) dengan `BaseEndpoint` dari `S3_ENDPOINT`. Dilarang import SDK MinIO/RustFS/AWS-spesifik di luar paket `internal/storage`.
2. **Operasi yang dipakai** (semua didukung S3 + RustFS + MinIO + R2): `PutObject, GetObject (Range), HeadObject, DeleteObject, DeleteObjects, ListObjectsV2, CreateBucket, HeadBucket, PutBucketVersioning`. Dilarang pakai API admin/ekstensi vendor (misal `mc admin`, policy console) dari kode aplikasi.
3. **Isolasi vendor di satu paket**: hanya `internal/storage/s3.go` yang tahu endpoint/region/path-style. Handler/service hanya panggil interface `Storage { Put(ctx,...), Get(ctx,...), Head, Delete, PresignGet(durasi) }` — presign hanya untuk kebutuhan internal/tooling, bukan URL publik ke browser.
4. **Env agnostik** (lihat §15.1): `S3_ENDPOINT, S3_REGION, S3_BUCKET, S3_ACCESS_KEY, S3_SECRET_KEY, S3_USE_PATH_STYLE=true, S3_PUBLIC_URL` (base URL publik via BFF, bukan host storage). Tidak ada `*_RUSTFS_*`/`*_MINIO_*` di `.env.example`, kode, compose, atau CI.
5. **Perilaku konsisten lintas vendor**: selalu `path-style` (agar bucket dengan titik aman di TLS), checksum `Content-SHA256` di `media_assets` sebagai sumber kebenaran (bukan ETag vendor yang formatnya beda-beda), versioning diaktifkan bila vendor mendukung (RustFS/MinIO/S3 ya, R2 perlu cek — kode tidak bergantung padanya, lifecycle delete async 7 hari ditangani worker, bukan lifecycle bucket vendor).
6. **Dev/prod parity**: lokal pakai service `storage:` (RustFS) dengan env yang sama persis skemanya dengan prod; e2e CI pakai container S3 fake yang kompatibel (`localstack`/`minio` sebagai stand-in API S3, bukan sebagai dependensi kode). Ganti vendor = ganti 5 env `S3_*` + restart, tanpa rebuild image.
7. **Contoh ganti vendor** (tanpa ubah kode):
   - RustFS lokal: `S3_ENDPOINT=http://storage:9000 S3_REGION=us-east-1 S3_BUCKET=grafikarsa S3_USE_PATH_STYLE=true`
   - Cloudflare R2: `S3_ENDPOINT=https://<account>.r2.cloudflarestorage.com S3_REGION=auto S3_BUCKET=grafikarsa S3_USE_PATH_STYLE=false`
   - AWS S3: `S3_ENDPOINT=https://s3.ap-southeast-1.amazonaws.com S3_REGION=ap-southeast-1 S3_USE_PATH_STYLE=false`

### 7.6 Embed (youtube/figma/canva/ppt/pdf/doc)

- Block `youtube`: simpan `video_id` (11 char regex) saja, render via `https://www.youtube-nocookie.com/embed/<id>` + `loading=lazy` + `sandbox allow-scripts allow-same-origin` + faceless preview (thumbnail maxresdefault) agar tidak load player sampai diklik (hemat 500KB/request).
- `figma/canva`: simpan URL asli, render via embed resmi (`figma.com/embed`, `canva.com/design/.../view?embed`) dengan allowlist + `isAllowedDomain` di BE (tolak URL pendek/bit.ly yang bisa redirect jahat).
- `ppt/pdf/doc`: simpan `media_id` (file di storage private) + render via `https://docs.google.com/viewer?url=<encoded /api/media/:id>&embedded=true` (Google fetch via BFF URL publik, bukan internal — jadi `/api/media` harus publik-read untuk file published, dengan token `?sig=` HMAC 1 jam agar tidak bisa di-scrape massal). Alternatif offline: worker konversi PDF halaman pertama ke webp preview (dipilih untuk v2.0 agar tidak bergantung ke Google).
- `embed` generik: hanya allowlist domain (`codepen.io, codesandbox.io, youtube-nocookie, figma, canva, docs.google, drive.google, slides.google`) + sanitasi `iframe` (hapus `allow-top-navigation`, paksa `sandbox`, `referrerpolicy=no-referrer`). Selain itu → tolak 422 `EMBED_DOMAIN_NOT_ALLOWED`.

---

## 8. Portfolio Lifecycle (diperbaiki dari v1)

### 8.1 State machine (final)

```
draft ──submit──▶ pending_review ──approve──▶ published
  ▲              │  ├─request-changes─▶ changes_requested ──resubmit──▶ pending_review
  │              │  └─reject (dengan note wajib)─▶ rejected ──resubmit──▶ pending_review
  │              └─cancel (oleh owner)──▶ draft
  └──unarchive── archived ◀──archive── published / rejected
  └──delete (soft)──▶ deleted (retensi 30 hari, bisa restore oleh admin)
```

- `draft`: hanya owner + guru pembimbing (jika di-assign) + admin. Autosave tiap 10 dtk ke `portfolios.draft_payload` (JSONB) + `blocks` via `reorder` debounce, tanpa validasi ketat.
- `submit`: validasi **wajib** thumbnail + ≥1 block + judul ≥10 char + ≥1 tag + series (jika jurusan mewajibkan via `majors.require_series`). Gagal → 422 `INCOMPLETE_PORTFOLIO` dengan `details` per-field (bukan pesan generik).
- `pending_review`: masuk antrean moderasi (scope jurusan untuk `admin_jurusan`, global untuk `admin`). SLA 3 hari kerja (diingatkan via notifikasi + dashboard badge). Owner tidak bisa edit konten (hanya bisa `cancel`), agar review konsisten (snapshot `submitted_snapshot` JSONB disimpan saat submit).
- `changes_requested`: seperti rejected tapi ringan (tanpa reset `submitted_at`), notif berisi checklist perbaikan.
- `published`: publik (feed, katalog, profil, search, leaderboard). Edit setelah publish → buat `pending_changes` (draft turunan) tanpa mengganggu versi published sampai di-approve (pendekatan Wikipedia-like, mencegah vandalisme konten lolos moderasi). v2.0 implementasi sederhana: edit langsung tapi `trending/quality` dihitung ulang + audit; `pending_changes` di fase 2 jika dibutuhkan (disiapkan kolom `parent_version_id`).
- `archive`: owner/admin, tidak tampil publik tapi tetap bisa dibuka via link langsung oleh owner.

Semua transisi tercatat di `portfolio_status_history` (implisit via `audit_logs` + kolom `reviewed_by/at/note`) + notifikasi (lihat §10).

### 8.2 Series sebagai template (diperbaiki)

`series_blocks` berisi `instruction` per posisi (misal "Slide 1: latar belakang masalah, max 200 kata"). Saat user pilih series, FE generate `portfolio_blocks` dari template dengan `is_from_series=true` + tampilkan instruksi sebagai placeholder (tidak lock total seperti v1 — user tetap bisa tambah/hapus, tapi penghapusan block wajib menampilkan peringatan + dicatat di audit agar guru bisa lihat kepatuhan template di assessment).

---

## 9. Feed, Leaderboard, Naik Kelas (algoritma proper)

### 9.1 Feed — arsitektur 2 lapis (precompute + personalize)

**Masalah v1**: skor dihitung per-request untuk `limit*2` portfolio + sort in-memory → O(N) per user, tidak konsisten, tidak bisa di-cache personal.

**Desain v2**:

- Lapisan 1 — `portfolio_stats` (global, non-personal, diupdate async): `likes, views_7d/30d, comments, assessment_avg, quality (0-1), engagement (0-1), trending (0-1)`, `computed_at`. Diupdate via **outbox + worker**:
  - Event `like_created`, `comment_created`, `view_batch (tiap 100 views)`, `assessment_scored` → `outbox` → worker agregat tiap 5 mnt (incremental, bukan full scan).
  - `trending` dihitung tiap 15 mnt via cron: `trending = 0.5*norm(likes_7d) + 0.3*norm(views_7d) + 0.2*norm(comments_7d)` dengan decay `exp(-age_days/7)`.
- Lapisan 2 — personalisasi saat request (murah, hanya untuk 50 kandidat dari lapisan 1):
  ```
  final = 0.30*Following + 0.25*Recency + 0.20*Engagement(precomputed) + 0.15*Relevance + 0.10*Quality(precomputed)
  Following: mutual 1.0, one-way 0.8, else 0 (dari follows, 1 query IN)
  Recency: exp(-age_hours/168) (half-life 7 hari, kontinu bukan bucket <24h/<7d)
  Relevance: 0.5*cosine(tag_weights user, tags portfolio) + 0.3*(jurusan sama? 0.5+0.5*log1p(major_weight)/5 : log1p/5) + 0.2*(kelas sama?1:0)
  ```
  Bobot dikunci di `system.settings.feed_weights` agar bisa di-tune tanpa deploy (audit tiap perubahan).
- Cache: `feed:<user_id>:<algo>:<page>` di Dragonfly 2 mnt (page 1) + `feed:global:recent` 1 mnt. Invalidasi via event (like baru tidak langsung invalidate personal, hanya global — personal stale 2 mnt acceptable).
- Mode: `smart` (default auth), `recent` (default guest + fallback), `following` (auth, hanya yang di-follow + mutual boost). Guest tidak ada personalisasi (Relevance=0).

Query SQL (ilustrasi sqlc, bukan N+1):

```sql
-- name: GetFeedCandidates :many
SELECT p.id, p.title, p.slug, p.thumbnail_media_id, p.published_at,
       ps.engagement, ps.quality, ps.trending, u.username, u.name
FROM portfolios p
JOIN portfolio_stats ps ON ps.portfolio_id = p.id
JOIN users u ON u.id = p.owner_id
WHERE p.status='published' AND p.published_at > now() - interval '90 days'
ORDER BY ps.trending DESC LIMIT 50;
-- lalu di Go hitung Following/Recency/Relevance untuk 50 ini + sort final + paginate 15
```

### 9.2 Best Student & Best Project (landing) — precompute harian, bukan live query

**Masalah v1**: `GetTopStudents/Projects` CTE berat tiap buka landing, tanpa histori rank, tanpa scope jurusan.

**Desain v2**:

- Cron `02:05 Asia/Jakarta` (setelah `02:00` backup): hitung untuk `period ∈ {weekly, monthly}` × `scope ∈ {global, jurusan:rpl,tkj,mm,dkv,ani}` × `kind ∈ {best_students, best_projects}`.
- Rumus (dinormalisasi 0-100 via min-max per scope agar jurusan kecil tidak tenggelam):

```
student_score = 30*norm(portfolio_count) + 25*norm(total_likes) + 30*norm(avg_assessment/10) + 15*norm(followers)
  syarat: ≥1 published dalam periode, tidak termasuk admin/guru, anti-gaming: likes dari akun <7 hari tidak dihitung (via users.created_at)
project_score = 50*(assessment_avg/10) + 30*norm(likes) + 20*recency_decay(exp(-age_days/30))
  syarat: published + ≥1 assessment (agar kualitas terjamin) ATAU likes top 5% jika belum dinilai (label "Rising")
```

- Simpan ke `leaderboards` + `leaderboard_entries (rank, score, breakdown JSONB {likes, views, assessment, followers}, delta_rank vs periode lalu)`.
- Serve: `GET /leaderboards?period=weekly&scope=global` → Dragonfly `lb:weekly:global:*` 1 jam → DB fallback. Landing tampilkan **Top 6** + `delta_rank` (▲2 badge) + link "Lihat semua".
- Kapan dihitung ulang: harian otomatis + manual `POST /admin/leaderboards/recompute` (ada audit + rate-limit 1/jam) + invalidasi cache otomatis.
- Anti-gaming: like spam (>20 like/jam dari 1 user) di-flag via worker + tidak dihitung + audit; view dihitung unik harian (bukan mentah).

### 9.3 Naik kelas (promotion) — proper, dry-run, audit

**Masalah v1**: tanpa job, tanpa histori imutabel, tanpa handle tinggal kelas/pindahan.

**Desain v2**:

Tabel: `academic_years (2024 aktif, 2025, ...)`, `classes`, `enrollments(is_current, status)`, `promotion_jobs`.

Alur (wajib via UI, dilarang SQL manual):

```
1. Admin buat tahun ajaran baru (2026) → status draft
2. Klik "Preview Kenaikan" → POST /admin/promotion/jobs {from:2025, to:2026, dry_run:true}
   → worker hitung: {naik: {10A→11A: 32, ...}, lulus: {12*: 400}, tinggal: [ids jika ada flag repeat], kelas_baru_dibuat: 12, error: []}
   → simpan dryrun_result di promotion_jobs(status=previewed), tampilkan tabel preview di UI
3. Admin centang pengecualian (tinggal kelas / pindah jurusan) → approve
4. Klik "Eksekusi" → POST /jobs/:id/execute (RequirePermission promote_execute, konfirmasi ketik tahun)
   → transaksi tunggal:
     a. buat classes tahun baru (copy dari tahun lama, group sama, homeroom dikosongkan)
     b. untuk tiap enrollment is_current: 10→11, 11→12 (class baru), 12→ status graduated + users.status=lulus + is_current=false
     c. repeat/transfer sesuai pengecualian
     d. set academic_years: lama is_active=false, baru true; promotion_jobs done
   → idempotent via UNIQUE(from,to) + job status guard (eksekusi 2× tidak ganda)
5. Notifikasi massal terjadwal (batch 100/5 mnt via outbox agar tidak spam): "Kamu naik ke XI-RPL-A" + "Akun alumni aktif"
```

Jadwal otomatis: cron `1 Juli 00:00` buat `promotion_jobs` draft + notif ke admin (tidak auto-execute — eksekusi tetap manual approve untuk keamanan). `promotion_month/day` per tahun tetap didukung untuk sekolah yang beda tanggal.

Edge yang ditangani: siswa pindahan (enrollment `transferred` + buat enrollment baru), tinggal kelas (`repeated`, tetap di grade sama tahun baru), guru wali pindah (homeroom dikosongkan, bukan di-copy), kelas kosong (<5 siswa → warning di preview, bisa merge manual).

---

## 10. Notifikasi (8 tipe, fan-out via worker)

Tipe: `new_follower, portfolio_liked, portfolio_approved, portfolio_changes_requested, portfolio_rejected, new_comment, reply_comment, feedback_updated` (+ `promotion_result`, `leaderboard_ranked`, `changelog_published` via tipe generik `announcement` dengan `data.kind`).

- Tulis notif via **outbox** dalam transaksi yang sama dengan aksi (like/comment/approve) → worker kirim (insert `notifications` + invalidasi `notifications:count:<user>` di Dragonfly). Tidak ada `go routine` liar di handler (masalah v1).
- Deep-link: `new_follower→/{username}`, `liked/approved/rejected/changes→/me/portfolios?highlight=<id>`, `comment→/{owner}/{slug}#comment-<id>`, `feedback→/settings/feedback#<id>`, `promotion→/me`, `leaderboard→/leaderboards`.
- Preferensi: user bisa matikan `portfolio_liked` (anti-spam seleb) via `notification_preferences` (default semua on kecuali `liked` untuk >1rb followers auto-digest harian).
- Retensi: notif read >90 hari dihapus via cron (kecuali `feedback/changelog` penting 1 tahun).

---

## 11. Feedback & Changelog (proper, lengkap)

### 11.1 Feedback (bukan sekadar form pesan)

- Thread model: `feedback_threads (category bug/idea/question/praise, title, body_md, status, priority, upvotes)` + `feedback_messages` (balasan threaded user↔admin, `is_internal` untuk catatan internal admin yang tidak terlihat user) + `feedback_votes` (1 vote/user, anti-brigading via rate-limit).
- Guest boleh lapor (tanpa login) tapi wajib Turnstile + email opsional untuk follow-up (token `?t=` HMAC agar bisa lihat thread tanpa akun).
- Status: `open → triaged → in_progress → resolved → closed` (+ `duplicate` dengan link ke thread kanonis). Setiap perubahan status wajib komentar (tidak bisa diam-diam) + notif `feedback_updated` + SLA badge (bug critical 1×24 jam, saran 14 hari).
- Prioritas otomatis awal via keyword + manual triage admin. Attachment via `media_assets(purpose=feedback)` (screenshot max 5MB).
- Publik: halaman `/feedback` tampilkan thread `idea/question` yang di-approve (voting publik ala Canny) + roadmap linkage (`linked_changelog_id` saat ide direalisasikan → notif otomatis ke voter "Ide kamu rilis di v2.3!").
- Moderasi: spam/hate → `closed` + `is_spam` + rate-limit reporter (3 thread/hari).

### 11.2 Changelog (bukan sekadar timeline)

- `changelogs (version semver unique, title, summary, channel stable/beta, audience, status draft/scheduled/published, publish_at)` + `sections (added/improved/fixed/removed)` dengan Markdown + `reads` per user.
- Alur: draft → schedule (publish_at masa depan, cron publish) → published (fan-out notif `changelog_published` ke audience, batch) → archived. Unpublish hanya oleh `changelog.publish` + audit (tidak bisa hapus yang sudah dibaca >100 user tanpa approval admin).
- Audience targeting: `all` vs `guru` vs `admin` (misal "Fitur nilai" hanya untuk guru). FE filter via `audience` + tampilkan badge `Baru` (unread) + `mark-all-read` saat buka halaman.
- Kontributor: `changelog_contributors (user_id, role dev/design/qa)` untuk apresiasi OSIS/dev.
- RSS `/changelogs/rss.xml` + SEO per-version `/changelogs/v2.3`.

---

## 12. Frontend Architecture (Next + shadcn + Hugeicons + Inter)

### 12.1 Struktur & konvensi

- `app/` routes sesuai §2.2. Semua halaman interaktif `'use client'`, halaman SEO (`/[username]/[slug]`, landing, changelogs) SSR via `fetch(INTERNAL_API_URL, {next:{revalidate:60}})` + `generateMetadata` + JSON-LD.
- `components/ui/*` **hanya dari shadcn CLI** (`pnpm dlx shadcn@latest add button card dialog table ...`). Dilarang duplikat (`MyButton`). Kustomisasi via `cva` variants + CSS vars, bukan file baru.
- Ikon: `hugeicons-react` free (`SearchIcon`, `HomeIcon`, `PlusIcon`, dsb). Dilarang `lucide-react` + emoji sebagai ikon (lint `no-restricted-imports`).
- Font: `Inter` via `next/font/google` + `JetBrains Mono`? **Tidak** — hanya Inter (sans + mono fallback system) sesuai permintaan. `font-sans: Inter, system-ui`.
- State: RQ untuk server-state, Zustand `auth-store (accessToken di memori, bukan persist token! persist hanya user + preferences)` + `ui-store (sidebar, viewMode)`. Refresh token **tidak pernah** di JS (hanya cookie HttpOnly, dibaca server).
- `middleware.ts`: cek cookie `__Host-rt` + panggil `GET /me` server-side (atau verifikasi JWT edge via `jose` dengan public secret? — dipilih panggil API agar single source of truth, dengan cache 60 dtk di Dragonfly edge). Redirect `/login` jika 401 + `?next=`. Tidak ada `localStorage` auth seperti v1.

### 12.2 BFF Route Handlers (kontrak)

```ts
// app/api/[...path]/route.ts
export async function GET(req, ctx) { return proxy(req, ctx.params.path) }
async function proxy(req, path) {
  const url = `${process.env.INTERNAL_API_URL}/${path.join('/')}${req.nextUrl.search}`;
  const headers = forwardHeaders(req); // x-request-id, cookie, x-csrf-token, content-type
  const res = await fetch(url, { method: req.method, headers, body: req.body, duplex: 'half' });
  return new Response(res.body, { status: res.status, headers: filterHeaders(res.headers) });
}
```

- Upload: `app/api/uploads/route.ts` dengan `export const config = { maxDuration: 60 }`, progress via `XMLHttpRequest` di FE (fetch tidak support upload progress).
- Media: `app/api/media/[id]/route.ts` dengan `Range` forward + `Cache-Control` + `Content-Disposition inline` (bukan attachment agar bisa preview).
- Error: BFF tidak mengubah `error.code` dari backend (FE toast via `sonner` pakai `error.message` Indonesia).

### 12.3 Halaman kunci (UX spec ringkas)

- Landing guest: hero + statistik live (total portfolio/siswa, dari `/leaderboards` cache) + Top Students/Projects mingguan (6 kartu) + FAQ + CTA login (tanpa data pribadi). Lighthouse ≥90, tanpa `useEffect` fetch (SSR semua).
- Feed auth: `SmartFeedList` infinite 15, switcher Smart/Terbaru/Following (persist via `PUT /feed/preferences`), skeleton shadcn, empty-state per-algoritma + CTA buat portfolio, share via `navigator.share` fallback clipboard.
- Editor portfolio: DnD `@dnd-kit`, autosave 10 dtk + indikator "Tersimpan 12:01", validasi inline Zod, thumbnail crop 4:3, block editors per-type (text Markdown toolbar, image dropzone + alt wajib, table grid editor, youtube URL→preview, button link+label, embed URL allowlist), tombol `Simpan Draft` vs `Kirim Review` + dialog konfirmasi + progress overlay.
- Moderasi: tabel shadcn + filter jurusan/status + preview Dialog + approve/reject drawer (note wajib untuk reject/changes) + bulk approve (max 20) + SLA badge.
- Admin users/roles: DataTable server-side (page/sort/search via URL `nuqs`), drawer create/edit, reset password + must_change flag, assign role dengan scope picker (jurusan), expiry date picker.
- Promotion: wizard 3 langkah (Pilih tahun → Preview tabel → Eksekusi ketik konfirmasi) + hasil + audit link.

---

## 13. Backend Architecture (Go Chi + sqlc + pgx)

### 13.1 Struktur paket (wajib)

```
cmd/api/main.go (wiring: config→pgxpool→dragonfly→s3→repos→services→handlers→chi router→graceful shutdown)
internal/config (env parse + validate, satu struct, fail-fast jika secret kosong di prod)
internal/db (NewPgxPool, goose embed, health, tx helper WithTx)
internal/repo (sqlc generated: db/, models/, querier) + db/queries/*.sql (sumber kebenaran)
internal/auth (jwt, argon2, session, csrf, turnstile, audit)
internal/iam (permissions catalog const, middleware RequirePermission, scope resolver)
internal/http (router.go chi groups /v1, handlers/*.go tipis, dto/*.go, errors.go problem+json, validate.go)
internal/service (portfolio, media, feed, leaderboard, promotion, notification, feedback, changelog, search)
internal/storage (s3 client agnostik, image pipeline webp, virus hook)
internal/cache (dragonfly wrapper: Get/Set/Incr/SlidingLimit/PubSub)
internal/jobs (scheduler gocron + workers: outbox dispatcher, leaderboard, promotion, thumbnails, cleanup, digest)
internal/observability (slog, otel trace, metrics expvar/prom)
```

Aturan: handler ≤100 baris, tanpa SQL mentah, tanpa `json.Marshal` manual (via dto), tanpa `go func` (via outbox). Service berisi logika + transaksi, repo hanya query.

### 13.2 Middleware Chi (urutan)

`RequestID → RealIP → Logger(slog) → Recoverer → Timeout(30s) → CORS(same-origin, praktis off karena BFF) → RateLimit(Dragonfly sliding 100/mnt/IP, 5000/mnt global) → AuthOptional/Required (JWT verify + blacklist check) → LoadPermissions (cache) → RequirePermission per-route → Validate Body (Zod-equivalent) → Handler → Audit (untuk mutasi admin)`.

### 13.3 Config & DX backend

`go run ./cmd/api` baca `.env` root (via `godotenv` + `os.Getenv`, bukan file terpisah). `make sqlc` generate, `make migrate-up/down`, `make seed`, `go test ./...` dengan `pgxmock` + testcontainers untuk integrasi (SQLite dilarang — harus Postgres asli agar perilaku `citext/trgm` sama).

---

## 14. Non-Functional Requirements & QA Plan

### 14.1 Performance budget (diukur di VPS prod, bukan laptop)

| Metrik | Target | Cara ukur |
|---|---|---|
| Feed smart p95 | < 300 ms | k6 100 VUs, 5 mnt, via `https://DOMAIN/api/feed` (BFF) |
| Profil + portfolio detail p95 | < 150 ms | k6 + RQ cache off |
| Upload 10 MB | < 8 s (Jakarta → VPS) | curl + FE progress e2e |
| Landing LCP | < 2.5 s, Lighthouse ≥90 | PageSpeed + `next build` analyze |
| DB query p95 | < 50 ms | `pg_stat_statements` + otel |
| Storage throughput | ≥ 80 MB/s internal (api→storage S3) | `mc speedtest`/`rclone check` + backend bench |

Optimasi wajib: pgx pool 25, sqlc `LIMIT` di semua list, Dragonfly untuk feed/lb/count, Next `revalidate` + `stale-while-revalidate`, image `webp` + `sizes`, `dynamic import` untuk editor/PDF, `request_body` Caddy 55MB + backend `MaxBytesReader` 52MB.

### 14.2 Security checklist (OWASP ASVS L2, QA wajib centang)

- [ ] Tidak ada secret di git (gitleaks di CI), `.env` gitignored, secret prod via GH Secrets + server `.env` 600.
- [ ] Auth: Argon2id, JWT 15 mnt + refresh rotation + reuse-detection + `__Host-` cookies + CSRF + Turnstile + rate-limit + session list.
- [ ] IAM: 403 untuk cross-jurusan (test direct-API dengan token `admin_jurusan RPL` akses `TKJ`), tidak ada IDOR (`/portfolios/:id` cek owner/scope, bukan sekadar login).
- [ ] Input: Zod FE + validator BE + sanitasi HTML + allowlist embed + MIME magic + SQL via sqlc (tanpa concat).
- [ ] Upload: svg/html ditolak, EXIF strip, quarantine, quota, virus hook (ClamAV opsional fase 2).
- [ ] Headers: HSTS, CSP nonce, nosniff, Referrer-Policy (verifikasi via `securityheaders.com`).
- [ ] Logging: tidak log password/token/PII berlebihan, audit untuk IAM/moderasi/promotion, retensi 90 hari.
- [ ] Backup: restore drill tiap rilis (lihat §11.4).

### 14.3 Testing pyramid

- Unit (Go `go test`, FE `vitest`): rumus feed/leaderboard/promotion, IAM scope, validators (target coverage ≥70% untuk `service/`, `jobs/`).
- Integration: API e2e via `testcontainers` (Postgres + Dragonfly + S3 fake kompatibel API), FE e2e via `playwright` (login→buat→submit→approve→feed→leaderboard).
- Contract: `openapi.yaml` divalidasi via `schemathesis` tiap CI.
- Load: `k6` skenario feed + upload + search, gagal jika p95 melebihi budget.
- A11y: `axe` di CI untuk landing + feed + editor (WCAG AA), keyboard nav, kontras 4.5:1, `alt` wajib untuk block image.
- SEO: `next-sitemap`, `robots.txt`, OG image per portfolio, JSON-LD `CreativeWork`.

---

## 15. Infra, Env, DX, CI/CD

### 15.1 Single `.env` root (final keys)

```bash
DOMAIN=grafikarsa.com
APP_ENV=production
DB_USER=grafikarsa
DB_PASSWORD=<openssl rand -base64 32>
DB_NAME=grafikarsa
DB_PORT=5432
# ── Object storage S3-compatible (agnostik vendor; default: RustFS self-host) ──
# Ganti ke R2/S3/MinIO cukup dengan mengubah 5 var ini + restart, tanpa ubah kode.
S3_ENDPOINT=http://storage:9000 # dev/compose; prod: http://storage:9000 (internal, tanpa port host)
S3_REGION=us-east-1 # R2 pakai `auto`; AWS pakai region bucket (ex: ap-southeast-1)
S3_BUCKET=grafikarsa
S3_ACCESS_KEY=<rand>
S3_SECRET_KEY=<rand 32>
S3_USE_PATH_STYLE=true # R2/AWS virtual-hosted bisa false
S3_PUBLIC_URL=https://grafikarsa.com/api/media # base URL publik via BFF, bukan host storage
DRAGONFLY_PASSWORD=<rand> # opsional, internal saja
JWT_ACCESS_SECRET=<rand 32> # HS256 256-bit
JWT_REFRESH_SECRET=<rand 32> # HMAC untuk opaque? atau hanya untuk signed metadata — tetap wajib
JWT_ACCESS_TTL=15m
REFRESH_ABSOLUTE_TTL=720h # 30 hari
REFRESH_IDLE_TTL=168h # 7 hari
CSRF_SECRET=<rand>
TURNSTILE_SITE_KEY=...
TURNSTILE_SECRET_KEY=...
INTERNAL_API_URL=http://api:8080/v1 # dibaca web server-side
NEXT_PUBLIC_APP_URL=https://grafikarsa.com
NEXT_PUBLIC_TURNSTILE_SITE_KEY=...
ADMIN_SEED_EMAIL=admin@grafikarsa.com
ADMIN_SEED_PASSWORD=<rand, wajib ganti saat init>
DOCKERHUB_USERNAME=rafapradana
IMAGE_TAG=latest # di-override CI ke sha
BACKUP_S3_PATH=/backup # volume host /opt/grafikarsa/backup
```

Tidak ada `.env` di `apps/*`. `.gitignore` root mengabaikan `.env`, `*.local`, `node_modules`, `.next`, `bin/`, `volumes/`, `backup/`.

### 15.2 Compose (ringkas)

- `docker-compose.yml` (dev): `db (postgres:16-alpine, 5432 host untuk debug) + dragonfly (6379 host) + storage (rustfs image, 9000/9001 host, console on) + api (air hot-reload, volume ./apps/api)`. Web via `pnpm dev` native (cepat, HMR).
- `docker-compose.prod.yml`: seperti dev tapi `api (built)` + `web (built, 3000 host)` untuk simulasi prod lokal.
- `docker-compose.deploy.yml` (server): `proxy (caddy:2-alpine, 80/443/443udp host, volume Caddyfile+caddy_data) + web + api + db + dragonfly + storage (image RustFS default, bisa diganti vendor via `S3_*`)` — **hanya proxy punya ports host**. Semua pakai `restart: unless-stopped`, `healthcheck` (`pg_isready`, `redis-cli ping` via dragonfly, `curl /health/live` endpoint S3 via storage, `wget /health` via api/web).

Caddyfile deploy (final, 20 baris):

```caddy
{$DOMAIN} {
  encode zstd gzip
  request_body { max_size 55MB }
  header {
    Strict-Transport-Security "max-age=63072000; includeSubDomains; preload"
    X-Content-Type-Options "nosniff"
    Referrer-Policy "strict-origin-when-cross-origin"
    Permissions-Policy "camera=(), microphone=(), geolocation=()"
    X-Frame-Options "SAMEORIGIN"
  }
  handle {
    reverse_proxy web:3000
  }
}
```

CSP lengkap di-set via Next `headers()` (nonce per-request) agar bisa inline-script Next tanpa `unsafe-inline` longgar.

### 15.3 Makefile DX (wajib ada)

```make
dev # compose up db+dragonfly+storage+api + info URLs
dev-web # pnpm --dir apps/web dev
dev-down dev-logs
migrate-up migrate-down migrate-new NAME=xxx
sqlc # generate
seed # seed majors/years/admin/roles/permissions/tags/series/metrics
backup # pg_dump + storage mirror ke ./backup/<date>
restore FILE=...
test # go test + pnpm test
test-e2e # playwright
lint # golangci + eslint + tsc
prod # compose prod up --build
deploy-check # validasi DOMAIN + secret tidak default
```

### 15.4 CI/CD (GitHub Actions + DockerHub + SSH proper)

```yaml
# .github/workflows/ci.yml: on push/PR
jobs: lint (go+fe), test-unit, test-integration (services postgres/dragonfly/s3-compatible), contract-test, build (next build + go build), trivy-scan, k6-smoke (opsional di main)
# .github/workflows/deploy.yml: on push main + workflow_dispatch
jobs:
  derive-urls: DOMAIN secret → APP_URL https://DOMAIN, INTERNAL_API_URL http://api:8080/v1
  build-push: buildx api+web → ${DOCKERHUB}/grafikarsa-api:${sha},:latest (cache gha)
  deploy:
    needs: build-push
    steps:
      - ssh via ed25519 (secrets SSH_HOST/SSH_PORT/SSH_USER/SSH_KEY, bukan password!) → mkdir /opt/grafikarsa
      - scp compose + Caddyfile + migrations (bukan .env — .env hanya di server, diupdate via secrets non-kosong)
      - ssh: backup (pg_dump + storage list) → pull → goose up (via api container one-off) → up -d → healthcheck 30×10s (public /health + internal api/db/storage) → prune hanya jika hijau, rollback ke tag sebelumnya jika merah
```

Perbaikan vs v1: tanpa password root di chat, tanpa `latest` saja (ada `sha` immutable untuk rollback), tanpa `rm -rf db/` via `busybox` tanpa backup, dengan `goose` versioned (bukan init SQL buta), dengan server-init.sh yang generate secret + `chown deploy` + `ufw allow 80,443` + `fail2ban`.

Secrets GH wajib: `DOMAIN, DOCKERHUB_USERNAME, DOCKERHUB_TOKEN, SSH_HOST, SSH_PORT, SSH_USER, SSH_KEY, JWT_ACCESS_SECRET, JWT_REFRESH_SECRET, DB_PASSWORD, S3_ACCESS_KEY, S3_SECRET_KEY, TURNSTILE_*`.

---

## 16. Migrasi v1 → v2 (DB 35 MB + Storage 5,1 GB, tanpa downtime panjang)

### 16.1 Prinsip

- Freeze tulis v1 selama migrasi (maintenance mode `NEXT_PUBLIC_MAINTENANCE_MODE=true` di v1, tampilkan halaman "Pindah ke v2").
- Migrasi idempotent + dry-run + checksum, bisa diulang tanpa duplikat (via `legacy_id` mapping table).
- URL lama `https://grafikarsa.com/storage/grafikarsa/*` tetap hidup via **redirect 301** di BFF (`/storage/*` lama → `/api/media/by-key/*`) selama 6 bulan, lalu sunset.

### 16.2 DB migration (`cmd/tools/migrate-old`)

```bash
# di VPS baru (atau lokal dengan dump prod):
pg_dump -Fc -U grafikarsa -h 103.134.78.58 -p 5432 grafikarsa > backup/v1.dump # via SSH tunnel, bukan expose DB
go run ./apps/api/cmd/tools/migrate-old --db-v1 backup/v1.dump --db-v2 $DATABASE_URL --dry-run
go run ./apps/api/cmd/tools/migrate-old --execute
```

Mapping:

| v1 | v2 | Aturan |
|---|---|---|
| `users.role student` + `tahun_masuk+3 <= now` | `role_primary=siswa, status=lulus` + enrollment `graduated` | Hitung dari `tahun_masuk`, jika null → `aktif` + flag `needs_class_fix` untuk admin_jurusan |
| `users.role admin` (1 row) | `admin` + semua permissions | Pertahankan password hash bcrypt (login pertama rehash Argon2) |
| `users.role teacher` (0 row saat ini, tapi enum ada) | `guru` (scope global, admin tugaskan jurusan manual) | — |
| `special_roles` text[] | `roles` custom + `role_permissions` via `ValidCapabilities` mapping lama→baru (`majors`→`academic.*`, `moderation`→`portfolios.moderate`, dsb) | Buat tabel mapping di tool, tampilkan diff untuk approval |
| `jurusan/tahun_ajaran/kelas` | `majors/academic_years/classes` 1:1 + buat `enrollments` dari `users.kelas_id` + `student_class_history` | `nama` auto-format dipertahankan, validasi ulang |
| `portfolios` + `content_blocks` | `portfolios` + `portfolio_blocks` + `media_assets` (untuk block image/doc) | `payload` divalidasi per type; yang gagal → `quarantine_blocks` + laporan (tidak di-drop diam-diam); `slug` keep; `status` mapping `pending_review→pending_review`, `rejected→rejected`, `archived→archived`, `draft→draft`, `published→published` + isi `portfolio_stats` awal |
| `portfolio_likes/follows/comments/notifications` | 1:1 (kecuali view mentah → agregat harian) | `notifications.data` dinormalisasi ke format v2 + `is_read` keep |
| `assessment_metrics/assessments/scores` | 1:1 | Hitung ulang `total_score` + isi `portfolio_stats.assessment_avg` |
| `feedback/changelogs` | `feedback_threads/messages` (1 pesan awal dari `pesan`) + `changelogs/sections` (split dari blocks) | `attachment_url` → `media_assets` via download-then-upload (dengan checksum) |
| `refresh_tokens/token_blacklist` | **Tidak dimigrasi** (semua sesi v1 di-revoke, user login ulang — diumumkan di changelog) | Keamanan > kenyamanan |

Validasi pasca-migrasi (gagal jika tidak lolos):

- `SELECT count(*)`: users ±0, portfolios ±0, blocks ±0 (toleransi quarantine tercatat).
- Checksum file: `COUNT(media_assets) == COUNT(objects di storage)` + spot-check 100 file acak `sha256`.
- Smoke: login 5 akun sampel (siswa/guru/admin), buka 20 portfolio acak (thumbnail + blocks + embed), feed smart 3 algoritma, leaderboard recompute.

### 16.3 Storage migration (MinIO 5,1 GB → storage S3 baru)

```bash
# di server prod lama (atau via mirror) — tool agnostik (mc/rclone), kredensial via env S3_*:
mc alias set v1 http://localhost:9000 <OLD_S3_KEYS> # via ssh -L 9000:localhost:9000 (tanpa expose)
mc alias set v2 http://<vps-baru-internal>:9000 <S3_ACCESS_KEY> # atau rclone remote
mc mirror --preserve --overwrite v1/grafikarsa v2/grafikarsa
# verifikasi: mc diff + cksum spot-check
```

- Layout key dipertahankan 100% (`avatars/...`, `thumbnails/...`, dst) agar mapping DB sederhana (hanya ganti host, bukan key).
- Setelah mirror, jalankan `migrate-old --migrate-media` yang: list objek storage → insert `media_assets(status=confirmed, checksum)` → update `users.avatar_media_id`, `portfolios.thumbnail_media_id`, `portfolio_blocks.asset_media_id` via parsing `object_key` dari URL lama.
- URL lama di DB (`https://.../storage/...`) **tidak di-overwrite buta**: disimpan di `legacy_urls` untuk redirect + audit, kolom baru diisi `media_id`. Rollback = kembalikan `maintenance off` di v1 (data v1 tidak dihapus sampai 30 hari setelah go-live).

Estimasi (VPS 4 GB, 5 GB): mirror ~15–30 mnt (internal 1 Gbps jika satu DC, atau ~1–2 jam via internet 50 Mbps) + DB migrate ~5 mnt + verifikasi ~15 mnt. Total maintenance window **2–3 jam** (dijadwalkan malam + diumumkan H-7 via changelog + notifikasi).

---

## 17. Observability, Backup, Runbook

- Log: `slog` JSON ke stdout (db `slow-query >200ms`, auth `reuse_detected`, upload `quarantine`, promotion `execute`), `docker logs --since` + rotasi via `logrotate` host.
- Health: `GET /health` (liveness, tanpa auth) + `GET /ready` (db+dragonfly+storage, untuk Caddy/API healthcheck + deploy gate).
- Metrics (fase 1 minimal): `api_requests_total{route,status}`, `api_duration_histogram`, `db_pool_in_use`, `jobs_queue_depth`, `media_quarantine_total` via `/metrics` (scrape manual/VictoriaMetrics fase 2).
- Backup: cron host `02:00` → `pg_dump -Fc` + `rclone sync` storage S3 → `/backup/storage` + retensi 7 harian + 4 mingguan (cek `df` — disk prod 81% harus dibersihkan/upgrade sebelum v2 go-live!). Restore drill tiap rilis minor.
- Runbook (`docs/runbook.md`): restart order (db→dragonfly→storage→api→web→proxy), cara buka `psql`/`dragonfly-cli`/`mc`, cara revoke sesi massal, cara re-run leaderboard/promotion, cara rollback deploy (`IMAGE_TAG=<prev-sha> up -d`), kontak on-call.

---

## 18. Roadmap Implementasi (4 fase, tiap fase bisa demo)

- **Fase 0 — Bootstrap (1 sprint)**: monorepo + `.env` + compose + Caddy (hanya web) + goose init + sqlc + CI lint/build + `make dev` hijau + seed admin/majors/years/roles.
- **Fase 1 — Auth/IAM/Users/Media (2 sprint)**: login/refresh/logout/sessions + RBAC scope + users CRUD + roles custom + upload/serve via storage S3 + BFF proxy + avatar/banner e2e.
- **Fase 2 — Portfolio/Social/Feed/Leaderboard (2 sprint)**: lifecycle + blocks + series + likes/comments/follows + feed 2-lapis + leaderboard cron + landing Top + search trigram+tsvector.
- **Fase 3 — Akademik/Assessment/Feedback/Changelog/Jobs (2 sprint)**: classes/enrollments + promotion dry-run/execute + assessments + feedback threads + changelog schedule + notifications fan-out.
- **Fase 4 — Migrasi & Go-live (1 sprint)**: `migrate-old` dry-run di staging (dump prod) → mirror storage → verifikasi → maintenance 2 jam → go-live → redirect `/storage/*` → monitor 7 hari → hapus v1 setelah 30 hari.

Kriteria go-live (hard gate): semua checklist §14.2 hijau + k6 p95 lolos + restore drill sukses + migrasi dry-run 0 error kritis + disk <70%.

---

## 19. Risiko & Mitigasi

| Risiko | Probabilitas | Mitigasi |
|---|---|---|
| Disk prod 81% penuh sebelum migrasi | Tinggi | Bersihkan log/docker prune + upgrade volume / tambah disk sebelum Fase 4; backup ke luar VPS (R2/S3) |
| Storage SNSD tanpa redundansi (1 disk, default RustFS) | Sedang | Versioning on + backup harian + dokumentasikan batas (scale ke multi-drive/K8s di fase 2 jika >100 GB; ganti vendor via `S3_*` tanpa ubah kode) |
| Dragonfly single-node hilang saat restart (jika tanpa AOF) | Rendah | AOF on + semua cache rebuildable (DB source of truth, bukan Redis sebagai store seperti v1 yang sudah benar) |
| BFF jadi bottleneck upload | Sedang | Streaming tanpa buffer + timeout 60s + limit 52MB + opsi presigned-redirect (§7.3) jika p95 >8s |
| Guru/Admin_jurusan bingung scope | Sedang | UI scope picker + banner "Kamu mengelola RPL saja" + QA test cross-jurusan + training 1 jam |
| URL lama mati / SEO drop | Rendah | Redirect 301 `/storage/*` + slug keep + sitemap baru + Search Console submit |

---

## Lampiran A — Permission matrix UI (untuk FE)

Lihat §1.2. Sidebar admin di-filter via `GET /me/permissions` (bukan hardcode role). Setiap tombol destruktif (`Hapus user`, `Eksekusi promosi`) wajib `ConfirmDialog` shadcn + ketik konfirmasi untuk irreversible.

## Lampiran B — Block JSON Schemas (ringkas)

- `text`: `{markdown: string(1..20000)}` → render Markdown (remark) + sanitasi.
- `image`: `{media_id: uuid, alt: string(5..200) wajib, caption?: string}`.
- `table`: `{headers: string[1..10], rows: string[][1..50]}`.
- `youtube`: `{video_id: ^[A-Za-z0-9_-]{11}$}`.
- `button`: `{label: 1..50, url: https allowlist, variant: primary|secondary}`.
- `embed|figma|canva|ppt|pdf|doc`: `{url: allowlist, media_id?: untuk ppt/pdf/doc}`.
- `link`: `{url, title}` (pengganti `embed` bebas v1).

## Lampiran C — Perintah DX (contoh)

```bash
cp .env.example .env && make dev
make dev-web # terminal 2
make migrate-up && make seed
make test && make lint
make prod # simulasi prod lokal
make backup && make restore FILE=backup/2026-10-09.dump
```

---

*Akhir PRD v2.0 — siap dijadikan issue/epic per fase. Keputusan yang belum final (perlu konfirmasi owner): 1) Argon2id vs bcrypt-12 untuk hash baru (default Argon2id), 2) ClamAV on/off di upload (default off fase 1, hook disiapkan), 3) Domain tetap `grafikarsa.com` atau pindah (asumsi tetap), 4) VPS tetap 4GB/50GB atau upgrade disk sebelum migrasi (direkomendasikan tambah 50GB karena 81% penuh).*
