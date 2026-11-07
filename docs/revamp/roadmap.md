# Grafikarsa Revamp — Roadmap & Backlog

> Fase 1 (MVP sampai go-live + migrasi v1) dan fase 2. Sprint = 2 minggu.
> Estimasi mengasumsikan **1–2 developer** dengan bantuan AI. Rollout ke sekolah diurus terpisah oleh pemilik proyek.

---

## 1. Definition of Done (berlaku untuk setiap item)

- [ ] Kontrak OpenAPI diperbarui, kode server & klien di-generate, CI cek drift hijau.
- [ ] Migrasi goose + query sqlc (jika menyentuh DB), bisa di-rollback.
- [ ] Unit test logika domain + integration test (Testcontainers) untuk alur utama.
- [ ] **Test otorisasi**: akses langsung ke API tanpa izin / beda scope jurusan ditolak (bukan hanya UI yang disembunyikan).
- [ ] UI memakai komponen **shadcn/ui** + font **Inter**, light & dark, responsif 360 px+, lolos axe.
- [ ] Teks UI bahasa Indonesia, pesan error manusiawi.
- [ ] Log terstruktur + metrik/trace untuk alur baru, aksi admin/moderator masuk audit log.
- [ ] Memenuhi anggaran performa (nfr.md) untuk halaman yang disentuh.
- [ ] Dokumen di `docs/revamp/` diperbarui bila perilaku berubah.

---

## 2. Fase 1 — MVP & go-live

### M0 · Fondasi (1 sprint)

| ID | Item |
|---|---|
| F-01 | Monorepo (pnpm workspace + Go module), `.env` root, `make dev-init`, `make dev` |
| F-02 | Docker Compose dev: Postgres 18, Valkey, RustFS; prod: + Caddy, web, api, worker |
| F-03 | Pipeline kontrak: `openapi.yaml` → oapi-codegen (Go) + orval (TS); skema blok JSON Schema → Go + zod |
| F-04 | `schema.sql` dipecah jadi migrasi goose (database.md §9) + sqlc + River migrate |
| F-05 | Kerangka Go: config, slog, OpenTelemetry, error problem+json, middleware (request id, recover, CSRF, rate limit, auth), healthz/readyz |
| F-06 | Kerangka Next.js: App Router, shadcn/ui, Inter, token desain (warna Grafika), dark mode, layout & navigasi (PRD §13.9), halaman error/404/maintenance |
| F-07 | Seed (seed.md) + `make seed-admin` |
| F-08 | CI: lint (golangci-lint, ESLint), typecheck, test, build image ke GHCR; Renovate |
| F-09 | Caddyfile (routing PRD §14.2, header keamanan, blok `/internal`) |

### M1 · Akun, akademik, IAM (2 sprint)

| ID | Item |
|---|---|
| A-01 | Login (NISN/NIP/username), sesi opaque, logout, daftar & cabut sesi |
| A-02 | Aktivasi: pilih username, ganti sandi awal, 8 recovery code + konfirmasi |
| A-03 | Lupa sandi via recovery code, ganti sandi, sandi sementara (`/ganti-sandi`), generate ulang recovery code |
| A-04 | RBAC: permission, role sistem, user_roles dengan scope jurusan, middleware `RequirePermission` + filter SQL |
| A-05 | Admin: CRUD jurusan, tahun ajaran, kelas (otomatis membuat grup) |
| A-06 | Admin: daftar/detail/buat/ubah/nonaktifkan user, reset sandi, terbitkan ulang sandi awal, cabut `content.create` |
| A-07 | Impor CSV siswa/guru/tendik (validasi → pratinjau → commit) + enrollment |
| A-08 | Ekspor kartu akun per kelas (PDF) |
| A-09 | Audit log + halaman `/admin/audit` |
| A-10 | Username: aturan, reserved, ganti 1×/30 hari, redirect 90 hari |

### M2 · Media (1 sprint)

| ID | Item |
|---|---|
| ME-01 | `POST /media` streaming ke RustFS (`gk-private`), validasi magic bytes & ukuran, kuota harian |
| ME-02 | Worker: hapus EXIF, varian AVIF/WebP multi-ukuran, blurhash, warna dominan, thumbnail PDF, validasi `.glb` |
| ME-03 | Akses media privat via api, publik via Caddy `/media/*`, promosi/pencabutan ke `gk-public` |
| ME-04 | Komponen gambar Next (srcset, placeholder, tanpa CLS) |
| ME-05 | Pembersihan media yatim |

### M3 · Profil (1 sprint)

| ID | Item |
|---|---|
| P-01 | Halaman profil `/@username` + tab (Portfolio, Blog, Experience, Kolaborasi), pengikut/mengikuti |
| P-02 | `/pengaturan/profil`: banner, avatar, nama, headline, bio, kota, link, skill, sematkan |
| P-03 | Experience CRUD + entri sistem SMKN 4 Malang otomatis |
| P-04 | `/onboarding` |

### M4 · Konten & editor (3 sprint)

| ID | Item |
|---|---|
| C-01 | State machine lifecycle di domain (PRD §4) + test semua transisi valid/tidak valid |
| C-02 | Studio: dashboard per status, buat konten, hapus/pulihkan, arsip/buka arsip |
| C-03 | Editor portfolio: blok (editor-blocks.md §3), drag & drop, unggah, embed allowlist |
| C-04 | Editor blog: TipTap dengan node/mark terbatas, slash command, daftar isi |
| C-05 | Autosave + `lockVersion` + presence + antrian offline IndexedDB |
| C-06 | Validasi submit (Go + zod), snapshot versi imutabel, `content_hash`, ekstraksi media |
| C-07 | Revisi konten tayang (draft dari versi tayang, buang revisi) |
| C-08 | Kolaborasi: undang, terima/tolak, level, label peran, urutan, keluar, transfer kepemilikan, kedaluwarsa 14 hari, batas author dari setting |
| C-09 | Renderer publik (dipakai juga untuk pratinjau) + halaman `/karya/{slug}-{id}` & `/blog/{slug}-{id}` + redirect slug kanonik + 404 status non-tayang |
| C-10 | Field umum: kategori, tag, tools, skill, konteks, periode, link, seri blog |

### M5 · Moderasi (1,5 sprint)

| ID | Item |
|---|---|
| MO-01 | Kasus moderasi dibuat saat submit, routing antrian per jurusan pemilik |
| MO-02 | `/moderasi`: antrian FIFO, filter, penanda umur, claim + kedaluwarsa |
| MO-03 | Ruang tinjau: pratinjau, diff revisi, anotasi per blok, keputusan setuju/revisi/tolak |
| MO-04 | Efek keputusan (transaksional): versi live, denormalisasi, index filter, promosi media, revalidasi, notifikasi |
| MO-05 | Takedown, sembunyikan/pulihkan komentar, reset profil/grup, hapus experience |
| MO-06 | Laporan (buat, antrian, resolve) |
| MO-07 | Banding (sekali per penolakan) + keputusan Admin |
| MO-08 | Statistik moderasi + ringkasan harian in-app |
| MO-09 | Filter kata (block/flag) |

### M6 · Sosial, notifikasi, discovery, grup (2 sprint)

| ID | Item |
|---|---|
| S-01 | Follow, apresiasi, simpan (optimistic), `/tersimpan` |
| S-02 | Komentar + balasan + mention + edit 15 menit + hapus oleh penulis/pemilik konten |
| S-03 | Beacon view & share, worker counter & `content_stats` |
| S-04 | Notifikasi (agregasi, preferensi) + SSE + `/notifikasi` |
| S-05 | Feed beranda: Mengikuti, Jurusanku, Terbaru (+ landing tamu) |
| S-06 | Katalog Karya & halaman Blog: filter + urut terbaru/populer (7d/30d/semua) |
| S-07 | Grup jurusan & kelas: halaman, mirror isi (`group_contents`), anggota, tentang, kelola tampilan, pengelola, status riwayat |
| S-08 | Pencarian + autocomplete |

### M7 · Kenaikan kelas & admin lengkap (1 sprint)

| ID | Item |
|---|---|
| K-01 | Promotion job: pemetaan rombel otomatis, item default, pengecualian manual + CSV |
| K-02 | Dry-run + konflik, approve/unapprove, eksekusi terjadwal (River cron), eksekusi manual |
| K-03 | Efek eksekusi (enrollment, alumni, grup riwayat, experience, notifikasi) + rollback 14 hari |
| K-04 | Pengingat 30 & 7 hari |
| K-05 | Admin: setting, taksonomi (kategori, skill, tools, tag, alasan, reserved username), grup & pengelola, dashboard |

### M8 · Hardening & migrasi v1 (2 sprint)

| ID | Item |
|---|---|
| H-01 | SEO: metadata, OG image, JSON-LD, sitemap, robots |
| H-02 | Performa: Lighthouse CI, anggaran bundle, k6 load test (nfr.md §7), perbaikan temuan |
| H-03 | Keamanan: checklist OWASP ASVS L2 (auth, sesi, akses, input, log), CSP ketat, uji bypass otorisasi |
| H-04 | Aksesibilitas: audit axe + pembaca layar di flow utama |
| H-05 | Produksi: compose prod, Caddy TLS, pgBackRest + arsip WAL, restic, uji restore otomatis, Grafana/Victoria*, alert Telegram, Uptime Kuma |
| H-06 | `migrate-v1` (inspect, dry-run, execute, media, verify, report) + pencocokan akun |
| H-07 | Redirect URL v1 (`legacy_redirects` + middleware) |
| H-08 | Gladi migrasi penuh di staging dengan dump produksi |
| H-09 | Runbook: deploy, rollback, restore, insiden, rotasi secret, kenaikan kelas |
| H-10 | Halaman `/pedoman`, `/tentang`, `/privasi`, `/syarat` |

**Total fase 1: ±14,5 sprint (±7 bulan)** untuk 1–2 developer. Dengan tim lebih besar, M3–M6 bisa paralel
setelah M4 C-01/C-06 selesai.

### Milestone demo

| Setelah | Bisa didemokan |
|---|---|
| M1 | Admin impor siswa → siswa aktivasi dan login |
| M4 | Siswa membuat portfolio/blog bersama kolaborator dan mengirim ke moderasi |
| M5 | Konten disetujui moderator dan tayang di URL publik |
| M6 | Platform terasa "hidup": feed, apresiasi, komentar, notifikasi realtime, grup |
| M8 | Siap go-live dengan data v1 |

---

## 3. Fase 2

| ID | Item | Estimasi |
|---|---|---|
| X-01 | Badge sistem otomatis + badge sekolah (katalog, beri, cabut) | 0,5 sprint |
| X-02 | Endorse skill | 0,25 sprint |
| X-03 | Kurasi "Pilihan Guru" + landing karya pilihan | 0,5 sprint |
| X-04 | Tab Trending (worker skor 15 menit) | 0,5 sprint |
| X-05 | Penugasan guru + rubrik + penilaian | 1,5 sprint |
| X-06 | Ekspor CV/portfolio ke PDF | 0,5 sprint |

**Total fase 2: ±4 sprint.**

---

## 4. Risiko jadwal

| Risiko | Mitigasi |
|---|---|
| Editor (blok + TipTap + autosave + offline) lebih rumit dari perkiraan | Dikerjakan paling awal di M4, offline queue boleh menyusul di M8 |
| Data siswa resmi sekolah terlambat / format berantakan | Template CSV + validator dibuat di M1, minta data ke sekolah sejak awal |
| Kualitas data v1 (duplikat, URL media rusak) | `migrate-v1 inspect` dijalankan sejak M1 pada dump terbaru, bukan menunggu M8 |
| Spek VPS ternyata kecil | Observability dibuat opsional (profil compose terpisah) |
