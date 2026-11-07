# Grafikarsa Revamp — Algoritma

> Feed, populer, trending, hitung view, counter, notifikasi, pencarian, isi grup, antrian moderasi, badge.
> Semua bobot dan ambang di bawah disimpan sebagai **setting** (`algo.*`) supaya bisa disetel tanpa deploy.

---

## 1. Prinsip

1. **Bisa dijelaskan.** Rumus sederhana yang bisa dijelaskan ke siswa dan guru, bukan kotak hitam.
2. **Adil untuk pemula.** Siswa baru dengan sedikit pengikut tetap punya peluang muncul (normalisasi audiens).
3. **Tahan manipulasi.** Interaksi dari author sendiri tidak dihitung, satu orang = satu suara, ambang minimum
   pelaku unik.
4. **Murah dijalankan.** Skor dihitung worker secara berkala, request baca hanya membaca index.

---

## 2. Feed beranda

| Tab | Isi | Urutan | Sumber data |
|---|---|---|---|
| **Mengikuti** (member) | Konten tayang yang salah satu author-nya (accepted) diikuti user | `first_published_at` desc | `follows` → `content_authors` (`ix_content_authors_user`) → `contents` |
| **Jurusanku** (member yang punya jurusan) | Isi grup jurusan user (alumni: jurusan enrollment terakhir, guru: `teacher_majors`) | `first_published_at` desc | `group_contents` |
| **Terbaru** | Semua konten tayang | `first_published_at` desc | `ix_contents_published` |
| **Trending** | Konten dengan `trending_score > 0` | `trending_score` desc | `ix_stats_trending` |

- Semua tab memakai **keyset pagination** (`(first_published_at, id) < (cursor)`), 20 item per halaman,
  bukan `OFFSET`.
- Filter tipe di setiap tab: Semua · Portfolio · Blog.
- Konten kolaborasi tidak muncul dobel (EXISTS, bukan JOIN langsung).
- Tab Mengikuti kosong → tampilkan saran akun untuk diikuti (teman sekelas dan sejurusan yang aktif) + Trending.
- Halaman pertama per user di-cache Valkey 30 detik. Halaman publik (Terbaru/Trending untuk tamu) di-cache Next.js
  dan direvalidasi lewat tag saat ada konten terbit/turun.

---

## 3. Skor engagement

### 3.1 Bobot interaksi

| Sinyal | Bobot | Catatan |
|---|---|---|
| View unik (per hari) | 1 | §4 |
| Apresiasi | 4 | |
| Komentar dari pengguna unik | 6 | Komentar author sendiri tidak dihitung, balasan beruntun dari orang yang sama dihitung sekali |
| Simpan (bookmark) | 5 | Sinyal kualitas kuat |
| Share | 3 | Dedupe per pelaku per hari |

Interaksi dari **author konten itu sendiri tidak pernah dihitung** di semua skor.

### 3.2 Populer (`urut=populer`)

Dipakai di Katalog Karya, Blog, grup, dan profil. Pilihan periode: **7 hari · 30 hari · Sepanjang waktu**.

```
E(periode)    = Σ bobot × jumlah sinyal dalam periode
popular_score = log10(1 + E(sepanjang waktu))        → disimpan di content_stats
```

- Periode 7 dan 30 hari dihitung worker tiap jam ke sorted set Valkey (`popular:7d`, `popular:30d`), diambil
  per halaman lalu dihidrasi dari Postgres.
- `log10` meredam selisih ekstrem supaya urutan tetap bergerak.

### 3.3 Trending (`tab=trending`)

Dihitung worker **tiap 15 menit** untuk konten yang terbit ≤ 14 hari terakhir:

```
raw      = Σ_sinyal  bobot × 2^(−usia_sinyal_jam / 24)            (half-life 24 jam, sinyal ≤ 7 hari)
audiens  = 1 + 0.25 × log10(1 + jumlah_pengikut_pemilik)
trending = raw / audiens          jika pelaku_unik_72_jam ≥ 3, selain itu 0
```

- **Half-life 24 jam**: interaksi baru jauh lebih berpengaruh dari interaksi lama.
- **Normalisasi audiens**: akun dengan banyak pengikut tidak otomatis mendominasi.
- **Ambang 3 pelaku unik**: satu-dua orang tidak bisa mendorong konten ke trending.
- Konten di luar `published` langsung `trending_score = 0`.

### 3.4 Karya pilihan di landing

1. Isi kurasi `pilihan-guru` (fase 2).
2. Jika kurang dari 8 item: diisi portfolio dengan `trending_score` tertinggi, maksimal 1 karya per pemilik.

---

## 4. Hitung view & share

```
Browser menampilkan konten
  → setelah konten terlihat ≥ 3 detik (IntersectionObserver), kirim beacon POST /api/v1/contents/{id}/view
  → api: abaikan jika bot (UA), author konten, atau status ≠ published
  → viewer_key = user_id (member) atau HMAC(ip + user_agent, salt_harian) (tamu, tanpa cookie)
  → Valkey:
       SET  vd:{content}:{viewer_key} 1 NX EX 1800      → hanya lanjut jika baru (dedupe 30 menit)
       INCR v:raw:{content}:{yyyymmdd}
       PFADD v:uniq:{content}:{yyyymmdd} viewer_key       (HyperLogLog, ±0,8% error)
       SADD stats:dirty {content}
  → worker tiap 5 menit: PFCOUNT + GET → upsert content_views_daily, update content_stats
```

- Salt harian membuat `viewer_key` tamu tidak bisa dilacak lintas hari (privasi).
- Share: tombol share memanggil `POST /api/v1/contents/{id}/share` (dedupe per pelaku per hari), lalu
  membuka share native / salin link.

---

## 5. Counter (apresiasi, komentar, simpan)

- API menulis baris (`INSERT ... ON CONFLICT DO NOTHING`) lalu `SADD stats:dirty {content}`. **Tidak** ada
  `UPDATE counter = counter + 1` di request.
- Worker tiap **30 detik** mengambil konten dari `stats:dirty` dan menghitung ulang dengan `COUNT(*)` berindex
  → nilai selalu tepat, tidak bisa drift, dan tidak ada hot row saat konten ramai.
- UI memakai optimistic update, jadi user langsung melihat angka berubah.
- Rekonsiliasi penuh semua konten setiap malam (pengaman).

---

## 6. Notifikasi

### 6.1 Agregasi

Notifikasi sejenis yang **belum dibaca** digabung lewat upsert pada `(recipient_id, group_key)`:

```sql
INSERT INTO notifications (recipient_id, type, group_key, actor_id, subject_type, subject_id, data)
VALUES (...)
ON CONFLICT (recipient_id, group_key) WHERE read_at IS NULL
DO UPDATE SET actor_id = EXCLUDED.actor_id,
              actor_count = notifications.actor_count + 1,
              updated_at = now();
```

| Tipe | `group_key` | Contoh tampilan |
|---|---|---|
| `appreciation` | `appreciation:{content_id}` | "Dinda dan 12 lainnya mengapresiasi *Poster HUT 80*" |
| `comment_on_content` | `comment:{content_id}` | "Bima dan 3 lainnya mengomentari *…*" |
| `new_follower` | `follower:{yyyy-mm-dd}` | "Sari dan 5 lainnya mulai mengikutimu" |
| Lainnya (moderasi, kolaborasi, badge, keamanan) | `{type}:{subject_id}:{uuid}` | Tidak digabung, selalu tampil sendiri |

### 6.2 Aturan

- Tidak ada notifikasi untuk aksi diri sendiri.
- Preferensi (`notification_preferences`) dicek sebelum insert. Notifikasi **keamanan & moderasi tidak bisa
  dimatikan**.
- Pembatalan: apresiasi dibatalkan dalam 10 menit dan notifikasi belum dibaca → `actor_count − 1` (hapus jika 0).
- Fan-out dilakukan job River yang di-enqueue **di transaksi yang sama** dengan perubahan data.
- Realtime: setelah commit, worker `PUBLISH notif:{user_id}` di Valkey → koneksi SSE user menerima event
  `unread_count`. Klien yang tidak terhubung melihatnya saat membuka halaman.

---

## 7. Pencarian

| Target | Metode | Urutan |
|---|---|---|
| Konten | `websearch_to_tsquery('indonesian', q)` pada `search_vector` (judul A, excerpt/tag/author B, isi C) | `ts_rank_cd × (1 + 0.1 × popular_score)` |
| Konten (fallback typo) | Jika hasil FTS < 5: `similarity(title, q) > 0.3` (pg_trgm) | similarity desc |
| User | Trigram pada `display_name` dan `username`. Prefix username dapat boost | similarity desc, lalu jumlah pengikut |
| Grup | Trigram pada nama jurusan/kelas | similarity desc |
| Skill / tools / tag (autocomplete editor) | Trigram + prefix, maks 8 saran | similarity desc |

- Autocomplete kotak cari: debounce 200 ms, maks 5 per kategori, hasil di-cache Valkey 60 detik per query.
- Query dinormalisasi (huruf kecil, trim, batas 100 karakter). Rate limit 30/menit per pelaku.

---

## 8. Isi grup (`group_contents`)

Job `recompute_group_contents(content_id)`:

```
jika konten tidak published → hapus semua baris konten itu, selesai
groups = ∅
untuk setiap author accepted bertipe student:
    untuk setiap enrollment author:
        groups += grup jurusan dari kelas enrollment itu
        jika first_published_at ∈ [starts_on, ends_on] tahun ajaran enrollment:
            groups += grup kelas enrollment itu
sinkronkan baris group_contents konten itu dengan groups (insert yang kurang, hapus yang lebih)
```

Dipicu oleh: konten disetujui pertama kali / dibuka dari arsip / dipulihkan, author accepted/keluar/dihapus,
enrollment dibuat atau dihapus (impor, kenaikan kelas, rollback, pindah jurusan) → job
`recompute_group_contents_for_user(user_id)` untuk semua konten user itu.

---

## 9. Antrian moderasi

- Urutan **FIFO** (`created_at`), revisi dan konten baru dalam satu antrian.
- Penanda umur: **> 24 jam** kuning, **> 48 jam** merah (target tinjau 2×24 jam).
- Kasus > 48 jam di antrian jurusan juga ditonjolkan di antrian global.
- Ringkasan harian in-app ke Admin dan Admin Jurusan: jumlah antrian, kasus tertua, rata-rata waktu tinjau.
- Metrik di dashboard: median waktu sampai keputusan, % diputuskan ≤ 48 jam, jumlah per moderator,
  rasio setuju / minta revisi / tolak.
- Claim kedaluwarsa 30 menit (setting), lalu kembali ke `open`.

---

## 10. Badge sistem otomatis

Dievaluasi job River setiap ada kejadian terkait (bukan cron penuh):

| Badge | Kriteria (`badges.criteria`) | Pemicu |
|---|---|---|
| Karya Pertama | portfolio tayang ≥ 1 (sebagai author accepted) | konten disetujui |
| Tulisan Pertama | blog tayang ≥ 1 | konten disetujui |
| Produktif | konten tayang ≥ 10 | konten disetujui |
| Kolaborator | kolaborasi tayang ≥ 5 (bukan pemilik) | undangan diterima / konten disetujui |
| Karya Pilihan | masuk kurasi `pilihan-guru` | item kurasi ditambah |
| Diapresiasi | total apresiasi di semua konten ≥ 100 | rekonsiliasi counter |

Badge sistem tidak dicabut otomatis saat angka turun (kecuali Karya Pilihan jika item kurasi dihapus).
