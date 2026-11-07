# Grafikarsa Revamp — Desain Database

> DDL lengkap: [`schema.sql`](schema.sql) (sudah diuji di PostgreSQL 18: 78 tabel, semua constraint jalan).
> Dokumen ini menjelaskan **kenapa** skemanya begitu, invariant yang dijaga, dan pola query-nya.

---

## 1. Konvensi

| Aturan | Alasan |
|---|---|
| Identifier DB bahasa Inggris (`contents`, `status = 'published'`), label UI bahasa Indonesia | Konsisten dengan kode Go/TS, terjemahan di satu tempat (UI) |
| PK `uuid` **v7** (`uuidv7()` bawaan PG18) | Urut waktu → insert ke B-tree tetap rapat, aman diekspos, bisa dibuat di aplikasi |
| `public_id` 8 karakter `[0-9a-z]` untuk konten | URL pendek `/karya/poster-hut-80-a7k2x9pq`. 36⁸ ≈ 2,8 triliun kombinasi, dibuat acak di aplikasi + retry bila bentrok |
| Semua waktu `timestamptz` (UTC) | Tampilan Asia/Jakarta di aplikasi |
| Enumerasi pakai `text + CHECK`, bukan `ENUM` | Menambah nilai cukup ganti CHECK di migrasi, tanpa masalah `ALTER TYPE` |
| Batas panjang teks di DB (`CHECK char_length`) | Pertahanan terakhir meski validasi aplikasi kebobolan |
| `citext` untuk username, slug master data | Unik tanpa peduli huruf besar/kecil |
| Tidak ada `ON DELETE CASCADE` ke `users` dari data penting (konten, komentar, moderasi) | User tidak pernah di-hard-delete selama punya jejak. Penghapusan akun = anonimisasi (§8) |
| Kolom JSON hanya untuk data yang memang dokumen (`body`, `meta`, `variants`, `data`) | Yang difilter/di-join selalu kolom biasa |

---

## 2. Peta domain

```
AKADEMIK      majors ─< classes >─ academic_years
                          │
PENGGUNA      users ─< enrollments        users ─┬─ profiles, profile_links
                ├─ sessions, recovery_codes      ├─ experiences ─< experience_skills / attachments
                ├─ user_roles >─ roles ─< role_permissions >─ permissions
                ├─ user_skills ─< skill_endorsements
                └─ user_badges >─ badges

KONTEN        contents ─┬─ content_drafts (1:1, salinan kerja)
                        ├─< content_versions (snapshot imutabel) ─< content_version_media
                        ├─< content_authors >─ users
                        ├─ content_stats (1:1), content_views_daily
                        ├─< content_tags / content_tools / content_skills / portfolio_facets
                        └─ blog_series_items >─ blog_series

GRUP          groups (major | class) ─< group_managers
                        └─< group_contents (index mirror isi grup)

MODERASI      moderation_cases (1 per versi) ─< moderation_annotations
                        └─ appeals (maks 1 per kasus ditolak)
              reports, moderation_actions, moderation_reasons, word_filters

SOSIAL        follows, appreciations, bookmarks, comments ─< comment_mentions, pinned_contents
NOTIFIKASI    notifications, notification_preferences
KENAIKAN      promotion_jobs ─< promotion_class_mappings, promotion_items
FASE 2        rubrics ─< rubric_criteria; assignments ─< assignment_submissions ─ assessments ─< assessment_scores
              curations ─< curation_items
SISTEM        settings, import_jobs, audit_logs, media_assets
LEGACY        legacy_id_map, legacy_redirects, legacy_quarantine
```

---

## 3. Akun & autentikasi

- `users.account_type`: `student` | `teacher` | `staff`. **Alumni bukan tipe**, tapi `status = 'alumni'`
  (hanya boleh untuk `student`, dijaga CHECK).
- Identitas login: `nisn` (siswa), `staff_number` (NIP/ID staf), atau `username`. Tidak ada email.
- `password_kind` menentukan alur:

  | `password_kind` | Arti | Setelah login diarahkan ke |
  |---|---|---|
  | `initial` | Sandi awal dari kartu akun | `/aktivasi` |
  | `temporary` | Sandi sementara hasil reset Admin | `/ganti-sandi` |
  | `user` | Sandi buatan user | Halaman tujuan |

  `initial`/`temporary` wajib punya `password_expires_at` (CHECK).
- `password_hash` format PHC. Argon2id untuk semua sandi baru. Hash bcrypt dari v1 tetap diterima,
  lalu **di-rehash ke Argon2id** saat login sukses pertama.
- `sessions.token_hash` = SHA-256 token acak 32 byte di cookie. Token asli tidak pernah disimpan.
  Sesi punya batas idle (sliding, default 7 hari) dan batas absolut (default 30 hari).
- `recovery_codes`: 8 kode per `batch_id`. Generate ulang = batch lama diisi `revoked_at`.
- `content_blocked_at` = capability `content.create` dicabut (tidak bisa membuat konten/diundang kolaborasi).

---

## 4. Akademik & grup

- `classes` unik per `(tahun ajaran, jurusan, tingkat, rombel)`. `slug` (`xi-rpl-b`) unik per tahun ajaran,
  jadi URL `/kelas/2026-2027/xi-rpl-b` deterministik.
- `enrollments` satu baris per siswa per tahun ajaran (UNIQUE). Tidak pernah diubah kecuali kolom `outcome`
  diisi di akhir tahun. Rollback kenaikan = hapus enrollment milik `promotion_job_id` itu + kosongkan `outcome`.
- `groups` dibuat otomatis (oleh service akademik) setiap ada jurusan/kelas baru. CHECK memastikan
  grup jurusan hanya punya `major_id`, grup kelas hanya `class_id`.
- **Keanggotaan grup tidak disimpan**, diturunkan dari `enrollments`:
  - Grup kelas: `enrollments.class_id = groups.class_id`.
  - Grup jurusan: siswa/alumni yang punya enrollment di kelas mana pun dengan `classes.major_id = groups.major_id`.
- **`group_contents`** adalah index materialisasi isi grup supaya feed grup cepat (satu range scan):
  - Diisi worker saat konten terbit, saat author berubah, saat enrollment berubah (kenaikan kelas/rollback).
  - Grup kelas hanya menerima konten yang `first_published_at`-nya di dalam rentang
    `academic_years.starts_on .. ends_on` kelas itu (aturan PRD §7.4).
  - Konten keluar dari `published` (arsip/takedown/hapus) → baris dihapus. Terbit lagi → dimasukkan lagi.

---

## 5. Konten

### 5.1 Tiga tabel inti

| Tabel | Isi | Bisa diubah? |
|---|---|---|
| `contents` | Identitas, status lifecycle, pointer `live_version_id`, salinan kolom untuk kartu/filter (judul, cover, kategori, excerpt) | Ya (oleh service lifecycle) |
| `content_drafts` | Salinan kerja editor (satu per konten) | Ya, oleh pemilik/editor, dengan `lock_version` |
| `content_versions` | **Snapshot imutabel** setiap kali dikirim ke moderasi | **Tidak.** Trigger menolak perubahan isi, hanya `state`/`decided_at` yang boleh berubah |

### 5.2 Pemetaan lifecycle (PRD §4) ke kolom

| Kejadian | `contents.status` | `revision_status` | `content_versions` | `moderation_cases` |
|---|---|---|---|---|
| Buat konten | `draft` | – | – | – |
| Kirim ke moderasi | `in_review` | – | insert `pending` (kind `initial`/`resubmission`) | insert `open` |
| Tarik kembali | `draft` | – | `withdrawn` | `withdrawn` |
| Moderator claim | `in_review` | – | `pending` | `claimed` |
| Setujui | `published` | – | `approved`, jadi `live_version_id` | `decided/approved` |
| Minta revisi | `changes_requested` | – | `changes_requested` | `decided/changes_requested` |
| Tolak | `rejected` | – | `rejected` | `decided/rejected` |
| Arsip / buka arsip | `archived` ↔ `published` | – | – | – |
| Takedown | `taken_down` | – | – | – (dicatat di `moderation_actions`) |
| Edit konten tayang | `published` | `editing` | – | – |
| Kirim revisi | `published` | `in_review` | insert `pending` (kind `revision`) | insert `open` |
| Revisi disetujui | `published` | NULL | `approved`, ganti `live_version_id` | `decided/approved` |
| Revisi diminta perbaikan | `published` | `changes_requested` | `changes_requested` | `decided` |
| Revisi ditolak | `published` | NULL (draft dikembalikan ke versi tayang) | `rejected` | `decided/rejected` |
| Hapus | `deleted` (+ `status_before_delete`, `purge_after`) | NULL | – | – |

Invariant yang dijaga DB:
- `status = 'published'` ⇒ `live_version_id` tidak NULL.
- `revision_status` hanya boleh ada saat `published`.
- Paling banyak **satu** versi `pending` per konten (`uq_versions_one_pending`).
- Satu `moderation_case` per versi (UNIQUE `version_id`).
- Tepat satu `owner` per konten (`uq_content_one_owner`), owner pasti `accepted`.
- Keputusan `changes_requested`/`rejected` wajib punya catatan, `rejected` wajib punya alasan.

Invariant yang dijaga service (tidak bisa diekspresikan CHECK):
- `contents.owner_id` = baris `content_authors` dengan `level = 'owner'` (diubah bersama dalam satu transaksi).
- Jumlah author accepted+pending ≤ batas setting (`collab.max_authors_portfolio|blog`).
- Unarchive tanpa review hanya jika `content_hash` versi tayang sama (selalu benar karena snapshot imutabel,
  dicek ulang sebagai pengaman).

### 5.3 Isi `body` dan `meta`

`body` mengikuti [`editor-blocks.md`](editor-blocks.md). `body_schema` menyimpan versi format supaya
konten lama bisa dimigrasi saat format berubah.

`meta` per tipe (divalidasi service, bukan DB):

```jsonc
// portfolio
{
  "context": "assignment|internship|competition|ukk|personal|other",
  "period": { "start": "2026-08", "end": "2026-10" },
  "links": [{ "label": "Demo", "url": "https://..." }],      // maks 5
  "credits": { "<user_id>": "Animator" }                     // label peran (opsional, sinkron ke content_authors)
}
// blog
{
  "series_id": "uuid | null",
  "toc": true
}
```

### 5.4 Index filter & denormalisasi

- Saat versi disetujui, service menulis ulang dalam satu transaksi: kolom denormal di `contents`,
  `search_vector`, `content_tags/tools/skills`, `portfolio_facets`, `blog_series_items`,
  lalu enqueue job: promosi media, revalidasi halaman, `group_contents`, notifikasi.
- `search_vector` = `setweight(to_tsvector('indonesian', title), 'A') || setweight(excerpt, 'B') || setweight(teks body, 'C')
  || setweight(nama author + tag, 'B')`.
- `content_stats` dipisah dari `contents` (`fillfactor = 80`) supaya update counter yang sering tidak membuat
  bloat dan tidak mengunci baris konten.

### 5.5 Media

- Semua file tercatat di `media_assets`. Object key acak, nama file asli hanya untuk tampilan.
- `content_draft_media` dan `content_version_media` mencatat media mana dipakai oleh draft/versi
  (diekstrak dari `body` + cover saat simpan/submit). Dipakai untuk:
  - menyalin varian ke `gk-public` saat disetujui (`is_public = true`),
  - mencabut varian saat konten keluar dari `published` dan tidak dipakai versi tayang lain,
  - mendeteksi media yatim (tidak direferensikan draft/versi/profil/grup > 7 hari) untuk dibersihkan.

---

## 6. Moderasi

- Antrian = `moderation_cases WHERE status IN ('open','claimed')`, diurut `created_at` (FIFO), difilter
  `queue_major_id` (NULL = global).
- `queue_major_id` = jurusan **pemilik** saat submit (dari enrollment tahun berjalan; alumni → jurusan
  enrollment terakhir; guru/tendik → NULL).
- **Claim** pakai `UPDATE ... SET status='claimed', claimed_by=$me, claim_expires_at=now()+'30 min'
  WHERE id=$1 AND (status='open' OR (status='claimed' AND claim_expires_at < now()))` dan cek baris
  terpengaruh = 1. Tidak butuh lock terpisah.
- Moderator yang ikut jadi author tidak boleh claim (dicek service: `NOT EXISTS content_authors`).
- Takedown, sembunyikan komentar, reset profil/grup dicatat di `moderation_actions` (bisa merujuk `reports`).

---

## 7. Sosial & notifikasi

- `appreciations`, `bookmarks`, `follows`: PK komposit = otomatis idempoten (`INSERT ... ON CONFLICT DO NOTHING`).
- Counter di `content_stats` diperbarui oleh worker (batch), bukan trigger, supaya interaksi ramai tidak
  menjadi hot row. UI memakai optimistic update.
- `comments`: balasan 1 level (service menolak `parent_id` yang sendirinya balasan). Komentar tidak
  dihapus fisik; `status` menyimpan alasan supaya thread tetap utuh ("komentar dihapus").
- `notifications.group_key` + unique index `(recipient_id, group_key) WHERE read_at IS NULL` membuat
  notifikasi sejenis yang belum dibaca **digabung** lewat upsert (`actor_count + 1`, `actor_id` terbaru).
  Detail di [`algorithms.md`](algorithms.md) §6.

---

## 8. Retensi & penghapusan

| Data | Retensi |
|---|---|
| Konten `deleted` | Dipurge setelah `purge_after` (30 hari): baris `contents` + cascade, media dicabut |
| Sesi | Baris kedaluwarsa/dicabut dihapus setelah 30 hari |
| Notifikasi | Sudah dibaca > 90 hari dihapus. Belum dibaca > 180 hari dihapus |
| `content_views_daily` | Disimpan 400 hari (cukup untuk statistik tahunan) |
| Audit log | Disimpan minimal 2 tahun |
| Media yatim | Dihapus setelah 7 hari tidak direferensikan |
| Akun yang dinonaktifkan | Data tetap (konten publik tetap tampil). Permintaan hapus akun = **anonimisasi**: `display_name` → "Pengguna nonaktif", profil/experience/link/skill dihapus, username dilepas, sesi & recovery code dihapus. Konten yang ia miliki sendirian ikut dihapus; konten kolaborasi tetap dengan kredit anonim |

---

## 9. Urutan migrasi goose (saat implementasi)

```
00001_extensions_util.sql        citext, pg_trgm, set_updated_at()
00002_academic.sql               majors, academic_years, classes
00003_media.sql                  media_assets
00004_users_auth.sql             users, reserved_usernames, username_history, sessions, recovery_codes
00005_iam.sql                    permissions, roles, role_permissions, user_roles, teacher_majors
00006_enrollments.sql            enrollments
00007_profiles.sql               profiles, profile_links, skills, user_skills, skill_endorsements,
                                 tools, experiences(+relasi), badges, user_badges
00008_taxonomy.sql               content_categories(+majors), tags, blog_series
00009_contents.sql               contents, content_versions(+trigger), content_drafts, *_media,
                                 content_authors, content_tags/tools/skills, portfolio_facets,
                                 blog_series_items, content_stats, content_views_daily
00010_groups.sql                 groups, group_managers, group_contents
00011_moderation.sql             moderation_*, appeals, reports, word_filters
00012_social.sql                 follows, appreciations, bookmarks, comments, comment_mentions, pinned_contents
00013_notifications.sql          notifications, notification_preferences
00014_promotion.sql              promotion_*
00015_assessment_curation.sql    rubrics…, curations… (fase 2, boleh dibuat sejak awal)
00016_system.sql                 settings, import_jobs, audit_logs
00017_legacy.sql                 legacy_* (dihapus setelah migrasi v1 selesai + 6 bulan)
river migrations                 dijalankan lewat `river migrate-up`
```

---

## 10. Estimasi volume & kapasitas

Asumsi: 3.300 siswa aktif + ±1.100 alumni baru/tahun, 2 konten/siswa/tahun.

| Tabel | Tahun 1 | Tahun 5 |
|---|---|---|
| `users` | ±5.500 (termasuk migrasi v1) | ±10.000 |
| `contents` | ±9.000 | ±35.000 |
| `content_versions` | ±20.000 | ±90.000 |
| `appreciations` | ±300 rb | ±2 jt |
| `comments` | ±60 rb | ±400 rb |
| `notifications` (setelah retensi) | ±500 rb | ±1 jt |
| `content_views_daily` | ±1 jt | ±5 jt (retensi 400 hari) |

Semua masih kecil untuk PostgreSQL di satu server. Jika kelak dibutuhkan, kandidat partisi pertama
adalah `audit_logs` dan `content_views_daily` (range per bulan).
