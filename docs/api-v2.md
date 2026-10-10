# Grafikarsa v2 — API (Fase 1)

Base URL (via BFF): `/api/v1` → `http://api:8080/v1` internal.

## Envelope

Sukses: `{ success: true, data, request_id }`
Gagal: `{ success: false, error: { code, message }, request_id }`

## Endpoint Fase 0–1

| Method | Path | Auth | Deskripsi |
|---|---|---|---|
| GET | `/health` | — | Liveness |
| GET | `/ready` | — | Readiness: `{ db, dragonfly, storage }` |
| GET | `/v1/health` | — | Liveness versi API |
| POST | `/v1/auth/login` | — | Login, set `__Host-rt` + `__Host-csrf` |
| POST | `/v1/auth/refresh` | cookie | Rotasi refresh + reuse-detection |
| POST | `/v1/auth/logout` | Bearer | Revoke sesi + blacklist access |
| POST | `/v1/auth/logout-all` | Bearer | Revoke semua sesi |
| GET | `/v1/auth/sessions` | Bearer | Daftar sesi aktif |
| GET/DELETE | `/v1/auth/sessions/:id` | Bearer | Revoke satu sesi |
| GET/PATCH | `/v1/me` | Bearer | Profil / edit profil (CSRF) |
| PATCH | `/v1/me/password` | Bearer | Ganti password (CSRF, revoke semua) |
| GET | `/v1/me/permissions` | Bearer | Union permission efektif |
| GET | `/v1/users`, `/v1/users/:username` | `users.view` | Direktori user |
| GET/POST | `/v1/admin/users` | `users.view/create` | List + buat (CSRF) |
| GET/PATCH/DELETE | `/v1/admin/users/:id` | `users.edit/delete` | Detail/ubah/hapus (CSRF) |
| POST | `/v1/admin/users/:id/activate`, `/reset-password` | khusus | Aktivasi + reset (CSRF) |
| GET/POST | `/v1/admin/roles` | `roles.view/create` | List + buat custom (CSRF) |
| POST/DELETE | `/v1/admin/roles/assign` | `roles.assign` | Assign/unassign (CSRF) |
| POST | `/v1/media/upload` | `media.upload` | Multipart (CSRF) → `{media_id}` |
| POST/DELETE | `/v1/media/:id/confirm`, `/v1/media/:id` | owner | Confirm / hapus (CSRF) |
| GET | `/v1/media/:id` | optional | Stream ETag + Range (avatar/banner publik) |

Kontrak lengkap: `packages/contracts/openapi.yaml`.

## Endpoint tambahan (pasca-audit)

| Method | Path | Auth | Deskripsi |
|---|---|---|---|
| GET | `/metrics` | — | Prometheus text (counter + runtime) |
| GET | `/v1/users` | optional | Direktori publik (tanpa email) |
| GET | `/v1/users/:username` | optional | Profil publik; email hanya self/admin/show_email |
| GET | `/v1/changelogs/:id` | optional | Detail + sections (draft hanya untuk manager) |
| GET | `/v1/changelogs/unread-count` | Bearer | Jumlah rilis belum dibaca |
| POST | `/v1/changelogs/mark-all-read` | Bearer+CSRF | Tandai semua rilis dibaca |
| POST | `/v1/admin/changelogs/:id/publish` \| `/unpublish` \| `/archive` | `changelog.publish` | Siklus rilis (unpublish >100 pembaca butuh admin) |
| PATCH | `/v1/admin/feedback/:id/triage` | `feedback.triage` | Prioritas/publikasi/spam (CSRF) |
| POST | `/v1/admin/leaderboards/recompute` | `system.jobs` | Recompute leaderboard (maks 1x/jam) |
| GET | `/v1/changelogs/rss.xml` (web) | — | RSS changelog |

### Catatan keamanan

- CSRF token terikat ke user (`HMAC(secret, hari:user_id)`) dan wajib pada semua mutasi termasuk logout/sesi.
- Refresh token: 30 hari absolut + 7 hari idle (sliding), rotasi mencatat `rotated_to`, reuse terdeteksi → seluruh family dicabut.
- Rate limit: 100 req/menit/IP global; login 5 gagal → Turnstile, 10 gagal → kunci 30 menit; upload 20/jam & 100/hari per user.
- `must_change_password` memblokir seluruh endpoint kecuali ganti password sendiri.
