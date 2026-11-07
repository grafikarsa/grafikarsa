# Grafikarsa Revamp — Desain API

> Katalog endpoint + konvensi. Saat implementasi, ini diterjemahkan ke `packages/contracts/openapi.yaml`
> (contract-first, lihat PRD §14.3). Semua path di bawah relatif ke **`/api/v1`**.

---

## 1. Konvensi

| Aspek | Aturan |
|---|---|
| Format | JSON UTF-8. Nama field **camelCase**. Waktu RFC 3339 UTC (`2026-11-07T03:21:00Z`) |
| ID | UUID untuk resource internal. Konten publik dirujuk dengan `publicId` (8 karakter) |
| Versi | Prefix `/v1`. Perubahan yang merusak klien = `/v2`. Field baru boleh ditambah kapan saja |
| Auth | Cookie sesi `__Host-gk_session` (HttpOnly, Secure, SameSite=Lax). Tidak ada token di header |
| CSRF | Semua method selain GET/HEAD wajib header `X-CSRF-Token` yang cocok dengan cookie `__Host-gk_csrf` (double-submit) |
| Paginasi | Cursor: `?cursor=<opaque>&limit=20` (maks 50). Respons `{ "items": [...], "nextCursor": "..." \| null }` |
| Filter & urut | Query string (`?type=portfolio&sort=popular&period=7d`) |
| Idempotensi | Header `Idempotency-Key` (UUID) **wajib** untuk aksi yang tidak boleh dobel: submit, upload, kirim komentar, impor, eksekusi kenaikan. Disimpan 24 jam di Valkey |
| Konkurensi | Draft memakai `lockVersion` di body. Bentrok → `409 LOCK_CONFLICT` berisi versi server |
| Cache | GET publik mengirim `ETag` + `Cache-Control`. Respons yang bergantung pada viewer: `Cache-Control: private, no-store` |
| Rate limit | Header `RateLimit-Limit`, `RateLimit-Remaining`, `RateLimit-Reset`. Lewat batas → `429` + `Retry-After` |
| Request ID | Setiap respons membawa `X-Request-Id` (juga ada di log & trace) |
| Kompresi | gzip/zstd oleh Caddy |

### 1.1 Format error (RFC 9457 `application/problem+json`)

```json
{
  "type": "https://grafikarsa.com/errors/validation-failed",
  "title": "Data tidak valid",
  "status": 422,
  "code": "VALIDATION_FAILED",
  "detail": "Beberapa field perlu diperbaiki.",
  "errors": [
    { "field": "title", "code": "too_short", "message": "Judul minimal 3 karakter." },
    { "field": "body.blocks[2]", "ref": "0193…", "code": "media_missing", "message": "Gambar belum selesai diunggah." }
  ],
  "requestId": "01J…"
}
```

| HTTP | `code` | Kapan |
|---|---|---|
| 400 | `BAD_REQUEST` | JSON rusak, parameter salah format |
| 401 | `UNAUTHENTICATED` | Belum login / sesi habis |
| 403 | `ACTIVATION_REQUIRED` | Akun belum aktivasi (hanya endpoint aktivasi yang boleh) |
| 403 | `PASSWORD_CHANGE_REQUIRED` | Login dengan sandi sementara |
| 403 | `CSRF_FAILED` | Token CSRF salah/tidak ada |
| 403 | `CONTENT_BLOCKED` | Capability `content.create` dicabut |
| 404 | `NOT_FOUND` | Tidak ada **atau tidak berwenang melihat** (tidak membocorkan keberadaan) |
| 409 | `CONFLICT` | Duplikat (username terpakai, dll) |
| 409 | `LOCK_CONFLICT` | Draft sudah diubah editor lain |
| 409 | `INVALID_TRANSITION` | Aksi lifecycle tidak valid untuk status saat ini (`detail` menyebut status sekarang) |
| 409 | `LIMIT_REACHED` | Batas setting tercapai (draf, in_review, author, pin, dll) |
| 413 | `PAYLOAD_TOO_LARGE` | File/badan request melebihi batas |
| 415 | `UNSUPPORTED_MEDIA_TYPE` | Tipe file tidak diizinkan (dicek dari magic bytes) |
| 422 | `VALIDATION_FAILED` | Validasi field/blok |
| 429 | `RATE_LIMITED` | Terlalu banyak request |
| 500 | `INTERNAL` | Error tak terduga (detail hanya di log) |
| 503 | `UNAVAILABLE` | Dependensi down / maintenance |

### 1.2 Akses

Kolom **Akses** di katalog:
- **Publik**: tanpa login.
- **Member**: login + aktif + sudah aktivasi.
- **Author**: author accepted konten terkait (level disebut jika perlu: Pemilik / Editor).
- **Mod**: punya `moderation.review` (scope jurusan dicek per kasus).
- **AdmJur**: Admin Jurusan, data otomatis dibatasi ke jurusannya.
- **Admin**: admin global.

### 1.3 Bentuk resource utama

```jsonc
// ContentCard (dipakai feed, katalog, profil, grup)
{
  "publicId": "a7k2x9pq", "type": "portfolio", "slug": "poster-hut-80", "title": "Poster HUT 80",
  "excerpt": "...", "cover": { "url": "/media/...", "w": 1200, "h": 675, "blurhash": "...", "color": "#1e293b" },
  "category": { "slug": "poster", "name": "Poster" },
  "authors": { "preview": [UserMini, UserMini, UserMini], "total": 49 },
  "stats": { "appreciations": 120, "comments": 8 },
  "viewer": { "appreciated": true, "bookmarked": false },          // hanya jika login
  "readingMinutes": null, "firstPublishedAt": "..."
}
// UserMini
{ "username": "rafa", "displayName": "Rafa Pradana", "avatar": { "url": "...", "blurhash": "..." }, "accountType": "student", "status": "active" }
```

---

## 2. Autentikasi & akun

| Method | Path | Akses | Keterangan |
|---|---|---|---|
| GET | `/auth/csrf` | Publik | Set cookie CSRF, kembalikan token |
| POST | `/auth/login` | Publik | `{ identifier, password }` (NISN/NIP/username). Respons berisi `nextStep`: `activate` \| `change_password` \| `none`. Rate limit ketat + lockout bertahap |
| POST | `/auth/logout` | Member | Cabut sesi ini |
| GET | `/auth/session` | Publik | `null` atau ringkasan user + `nextStep` + permission efektif (untuk gating UI) |
| GET | `/auth/username-availability?u=` | Login (belum aktivasi juga) | Cek format, reserved, terpakai |
| POST | `/auth/activate` | Login belum aktivasi | `{ username, newPassword }` → respons berisi **8 recovery code** (sekali tampil) |
| POST | `/auth/recovery-codes/confirm` | Login | Konfirmasi sudah disimpan → aktivasi selesai |
| POST | `/auth/password/recover` | Publik | `{ identifier, recoveryCode, newPassword }`. Mencabut semua sesi |
| POST | `/auth/password/change` | Login | `{ currentPassword?, newPassword }`. `currentPassword` wajib kecuali `password_kind = temporary` |
| POST | `/auth/recovery-codes/regenerate` | Member | `{ password }` → 8 kode baru, batch lama hangus |
| GET | `/auth/sessions` | Member | Daftar sesi aktif (perangkat, IP kasar, terakhir aktif) |
| DELETE | `/auth/sessions/{id}` | Member | Cabut satu sesi |
| POST | `/auth/sessions/revoke-others` | Member | Keluar dari semua perangkat lain |

## 3. Profil saya

| Method | Path | Akses | Keterangan |
|---|---|---|---|
| GET | `/me` | Member | Data lengkap milik sendiri |
| PATCH | `/me/profile` | Member | `displayName`, `headline`, `bio`, `city`, `teachingSubjects` |
| PUT / DELETE | `/me/avatar` | Member | `{ mediaId }` (purpose `avatar`) |
| PUT / DELETE | `/me/banner` | Member | `{ mediaId }` (purpose `banner`) |
| PUT | `/me/links` | Member | Ganti seluruh daftar link (maks 10) |
| PUT | `/me/skills` | Member | Ganti daftar skill berurutan (maks 20) |
| PUT | `/me/pinned` | Member | `{ publicIds: [...] }` maks 3, harus konten tayang di mana user author accepted |
| PATCH | `/me/username` | Member | 1× per 30 hari, username lama redirect 90 hari |
| GET / PUT | `/me/notification-preferences` | Member | Per tipe (tipe keamanan & moderasi tidak bisa dimatikan) |
| GET | `/me/experiences` | Member | |
| POST | `/me/experiences` | Member | |
| PATCH / DELETE | `/me/experiences/{id}` | Member | Entri sistem (SMKN 4 Malang) ditolak |
| GET | `/me/bookmarks` | Member | Konten tersimpan (hanya yang masih tayang) |
| GET / POST | `/me/blog-series` | Member | |
| PATCH / DELETE | `/me/blog-series/{id}` | Member | |

## 4. Pengguna (publik)

| Method | Path | Akses | Keterangan |
|---|---|---|---|
| GET | `/users/{username}` | Publik | Profil + badge + skill (+ jumlah endorse) + statistik + konten disematkan + `viewer.following`. Username lama → `301` ke yang baru |
| GET | `/users/{username}/contents?type=&relation=all\|owner\|collab&sort=&cursor=` | Publik | Tab Portfolio / Blog / Kolaborasi |
| GET | `/users/{username}/experiences` | Publik | |
| GET | `/users/{username}/followers?cursor=` | Publik | |
| GET | `/users/{username}/following?cursor=` | Publik | |
| PUT / DELETE | `/users/{username}/follow` | Member | Idempoten |
| PUT / DELETE | `/users/{username}/skills/{skillSlug}/endorsement` | Member | Tidak bisa endorse diri sendiri |
| GET | `/users/suggestions` | Member | Saran untuk diikuti |

## 5. Konten (publik & interaksi)

| Method | Path | Akses | Keterangan |
|---|---|---|---|
| GET | `/contents?type=&sort=latest\|popular&period=7d\|30d\|all&category=&major=&tool=&skill=&tag=&context=&cursor=` | Publik | Katalog Karya & daftar Blog |
| GET | `/contents/{publicId}` | Publik* | Versi tayang + author + stats + `viewer`. *Status lain hanya untuk author & mod (dengan `status`), selain itu 404 |
| GET | `/contents/{publicId}/related` | Publik | Konten terkait (kategori/tag/author sama), maks 6 |
| POST | `/contents/{publicId}/view` | Publik | Beacon view (§4 algorithms), selalu `204` |
| POST | `/contents/{publicId}/share` | Publik | Beacon share, `204` |
| PUT / DELETE | `/contents/{publicId}/appreciation` | Member | Idempoten |
| PUT / DELETE | `/contents/{publicId}/bookmark` | Member | Idempoten |
| GET | `/contents/{publicId}/comments?cursor=` | Publik | Komentar level atas + 3 balasan pertama |
| POST | `/contents/{publicId}/comments` | Member | `{ body, parentId? }`. Filter kata, mention diproses |
| GET | `/comments/{id}/replies?cursor=` | Publik | |
| PATCH | `/comments/{id}` | Member (penulis) | Edit dalam 15 menit setelah kirim |
| DELETE | `/comments/{id}` | Member (penulis / pemilik konten) | Soft delete dengan status sesuai pelaku |

## 6. Feed, pencarian, taksonomi

| Method | Path | Akses | Keterangan |
|---|---|---|---|
| GET | `/feed?tab=following\|major\|latest\|trending&type=&cursor=` | Publik (`latest`, `trending`) / Member | |
| GET | `/search?q=&type=content\|user\|group&contentType=&cursor=` | Publik | Tanpa `type` = ringkasan gabungan |
| GET | `/search/suggest?q=` | Publik | Autocomplete (maks 5 per kategori) |
| GET | `/taxonomy/categories?type=portfolio\|blog` | Publik | |
| GET | `/taxonomy/skills?q=` · `/taxonomy/tools?q=` · `/taxonomy/tags?q=` | Publik | Autocomplete |
| GET | `/moderation-reasons?appliesTo=` | Publik | Untuk dialog laporan & keputusan |
| GET | `/majors` | Publik | Daftar jurusan + ringkasan grup |

## 7. Grup

| Method | Path | Akses | Keterangan |
|---|---|---|---|
| GET | `/groups/majors/{code}` | Publik | Info grup jurusan + pengelola |
| GET | `/groups/majors/{code}/contents?type=&category=&sort=&cursor=` | Publik | Mirror isi grup |
| GET | `/groups/majors/{code}/members?cursor=` | Publik | |
| GET | `/groups/majors/{code}/classes` | Publik | Kelas aktif + riwayat |
| GET | `/groups/classes/{year}/{slug}` | Publik | `year` = `2026-2027` |
| GET | `/groups/classes/{year}/{slug}/contents?…` | Publik | |
| GET | `/groups/classes/{year}/{slug}/members?cursor=` | Publik | |
| PATCH | `/groups/{id}` | Pengelola / AdmJur / Admin | `avatarMediaId`, `bannerMediaId`, `description`. Grup `history`: hanya Admin |

## 8. Studio (ruang kerja author)

| Method | Path | Akses | Keterangan |
|---|---|---|---|
| GET | `/studio/contents?status=&type=&cursor=` | Member | Konten di mana user author accepted |
| POST | `/studio/contents` | Member | `{ type }` → konten `draft` + draft kosong. Cek batas draf |
| GET | `/studio/contents/{id}` | Author | Draft + status + kasus moderasi terakhir (keputusan, catatan, anotasi) + author |
| PUT | `/studio/contents/{id}/draft` | Pemilik / Editor | Autosave seluruh draft + `lockVersion`. Ditolak saat `in_review` / `rejected` |
| POST | `/studio/contents/{id}/submit` | Pemilik | Validasi penuh → versi `pending` + kasus moderasi. `Idempotency-Key` wajib |
| POST | `/studio/contents/{id}/withdraw` | Pemilik | Hanya selama kasus belum di-claim |
| POST | `/studio/contents/{id}/revision` | Pemilik / Editor | Mulai revisi atas konten tayang (draft diisi dari versi tayang) |
| DELETE | `/studio/contents/{id}/revision` | Pemilik | Buang revisi yang belum dikirim |
| POST | `/studio/contents/{id}/archive` | Pemilik | Ditolak jika ada revisi `in_review` |
| POST | `/studio/contents/{id}/unarchive` | Pemilik | Kembali tayang tanpa review |
| DELETE | `/studio/contents/{id}` | Pemilik | Soft delete (30 hari) |
| POST | `/studio/contents/{id}/restore` | Pemilik | Kembali ke `status_before_delete` |
| GET | `/studio/contents/{id}/versions` | Author | Riwayat versi + keputusan |
| GET | `/studio/contents/{id}/versions/{versionId}` | Author | Snapshot lengkap |
| POST | `/studio/contents/{id}/appeal` | Pemilik | `{ message }`, sekali per penolakan |
| GET | `/studio/contents/{id}/authors` | Author | |
| POST | `/studio/contents/{id}/authors` | Pemilik | Undang `{ userId, level, roleLabel? }` |
| PATCH | `/studio/contents/{id}/authors/{userId}` | Pemilik | `level`, `roleLabel`, `position` |
| DELETE | `/studio/contents/{id}/authors/{userId}` | Pemilik | Keluarkan / batalkan undangan |
| POST | `/studio/contents/{id}/transfer-ownership` | Pemilik | `{ userId }` (harus author accepted). Pemilik lama jadi Editor |
| POST | `/studio/contents/{id}/leave` | Editor / Kontributor | Keluar dari konten |
| GET | `/studio/invitations?cursor=` | Member | Undangan masuk |
| POST | `/studio/invitations/{contentId}/accept` | Member | |
| POST | `/studio/invitations/{contentId}/decline` | Member | |
| GET | `/studio/summary` | Member | Hitungan per status + undangan pending (badge menu) |

## 9. Media

| Method | Path | Akses | Keterangan |
|---|---|---|---|
| POST | `/media` | Member | `multipart/form-data`: `file`, `purpose`. Di-stream ke RustFS. Respons `202` + `MediaAsset` (`status: processing`). `Idempotency-Key` wajib |
| GET | `/media/{id}` | Pemilik media / Author konten terkait / Mod | Status + daftar varian (untuk polling editor) |
| GET | `/media/{id}/{variant}` | Sama dengan di atas | Stream varian privat (draft). Media publik dilayani Caddy di `/media/...` |
| DELETE | `/media/{id}` | Pemilik media | Hanya jika tidak direferensikan |

## 10. Notifikasi & realtime

| Method | Path | Akses | Keterangan |
|---|---|---|---|
| GET | `/notifications?cursor=` | Member | |
| GET | `/notifications/unread-count` | Member | |
| POST | `/notifications/read` | Member | `{ ids: [...] }` atau `{ all: true }` |
| GET | `/stream` | Member | **SSE**: event `unread_count`, `studio_changed` (status konten berubah), `media_ready`. Heartbeat 25 detik |

## 11. Laporan

| Method | Path | Akses | Keterangan |
|---|---|---|---|
| POST | `/reports` | Member | `{ targetType, targetId, reasonId, note? }`. Satu laporan terbuka per pelapor per target |

## 12. Moderasi

| Method | Path | Akses | Keterangan |
|---|---|---|---|
| GET | `/moderation/queue?major=&kind=&age=&cursor=` | Mod | Kasus `open`/`claimed` sesuai scope. Kasus di mana mod adalah author tidak muncul |
| GET | `/moderation/cases/{id}` | Mod | Snapshot + diff (revisi) + riwayat versi & keputusan sebelumnya + info author |
| POST | `/moderation/cases/{id}/claim` | Mod | Claim / perpanjang claim |
| POST | `/moderation/cases/{id}/release` | Mod (pemegang claim) | |
| POST | `/moderation/cases/{id}/decision` | Mod (pemegang claim) | `{ decision, reasonId?, note?, annotations: [{ targetRef, note }] }` |
| GET | `/moderation/reports?status=&targetType=&cursor=` | Mod | |
| GET | `/moderation/reports/{id}` | Mod | Detail + target + laporan lain untuk target yang sama |
| POST | `/moderation/reports/{id}/resolve` | Mod | `{ action, reasonId?, note? }` (`dismiss` atau tindakan di bawah) |
| POST | `/moderation/contents/{publicId}/take-down` | Mod | `{ reasonId, note }` |
| POST | `/moderation/comments/{id}/hide` · `/restore` | Mod | |
| POST | `/moderation/users/{username}/reset` | Mod | `{ fields: ["avatar","banner","bio","headline"], reasonId, note }` |
| DELETE | `/moderation/experiences/{id}` | Mod | Hapus experience yang melanggar |
| POST | `/moderation/groups/{id}/reset` | Mod | `{ fields: [...] }` |
| GET | `/moderation/history?moderator=&cursor=` | Mod | Log keputusan & tindakan |
| GET | `/moderation/stats?period=` | Mod / AdmJur / Admin | Waktu tinjau, % ≤ 48 jam, per moderator |

## 13. Admin

### 13.1 Pengguna & akun

| Method | Path | Akses | Keterangan |
|---|---|---|---|
| GET | `/admin/users?type=&status=&major=&class=&q=&cursor=` | AdmJur / Admin | |
| POST | `/admin/users` | AdmJur / Admin | Buat satu akun (guru/tendik/siswa) |
| GET | `/admin/users/{id}` | AdmJur / Admin | Data akademik, role, sesi aktif, riwayat moderasi |
| PATCH | `/admin/users/{id}` | AdmJur / Admin | Nama, NISN/NIP, tipe, jurusan diampu |
| POST | `/admin/users/{id}/deactivate` · `/reactivate` | AdmJur / Admin | Nonaktif = cabut semua sesi |
| POST | `/admin/users/{id}/reset-password` | AdmJur / Admin | Sandi sementara (sekali tampil, 72 jam) |
| POST | `/admin/users/{id}/reissue-initial-password` | AdmJur / Admin | Untuk akun belum aktivasi |
| POST | `/admin/users/{id}/content-block` · `/content-unblock` | Admin | Cabut / pulihkan `content.create` |
| POST | `/admin/users/{id}/anonymize` | Admin | Penghapusan akun (database.md §8). Wajib konfirmasi ketik username |
| POST | `/admin/imports` | AdmJur / Admin | `multipart`: `kind`, `file` (CSV) → validasi async |
| GET | `/admin/imports/{id}` | AdmJur / Admin | Hasil validasi per baris |
| POST | `/admin/imports/{id}/commit` | AdmJur / Admin | Terapkan. `Idempotency-Key` wajib |
| GET | `/admin/account-cards?classId=` | AdmJur / Admin | PDF kartu akun (NISN + sandi awal **yang belum dipakai**). Dicatat di audit log |

### 13.2 Role & izin

| Method | Path | Akses |
|---|---|---|
| GET | `/admin/permissions` | Admin |
| GET / POST | `/admin/roles` | Admin |
| PATCH / DELETE | `/admin/roles/{id}` | Admin (role sistem tidak bisa dihapus) |
| POST | `/admin/users/{id}/roles` | Admin / AdmJur (hanya role & scope di bawah wewenangnya) |
| DELETE | `/admin/user-roles/{id}` | Admin / AdmJur |

### 13.3 Akademik & kenaikan kelas

| Method | Path | Akses | Keterangan |
|---|---|---|---|
| GET / POST | `/admin/majors` | Admin | Membuat jurusan = membuat grup jurusan |
| PATCH | `/admin/majors/{id}` | Admin | Nonaktif = grup jadi riwayat |
| GET / POST | `/admin/academic-years` | Admin | |
| PATCH | `/admin/academic-years/{id}` | Admin | |
| GET | `/admin/classes?academicYear=&major=` | AdmJur / Admin | |
| POST | `/admin/classes` | AdmJur / Admin | Membuat kelas = membuat grup kelas |
| PATCH / DELETE | `/admin/classes/{id}` | AdmJur / Admin | Hapus hanya jika belum punya enrollment |
| POST | `/admin/promotions` | Admin | `{ toAcademicYearId }` → job draft + pemetaan + item default |
| GET | `/admin/promotions` · `/admin/promotions/{id}` | AdmJur / Admin | |
| GET | `/admin/promotions/{id}/items?action=&major=&class=&exception=&cursor=` | AdmJur / Admin | |
| PUT | `/admin/promotions/{id}/mappings` | Admin | Pemetaan rombel lama → baru |
| PATCH | `/admin/promotions/{id}/items/{itemId}` | AdmJur / Admin | Ubah aksi / rombel tujuan / catatan |
| POST | `/admin/promotions/{id}/items/import` | AdmJur / Admin | CSV pengecualian hasil rapat |
| POST | `/admin/promotions/{id}/dry-run` | AdmJur / Admin | Ringkasan + konflik |
| POST | `/admin/promotions/{id}/approve` · `/unapprove` | Admin | Approve ditolak jika konflik > 0 |
| POST | `/admin/promotions/{id}/execute` | Admin | Hanya jika tanggal sudah lewat dan job approved tapi belum jalan. `Idempotency-Key` wajib |
| POST | `/admin/promotions/{id}/rollback` | Admin | Dalam 14 hari setelah eksekusi |
| POST | `/admin/promotions/{id}/cancel` | Admin | Hanya draft/approved |

### 13.4 Grup, badge, taksonomi, sistem

| Method | Path | Akses |
|---|---|---|
| GET | `/admin/groups?kind=&status=&q=` | AdmJur / Admin |
| POST / DELETE | `/admin/groups/{id}/managers` · `/admin/groups/{id}/managers/{userId}` | AdmJur / Admin |
| GET / POST / PATCH | `/admin/badges` · `/admin/badges/{id}` | Admin |
| POST | `/admin/badges/{id}/awards` | Admin (`{ userIds, note, dedupeKey, expiresAt? }`) |
| DELETE | `/admin/user-badges/{id}` | Admin |
| CRUD | `/admin/taxonomy/categories`, `/skills`, `/tools`, `/tags` (+ `merge`, `block`) | Admin |
| CRUD | `/admin/moderation-reasons`, `/admin/word-filters`, `/admin/reserved-usernames` | Admin |
| GET | `/admin/appeals?status=` | Admin |
| POST | `/admin/appeals/{id}/decision` | Admin (`upheld` / `overturned` + catatan) |
| GET / PATCH | `/admin/settings` | Admin |
| GET | `/admin/audit-logs?actor=&action=&target=&from=&to=&cursor=` | Admin (AdmJur: scope jurusannya) |
| GET | `/admin/dashboard` | AdmJur / Admin |

## 14. Fase 2

| Method | Path | Akses |
|---|---|---|
| GET | `/curations` · `/curations/{slug}` | Publik |
| CRUD | `/admin/curations`, `/admin/curations/{id}/items` | Guru (dengan izin) / Admin |
| CRUD | `/assignments` (guru), `/assignments/{id}/submissions`, `/submissions/{id}/assessment` | Guru / Siswa terkait |
| CRUD | `/rubrics` | Guru |

## 15. Operasional (tidak lewat Caddy publik)

| Method | Path | Keterangan |
|---|---|---|
| GET | `/healthz` | Liveness (proses hidup) |
| GET | `/readyz` | Postgres + Valkey + RustFS terjangkau (dipakai deploy gate) |
| GET | `/metrics` | OpenTelemetry/Prometheus, hanya jaringan internal |
| POST | `web:/internal/revalidate` | Dipanggil api ke Next.js: `{ tags: [...] }` + shared secret |

---

## 16. Rate limit default (setting `ratelimit.*`)

| Aksi | Batas |
|---|---|
| Login | 5 / 15 menit per identitas + 20 / 15 menit per IP, lalu lockout bertahap |
| Lupa sandi (recovery code) | 5 / jam per identitas |
| Aktivasi / ganti sandi | 10 / jam per user |
| Komentar | 10 / menit, 200 / hari per user |
| Apresiasi / simpan / follow | 60 / menit per user |
| Undangan kolaborasi | 50 / jam per user |
| Laporan | 20 / jam per user |
| Upload | 60 / jam per user, total 1 GB / hari |
| Pencarian | 30 / menit per pelaku |
| Beacon view/share | 120 / menit per pelaku |
| Lainnya (baca) | 600 / menit per pelaku |
