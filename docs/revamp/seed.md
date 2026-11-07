# Grafikarsa Revamp — Data Awal (Seed)

> Isi awal master data. Saat implementasi menjadi `db/seeds/*.sql` yang idempoten
> (`INSERT ... ON CONFLICT DO UPDATE`), dijalankan `make seed`. Semua daftar bisa diubah Admin setelah live.

---

## 1. Jurusan

Kode final **mengikuti kode jurusan di data v1/sekolah** saat migrasi supaya pemetaan tidak salah. Usulan awal:

| `code` (URL) | `name` | `short_name` (nama kelas) |
|---|---|---|
| `pg` | Teknik Grafika — Produksi Grafika | PG |
| `dg` | Teknik Grafika — Desain Grafika | DG |
| `dkv` | Desain Komunikasi Visual | DKV |
| `animasi` | Animasi | ANI |
| `rpl` | Pengembangan Perangkat Lunak dan Gim (RPL) | RPL |
| `tkj` | Teknik Komputer dan Jaringan | TKJ |
| `mekatronika` | Teknik Mekatronika | MKT |
| `logistik` | Teknik Logistik | LOG |
| `perhotelan` | Perhotelan | PH |

Nama kelas otomatis: `{X|XI|XII} {short_name} {rombel}` → "XI RPL B", slug `xi-rpl-b`.

## 2. Tahun ajaran

Tahun ajaran berjalan diambil dari data sekolah saat go-live (mis. 2026/2027, `starts_on` sesuai kalender
pendidikan Jawa Timur). Tahun ajaran lama dari v1 ikut dimigrasi sebagai riwayat.

---

## 3. Izin (permission catalog)

| Key | Scopable | Keterangan |
|---|---|---|
| `content.create` | – | Membuat portfolio/blog & diundang kolaborasi |
| `content.moderate` | ✅ | Meninjau antrian moderasi |
| `content.take_down` | ✅ | Menurunkan konten tayang |
| `comment.moderate` | ✅ | Menyembunyikan/memulihkan komentar |
| `profile.moderate` | ✅ | Reset avatar/banner/bio/headline, hapus experience |
| `report.handle` | ✅ | Memproses laporan |
| `moderation.stats` | ✅ | Melihat statistik moderasi |
| `appeal.decide` | – | Memutus banding |
| `group.manage` | ✅ | Mengubah tampilan grup (di luar pengelola) |
| `group.assign_manager` | ✅ | Menunjuk/mencabut pengelola grup |
| `user.view` | ✅ | Melihat data akun di admin |
| `user.create` | ✅ | Membuat akun |
| `user.update` | ✅ | Mengubah data akun |
| `user.reset_password` | ✅ | Sandi sementara, terbitkan ulang sandi awal, kartu akun |
| `user.deactivate` | ✅ | Nonaktif / aktifkan akun |
| `user.block_content` | – | Mencabut `content.create` |
| `user.anonymize` | – | Menghapus (menganonimkan) akun |
| `role.manage` | – | Membuat/mengubah role |
| `role.assign` | ✅ | Memberi role (hanya subset izin milik sendiri) |
| `academic.manage_majors` | – | |
| `academic.manage_years` | – | |
| `academic.manage_classes` | ✅ | |
| `academic.import` | ✅ | Impor CSV |
| `promotion.prepare` | ✅ | Mengisi pengecualian, dry-run |
| `promotion.approve` | – | Menyetujui, eksekusi manual, batalkan |
| `promotion.rollback` | – | |
| `badge.manage` | – | Katalog badge |
| `badge.award` | – | Memberi/mencabut badge sekolah |
| `taxonomy.manage` | – | Kategori, skill, tools, tag, alasan moderasi, filter kata, username terlarang |
| `settings.manage` | – | |
| `audit.view` | ✅ | |
| `dashboard.view` | ✅ | |
| `curation.manage` | – | Fase 2 |
| `assignment.manage` | – | Fase 2 |
| `assessment.score` | – | Fase 2 |

## 4. Role sistem

| Role (`slug`) | Diberikan ke | Izin |
|---|---|---|
| `member` | Semua akun aktif (otomatis) | `content.create` |
| `teacher` | Semua akun guru (otomatis) | `curation.manage`, `assignment.manage`, `assessment.score` (fase 2) |
| `moderator` | Ditunjuk (scope jurusan atau global) | `content.moderate`, `content.take_down`, `comment.moderate`, `profile.moderate`, `report.handle`, `moderation.stats` |
| `major_admin` | Pengelola jurusan / Kaprog (scope jurusan wajib) | Semua izin `moderator` + `group.manage`, `group.assign_manager`, `user.view/create/update/reset_password/deactivate`, `role.assign` (hanya `moderator` di jurusannya), `academic.manage_classes`, `academic.import`, `promotion.prepare`, `audit.view`, `dashboard.view` |
| `admin` | Admin sekolah (scope global) | Semua izin |

**Tanggung jawab moderasi** ada di **Admin** dan **Admin Jurusan** (`major_admin`). Role `moderator` tersedia
untuk menambah tenaga (mis. guru per jurusan) tanpa memberi akses admin.

Akun admin pertama dibuat lewat CLI (`make seed-admin`), sandi awal ditampilkan sekali di terminal.

---

## 5. Setting default (`settings`)

| Key | Default |
|---|---|
| `collab.max_authors_portfolio` | 50 |
| `collab.max_authors_blog` | 10 |
| `collab.invite_ttl_days` | 14 |
| `content.max_active_drafts` | 20 |
| `content.max_in_review` | 5 |
| `content.trash_retention_days` | 30 |
| `profile.max_pinned` | 3 |
| `profile.max_links` | 10 |
| `profile.max_skills` | 20 |
| `profile.username_change_days` | 30 |
| `profile.username_redirect_days` | 90 |
| `moderation.claim_ttl_minutes` | 30 |
| `moderation.sla_hours` | 48 |
| `auth.initial_password_ttl_days` | 30 |
| `auth.temporary_password_ttl_hours` | 72 |
| `auth.recovery_code_count` | 8 |
| `auth.session_idle_days` | 7 |
| `auth.session_absolute_days` | 30 |
| `promotion.reminder_days` | [30, 7] |
| `promotion.rollback_days` | 14 |
| `media.max_image_mb` | 10 |
| `media.max_pdf_mb` | 25 |
| `media.max_model3d_mb` | 30 |
| `media.daily_upload_mb` | 1024 |
| `media.orphan_days` | 7 |
| `comment.edit_window_minutes` | 15 |
| `algo.*` | Bobot & ambang di `algorithms.md` |
| `ratelimit.*` | Batas di `api.md` §16 |

---

## 6. Kategori konten

### 6.1 Portfolio

| `slug` | Nama | Jurusan yang disarankan |
|---|---|---|
| `cetak-kemasan` | Cetak & Kemasan | PG, DG |
| `pracetak-layout` | Pracetak & Layout | DG, PG |
| `branding-identitas` | Branding & Identitas Visual | DG, DKV |
| `poster-publikasi` | Poster & Publikasi | DG, DKV |
| `ilustrasi` | Ilustrasi | DKV, Animasi, DG |
| `fotografi` | Fotografi | DKV, Animasi |
| `ui-ux` | UI/UX Design | DKV, RPL |
| `video-film` | Video & Film | DKV, Animasi |
| `animasi-2d` | Animasi 2D | Animasi |
| `animasi-3d` | Animasi & Modeling 3D | Animasi |
| `motion-graphic` | Motion Graphic | Animasi, DKV |
| `web-app` | Aplikasi Web | RPL |
| `mobile-app` | Aplikasi Mobile | RPL |
| `game` | Game | RPL, Animasi |
| `jaringan-infrastruktur` | Jaringan & Infrastruktur | TKJ |
| `keamanan-siber` | Keamanan Siber | TKJ, RPL |
| `iot-embedded` | IoT & Embedded | Mekatronika, TKJ |
| `otomasi-robotik` | Otomasi & Robotik | Mekatronika |
| `logistik-rantai-pasok` | Logistik & Rantai Pasok | Logistik |
| `layanan-perhotelan` | Layanan Perhotelan | Perhotelan |
| `tata-boga-minuman` | Tata Hidang & Minuman | Perhotelan |
| `lainnya` | Lainnya | Semua |

### 6.2 Blog

| `slug` | Nama |
|---|---|
| `tutorial` | Tutorial |
| `cerita-pengalaman` | Cerita & Pengalaman (PKL, lomba, kuliah) |
| `opini` | Opini |
| `review` | Review |
| `tips` | Tips |
| `lainnya` | Lainnya |

### 6.3 Konteks portfolio (`meta.context`)

Tugas Sekolah · PKL/Magang · Lomba · UKK · Proyek Pribadi · Lainnya.

---

## 7. Skill (awal, ±60)

| Kategori | Skill |
|---|---|
| `design` | Desain Grafis, Tipografi, Layout, Branding, Ilustrasi Digital, UI Design, UX Research, Desain Kemasan |
| `printing` | Pracetak, Separasi Warna, Cetak Offset, Cetak Digital, Sablon, Finishing Cetak, Manajemen Warna |
| `media` | Fotografi, Videografi, Editing Video, Animasi 2D, Animasi 3D, Modeling 3D, Rigging, Motion Graphic, Storyboard, Sound Design, VFX |
| `software` | Frontend, Backend, Mobile Development, Game Development, Basis Data, API Design, DevOps, Testing |
| `network` | Administrasi Jaringan, Administrasi Server, Keamanan Jaringan, Cloud Computing, Troubleshooting Hardware |
| `engineering` | PLC, Pneumatik & Hidrolik, Mikrokontroler, Elektronika, CAD Mekanik, Robotik |
| `logistics` | Manajemen Gudang, Inventory Control, Distribusi, Administrasi Logistik |
| `hospitality` | Front Office, Housekeeping, Food & Beverage Service, Barista, Pelayanan Tamu |
| `soft_skill` | Public Speaking, Kepemimpinan, Kerja Tim, Manajemen Proyek, Bahasa Inggris, Copywriting |

## 8. Tools (awal, ±50)

| Kategori | Tools |
|---|---|
| `design_software` | Adobe Photoshop, Adobe Illustrator, Adobe InDesign, CorelDRAW, Figma, Canva, Affinity Designer, Procreate, Krita |
| `media_software` | Adobe Premiere Pro, Adobe After Effects, DaVinci Resolve, CapCut, Blender, Autodesk Maya, Toon Boom Harmony, Clip Studio Paint, Audacity |
| `language` | HTML, CSS, JavaScript, TypeScript, PHP, Python, Go, Java, Kotlin, Dart, C++, C#, SQL |
| `framework` | React, Next.js, Vue, Laravel, Flutter, Express, Tailwind CSS, Unity, Godot |
| `platform` | Linux, Windows Server, MikroTik, Cisco Packet Tracer, Docker, Git, Arduino IDE |
| `hardware` | Arduino, ESP32, Raspberry Pi, Router/Switch |
| `machine` | Mesin Offset, Printer Digital, Mesin Potong, PLC Siemens, PLC Omron |

---

## 9. Badge

### 9.1 Sistem (otomatis, lihat `algorithms.md` §10)

Karya Pertama · Tulisan Pertama · Produktif · Kolaborator · Karya Pilihan · Diapresiasi.

### 9.2 Sekolah (diberikan Admin)

| `slug` | Nama | Repeatable |
|---|---|---|
| `juara-lks` | Juara LKS | ✅ (`dedupe_key` = bidang + tahun) |
| `juara-lomba` | Juara Lomba | ✅ |
| `lolos-snbp` | Lolos SNBP | – |
| `lolos-snbt` | Lolos SNBT | – |
| `sertifikasi-lsp` | Bersertifikat LSP | ✅ |
| `pengurus-osis` | Pengurus OSIS/MPK | ✅ (per periode, dengan `expires_at`) |
| `karya-gkp` | Peserta Gelar Karya Pembelajaran | ✅ (per tahun) |

---

## 10. Alasan moderasi

| `slug` | Label | Berlaku untuk |
|---|---|---|
| `tidak-pantas` | Konten tidak pantas (kekerasan, seksual, kata kasar) | konten, komentar, profil, grup, laporan |
| `sara-ujaran-kebencian` | SARA / ujaran kebencian | semua |
| `perundungan` | Perundungan / pelecehan | komentar, profil, laporan |
| `data-pribadi` | Membuka data pribadi orang lain | semua |
| `plagiarisme` | Plagiarisme / karya orang lain tanpa kredit | konten, laporan |
| `hak-cipta` | Pelanggaran hak cipta (aset, musik, font berbayar) | konten, laporan |
| `spam` | Spam / promosi | konten, komentar, laporan |
| `tipe-tidak-sesuai` | Tidak sesuai tipe (tulisan sebagai portfolio, dll) | konten |
| `kualitas-kelengkapan` | Belum lengkap / kualitas belum layak tayang | konten |
| `informasi-menyesatkan` | Informasi menyesatkan | konten, laporan |
| `lainnya` | Lainnya (wajib catatan) | semua |

## 11. Username terlarang

- **Route:** `admin`, `api`, `studio`, `karya`, `blog`, `jurusan`, `kelas`, `cari`, `masuk`, `aktivasi`,
  `onboarding`, `lupa-sandi`, `ganti-sandi`, `pengaturan`, `moderasi`, `notifikasi`, `tersimpan`, `kurasi`,
  `penugasan`, `pedoman`, `tentang`, `privasi`, `syarat`, `media`, `internal`, `static`, `_next`.
- **Resmi:** `grafikarsa`, `grafika`, `smkn4`, `smkn4malang`, `smkn4mlg`, `official`, `resmi`, `admin`,
  `administrator`, `moderator`, `support`, `bantuan`, `sekolah`, `osis`, `humas`.
- **Kata kasar:** dari daftar filter kata (dikelola Admin, tidak disimpan di repo publik).

## 12. Filter kata

Daftar awal disusun bersama sekolah dan diimpor lewat Admin (`/admin/taksonomi`), **tidak di-commit ke repo**.
Dua mode: `block` (simpan ditolak dengan pesan sopan) dan `flag` (tersimpan, otomatis jadi laporan).

---

## 13. Pedoman komunitas (`/pedoman`) — kerangka isi

1. **Grafikarsa untuk warga SMKN 4 Malang**: semua karya dan tulisan publik, jadi tampilkan yang kamu
   bangga orang lain melihatnya.
2. **Karya sendiri, kredit yang jujur**: cantumkan kolaborator, sumber aset, dan referensi. Plagiarisme ditolak.
3. **Hormati hak cipta**: gunakan aset berlisensi atau buatan sendiri.
4. **Aman untuk semua umur**: tanpa konten seksual, kekerasan, kata kasar, SARA, atau perundungan.
5. **Jaga data pribadi**: jangan membagikan alamat, nomor HP, atau data orang lain.
6. **Komentar yang membangun**: kritik karya, bukan orangnya.
7. **Semua konten dimoderasi**: penjelasan alur (draf → ditinjau → tayang / revisi / ditolak) dan target waktu tinjau.
8. **Laporkan, jangan balas**: cara melaporkan dan apa yang terjadi setelahnya.
9. **Konsekuensi**: komentar disembunyikan, konten diturunkan, hak membuat konten dicabut, akun dinonaktifkan.
10. **Banding**: satu kali per penolakan, diputus Admin.

## 14. Preferensi notifikasi default

Semua tipe aktif. Yang bisa dimatikan user: `appreciation`, `new_follower`, `comment_on_content`,
`comment_reply`, `comment_mention`, `badge_awarded`. Tipe lain (kolaborasi, moderasi, keamanan) selalu aktif.
