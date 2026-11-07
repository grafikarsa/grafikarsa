# Grafikarsa Revamp — Migrasi dari v1

> Sumber: **v1 yang berjalan di produksi** (`grafikarsa.com`). v2 (`feat/grafikarsa-v2-rebuild`) tidak pernah
> di-deploy, jadi tidak ikut dimigrasi. Skema v1 diambil dari `apps/backend/internal/domain/models.go`
> (branch kode v1) dan hasil inspeksi produksi di PRD v2.

---

## 1. Gambaran data v1 (per inspeksi terakhir)

| Data | Jumlah | Catatan |
|---|---|---|
| Users | ±1.687 (1.686 siswa + 1 admin) | Punya `username`, `email`, `nisn`/`nis`/`nip` (opsional), `kelas_id`, `tahun_masuk/lulus`, `role` (`student`/`alumni`/`teacher`/`admin`), `special_roles` |
| Portfolio | ±1.705 | published 861 · draft 502 · pending_review 252 · rejected 74 · archived 16 |
| Content blocks | ±6.903 | text 3.603 · image 2.251 · youtube 931 · sisanya pdf/button/figma/doc/table/canva/ppt/embed |
| Media (MinIO) | ±5,1 GB | URL absolut `https://grafikarsa.com/storage/grafikarsa/...` tersebar di kolom & payload |
| Sosial | likes, follows, comments (berbalas), notifications, portfolio_views | |
| Lainnya | tags, series, assessment, feedback, changelog, special roles, app settings, AI ideas | |

**Angka harus diperbarui** dengan menjalankan `migrate-v1 inspect` pada dump terbaru sebelum migrasi.

---

## 2. Prinsip

1. **Tidak ada data yang hilang diam-diam.** Yang tidak bisa dipetakan masuk `legacy_quarantine` + laporan.
2. **Idempoten & bisa diulang.** Semua baris tercatat di `legacy_id_map`. Menjalankan ulang tidak membuat duplikat.
3. **Dry-run dulu.** Mode `--dry-run` menjalankan semua transformasi dan validasi tanpa menulis, menghasilkan laporan.
4. **URL lama tidak mati.** Semua URL publik v1 di-redirect 301 ke URL baru.
5. **v1 tidak dihapus** sampai 30 hari setelah go-live (jalur rollback).
6. **Tidak ada moderasi ulang massal** untuk konten yang sudah tayang di v1 (sudah lolos moderasi v1).

---

## 3. Keputusan pemetaan

### 3.1 Akun

| v1 | Revamp | Aturan |
|---|---|---|
| `role = student`, masih bersekolah | `account_type = student`, `status = active` | Dicocokkan ke data siswa resmi sekolah (§4) |
| `role = alumni` atau `student` yang sudah lulus | `student`, `status = alumni` | `graduation_year` dari v1 atau `tahun_masuk + 3` |
| `role = teacher` | `teacher` | |
| `role = admin` | `staff` + role `admin` | Ditinjau manual |
| `special_roles` | Role revamp | Pemetaan manual oleh Admin (daftar ditampilkan di laporan dry-run). Default: **tidak** dibawa |
| `username` | `username` (dinormalisasi) | Huruf kecil, karakter tidak valid diganti. Bentrok dengan reserved/username lain → `_v1` + laporan |
| `email`, `phone`, `address`, `show_*` | **Tidak dibawa** | Revamp tidak memakai email; data sensitif tidak dibutuhkan (minimisasi data) |
| `password_hash` (bcrypt) | `password_hash`, `password_kind = user` | User v1 **bisa login dengan username + sandi lamanya**. Rehash ke Argon2id saat login |
| `bio`, `avatar_url`, `banner_url` | `profiles` | URL → `media_assets` lewat peta media |
| `social_links` | `profile_links` | Platform dipetakan (`twitter` → `x`, `personal_website` → `website`). Non-https dibuang + laporan |
| `refresh_tokens`, `token_blacklist` | **Tidak dibawa** | Semua sesi v1 berakhir |

**Alur login user v1 setelah migrasi:**

```
/masuk (username + sandi v1)
  → login sukses, akun ditandai "migrasi, belum selesai aktivasi" (activated_at NULL, password_kind user)
  → /aktivasi versi singkat: username sudah terisi (boleh diganti sekali), sandi tidak wajib diganti,
    WAJIB simpan 8 recovery code
  → /
```

Siswa yang cocok dengan data sekolah juga bisa login memakai NISN (+ sandi v1). Siswa yang belum pernah punya
akun v1 menerima kartu akun seperti biasa.

### 3.2 Akademik

| v1 | Revamp | Aturan |
|---|---|---|
| `jurusan` (`kode`, `nama`) | `majors` | Dipetakan ke 9 jurusan seed (seed.md §1). Kode v1 dipakai jika sesuai |
| `tahun_ajaran` | `academic_years` | `starts_on`/`ends_on` dari kalender (default 1 Juli – 30 Juni bila tidak ada) |
| `kelas` | `classes` (+ grup kelas) | `tingkat` 10–12, `rombel` jadi `section` |
| `student_class_history` | `enrollments` (`source = legacy`) | Satu baris per siswa per tahun ajaran. Duplikat dalam satu tahun → ambil yang `is_current`/terbaru |
| `users.kelas_id` tanpa riwayat | `enrollments` tahun berjalan | |

### 3.3 Portfolio

| v1 | Revamp | Aturan |
|---|---|---|
| `portfolios` | `contents` (`type = portfolio`), `legacy_v1_id` | `public_id` baru, slug dari v1 (dinormalisasi) |
| `user_id` | Pemilik (`content_authors` level `owner`) | v1 tidak punya kolaborasi |
| `judul` | `title` | > 150 karakter dipotong + laporan |
| `thumbnail_url` | `cover_media_id` | Tidak ada thumbnail → gambar pertama dari blok. Tidak ada gambar sama sekali → lihat aturan status di bawah |
| `content_blocks` | `body.blocks` | Pemetaan blok di editor-blocks.md §7 |
| `tags` | `content_tags` | Tag v1 → `tags` (slug dinormalisasi, digabung bila sama) |
| `series_id` | Tidak dibawa sebagai struktur | Nama series dijadikan tag (`series-<nama>`) agar jejaknya tidak hilang |
| Kategori, konteks | Kategori `lainnya`, konteks `other` | v1 tidak punya. Pemilik bisa memperbaiki lewat revisi |

**Pemetaan status:**

| Status v1 | Status revamp | Versi yang dibuat |
|---|---|---|
| `published` | `published` | 1 versi `approved` (`kind = legacy`), menjadi `live_version_id`. `first_published_at` = `published_at` v1 |
| `archived` | `archived` | 1 versi `approved` (`kind = legacy`) |
| `pending_review` | `draft` | Tidak ada versi. Draft diisi isi v1. **Notifikasi ke pemilik**: "Portfolio dari Grafikarsa lama perlu dikirim ulang ke moderasi" |
| `draft` | `draft` | Tidak ada versi |
| `rejected` | `draft` | Tidak ada versi. Catatan `admin_review_note` v1 ditampilkan di Studio sebagai info |

Alasan `pending_review` → `draft`: antrian v1 (±252) tidak tahu aturan revamp (kategori wajib, cover wajib).
Lebih adil pemilik melengkapi dulu lalu mengirim ulang, daripada moderator menolak massal.

**Portfolio tayang yang tidak memenuhi aturan revamp** (tanpa cover, tanpa blok media): tetap `published`
(sudah disetujui di v1), diberi penanda `meta.legacy_incomplete = true`. Saat pemilik membuat revisi,
aturan baru berlaku penuh.

### 3.4 Sosial & lainnya

| v1 | Revamp | Aturan |
|---|---|---|
| `follows` | `follows` | Abaikan follow ke diri sendiri / user yang tidak dimigrasi |
| `portfolio_likes` | `appreciations` | |
| `comments` (`parent_id`) | `comments` | Balasan > 1 level diratakan ke komentar akar. Teks > 1.000 karakter dipotong + laporan. `is_edited` → `edited_at` |
| `portfolio_views` | `content_views_daily` + `content_stats` | Diagregasi per hari (unique per user/session) |
| `notifications` | **Tidak dibawa** | Format & tipe berbeda. User mulai dengan kotak notifikasi kosong |
| `portfolio_assessments` (+ scores, metrics) | **Diarsipkan** ke `legacy_quarantine` (`reason = archived_assessment`) | Model penilaian revamp (fase 2) berbasis penugasan. Data v1 tetap bisa diekspor CSV untuk guru |
| `feedback`, `changelogs` | **Tidak dibawa** (diekspor ke arsip JSON) | Fitur tidak ada di revamp |
| `user_interests`, `user_feed_preferences` | Tidak dibawa | Algoritma baru |
| `ai ideas` | Tidak dibawa | Fitur tidak ada di revamp |
| `app_settings` | Tidak dibawa | Setting baru |

### 3.5 Media

```
1. rclone sync MinIO v1 (bucket grafikarsa) → RustFS gk-private/legacy/...  (layout key dipertahankan)
2. Untuk setiap URL /storage/... yang dirujuk DB v1 (kolom + isi payload JSON):
     - cari object → insert media_assets (original_key = legacy/..., checksum)
     - object tidak ada → karantina (blok tetap dibuat tanpa media, ditandai)
3. Worker memproses semua media legacy: hapus EXIF, buat varian, blurhash
4. Media milik konten published/archived + avatar/banner → disalin ke gk-public
5. legacy_redirects: /storage/grafikarsa/<key> → /media/<public-key> (untuk link lama yang tersebar di luar)
```

File di MinIO yang **tidak dirujuk** oleh data mana pun tidak dimigrasi (dilaporkan jumlah & ukurannya).

---

## 4. Pencocokan akun v1 dengan data sekolah

Kunci pencocokan (berurutan, berhenti di yang pertama cocok):

1. `nisn` v1 = NISN data sekolah.
2. `nis` v1 = NIS data sekolah.
3. Nama ternormalisasi + jurusan + angkatan sama, **dan hanya satu kandidat**.

Hasil:

| Hasil | Tindakan |
|---|---|
| Cocok pasti (1 atau 2) | Akun v1 dilengkapi NISN + enrollment resmi |
| Cocok nama (3) | Masuk daftar **verifikasi manual** Admin Jurusan sebelum digabung |
| Tidak cocok | Akun tetap dimigrasi dengan status dari v1 (alumni/aktif), ditandai `needs_review` di admin |
| Data sekolah tanpa akun v1 | Akun baru dibuat dari impor, dapat kartu akun |
| Dua akun v1 untuk satu siswa | Laporan duplikat → Admin memilih akun utama, konten akun lain dipindah ke akun utama |

---

## 5. URL lama → baru (301)

| URL v1 | URL revamp |
|---|---|
| `/{username}` | `/@{username}` |
| `/{username}/{slug}` | `/karya/{slug}-{publicId}` (dari `legacy_id_map`) |
| `/{username}/{slug}` untuk portfolio yang tidak tayang | `/@{username}` |
| `/portfolios` | `/karya` |
| `/users` | `/cari?tipe=user` |
| `/notifications` | `/notifikasi` |
| `/login` | `/masuk` |
| `/settings`, `/{username}/edit*` | `/pengaturan/profil` |
| `/changelog`, `/ai-ideas` | `/` |
| `/admin/*` | `/admin` |
| `/storage/grafikarsa/*` | `/media/*` (tabel `legacy_redirects`) |

Dilayani middleware Next.js dengan lookup ke `legacy_redirects` (di-cache Valkey). Karena revamp memakai
prefix `@` untuk profil dan path tingkat atas yang sudah dipesan, `/{segmen}` yang bukan route revamp
diperlakukan sebagai username v1. Redirect dipertahankan **minimal 12 bulan**, dipantau lewat `hit_count`.

---

## 6. Tool & langkah

Tool: `apps/api/cmd/migrate-v1` (Go), memakai skema adaptif seperti tool v2 (`cmd/tools/migrate-old`),
yang logikanya bisa dipakai ulang.

```
migrate-v1 inspect   --v1 <dsn>                 # hitung & profil data, cek kualitas
migrate-v1 run       --v1 <dsn> --dry-run       # semua transformasi, laporan, tanpa menulis
migrate-v1 run       --v1 <dsn> --execute       # tulis (idempoten)
migrate-v1 media     --execute                  # daftarkan media legacy + enqueue proses
migrate-v1 verify                               # validasi pasca-migrasi (§7)
migrate-v1 report    --out report.html          # laporan karantina, duplikat, verifikasi manual
```

### 6.1 Urutan go-live

| Kapan | Langkah |
|---|---|
| H-30 | Dump v1 → staging. `inspect` + `run --dry-run`. Perbaiki pemetaan sampai karantina minimal |
| H-21 | Impor data siswa resmi sekolah ke staging, jalankan pencocokan (§4), Admin Jurusan memverifikasi daftar manual |
| H-14 | Uji penuh di staging: login akun v1 sampel, buka 50 portfolio acak, redirect URL lama, load test |
| H-7 | Pengumuman di v1 (banner): tanggal pindah, "simpan sandimu", portfolio pending harus dikirim ulang nanti |
| H-1 | `rclone sync` media awal (bulk, tanpa downtime) |
| **H (malam, di luar jam sekolah)** | v1 mode maintenance (read-only) → dump final → `rclone sync` selisih media → `run --execute` → `media --execute` → `verify` → DNS/Caddy pindah ke revamp → smoke test |
| H+1 … H+7 | Pantau error, `hit_count` redirect, login gagal user v1. Admin membantu reset sandi |
| H+30 | Matikan v1, simpan dump & bucket v1 sebagai arsip offline |

Perkiraan jendela maintenance: **2–3 jam** (dump & migrasi DB < 15 menit, sisanya sinkron media + verifikasi).

### 6.2 Rollback

Jika `verify` gagal atau ada masalah kritis dalam 24 jam pertama: arahkan Caddy/DNS kembali ke v1
(yang hanya read-only, jadi tidak ada data baru yang hilang), matikan maintenance v1, perbaiki, ulangi
di jadwal berikutnya. Setelah 24 jam (data baru sudah masuk di revamp), rollback tidak lagi dilakukan;
perbaikan dilakukan maju.

---

## 7. Verifikasi pasca-migrasi (`migrate-v1 verify`, wajib lolos)

- Jumlah: users, portfolio per status, blok, like, follow, komentar = v1 − karantina (karantina tercatat & dijelaskan).
- Semua `contents.status = published` punya `live_version_id` dan cover (atau penanda `legacy_incomplete`).
- Semua media yang dirujuk konten tayang punya varian `ready` dan `is_public = true`.
- 100 file acak: checksum SHA-256 sama dengan MinIO.
- 100 URL v1 acak (`/{username}/{slug}`, `/storage/...`) → 301 ke halaman yang benar (200).
- Login 10 akun v1 sampel (siswa aktif, alumni, guru, admin) dengan sandi lama berhasil.
- Feed Terbaru, Katalog, profil, grup jurusan menampilkan data.
- `group_contents` terisi untuk semua konten tayang.
