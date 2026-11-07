# Grafikarsa Revamp — PRD (Product Requirements Document)

> Status: **draft keputusan produk** · Branch: `revamp` · Dibangun ulang dari nol.
> Dokumen ini fokus ke *apa* dan *aturan main*. Tech stack dan arsitektur ditulis terpisah.

---

## 0. Ringkasan

Grafikarsa adalah platform **jejaring sosial + katalog portfolio + blogging** khusus warga
**SMKN 4 Malang ("Grafika")**. Setiap warga sekolah punya halaman profil, bisa memamerkan karya
(portfolio) dan tulisan (blog), berkolaborasi lintas jurusan, dan saling mengapresiasi.

Prinsip inti:

1. **Semua publik.** Tidak ada akun privat, tidak ada pengaturan privasi per konten.
   Publik (tanpa login) hanya bisa melihat, membaca, dan membagikan.
2. **Interaksi hanya untuk warga.** React, komentar, follow, simpan, dan lapor butuh akun Grafikarsa.
3. **Semua konten selalu lewat moderasi.** Setiap terbit dan setiap perubahan isi konten yang
   sudah terbit. Tidak ada auto-pass, tidak ada trust level.
4. **Konten itu umum.** Portfolio ya portfolio, blog ya blog. Tidak ada "blog guru" atau
   "blog alumni". Siapa pun yang punya capability bisa membuat keduanya.
5. **Kolaborasi multi-author** lintas user, jurusan, dan role.

### 0.1 Konteks sekolah (hasil riset, Nov 2026)

| | |
|---|---|
| Sekolah | SMK Negeri 4 Malang (brand "Grafika"), Jl. Tanimbar 22, Kasin, Klojen, Malang · NPSN 20533816 |
| Skala | ±3.300 siswa (±1.100 per angkatan), ±117 guru |
| Sejarah | Berakar dari sekolah percetakan 1938 → SMT Grafika (1959) → SMKN 4 Malang (1996) |
| Konsentrasi keahlian (9) | Produksi Grafika, Desain Grafika, DKV/Multimedia, Animasi, PPLG/RPL, TKJ, Mekatronika, Logistik, Perhotelan |
| Implikasi produk | Jenis karya sangat beragam (cetak, desain, video, kode, jaringan, mesin, jasa), jadi portfolio harus mendukung banyak tipe media, bukan sekadar galeri gambar |
| Aplikasi sekolah yang sudah ada | Rapor, Kelulusan, Inventaris, Ujian Online, SIPRAKERIN, Alufika, situs OSIS. Grafikarsa **tidak** menduplikasi fungsi ini |

### 0.2 Non-goals (sengaja dikeluarkan)

- Akun **mitra industri** dan **orang tua**.
- **DM / chat**.
- **Aspirasi**, **acara/event**.
- Lowongan PKL/kerja (BKK), direktori alumni & tracer study, mentoring alumni.
- Kanal redaksi resmi / majalah digital.
- Tipe konten khusus per role (blog guru, blog alumni, dll).
- Pengaturan privasi akun atau konten.
- Grup selain **jurusan** dan **kelas** (organisasi/ekskul tidak dibuat sebagai grup).
- Postingan khusus di grup.

---

## 1. Akun & Role

### 1.1 Tipe akun (identitas, diverifikasi sekolah)

Tidak ada pendaftaran bebas. Akun dibuat dari data sekolah (impor/admin).

| Tipe | Keterangan |
|---|---|
| **Siswa** | Terikat ke kelas (rombel), jurusan, angkatan, tahun ajaran |
| **Alumni** | Otomatis dari Siswa saat proses kelulusan. Bukan role, tapi status |
| **Guru** | Bisa ditandai mengampu jurusan/mapel |
| **Tendik/Staf** | TU, BK, pustakawan, laboran, humas, dll |

Status akun: `aktif`, `alumni`, `mutasi`, `nonaktif`.

### 1.2 Role platform (izin)

| Role | Wewenang |
|---|---|
| **Member** | Default untuk semua tipe akun. Memegang capability `content.create` (bisa dicabut Admin per user) |
| **Moderator** | Scope `global` atau `jurusan`. Memproses antrian konten dan laporan |
| **Admin Jurusan** (Kaprog) | Kelola user & kelas di jurusannya, moderasi jurusannya, kelola grup di jurusannya |
| **Admin** | Akses penuh |

Role bisa diberi `expires_at` opsional. Semua pemberian/pencabutan role masuk audit log.

### 1.3 Matriks aksi publik vs member

| Aksi | Publik | Member |
|---|---|---|
| Lihat profil, grup, portfolio, blog, komentar | ✅ | ✅ |
| Cari & jelajah katalog | ✅ | ✅ |
| Share (salin link / share native) | ✅ | ✅ |
| Apresiasi (react), komentar, balas | ❌ | ✅ |
| Follow, simpan ke koleksi, endorse skill, lapor | ❌ | ✅ |
| Buat konten & jadi kolaborator | ❌ | ✅ (butuh `content.create`) |

- Hitungan view/share tetap dicatat dari publik (analitik, bukan perubahan konten).
- **Data identitas tidak pernah tampil publik**: email, NISN, nomor HP, tanggal lahir, alamat.
- Draf bukan "privat", hanya belum terbit, jadi hanya terlihat oleh author-nya.

---

## 2. Konten: Portfolio vs Blog

### 2.1 Definisi

- **Portfolio = karya.** "Ini yang aku/kami **buat**." Fokus pada hasil/artefak.
- **Blog = tulisan.** "Ini yang aku/kami **pikirkan, ketahui, atau alami**." Fokus pada narasi.

**Tes cepat:** hapus semua teksnya. Kalau masih ada yang bisa dinikmati, itu portfolio.
Hapus semua gambarnya. Kalau tulisannya masih utuh dan bisa dibaca, itu blog.

### 2.2 Perbedaan struktural

| | **Portfolio** | **Blog** |
|---|---|---|
| Editor | **Block-based layout** (susun blok seperti kanvas) | **Dokumen rich-text** satu alur |
| Blok yang diizinkan | Gambar, galeri, video, embed (YouTube, Figma, Canva, Sketchfab, GitHub), file PDF, model 3D, teks pendek | Heading, paragraf, list, kutipan, kode, callout, gambar inline, embed, pembatas, kartu portfolio |
| Wajib diisi | Judul, thumbnail/cover, ≥1 blok media, kategori karya, konteks | Judul, excerpt, isi ≥150 kata, kategori tulisan |
| Metadata khusus | Kredit/peran per kolaborator, tools/software, skill, periode pengerjaan, konteks (Tugas, PKL, Lomba, UKK, Pribadi), link eksternal (demo/repo/Behance) | Waktu baca, daftar isi otomatis, tag, seri artikel |
| Kategori | Per bidang: cetak, branding, ilustrasi, animasi 2D/3D, film, web app, mobile app, jaringan, robotik/otomasi, layanan hotel, logistik, dst. (dikelola Admin) | Tutorial, Cerita/Pengalaman, Opini, Review, Tips (dikelola Admin) |
| Kartu | Grid visual, thumbnail dominan | List: judul, excerpt, waktu baca, cover opsional |
| Halaman detail | Hero media → kredit → blok → tools/skill | Judul → author → artikel → daftar isi samping |
| Tempat muncul | Tab Portfolio di profil, **Katalog Karya**, grup, bisa ditautkan ke experience, bisa diekspor ke CV PDF | Tab Blog di profil, halaman **Blog**, grup |
| Penilaian guru | Bisa (lewat penugasan) | Tidak |

### 2.3 Yang dibagi bersama

Lifecycle, moderasi, kolaborasi, apresiasi, komentar, simpan, share, laporan, slug/URL,
dan riwayat versi. Model data: satu entitas `content` dengan `type = portfolio | blog`, ditambah
tabel detail per tipe.

---

## 3. Kolaborasi Multi-Author

### 3.1 Batas

| | Default | Catatan |
|---|---|---|
| Portfolio | **50 author** (1 pemilik + 49) | Muat 1 rombel (±36) + guru + cadangan |
| Blog | **10 author** | |

Keduanya **configurable oleh Admin** (setting, bukan hardcode). Menurunkan batas tidak
memengaruhi konten yang sudah melebihi batas baru. Konten itu hanya tidak bisa menambah author lagi.

### 3.2 Level author

| Level | Edit isi | Kelola kolaborator | Submit / tarik review | Arsip / hapus | Tampil di profilnya |
|---|---|---|---|---|---|
| **Pemilik** (tepat 1) | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Editor** | ✅ | ❌ | ❌ | ❌ | ✅ |
| **Kontributor** | ❌ | ❌ | ❌ | ❌ | ✅ |

### 3.3 Aturan

1. Hanya akun **aktif** dengan `content.create` yang bisa diundang, dari user/jurusan/role mana pun.
2. Undangan **wajib diterima** sebelum nama tampil. Undangan kedaluwarsa setelah **14 hari**.
3. Kolaborator bisa **keluar** kapan saja. Pemilik harus **transfer kepemilikan** dulu sebelum keluar.
4. Perubahan daftar author pada konten `published` **tidak perlu moderasi ulang** (metadata,
   disetujui oleh yang diundang), tapi selalu masuk audit log.
5. Author yang akunnya dinonaktifkan tampil sebagai "Pengguna nonaktif". Jika pemilik
   dinonaktifkan, Admin dapat memindahkan kepemilikan.
6. Untuk portfolio, setiap author bisa diberi **label peran** (mis. "Animator", "Backend")
   dan urutan tampil diatur pemilik.
7. Tampilan: tumpukan avatar ("Rafa, Dinda, dan 47 lainnya") → daftar kredit lengkap.

---

## 4. Lifecycle Konten

Berlaku identik untuk portfolio dan blog.

### 4.1 Status

| Status | Terlihat oleh | Keterangan |
|---|---|---|
| `draft` | Author | Sedang dikerjakan, autosave |
| `in_review` | Author + moderator | **Terkunci**, tidak bisa diedit |
| `changes_requested` | Author + moderator | Moderator minta revisi dengan catatan |
| `published` | Semua orang | Tayang |
| `archived` | Author | Disembunyikan sendiri oleh pemilik |
| `taken_down` | Author + moderator | Diturunkan moderator setelah tayang |
| `rejected` | Author + moderator | **Final**, alasan wajib |
| `deleted` | — | Soft delete 30 hari, lalu purge permanen |

### 4.2 Transisi

| Dari → Ke | Oleh | Syarat / efek |
|---|---|---|
| (baru) → `draft` | Member dengan `content.create` | Batas draf aktif per user (default 20, configurable) |
| `draft` → `in_review` | Pemilik | Validasi field wajib. Sistem membuat **snapshot imutabel** versi yang disubmit. Batas `in_review` per user (default 5, configurable) |
| `in_review` → `draft` | Pemilik | Tarik kembali, hanya selama belum di-*claim* moderator |
| `in_review` → `published` | Moderator | Setujui snapshot. `published_at` diisi saat terbit pertama |
| `in_review` → `changes_requested` | Moderator | **Catatan wajib**, bisa ditandai per blok |
| `in_review` → `rejected` | Moderator | Kategori pelanggaran + alasan wajib |
| `changes_requested` → `in_review` | Pemilik | Setelah revisi, snapshot baru |
| `changes_requested` → `draft` | Pemilik | Opsional, untuk lanjut mengerjakan lebih lama |
| `published` → `archived` | Pemilik | Langsung hilang dari publik, profil, grup, katalog |
| `archived` → `published` | Pemilik | **Tanpa review ulang**: isinya identik dengan snapshot yang sudah disetujui (diverifikasi lewat hash snapshot) |
| `published` → `taken_down` | Moderator | Alasan wajib. Semua author dinotifikasi |
| `taken_down` → `in_review` | Pemilik | Setelah diperbaiki, wajib review lagi |
| `rejected` → — | — | **Final.** Tidak bisa disubmit ulang atau diedit. Yang masih bisa diperbaiki harus lewat `changes_requested`, bukan `rejected` |
| status apa pun kecuali `in_review` → `deleted` | Pemilik / Admin | Bisa dipulihkan dalam 30 hari |

Diagram:

```
            ┌───────── tarik kembali ─────────┐
            ▼                                  │
 (baru) → draft ──submit──► in_review ─────────┤
            ▲                 │  │  │          │
            │      minta revisi  │  └─tolak──► rejected (final)
            │                 ▼  │
            └──── changes_requested ◄┘ (resubmit → in_review)
                              │
                         setujui
                              ▼
           archived ◄──► published ──turunkan──► taken_down ──perbaiki──► in_review
```

### 4.3 Mengedit konten yang sudah terbit (revisi)

Konten `published` **tidak pernah diedit langsung**. Kalau bisa, aturan "selalu lewat moderasi"
jebol: submit versi aman, lolos, lalu isinya diganti.

1. "Edit" pada konten `published` membuat **revisi draf** terpisah. **Versi tayang tidak berubah.**
2. Revisi menjalani alur yang sama: `draft` → `in_review` → disetujui / `changes_requested` / ditolak.
3. Moderator melihat **diff** antara versi tayang dan revisi.
4. **Disetujui**: revisi menggantikan versi tayang. URL, komentar, apresiasi, `published_at`
   tetap. `updated_at` diperbarui, dan revisi **tidak** mengangkat konten ke atas urutan "terbaru".
5. **Ditolak**: revisi ditutup, versi tayang tetap utuh. Pemilik boleh membuat revisi baru
   (yang final adalah revisinya, bukan kontennya).
6. Maksimal **1 revisi terbuka** per konten. Semua versi yang pernah disetujui disimpan sebagai riwayat.
7. Yang **tidak** butuh revisi: daftar author/peran (lihat §3.3) dan urutan sematkan di profil.

### 4.4 Moderasi

- **Tidak ada auto-pass dan tidak ada trust level.** Semua submit dan semua revisi masuk antrian.
- Moderator **tidak boleh memoderasi konten yang ia ikut jadi author-nya**.
- Pembagian antrian: berdasarkan jurusan **pemilik**. Pemilik Guru/Tendik (tanpa jurusan) masuk
  antrian global. Moderator global melihat semua antrian.
- Antrian FIFO dengan **claim** (mencegah dua moderator meninjau item yang sama). Claim
  kedaluwarsa otomatis kalau tidak diproses (default 30 menit).
- Indikator item yang sudah terlalu lama menunggu.
- Kategori alasan (dikelola Admin): konten tidak pantas, plagiarisme, SARA, data pribadi,
  spam, tidak sesuai tipe (mis. tulisan diposting sebagai portfolio), kualitas/kelengkapan, lainnya.
- Setiap keputusan masuk audit log dan dinotifikasi ke **semua author**.

---

## 5. Sosial

- **Follow** user.
- **Apresiasi** (react) pada konten `published`.
- **Komentar** dengan balasan 1 level, mention `@username`.
  - Bukan "konten", jadi **tidak** diantrikan sebelum tayang. Pengamannya filter kata otomatis,
    laporan, moderator bisa menyembunyikan, dan **pemilik konten bisa menghapus komentar di kontennya sendiri**.
- **Simpan** ke koleksi pribadi (daftar hanya terlihat oleh pemiliknya, karena ini bookmark, bukan konten).
- **Share**: salin link / share native + Open Graph image otomatis.
- **Laporkan**: konten, komentar, profil, grup.
- Saat konten `archived` / `taken_down` / `deleted`: komentar & apresiasi ikut tersembunyi, tidak dihapus
  (kecuali purge permanen).
- **Notifikasi**: in-app + email digest opsional. Pemicunya undangan kolaborasi, keputusan moderasi,
  komentar, balasan, mention, apresiasi (diringkas), follower baru, badge baru.

### 5.1 Feed & discovery

- Feed beranda: **Mengikuti** · **Jurusanku** · **Terbaru** · **Trending**.
- **Katalog Karya** (portfolio): filter jurusan, kategori, tools, skill, konteks.
- **Blog**: filter kategori, tag.
- Pencarian: user, konten, skill, grup.
- **Kurasi "Pilihan Guru"**: guru/moderator menandai konten `published` sebagai pilihan dan
  memasukkannya ke koleksi kurasi.

---

## 6. Profil & Experience

### 6.1 Halaman profil (selalu publik)

- Banner, foto profil, nama, @username, headline, chip status (Siswa/Alumni/Guru/Tendik),
  jurusan, kelas/angkatan.
- Bio, kota (opsional), link sosial (GitHub, Behance, Dribbble, ArtStation, YouTube, LinkedIn, website).
- Statistik: pengikut, mengikuti, total apresiasi, jumlah portfolio & blog.
- **Sematkan** maksimal 3 konten.
- **Skill** dari katalog skill, bisa di-*endorse* member lain.
- **Badge** (lihat §6.3).
- Tab: **Portfolio · Blog · Experience · Kolaborasi**.
- Guru/Tendik: tambahan "Mengampu" (jurusan/mapel).

### 6.2 Experience (self-declared, ala LinkedIn)

- Tipe: Pendidikan, PKL/Magang, Pekerjaan, Freelance, Organisasi, Kepanitiaan, Lomba & Prestasi,
  Sertifikasi, Volunteer.
- Field: judul/posisi, instansi, lokasi, tipe kerja, mulai–selesai (bulan/tahun) atau
  "masih berjalan", deskripsi, skill terkait, tautan ke portfolio, lampiran (sertifikat/foto).
- Entri Pendidikan **SMKN 4 Malang dibuat otomatis** dari data akademik dan tidak bisa dihapus/diubah user.

### 6.3 Badge (terverifikasi, beda dengan experience)

- **Badge sistem** (otomatis): Karya Pertama, 10 Karya Terbit, Tulisan Pertama, Karya Pilihan, dll.
- **Badge sekolah** (diberikan Admin): Juara LKS, Lolos SNBP/SNBT, Bersertifikat LSP,
  Pengurus OSIS 2026/27, dll. Bisa diberi masa berlaku.
- Admin mengelola katalog badge (nama, ikon, deskripsi, kriteria).

### 6.4 Moderasi profil

Profil, experience, avatar, dan banner **post-moderasi**: langsung tayang, dengan filter otomatis,
bisa dilaporkan, dan moderator bisa mereset avatar/banner/bio/experience yang melanggar
(dengan notifikasi ke user).

---

## 7. Grup Jurusan & Kelas

### 7.1 Jenis & pembuatan

- Hanya dua jenis grup: **Jurusan** dan **Kelas** (rombel per tahun ajaran, mis. "XI RPL B · 2026/2027").
- **Dibuat otomatis** saat data jurusan/kelas dibuat di modul akademik. Tidak bisa dibuat atau dihapus
  manual.
- Grup punya status `aktif` atau `riwayat`. Istilah "riwayat" dipakai, bukan "arsip", supaya tidak
  tertukar dengan status konten `archived`.
- Kelas yang tahun ajarannya sudah lewat → grup jadi **riwayat**. Jurusan yang dinonaktifkan → grupnya
  juga jadi riwayat.
- Yang berubah saat grup jadi riwayat **hanya grupnya, bukan kontennya**:
  - Tetap publik, URL sama, isi tetap tampil seperti biasa.
  - Diberi label, mis. "Riwayat · 2026/2027".
  - Foto, banner, dan deskripsi **dikunci** untuk pengelola. Admin masih bisa mengubah.
  - Tidak muncul di daftar kelas aktif / menu "Kelasku". Pindah ke daftar riwayat kelas di profil
    anggota dan di grup jurusan.
  - Daftar anggota tetap (enrollment imutabel).
- Tidak ada yang dihapus. Konten anggota tidak terpengaruh sama sekali.

### 7.2 Keanggotaan (diturunkan dari data akademik, tidak bisa join/leave manual)

- **Grup kelas**: siswa yang terdaftar (enrollment) di rombel tersebut pada tahun ajaran itu.
- **Grup jurusan**: semua siswa dan alumni yang pernah terdaftar di jurusan tersebut.
- Guru/Tendik bukan anggota otomatis. Mereka bisa dijadikan **pengelola** grup.

### 7.3 Pengelolaan

- **Admin** (dan **Admin Jurusan** untuk grup di jurusannya) bisa mengubah foto profil, banner,
  dan deskripsi grup, serta menunjuk/mencabut **Pengelola Grup**.
- **Pengelola Grup** (user mana pun yang ditunjuk) hanya bisa mengubah **foto profil, banner, dan
  deskripsi** grup tersebut. Tidak ada wewenang lain.
- Nama grup mengikuti data akademik (tidak bisa diubah pengelola).
- Perubahan tampilan grup post-moderasi (sama seperti profil) dan masuk audit log.

### 7.4 Isi grup = mirror, tanpa postingan khusus

- Isi grup adalah **portfolio & blog `published` yang salah satu author-nya anggota grup**.
  Tidak ada postingan khusus di grup.
- **Grup kelas** hanya menampilkan konten yang `published_at`-nya berada dalam periode tahun ajaran
  kelas itu, jadi grup "X RPL A · 2025/2026" tetap jadi potret karya kelas itu di tahun itu.
- **Grup jurusan** menampilkan semua konten anggota, tanpa batas waktu.
- Konten kolaborasi lintas kelas/jurusan muncul di **setiap** grup yang punya author di dalamnya.
- Default urutan: **terbaru** (`published_at` desc). Opsi urutan lain: terpopuler.
- Filter: tipe (portfolio/blog), kategori.
- Tab lain: **Anggota** (daftar anggota) dan **Tentang** (deskripsi, pengelola).

---

## 8. Akademik & Penilaian

### 8.1 Data master

- Jurusan, tahun ajaran, kelas/rombel, enrollment.
- Tingkat kelas: **X, XI, XII** untuk semua jurusan (masa studi 3 tahun, tidak ada kelas XIII).
- Siswa baru kelas X masuk lewat **impor CSV** terpisah, bukan lewat alur kenaikan.
- Enrollment **imutabel**: setiap tahun ajaran membuat baris baru, dan riwayat lama tidak pernah diubah.

### 8.2 Kenaikan kelas & kelulusan (semi-otomatis, terjadwal)

Sistem menyiapkan semuanya, admin meninjau pengecualian dan menyetujui, lalu eksekusi berjalan
otomatis di tanggal mulai tahun ajaran. Tidak full otomatis karena hasil kenaikan ditentukan rapat
sekolah (tinggal kelas, mutasi, pindah rombel, lulus ditunda).

1. **Admin membuat tahun ajaran baru** beserta tanggal mulainya.
2. **Sistem membuat rombel & grup tahun depan** dengan memetakan rombel pasangan:
   X RPL A → XI RPL A, XI RPL A → XII RPL A. Kelas XII dipetakan ke **lulus**.
   Admin bisa menyesuaikan pemetaan (mis. rombel digabung/dipecah).
3. **Sistem menyusun draf promotion job**: default setiap siswa aktif naik ke rombel pasangannya.
4. **Admin / Admin Jurusan menandai pengecualian** per siswa, manual atau impor CSV hasil rapat:
   - **Tinggal kelas**: tetap di tingkat yang sama, rombel tahun depan dipilih.
   - **Pindah rombel**.
   - **Mutasi keluar**: status → `mutasi`, akun nonaktif untuk login. Konten yang sudah tayang tetap.
   - **Lulus ditunda** (kelas XII): tetap berstatus siswa, enrollment XII baru.
   Admin Jurusan hanya bisa mengubah siswa di jurusannya.
5. **Dry-run**: ringkasan jumlah naik / tinggal / lulus / mutasi per jurusan & rombel, plus daftar
   konflik (siswa tanpa rombel tujuan, rombel tujuan kosong, siswa ganda). Job **tidak bisa disetujui
   selama masih ada konflik**.
6. **Persetujuan oleh Admin** (bukan Admin Jurusan). Setelah disetujui, job terkunci dan dijadwalkan
   ke tanggal mulai tahun ajaran. Admin bisa membatalkan persetujuan sebelum tanggal itu.
7. **Eksekusi otomatis** di tanggal mulai, dalam **satu transaksi** (semua berhasil atau tidak ada yang berubah):
   - Enrollment baru dibuat untuk tahun ajaran baru.
   - Grup kelas tahun lalu → status `riwayat`, grup kelas tahun baru → `aktif`.
   - Siswa XII yang lulus → status `alumni`. Entri pendidikan "SMKN 4 Malang" di experience
     otomatis terisi tahun lulus.
   - Akun, profil, konten, kolaborasi, follower **tidak berubah**. Alumni tetap memegang `content.create`.
   - Konten `in_review` dan antrian moderasi tidak terpengaruh.
   - Semua anggota yang terdampak menerima notifikasi (mis. "Kamu sekarang di XI RPL A").
8. **Jaring pengaman**:
   - Pengingat ke Admin **30 hari** dan **7 hari** sebelum tahun ajaran baru jika job belum disetujui.
   - Jika tanggal mulai tiba dan job belum disetujui, tidak ada yang dieksekusi. Admin dinotifikasi,
     dan job tetap bisa disetujui lalu dieksekusi manual.
   - **Rollback** seluruh job dalam **14 hari** setelah eksekusi: enrollment baru dihapus, status
     dan grup dikembalikan. Hanya Admin.
   - Setiap langkah (buat draf, ubah pengecualian, setujui, eksekusi, rollback) masuk audit log.

### 8.3 Penugasan guru

- **Penugasan guru** (portfolio saja): guru membuat penugasan untuk kelas, siswa menautkan portfolio,
  dan guru menilai dengan rubrik. Nilai **tidak** tampil publik (hanya siswa & guru terkait).
  Portfolio tetap wajib lewat moderasi biasa untuk bisa tayang.

---

## 9. Admin & Setting

- Manajemen user, role, capability `content.create`, jurusan, kelas, grup & pengelola, badge,
  kategori portfolio/blog, katalog skill & tools, kategori alasan moderasi, filter kata.
- **Setting configurable**:

| Setting | Default |
|---|---|
| Maks author portfolio | 50 |
| Maks author blog | 10 |
| Maks draf aktif per user | 20 |
| Maks konten `in_review` per user | 5 |
| Kedaluwarsa undangan kolaborasi | 14 hari |
| Kedaluwarsa claim moderasi | 30 menit |
| Retensi soft delete | 30 hari |
| Maks konten disematkan | 3 |
| Pengingat promotion job sebelum tahun ajaran | 30 & 7 hari |
| Jendela rollback promotion job | 14 hari |

- Dasbor: antrian moderasi, laporan, statistik konten per jurusan/kelas, aktivitas.
- Audit log untuk semua aksi admin/moderator.

---

## 10. Keamanan & Keselamatan (mayoritas user di bawah umur)

- Tidak ada DM. Interaksi hanya lewat komentar publik.
- Data identitas tidak pernah tampil publik (§1.3).
- Semua konten pre-moderasi. Profil, komentar, dan grup post-moderasi dengan filter + laporan.
- Rate limit untuk komentar, apresiasi, undangan kolaborasi, laporan.
- Strip metadata EXIF (termasuk lokasi) dari semua unggahan gambar.

---

## 11. Konsep model data (tingkat tinggi)

```
users ─┬─ profiles (banner, avatar, headline, bio, links)
       ├─ experiences
       ├─ user_skills ── skill_endorsements
       ├─ user_badges ── badges
       ├─ user_roles ── roles ── role_permissions
       └─ enrollments ── classes ── academic_years
                              └── majors

groups (type: major|class, ref_id, banner, avatar, description, archived_at)
group_managers (group_id, user_id, assigned_by)

contents (type: portfolio|blog, status, owner_id, slug, published_at, live_version_id)
  ├─ content_authors (user_id, level: owner|editor|contributor, role_label, position, invite_status)
  ├─ content_versions (immutable snapshot, hash, status of revision, submitted_at)
  ├─ portfolio_details / blog_details
  ├─ moderation_cases (version_id, moderator_id, claim_until, decision, reason_category, notes)
  ├─ comments ── comment_reports
  ├─ reactions
  └─ bookmarks

promotion_jobs (academic_year_id, status: draft|approved|executed|rolled_back, scheduled_for)
  └─ promotion_items (student_id, action: promote|retain|move|transfer_out|graduate|defer, target_class_id)

follows, notifications, reports, audit_logs, settings
assignments ── assignment_submissions ── assessments
```

---

## 12. Fase rilis

1. **Fase 1 (MVP)**: akun dari data sekolah, profil + experience, portfolio, blog, kolaborasi,
   lifecycle + moderasi + revisi, follow/apresiasi/komentar/simpan/share, notifikasi, grup jurusan & kelas.
2. **Fase 2**: badge, skill endorsement, kurasi "Pilihan Guru", penugasan & penilaian guru,
   ekspor CV PDF, trending feed.

---

## 13. Halaman, Routing & Flow

### 13.1 Prinsip URL

1. **Slug URL berbahasa Indonesia** (`/karya`, `/masuk`, `/pengaturan`).
2. **Profil di `/@{username}`**.
3. **URL konten tidak memuat username**, karena konten bisa multi-author dan kepemilikannya bisa
   dipindah. Format: `/karya/{slug}-{id}` dan `/blog/{slug}-{id}`.
   - `{id}` adalah ID pendek (mis. `a7k2`) dan menjadi kunci lookup. `{slug}` hanya kosmetik.
   - Slug salah/lama (mis. judul berubah lewat revisi) → **301 redirect** ke slug kanonik.
4. **Satu URL untuk semua status konten.**
   - `published` → semua orang.
   - Status lain (`draft`, `in_review`, `changes_requested`, `archived`, `taken_down`, `rejected`) →
     author & moderator melihatnya dengan **banner status**. Publik dan member lain mendapat **404**
     (tidak membocorkan keberadaan konten).
   - `deleted` → 404 untuk semua kecuali pemilik (lewat Studio › Sampah) dan Admin.
5. **Ruang kerja author dipisah** di `/studio`, terpisah dari halaman publik.
6. Halaman grup yang berstatus `riwayat` tetap memakai URL yang sama.
7. Query string dipakai untuk filter/urutan/tab yang bisa di-share (mis. `/karya?jurusan=rpl&urut=populer`).

### 13.2 Aturan username

- 3–30 karakter, huruf kecil, angka, titik, underscore. Tidak boleh diawali/diakhiri titik.
- Unik, case-insensitive.
- Dipilih saat aktivasi. Bisa diganti di `/pengaturan/akun` maksimal **1× per 30 hari**.
- Username lama di-*redirect* ke yang baru selama 90 hari, lalu dilepas.
- **Username terlarang (reserved)**: semua segmen route tingkat atas (`admin`, `studio`, `karya`, `blog`,
  `jurusan`, `kelas`, `cari`, `masuk`, `aktivasi`, `onboarding`, `pengaturan`, `moderasi`, `notifikasi`,
  `tersimpan`, `kurasi`, `penugasan`, `pedoman`, `tentang`, `privasi`, `syarat`, `api`), plus nama
  yang menyaru pihak resmi (`grafikarsa`, `smkn4malang`, `grafika`, `official`, `moderator`, `support`, dll).
  Daftar dikelola di kode + bisa ditambah Admin.

### 13.3 Halaman publik (tanpa login)

| URL | Halaman | Catatan |
|---|---|---|
| `/` | Beranda | Tamu: landing (hero, karya pilihan, terbaru). Member: feed |
| `/?tab=mengikuti` · `jurusanku` · `terbaru` · `trending` | Tab feed | `mengikuti` & `jurusanku` khusus member. Default member: `mengikuti`, tamu: `terbaru` |
| `/karya` | Katalog Karya | Filter: `jurusan`, `kategori`, `tools`, `skill`, `konteks`. Urut: `terbaru` (default) / `populer` |
| `/karya/{slug}-{id}` | Detail portfolio | |
| `/blog` | Daftar blog | Filter: `kategori`, `tag`. Urut: `terbaru` / `populer` |
| `/blog/{slug}-{id}` | Detail blog | |
| `/blog/tag/{tag}` | Blog per tag | |
| `/@{username}` | Profil, tab Portfolio (default) | |
| `/@{username}/blog` | Tab Blog | |
| `/@{username}/experience` | Tab Experience | |
| `/@{username}/kolaborasi` | Tab Kolaborasi (konten di mana user bukan pemilik) | |
| `/@{username}/pengikut` · `/@{username}/mengikuti` | Daftar pengikut / diikuti | |
| `/jurusan` | Daftar jurusan | |
| `/jurusan/{kode}` | Grup jurusan: feed konten anggota | Mis. `/jurusan/rpl`. Filter `tipe`, `kategori`; urut `terbaru` / `populer` |
| `/jurusan/{kode}/kelas` | Daftar kelas di jurusan (aktif + riwayat) | |
| `/jurusan/{kode}/anggota` | Anggota jurusan | |
| `/jurusan/{kode}/tentang` | Deskripsi, pengelola | |
| `/kelas/{tahun}/{rombel}` | Grup kelas: feed konten anggota | Mis. `/kelas/2026-2027/xi-rpl-b` |
| `/kelas/{tahun}/{rombel}/anggota` | Anggota kelas | |
| `/kelas/{tahun}/{rombel}/tentang` | Deskripsi, pengelola | |
| `/cari?q=…&tipe=user\|karya\|blog\|grup` | Pencarian | Tanpa `tipe` = hasil gabungan per bagian |
| `/pedoman` | Pedoman komunitas | Dirujuk oleh setiap alasan moderasi |
| `/tentang` · `/privasi` · `/syarat` | Halaman statis | |

### 13.4 Autentikasi

| URL | Halaman | Catatan |
|---|---|---|
| `/masuk` | Login | Identitas: **NISN** (siswa/alumni) / **NIP atau ID staf** (guru/tendik) / username / email. `?kembali=` untuk kembali ke halaman asal (hanya path internal) |
| `/aktivasi` | Login pertama | Wajib: ganti sandi awal, isi email, pilih username. Tidak bisa ke halaman lain sebelum selesai |
| `/onboarding` | Lengkapi profil | Foto, headline, bio, skill. Boleh dilewati |
| `/lupa-sandi` | Minta reset sandi | Butuh email terdaftar. Tanpa email → reset oleh Admin |
| `/reset-sandi?token=` | Atur sandi baru | Token sekali pakai, kedaluwarsa singkat |
| `/verifikasi-email?token=` | Verifikasi email | |

Akun dibagikan sekolah berupa **NISN/NIP + sandi awal**. Sandi awal hanya berlaku untuk aktivasi
dan kedaluwarsa jika tidak dipakai dalam waktu tertentu (Admin bisa menerbitkan ulang).

### 13.5 Halaman member (wajib login)

| URL | Halaman |
|---|---|
| `/notifikasi` | Semua notifikasi |
| `/tersimpan` | Koleksi tersimpan (hanya terlihat pemiliknya) |
| `/studio` | Dashboard konten saya. Tab: Draf · Ditinjau · Perlu Revisi · Terbit · Diarsipkan · Diturunkan · Ditolak · Sampah |
| `/studio/baru` | Pilih tipe: Portfolio / Blog. Draf dibuat lewat aksi POST, lalu redirect ke editor |
| `/studio/{id}` | Editor (blok untuk portfolio, dokumen untuk blog) + panel status & catatan moderasi |
| `/studio/{id}/kolaborator` | Kelola author: undang, level, label peran, urutan, transfer kepemilikan |
| `/studio/{id}/riwayat` | Versi yang pernah disetujui + log keputusan moderasi |
| `/studio/{id}/revisi` | Editor revisi untuk konten yang sudah terbit |
| `/studio/undangan` | Undangan kolaborasi masuk (terima/tolak) |
| `/pengaturan/profil` | Banner, foto, nama tampilan, headline, bio, kota, link sosial, skill, konten disematkan |
| `/pengaturan/experience` | Tambah/ubah/hapus experience |
| `/pengaturan/akun` | Username, email, sandi |
| `/pengaturan/sesi` | Perangkat aktif, logout perangkat lain |
| `/pengaturan/notifikasi` | Preferensi email digest |
| `/jurusan/{kode}/kelola` · `/kelas/{tahun}/{rombel}/kelola` | Edit foto, banner, deskripsi grup (pengelola grup, Admin Jurusan, Admin) |

Editor (`/studio/{id}`) hanya bisa dibuka **pemilik & editor**. Kontributor hanya bisa melihat
preview dan keluar dari konten.

### 13.6 Halaman moderator

| URL | Halaman |
|---|---|
| `/moderasi` | Antrian konten. Filter: tipe, jurusan, submit baru vs revisi, umur antrian. Konten milik moderator sendiri tidak muncul |
| `/moderasi/{caseId}` | Ruang tinjau: preview, **diff** (untuk revisi), catatan per blok, Setujui / Minta Revisi / Tolak. Membuka halaman = *claim* |
| `/moderasi/laporan` | Antrian laporan (konten, komentar, profil, grup) |
| `/moderasi/laporan/{id}` | Detail laporan + aksi |
| `/moderasi/riwayat` | Log keputusan |

### 13.7 Halaman admin

| URL | Halaman |
|---|---|
| `/admin` | Dashboard |
| `/admin/pengguna` | Daftar user (filter tipe, status, jurusan, kelas) |
| `/admin/pengguna/{id}` | Detail user: data akademik, role, capability, terbitkan ulang sandi awal, nonaktifkan |
| `/admin/pengguna/impor` | Impor CSV (siswa baru, guru, tendik) dengan preview & validasi |
| `/admin/role` | Role, izin, penugasan role |
| `/admin/akademik/jurusan` | Data jurusan |
| `/admin/akademik/tahun-ajaran` | Tahun ajaran |
| `/admin/akademik/kelas` | Kelas/rombel |
| `/admin/akademik/kenaikan` | Daftar promotion job |
| `/admin/akademik/kenaikan/{jobId}` | Pemetaan rombel, pengecualian, dry-run, setujui, rollback |
| `/admin/grup` | Grup & pengelolanya |
| `/admin/badge` | Katalog badge & pemberian badge |
| `/admin/taksonomi` | Kategori portfolio/blog, skill, tools, alasan moderasi, filter kata, username terlarang |
| `/admin/pengaturan` | Setting configurable (§9) |
| `/admin/audit` | Audit log |

Admin Jurusan memakai halaman `/admin/...` yang sama, dengan data **otomatis dibatasi ke jurusannya**
dan menu yang bukan wewenangnya disembunyikan (enforcement tetap di server).

**Fase 2:** `/kurasi`, `/kurasi/{slug}` (Pilihan Guru); `/penugasan`, `/penugasan/{id}`.

### 13.8 Aturan akses route

| Kelompok | Tamu | Member | Moderator | Admin Jurusan | Admin |
|---|---|---|---|---|---|
| Publik (§13.3) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Auth (§13.4) | ✅ | redirect ke `/` (kecuali `/aktivasi` saat wajib) | ← | ← | ← |
| Member (§13.5) | redirect `/masuk?kembali=` | ✅ | ✅ | ✅ | ✅ |
| `/moderasi/*` | redirect `/masuk` | 404 | ✅ (sesuai scope) | ✅ (jurusannya) | ✅ |
| `/admin/*` | redirect `/masuk` | 404 | 404 | ✅ (jurusannya) | ✅ |

- Area tanpa wewenang mengembalikan **404**, bukan 403, supaya struktur area admin tidak terbaca.
- Akun yang belum aktivasi selalu diarahkan ke `/aktivasi`.

### 13.9 Navigasi

- **Desktop (top bar):** Logo · Beranda · Karya · Blog · Jurusan · Cari | **+ Buat** · Notifikasi · Avatar.
- **Menu avatar:** Profil saya · Studio · Undangan · Tersimpan · *Moderasi* · *Admin* · Pengaturan · Keluar
  (menu miring hanya tampil untuk role terkait).
- **Mobile (bottom bar):** Beranda · Karya · **+** · Blog · Profil. Notifikasi & cari di header.
- **Tamu:** tombol "+ Buat" dan Notifikasi diganti tombol **Masuk**.

### 13.10 Flow utama

**1. Login pertama**
```
/masuk (NISN/NIP + sandi awal dari sekolah)
  → /aktivasi: ganti sandi, isi email, pilih username        [wajib]
  → email verifikasi dikirim (boleh diverifikasi nanti)
  → /onboarding: foto, headline, bio, skill                   [boleh dilewati]
  → /
```

**2. Tamu mencoba berinteraksi**
```
/karya/x → klik Apresiasi / Komentar / Follow / Simpan
  → dialog "Masuk untuk berinteraksi" → /masuk?kembali=/karya/x
  → setelah login kembali ke /karya/x (aksi tidak dijalankan otomatis)
```

**3. Membuat konten sampai terbit**
```
+ Buat → /studio/baru → pilih tipe → draf dibuat → /studio/{id}
  → tulis (autosave) → /studio/{id}/kolaborator: undang author (opsional)
  → "Kirim ke Moderasi" → validasi field wajib (gagal = tunjukkan yang kurang)
  → status Ditinjau (editor terkunci, tombol "Tarik Kembali" selama belum di-claim)
  → moderator di /moderasi/{caseId}:
       Setujui      → notifikasi semua author → konten tayang di /karya|blog/{slug}-{id}
       Minta Revisi → author membuka /studio/{id} dengan catatan per blok → perbaiki → kirim lagi
       Tolak        → final, alasan tampil di /studio/{id}
```

**4. Merevisi konten yang sudah terbit**
```
/karya/{slug}-{id} (sebagai pemilik/editor) → "Edit" → /studio/{id}/revisi
  (versi tayang tetap online, banner "Kamu sedang mengedit revisi")
  → kirim → moderator melihat diff
  → Setujui: versi tayang diganti | Minta Revisi: kembali ke editor revisi | Tolak: revisi dibuang, versi tayang aman
```

**5. Undangan kolaborasi**
```
Pemilik mengundang di /studio/{id}/kolaborator → notifikasi → /studio/undangan
  → Terima: nama tampil di konten + muncul di profilnya
  → Tolak / 14 hari tidak dijawab: undangan hangus
```

**6. Arsip, hapus, pulihkan**
```
/studio (tab Terbit) → Arsipkan → hilang dari publik → tab Diarsipkan → Terbitkan lagi (tanpa review)
/studio → Hapus → tab Sampah (30 hari) → Pulihkan ke status sebelumnya | purge otomatis
```

**7. Melaporkan**
```
Menu ⋯ di konten / komentar / profil / grup → pilih alasan → kirim
  → /moderasi/laporan → moderator: abaikan / sembunyikan komentar / turunkan konten / reset profil atau grup
  → pelapor dinotifikasi hasilnya (tanpa detail), pihak terlapor dinotifikasi jika ada tindakan
```

**8. Kelola grup**
```
Admin menunjuk pengelola di /admin/grup
  → pengelola membuka /jurusan/{kode}/kelola atau /kelas/{tahun}/{rombel}/kelola
  → ubah foto/banner/deskripsi → langsung tayang (post-moderasi)
  (grup berstatus riwayat: hanya Admin yang bisa mengubah)
```

**9. Kenaikan kelas**
```
/admin/akademik/tahun-ajaran: buat tahun ajaran baru
  → /admin/akademik/kenaikan/{jobId}: draf + pemetaan rombel otomatis
  → tandai pengecualian / impor CSV → dry-run → selesaikan konflik
  → Setujui → terjadwal → eksekusi otomatis di tanggal mulai → rollback ≤ 14 hari
```
