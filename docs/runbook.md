# Grafikarsa v2 — Runbook

## Perintah harian

| Perlu | Jalankan |
|---|---|
| Start dev | `cp .env.example .env && make dev`, terminal 2: `make dev-web` |
| Migrasi + seed | `make migrate-up && make seed && make verify-seed` |
| Cek versi migrasi | `make migrate-status` |
| Migrasi baru | `make migrate-new NAME=add_x` lalu tulis Up/Down-nya |
| Query type-safe | `make sqlc` (setelah mengubah `apps/api/db/queries/*.sql`) |
| Kontrak API | `make openapi` setelah mengubah `routes.go`; `make openapi-check` untuk memverifikasi |
| Test & lint | `make test`, `make lint` |
| Lihat log | `make dev-logs` |
| Matikan | `make dev-down` |
| Simulasi prod | `make prod` / `make prod-down` |

## Deploy & rollback

Alur `deploy.yml` (push ke `main`): build image (`sha-<commit>` + `latest`) →
scan Trivy → backup DB di server → tulis `IMAGE_TAG` ke `.env` → `goose up` →
`up -d` → healthcheck 30×10 detik → seed idempoten → prune image lama. Bila
healthcheck gagal, workflow mengembalikan `IMAGE_TAG` ke tag sebelumnya dan
menjalankan `up -d` lagi.

Rollback manual:

```bash
cd /opt/grafikarsa
sed -i 's/^IMAGE_TAG=.*/IMAGE_TAG=sha-<commit-sebelumnya>/' .env
docker compose -f docker-compose.deploy.yml up -d
```

Validasi sebelum deploy: `make deploy-check` (menolak secret default, `DOMAIN`
localhost, dan `NEXT_PUBLIC_APP_URL` localhost yang membuat semua mutasi 403).

## Restart order (server)

`db` → `dragonfly` → `storage` → `api` → `web` → `proxy`

## Observability

| Perlu | Cara |
|---|---|
| Health/readiness | `curl -s https://DOMAIN/api/v1/health`, `curl -s http://api:8080/ready` |
| Metrics | `curl -s -H "Authorization: Bearer <token admin>" http://api:8080/metrics` (butuh permission `system.jobs`) |
| Log | `docker compose logs -f api worker` (JSON via slog, ada `request_id`) |
| Backup manual | `cd /opt/grafikarsa && ./backup.sh` (pg_dump + mirror storage + retensi + warning disk) |
| Backup terjadwal | crontab: `0 2 * * * /opt/grafikarsa/backup.sh >> /var/log/grafikarsa-backup.log 2>&1` |
| Restore DB | `docker exec -i grafikarsa-db pg_restore -c -U grafikarsa -d grafikarsa < backup/db-YYYY-MM-DD.dump` |
| Recompute leaderboard | `POST /api/v1/admin/leaderboards/recompute` (permission `system.jobs`, maks 1×/jam) |
| Re-run promotion | UI `/admin/promotion` → Preview → Setujui → ketik tahun tujuan → Eksekusi |
| Kirim digest manual | `docker compose exec worker /worker digest` |
| Revoke semua sesi user | admin: reset password user (otomatis revoke + wajib ganti) atau SQL `UPDATE sessions SET revoked_at=NOW() WHERE user_id=...` |
| Media nyangkut | job `cleanup-media` (per jam) hanya menghapus media `pending` yang tidak direferensikan |

## Job terjadwal

Outbox 30 detik · stats 5 menit · publish changelog 5 menit · cleanup media 1 jam ·
retensi 24 jam · leaderboard 02:05 WIB · draft kenaikan kelas 00:10 WIB ·
digest 07:00 WIB. Semua job dibungkus `recover` sehingga panic satu job tidak
mematikan scheduler. Jalankan satu job manual:
`go run ./cmd/worker <outbox|stats|leaderboard|cleanup-media|publish-changelogs|retention|promotion-draft|digest>`.

## Catatan operasional

- **Gate wajib ganti password**: user hasil import/seed hanya bisa memakai
  `/me`, `/me/password`, `/auth/logout*`, `/auth/sessions`, dan halaman
  `/settings/password` sampai password diganti.
- **Scope jurusan**: `admin_jurusan` hanya melihat/memoderasi jurusannya
  (claim `scope_major_id` di JWT + filter SQL fail-closed). Bila scope kosong,
  cek `user_roles.scope_major_id` atau `users.major_id` — akses akan ditolak,
  bukan dibuka.
- **Eskalasi role**: role global (mis. `admin`) hanya bisa diberikan admin
  global; role lain wajib subset permission pemberi dan target harus sedalam
  scope pemberi.
- **Outbox**: `processing → sent/failed` dengan backoff; event `processing`
  yang tertinggal dikembalikan ke `pending` oleh reaper setiap kali dispatcher
  jalan (maks 5 percobaan).
- **Rate limit** dibatasi per user (bila login) atau IP asli klien dari BFF
  (`x-client-ip`); header `X-Forwarded-For` mentah tidak dipercaya.
