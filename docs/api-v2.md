# Grafikarsa v2 — API

Dokumen ini ringkas saja: **daftar endpoint tidak ditulis manual di sini** agar
tidak pernah stale. Sumber kebenarannya adalah kode, dan kontraknya di-generate:

```bash
make openapi        # regenerate packages/contracts/openapi.yaml
make openapi-check  # verifikasi kontrak tidak drift (dipakai CI)
```

Kontrak OpenAPI (semua endpoint, security, permission, CSRF):
`packages/contracts/openapi.yaml` — 100 path / 126 operasi.

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

Jalankan manual: `go run ./cmd/worker <outbox|stats|leaderboard|cleanup-media|publish-changelogs|retention|promotion-draft|digest|serve>`.
