# Grafikarsa v2 — Runbook (Fase 0)

## Perintah harian

| Perlu | Jalankan |
|---|---|
| Start dev | `cp .env.example .env && make dev` + `make dev-web` |
| Migrasi + seed | `make migrate-up && make seed && make verify-seed` |
| Lihat log | `make dev-logs` |
| Matikan | `make dev-down` |
| Simulasi prod | `make prod` / `make prod-down` |

## Restart order (server)

`db` → `dragonfly` → `storage` → `api` → `web` → `proxy`

## Lokasi log

`docker compose logs -f <service>` (dev) / `docker compose -f docker-compose.deploy.yml logs -f` (server).

## Observability

| Perlu | Cara |
|---|---|
| Health/readiness | `curl -s https://DOMAIN/api/v1/health`, `curl -s http://api:8080/ready` |
| Metrics | `curl -s http://api:8080/metrics` (counter request + duração + goroutine/memori) |
| Log | `docker compose logs -f api worker` (JSON via slog; ada request_id) |
| Backup manual | `cd /opt/grafikarsa && ./backup.sh` (pg_dump + mirror storage + retensi) |
| Backup terjadwal | crontab: `0 2 * * * /opt/grafikarsa/backup.sh >> /var/log/grafikarsa-backup.log 2>&1` |
| Restore DB | `docker exec -i grafikarsa-db pg_restore -c -U grafikarsa -d grafikarsa < backup/db-YYYY-MM-DD.dump` |
| Rollback deploy | ubah `IMAGE_TAG` di `.env` ke `sha-<commit-sebelumnya>` lalu `docker compose -f docker-compose.deploy.yml up -d` |
| Recompute leaderboard | `POST /api/v1/admin/leaderboards/recompute` (admin, maks 1x/jam) |
| Re-run promotion | UI `/admin/promotion` → Preview → Setujui → ketik tahun tujuan → Eksekusi |
| Revoke semua sesi user | admin: reset password user (otomatis revoke) atau SQL `UPDATE sessions SET revoked_at=NOW() WHERE user_id=...` |

## Catatan operasional

- **Migrasi otomatis**: deploy workflow menjalankan `goose up` sebelum menaikkan service; bila healthcheck gagal, image di-rollback ke tag sebelumnya.
- **Gate wajib ganti password**: user hasil import/seed tidak bisa memakai API selain `/me`, `/me/password`, `/auth/logout*`, `/auth/sessions` sampai password diganti.
- **Scope jurusan**: `admin_jurusan` hanya melihat/memoderasi jurusannya (claim `scope_major_id` di JWT + filter SQL). Bila scope tidak terisi, cek `user_roles.scope_major_id` atau `users.major_id`.
- **Outbox**: status `processing → sent/failed` dengan backoff; event yang gagal diulang otomatis (maks 5x).
