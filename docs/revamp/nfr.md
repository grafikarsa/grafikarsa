# Grafikarsa Revamp — Kebutuhan Non-Fungsional

> Target performa, keandalan, UX, SEO, aksesibilitas, dan cara mencapainya.
> Target diukur di **HP Android kelas menengah + 4G** (kondisi mayoritas siswa), bukan laptop developer.

---

## 1. Target utama

### 1.1 Web (Core Web Vitals, persentil 75)

| Metrik | Target | Ambang "good" Google |
|---|---|---|
| LCP | **≤ 2,0 s** | 2,5 s |
| INP | **≤ 150 ms** | 200 ms |
| CLS | **≤ 0,05** | 0,1 |
| TTFB halaman publik (cache) | ≤ 200 ms | – |
| TTFB halaman dinamis | ≤ 600 ms | – |
| Navigasi antar halaman (klien) | terasa instan, skeleton ≤ 100 ms | – |
| JS awal halaman publik | ≤ 170 KB gzip | – |
| Lighthouse (mobile) landing, karya, blog, profil | Performance ≥ 90, A11y ≥ 95, SEO = 100 | – |

### 1.2 API

| Jenis | p95 | p99 |
|---|---|---|
| Baca (cache hit) | ≤ 50 ms | ≤ 120 ms |
| Baca (DB) | ≤ 150 ms | ≤ 400 ms |
| Tulis (interaksi, autosave) | ≤ 200 ms | ≤ 500 ms |
| Pencarian | ≤ 250 ms | ≤ 600 ms |
| Submit ke moderasi | ≤ 500 ms | ≤ 1 s |
| Upload diterima (10 MB, jaringan sekolah) | ≤ 3 s | – |
| Varian gambar siap setelah upload | ≤ 10 s | ≤ 30 s |
| Notifikasi realtime sampai ke klien | ≤ 2 s | ≤ 5 s |

### 1.3 Kapasitas (dibuktikan dengan load test, §7)

Di server 4 vCPU / 8 GB:

| Skenario | Target |
|---|---|
| Pengguna aktif bersamaan | 1.500 (≈ setengah sekolah online sesudah pengumuman) |
| Request baca API | 800 RPS berkelanjutan, error < 0,1% |
| Request tulis API | 80 RPS |
| Koneksi SSE bersamaan | 3.000 |
| Upload bersamaan | 50 |

### 1.4 Keandalan

| Metrik | Target |
|---|---|
| Ketersediaan bulanan | **99,5%** (±3,6 jam/bulan, mayoritas untuk maintenance terjadwal di luar jam sekolah) |
| RPO (data hilang maksimal) | **≤ 1 menit** (arsip WAL `archive_timeout = 60s`) |
| RTO (pulih total di VPS baru) | **≤ 2 jam**, pulih DB saja ≤ 30 menit |
| Deploy | Tanpa downtime yang terasa (≤ 10 detik), rollback ≤ 5 menit |
| Kehilangan job | 0 (River, transaksional, retry) |
| Uji restore | Otomatis mingguan ke container sementara + manual tiap rilis besar |

---

## 2. Strategi rendering (Next.js)

| Halaman | Strategi |
|---|---|
| Landing, Katalog Karya, Blog, Detail konten, Profil, Grup | **Server-rendered + cache berbasis tag** (revalidasi saat konten/profil berubah). Bagian yang bergantung pada viewer (sudah diapresiasi, follow, tombol edit) diambil di klien, sehingga HTML-nya sama untuk semua orang dan bisa di-cache |
| Feed member, Studio, Notifikasi, Moderasi, Admin | Render klien di dalam shell server, data via TanStack Query |
| Editor | Dimuat *lazy* (TipTap, dnd-kit hanya di Studio), tidak membebani halaman publik |

- Streaming + `Suspense` supaya kerangka halaman tampil sebelum data lambat selesai.
- Cache Next disimpan di Valkey (konsisten antar-replica). Tag: `content:{id}`, `user:{id}`, `group:{id}`,
  `feed:latest`, `catalog`.

## 3. Teknik "cepat & menyenangkan"

| Area | Teknik |
|---|---|
| Interaksi | **Optimistic update** untuk apresiasi, simpan, follow, komentar, tandai baca. Rollback halus + toast kalau gagal |
| Navigasi | Prefetch link saat terlihat/hover, cache TanStack Query, skeleton yang **sama bentuk** dengan konten asli (tidak loncat) |
| Transisi | View Transitions API untuk kartu → detail (dimatikan bila `prefers-reduced-motion`) |
| Gambar | AVIF/WebP, `srcset` 320/640/960/1280/1920, ukuran eksplisit (tanpa CLS), **blurhash + warna dominan** sebagai placeholder, gambar LCP diprioritaskan, sisanya lazy |
| Embed berat | Facade (thumbnail + tombol play) untuk YouTube/Figma/Canva, iframe baru dimuat saat diklik/terlihat |
| Font | **Inter** via `next/font` (self-hosted, variable, subset latin, `size-adjust` → tanpa layout shift) |
| Ikon | Komponen tree-shakeable (hanya ikon yang dipakai masuk bundle) |
| Editor | Autosave + **antrian offline** (IndexedDB): koneksi putus, tulisan tidak hilang, tersinkron saat online |
| Upload | Progress per file, unggah paralel (maks 3), bisa dilanjutkan bekerja sambil menunggu varian |
| Realtime | Badge notifikasi & status moderasi berubah tanpa refresh (SSE) |
| Feedback | Toast (shadcn `Sonner`) untuk setiap aksi, pesan error manusiawi berbahasa Indonesia |
| Keyboard | Shortcut: `/` cari, `n` buat konten, `g` + `h/k/b` navigasi, `?` daftar shortcut |
| PWA ringan | Manifest + ikon (bisa "Tambahkan ke layar utama"), tanpa mode offline penuh |

## 4. Teknik backend

- **pgxpool** dengan ukuran pool sesuai CPU, prepared statements, `statement_timeout = 5s`,
  `idle_in_transaction_session_timeout = 10s`, log query > 200 ms.
- **Anggaran query**: halaman detail ≤ 5 query. N+1 dilarang (batch loading per request).
- **Cache Valkey** untuk data panas (profil, konten tayang, izin user, hitungan): TTL ber-jitter +
  *singleflight* agar tidak terjadi cache stampede. Invalidasi berbasis kejadian, bukan menunggu TTL.
- **Keyset pagination** di semua list.
- Counter dan statistik lewat worker batch (algorithms.md §5), bukan update per request.
- Timeout HTTP server (read header 5 s, read 30 s, write 60 s, idle 120 s), batas ukuran body per endpoint,
  graceful shutdown (selesaikan request berjalan ≤ 25 s).
- Kompresi zstd/gzip di Caddy. HTTP/2 + HTTP/3.

## 5. Degradasi yang anggun

| Jika ini mati | Perilaku |
|---|---|
| Valkey | Aplikasi tetap jalan: cache miss → langsung DB, SSE jatuh ke polling 30 s, view/share sementara tidak terhitung. Rate limit login **fail-closed** (tolak), endpoint baca **fail-open** |
| RustFS | Upload dinonaktifkan dengan pesan jelas. Halaman tetap tampil, gambar diganti placeholder warna dominan |
| Worker | Interaksi tetap tersimpan. Notifikasi, counter, varian gambar tertunda lalu menyusul (job tersimpan di Postgres) |
| Next.js | Caddy menampilkan halaman maintenance statis |
| PostgreSQL | Halaman publik yang sudah di-cache tetap tampil (stale), aksi tulis ditolak dengan pesan maintenance |

## 6. Observability & alert

| Sinyal | Alert (Telegram) |
|---|---|
| Rasio 5xx | > 1% selama 5 menit |
| Latensi API p95 | > 500 ms selama 10 menit |
| Disk host / volume RustFS | > 80% |
| Backup pgBackRest / restic | Gagal atau tidak jalan > 26 jam |
| Arsip WAL | Tertunda > 5 menit |
| Antrian River | Job tertua > 10 menit / job gagal permanen |
| Antrian moderasi | Ada kasus > 48 jam (ringkasan harian, bukan alert malam) |
| Sertifikat TLS | Kedaluwarsa < 14 hari |
| Uptime eksternal | Situs tidak merespons 2 menit |

Dashboard Grafana: RED per endpoint (rate, error, duration), pool DB, query lambat, Valkey hit ratio,
antrian River, koneksi SSE, Web Vitals nyata dari pengguna (dikirim lewat beacon, disimpan di VictoriaMetrics).

## 7. Pengujian performa

- **k6** dengan skenario: tamu menjelajah katalog, member scroll feed + apresiasi, autosave editor,
  upload, ledakan login pagi hari, 3.000 koneksi SSE.
- Dijalankan sebelum go-live dan sebelum rilis besar di server staging dengan spek sama.
- **Lighthouse CI** di pipeline untuk landing, karya, blog, profil (gagal jika di bawah target §1.1).
- Cek ukuran bundle per route di CI (gagal jika melewati anggaran).

---

## 8. SEO & share

- Halaman publik di-render server, URL kanonik (`<link rel="canonical">`), slug salah → 301.
- `sitemap.xml` (indeks: konten, profil, grup), diperbarui tiap jam oleh worker.
- `robots.txt`: blokir `/studio`, `/moderasi`, `/admin`, `/api`, `/pengaturan`, `/notifikasi`, `/tersimpan`.
- Open Graph & Twitter Card untuk konten, profil, grup. Gambar OG otomatis (`next/og`) bila tidak ada cover.
- JSON-LD: `CreativeWork` (portfolio), `BlogPosting` (blog), `ProfilePage`/`Person`, `BreadcrumbList`,
  `EducationalOrganization` (beranda).
- `lang="id"`, judul & deskripsi unik per halaman.

## 9. Aksesibilitas

- **WCAG 2.2 AA**. Komponen shadcn/ui (berbasis Radix) sudah aksesibel; komponen kustom wajib mengikuti pola ARIA yang sama.
- Kontras ≥ 4,5:1, fokus selalu terlihat, target sentuh ≥ 44 px, semua aksi bisa via keyboard.
- `alt` gambar diminta di editor (peringatan jika kosong), caption video/embed disarankan.
- Menghormati `prefers-reduced-motion` dan `prefers-color-scheme` (light/dark).
- Diuji dengan axe (otomatis di Playwright) + uji manual pembaca layar (TalkBack / VoiceOver) di flow utama.

## 10. Kompatibilitas

| | Dukungan |
|---|---|
| Browser | 2 versi terakhir Chrome, Edge, Firefox, Safari. Chrome Android ≤ 2 tahun. iOS Safari 16+ |
| Layar | Lebar minimal 360 px, responsif sampai layar lebar (konten dibatasi lebar baca) |
| Bahasa | Bahasa Indonesia. Tanggal/angka format `id-ID`, zona waktu Asia/Jakarta |
| Tema | Light & dark (mengikuti sistem, bisa diganti user) |
