# Grafikarsa v2 — API

Dokumen ini ringkas saja: **daftar endpoint tidak ditulis manual di sini** agar
tidak pernah stale. Sumber kebenarannya adalah kode, dan kontraknya di-generate:

```bash
make openapi        # regenerate packages/contracts/openapi.yaml
make openapi-check  # verifikasi kontrak tidak drift (dipakai CI)
```

Kontrak OpenAPI (semua endpoint, security, permission, CSRF):
`packages/contracts/openapi.yaml` — 118 path / 146 operasi.

## Topologi

```
Browser → Caddy (:80/:443) → web Next.js (:3000) → BFF /api/* → api Go (:8080, internal)
```

Backend, database, cache, dan object storage **tidak** diekspos publik. Semua
panggilan browser melewati Route Handler di `apps/web/app/api/[...path]/route.ts`
yang meneruskan cookie (`__Host-rt`, `__Host-csrf`), `Range`, dan ETag.

## Envelope respons

```jsonc
// sukses
{ "success": true, "data": { /* payload */ }, "meta": { /* paginasi */ }, "request_id": "..." }
// gagal
{ "success": false, "error": { "code": "VALIDATION_ERROR", "message": "...", "details": [/* opsional */] }, "request_id": "..." }
```

Kode error stabil ada di `components.schemas.ErrorCode` pada kontrak.

## Autentikasi & sesi

| Aspek | Ketentuan |
|---|---|
| Access token | JWT HS256, **15 menit**, hanya di memori FE (`lib/auth-store.ts`), tidak di localStorage |
| Refresh token | Opaque, cookie `__Host-rt` (HttpOnly, Secure, SameSite=Lax), **idle 7 hari + absolut 30 hari** |
| Rotasi | Setiap refresh membuat sesi baru (`rotated_to`); token lama dipakai lagi → seluruh family dicabut + audit `reuse_detected` |
| CSRF | Double-submit: cookie `__Host-csrf` + header `X-CSRF-Token`; token = `HMAC(secret, hari:user_id)` sehingga terikat user; wajib di **semua** mutasi kecuali login, refresh, dan feedback guest |
| Claim JWT | `role_primary`, `scope_major_id` (batas jurusan), `mcp` (wajib ganti password) |
| Wajib ganti password | `mcp=true` memblokir seluruh endpoint kecuali `/v1/me`, `/v1/me/password`, `/v1/auth/logout*`, `/v1/auth/sessions` |
| Otorisasi | `requirePerm` / `requireAnyPerm`; role `admin` melewati daftar permission, `admin_jurusan` **selalu** difilter row-level `scope_major_id` (fail-closed) |

## Rate limit & anti-abuse

| Cakupan | Batas |
|---|---|
| Global `/v1` | `RATE_LIMIT_REQUESTS` (default **300**/menit), dikunci per **user** (login) atau IP asli klien dari BFF (`x-client-ip`) |
| Login | 5 gagal → Turnstile; 10 gagal → kunci 15 menit (per username+IP); 50 percobaan/10 menit per IP |
| Upload | 20 file/jam dan 100 file/hari per user, kuota 200 MB |
| Feedback guest | Turnstile wajib (bila `TURNSTILE_SECRET_KEY` diisi) |
| Recompute leaderboard | 1×/jam |

Counter gagal login hanya bertambah saat password benar-benar salah, sehingga
tidak bisa dipakai mengunci akun orang lain.

## Media

- Upload: `POST /v1/media/upload` (multipart, `purpose` ∈ avatar/banner/thumbnail/block_image/document/feedback/changelog_cover).
- Gambar di-decode ulang (EXIF/GPS hilang), dinormalkan ke JPEG q82 (PNG transparan tetap PNG), disimpan bersama varian `?w=320` / `?w=800`.
- Dedup sha256 per owner+purpose; magic bytes diperiksa (header klien tidak dipercaya); SVG/HTML ditolak.
- Lifecycle: `pending` → `confirmed` saat dikaitkan (confirm `link_to` atau PATCH portfolio/block). Cleanup hanya menghapus media `pending` yang tidak direferensikan siapa pun.
- Delete menghapus objek utama + semua varian dari storage.
- Media non-publik bisa diakses tamu hanya bila dipakai portfolio `published`.

## PDF & karantina media

- **Preview PDF**: worker `pdf-preview` merender halaman pertama PDF memakai
  `pdftoppm` (poppler-utils; sudah ada di image) menjadi media JPEG dengan
  varian 320/800. FE menampilkannya lewat `GET /v1/media/{id}?preview=1`
  (visibilitas mengikuti media induknya). Bila poppler tidak ada, job dilewati
  dengan peringatan dan FE hanya menampilkan tautan file.
- **Karantina**: `scan-media` memeriksa objek media `confirmed` (HeadObject +
  ukuran) dan mengubah statusnya menjadi `quarantined` bila objek hilang atau
  berbeda ukuran. Admin melihatnya di `/admin/media/quarantine` dan memulihkan
  dengan `POST /v1/admin/media/{id}/unquarantine` (menolak bila objek belum ada).

## Statistik & minat

- `stats` berjalan **inkremental**: hanya portfolio yang aktivitasnya lebih baru
  dari `portfolio_stats.computed_at` (full sweep tiap 12 run) dan memakai satu
  query agregat, bukan 4 query per portfolio. Penyebut normalisasi dihitung dari
  seluruh katalog agar skor tetap sebanding.
- Bobot minat (`user_interests`) diluruhkan 0.98/hari oleh job `decay-interests`.

## Batas anti-abuse sosial

| Aksi | Batas |
|---|---|
| Suka | 20/jam per akun (lebih dari itu ditolak + audit) |
| Komentar | 5/menit dan 30/jam per akun |
| Mengikuti | 60/jam per akun |
| Balasan guest | 10/jam per thread+IP |

## Job terjadwal (worker)

| Job | Interval |
|---|---|
| Outbox dispatcher (notifikasi, fan-out changelog, recompute) | 30 detik |
| Recompute `portfolio_stats` | 5 menit |
| Publish changelog terjadwal | 5 menit |
| Cleanup media pending | 1 jam |
| Retensi notifikasi (90 hari; feedback & pengumuman 1 tahun) | 24 jam |
| Recompute leaderboard (weekly + monthly, semua scope) | harian 02:05 WIB |
| Draft kenaikan kelas | harian 00:10 WIB |
| Digest notifikasi harian | harian 07:00 WIB |
| Flush view counter (HyperLogLog → agregat harian) | 1 menit |
| Peluruhan bobot minat (decay 0.98/hari) | harian 03:00 WIB |
| Pemindaian integritas media (karantina) | harian 03:30 WIB |
| Preview PDF halaman-1 | 30 menit |

Jalankan manual:
`go run ./cmd/worker <outbox|stats|leaderboard|cleanup-media|publish-changelogs|retention|promotion-draft|digest|flush-views|decay-interests|scan-media|pdf-preview|serve>`.

## View counter

View dicatat di **HyperLogLog Dragonfly** (`views:hll:<portfolio>:<hari>`, unik
per pengunjung/hari) lalu di-flush tiap menit ke `portfolio_view_counts`. Tidak
ada INSERT per request; statistik membaca agregat harian ini.

## Migrasi v1 → v2

| Perintah | Fungsi |
|---|---|
| `migrate-old --v1 <dsn> --v2 <dsn> --dry-run` | Hitung + validasi tanpa menulis |
| `migrate-old … --execute` | Tulis ke v2 (idempotent via `legacy_id_map`) |
| `migrate-old … --execute --migrate-media` | Daftarkan URL `/storage/*` lama ke `media_assets` + `legacy_urls` |

Block yang gagal validasi masuk `quarantine_blocks` (tidak hilang diam-diam).
URL lama: `GET /v1/storage/*` → 301 ke `/v1/media/by-key/*` bila dikenal, 410
bila tidak (6 bulan, PRD §16.1).

## Endpoint yang sering dipakai FE

| Method | Path | Deskripsi |
|---|---|---|
| GET/POST | `/v1/search` | Cari karya/user (GET `?q=`, POST `{"query"}`); query tanpa alfanumerik ditolak |
| GET | `/v1/majors`, `/v1/classes` | Referensi publik (kelas tahun aktif) |
| GET | `/v1/changelogs/latest`, `/v1/changelogs/version/{version}` | Rilis terbaru & halaman per versi |
| GET | `/v1/portfolios?user=&major_id=&tag_id=&search=&sort=` | Filter gabungan (latest/top/relevance) |
| GET | `/v1/admin/dashboard/stats` | Angka ringkas (scope global/jurusan) |
| GET | `/v1/admin/jobs` | Status outbox |
| GET/PATCH | `/v1/admin/settings`, `/v1/admin/settings/{key}` | `feed_weights`, `upload_limits` tanpa deploy |
| GET/POST/DELETE | `/v1/admin/changelogs/{id}/contributors` | Kontributor rilis |
| GET | `/v1/users/{username}/followers` dan `/following` | Pengikut/mengikuti |
| POST | `/v1/feedback` | Guest menerima `follow_up_token` (`?t=`, berlaku 6 bulan) untuk membuka & membalas thread-nya |
| GET | `/v1/media/{id}?preview=1` | Gambar preview halaman-1 PDF (bila sudah dibuat worker) |
| GET | `/v1/admin/media/quarantine` | Daftar media yang dikarantina |
| POST | `/v1/admin/media/{id}/unquarantine` | Pulihkan media setelah objeknya kembali |
| GET | `/v1/admin/assessments?filter=` | Antrean penilaian (assessed/unassessed, ter-scope jurusan) |
| GET/POST/DELETE | `/v1/admin/series/{id}/blocks` | Template block series |
