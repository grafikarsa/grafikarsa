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
