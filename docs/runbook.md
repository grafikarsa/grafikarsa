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

Alur `deploy.yml`: build image (`sha-<commit>` + `latest`) → scan Trivy →
backup DB di server → tulis `IMAGE_TAG` ke `.env` → `goose up` → `up -d` →
healthcheck 30×10 detik → seed idempoten → prune image lama. Bila
healthcheck gagal, workflow mengembalikan **image dan skema** ke versi
sebelumnya lalu menjalankan `up -d` lagi.

**Kapan workflow ini jalan:** hanya dari **tag versi** (`v2.0.0`) atau
**manual** (`workflow_dispatch`).

Push ke `main` sengaja **tidak** men-deploy. Alasannya: go-live v2 punya
prasyarat yang harus selesai lebih dulu — storage mirror 5,1 GB, migrasi
DB v1→v2 di server, jendela maintenance, dan restore drill. Volume v2
(`postgres_data`, `storage_data`) berbeda nama dari volume yang sedang jalan
(`grafikarsa_postgres_data`, `grafikarsa_minio_data`), jadi data v1 tidak ikut
terhapus — tapi kalau stack v2 start sebelum dimigrasi, **situsnya kosong**.
Deploy karena itu harus tindakan yang disengaja, bukan efek samping merge.

Untuk go-live:

```bash
# 1. Maintenance mode aktif (v1 menulis beku) + storage mirror + migrasi DB.
#    Lihat bagian "Mirror storage saat go-live" dan prosedur migrasi.
# 2. Deploy (setelah migrasi selesai & diverifikasi):
git tag v2.0.0 && git push origin v2.0.0
#    atau manual: gh workflow run deploy.yml
```

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
| Backup manual | `cd /opt/grafikarsa && ./gk-backup.sh` (pg_dump + arsip volume storage + cache + inventaris + manifest sha256 + retensi + warning disk) |
| Backup terjadwal | crontab: `0 2 * * * /opt/grafikarsa/gk-backup.sh >> /var/log/grafikarsa-backup.log 2>&1` |
| Verifikasi backup | `cd /opt/grafikarsa && ./verify-backup.sh backup/<timestamp>` (checksum, arsip storage terbaca, dump pg bisa dilist — read-only) |
| Restore DB | `docker exec -i grafikarsa-db pg_restore -c -U grafikarsa -d grafikarsa < backup/db-YYYY-MM-DD.dump` |
| Recompute leaderboard | `POST /api/v1/admin/leaderboards/recompute` (permission `system.jobs`, maks 1×/jam) |
| Re-run promotion | UI `/admin/promotion` → Preview → Setujui → ketik tahun tujuan → Eksekusi |
| Kirim digest manual | `docker compose exec worker /worker digest` |
| Revoke semua sesi user | admin: reset password user (otomatis revoke + wajib ganti) atau SQL `UPDATE sessions SET revoked_at=NOW() WHERE user_id=...` |
| Media nyangkut | job `cleanup-media` (per jam) hanya menghapus media `pending` yang tidak direferensikan |
| Flush view manual | `docker compose exec worker /worker flush-views` |
| Migrasi v1→v2 | `go run ./cmd/tools/migrate-old --v1 <dsn-v1> --v2 <dsn-v2> --dry-run` lalu `--execute` |
| URL lama | `GET /v1/storage/*`: 301 bila ada di `legacy_urls`, 410 bila tidak |
| Ubah bobot feed | `/admin/settings` atau `PATCH /v1/admin/settings/feed_weights` |
| Media karantina | `/admin/media/quarantine` → "Pulihkan" setelah objek di-restore |
| Preview PDF | `docker compose exec worker /worker pdf-preview` (butuh poppler-utils) |
| Uji beban (k6) | `make test-load` |
| Uji kontrak (schemathesis) | `make test-contract` |

## Backup & restore (v2)

Backup harian dijalankan cron: `0 2 * * * /opt/grafikarsa/gk-backup.sh`.
Script-nya ada di repo (`tools/gk-backup.sh`) dan disalin ke server saat
deploy — nama file di server **harus** `gk-backup.sh` karena itu yang dipanggil
crontab.

Isi satu backup (direktori `backup/<timestamp>/`):

| Isi | Keterangan |
|---|---|
| `db/<db>-<ts>.dump` | `pg_dump -Fc` (format custom, bisa dipulihkan per tabel) |
| `storage/storage-data-<ts>.tar.gz` | seluruh volume object storage (data bucket + metadata RustFS) |
| `cache/` | snapshot Dragonfly (best effort — cache disposable) |
| `config/` | `.env`, compose, Caddyfile, daftar image yang jalan |
| `inventory/` | row count per tabel + daftar objek storage (ukuran & path) |
| `MANIFEST.txt` | checksum sha256 semua file (dicek saat restore drill) |

Volume object storage dibaca dari **dalam container helper**, bukan lewat path
host (`/var/lib/docker/volumes/...` mode 700 root). Jadi script jalan sebagai
user mana pun yang punya akses docker, dan image yang dipakai adalah image
milik service storage itu sendiri — tidak perlu menarik image tambahan.

Verifikasi (read-only, tidak menyentuh data asli):

```bash
cd /opt/grafikarsa && ./verify-backup.sh backup/<timestamp>
```

Restore drill — buktikan backup benar-benar bisa dipulihkan, jangan hanya
"exit 0":

```bash
DC="docker compose -f docker-compose.deploy.yml"

# 1. DB ke database baru (TIDAK ADA database lama yang disentuh)
$DC exec -T db psql -U "$DB_USER" -d postgres -c "create database restore_test owner $DB_USER;"
$DC exec -T db pg_restore -U "$DB_USER" -d restore_test --no-owner --no-privileges \
  < backup/<timestamp>/db/*.dump

# 2. Object storage ke direktori sementara
mkdir -p /tmp/gk-restore && tar -xzf backup/<timestamp>/storage/*.tar.gz -C /tmp/gk-restore
ls -la /tmp/gk-restore/grafikarsa/

# 3. Bandingkan jumlah objek dengan inventaris
cat backup/<timestamp>/inventory/object-count-*.txt
find /tmp/gk-restore/grafikarsa -type f | wc -l
```

Kalau jumlah objek tidak sama, **jangan** pakai backup itu — cari sumber lain.
Restore DB yang gagal di tengah jalan tidak merusak data asli: database
sasaran baru dibuang dan diulang dari dump.

## Job terjadwal

Outbox 30 detik · stats 5 menit · publish changelog 5 menit · cleanup media 1 jam ·
retensi 24 jam · leaderboard 02:05 WIB · draft kenaikan kelas 00:10 WIB ·
digest 07:00 WIB · flush view 1 menit · decay minat 03:00 WIB · scan media
03:30 WIB · preview PDF tiap 30 menit. Semua job dibungkus `recover` sehingga panic satu job tidak
mematikan scheduler. Jalankan satu job manual:
`go run ./cmd/worker <outbox|stats|leaderboard|cleanup-media|publish-changelogs|retention|promotion-draft|digest>`.

## Maintenance mode (jendela migrasi v1 → v2)

Saat data dimindahkan, aplikasi v1 perlu di-*freeze* supaya tidak ada tulisan
baru yang masuk selama migrasi berjalan. Pengunjung melihat halaman
"Pindah ke v2" (HTTP 503, `Retry-After: 600`).

Dua cara menyalakannya:

| Cara | Kapan aktif | Catatan |
|---|---|---|
| `NEXT_PUBLIC_MAINTENANCE_MODE=true` saat build | saat build ulang | Nama flag yang dipakai PRD §16.1; perlu build + deploy ulang |
| Header `x-maintenance-mode: on` dari reverse proxy | **sekali reload proxy** | Tanpa build ulang — dipakai untuk go-live |

Contoh Caddy (reverse proxy di depan Next.js):

```caddy
# buka maintenance
grafikarsa.com {
    header x-maintenance-mode on
    reverse_proxy web:3000
}
# tutup maintenance: hapus baris "header x-maintenance-mode on", lalu
#   docker compose exec caddy caddy reload
```

`/api/*` **tidak** disentuh maintenance — health check, BFF, dan API tetap
hidup, jadi pod masih lolos health check dan rollback tetap mungkin.

Aktifkan **H-1** (bukan saat migrasi dimulai) supaya orang tidak sempat
menulis karya yang tidak ikut termigrasi.

## Mirror storage saat go-live (PRD §16.3)

Bucket lama (MinIO v1) disalin ke bucket baru (RustFS/S3 v2) memakai
`tools/storage-mirror.sh`. Layout key dipertahankan 100% — hanya host yang
ganti — supaya mapping DB sesederhana mungkin.

```bash
# di server, setelah service storage baru menyala & maintenance mode aktif
export SRC_ENDPOINT=http://127.0.0.1:9000   # bucket lama
export SRC_KEY=... SRC_SECRET=...
export DST_ENDPOINT=http://127.0.0.1:9020   # bucket baru
export DST_KEY=... DST_SECRET=...
export BUCKET=grafikarsa

# 1. plan dulu (tidak menulis apa pun)
MIRROR_DRY_RUN=1 ./tools/storage-mirror.sh

# 2. eksekusi + verifikasi (15-30 mnt untuk 5,1 GB)
./tools/storage-mirror.sh
```

Verifikasi di dalam skrip: `mc diff` (nama & ukuran semua objek) +
checksum md5 pada sampel objek yang tersebar. Kalau script keluar dengan
error, jangan lanjut ke migrasi DB — obyek yang tidak lengkap akan membuat sebagian gambar hilang di karya yang sudah tayang.

**`mc` wajib memakai `--api S3v4`.** Tanpa itu mc memakai S3v2 yang ditolak
MinIO modern & RustFS, dan pesannya ("The Access Key Id you provided does
not exist in our records") tidak menyiratkannya bahwa masalahnya versi API.
Flag ini sudah dipakai di dalam skrip.

Setelah mirror selesai:

```bash
# daftarkan objek storage jadi media_assets + legacy_urls
go run ./cmd/tools/migrate-old --v1 ... --v2 ... --execute --migrate-media
```

## Migrasi DB v1 → v2 di server (go-live)

Skema v1 dan v2 **berbeda total** — jadi migrasi bukan "jalanin migration
goose di atas data v1", tapi: buat DB v2 kosong → jalankan migrasi goose v2 →
salin data dari DB v1 ke DB v2 dengan `migrate-old`.

**Kendala yang menentukan bentuknya:** compose v2 memakai
`container_name: grafikarsa-db` yang **sama** dengan stack v1 yang sedang
jalan. Dua stack dengan nama container yang sama tidak bisa hidup
berdampingan. Jadi v1 **dimatikan** lebih dulu, dan datanya dipulihkan dari
dump (bukan dibaca dari container yang hidup).

Semua perintah dijalankan di server (`/opt/grafikarsa`).

| Volume | v1 (sekarang) | v2 (setelah go-live) |
|---|---|---|
| PostgreSQL | `grafikarsa_postgres_data` (postgres:**15**) | `grafikarsa_v2_postgres_data` (postgres:**16**) |
| Object storage | `grafikarsa_minio_data` (MinIO) | `grafikarsa_v2_storage_data` (RustFS) |
| Cache | `grafikarsa_redis_data` | `grafikarsa_v2_dragonfly_data` |
| TLS Caddy | `grafikarsa_caddy_data` | `grafikarsa_caddy_data` (**sengaja sama** — sertifikat tidak perlu diterbitkan ulang) |

Nama volume v2 ditulis eksplisit di compose. Kalau tidak, key `postgres_data`
resolve ke volume yang sama dengan v1 — dan image postgres:16 **menolak
start** di data dir postgres:15 ("database files are incompatible with
server"). Jadi volume v1 bukan sekadar "dibaca" oleh v2: container db-nya
gagal start sama sekali.

```bash
DC="docker compose -f docker-compose.deploy.yml"

# 0. Maintenance mode v1 aktif (v1 tidak boleh menerima tulisan baru selama
#    migrasi) — lihat bagian "Maintenance mode".

# 1. Backup v1 terakhir (dipakai sebagai sumber di langkah 3).
$DC exec -T db pg_dump -Fc -U "$DB_USER" "$DB_NAME" > "/tmp/v1-pre-migrate.dump"
ls -la /tmp/v1-pre-migrate.dump

# 2. Matikan stack v1. Volume v1 (grafikarsa_postgres_data) TIDAK dihapus —
#    data v1 tetap utuh sebagai jaring pengaman.
$DC down

# 3. Nyalakan DB v2 (volume grafikarsa_v2_postgres_data, kosong) + pulihkan
#    dump v1 ke database terpisah di container yang sama, sebagai sumber
#    migrasi. v1_src hidup di DB v2, tapi TIDAK disruptif: schema-nya milik
#    v1 dan dibaca sebagai data biasa.
$DC up -d db
$DC exec -T db psql -U "$DB_USER" -d postgres -c "CREATE DATABASE v1_src OWNER $DB_USER;"
$DC exec -T db pg_restore -U "$DB_USER" -d v1_src --no-owner --no-privileges \
  < /tmp/v1-pre-migrate.dump

# 4. Skema v2 (goose) di database v2.
$DC exec -T db psql -U "$DB_USER" -d postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;"
$DC run --rm --entrypoint /migrate -e DB_NAME="$DB_NAME" api up

# 5. Salin data v1 -> v2. KERINGKASAN dulu (wajib):
$DC run --rm --entrypoint /migrate-old \
  -e DB_NAME="$DB_NAME" api \
  --v1 "postgres://$DB_USER:$DB_PASSWORD@db:5432/v1_src?sslmode=disable" \
  --v2 "postgres://$DB_USER:$DB_PASSWORD@db:5432/$DB_NAME?sslmode=disable" \
  --dry-run
#    Periksa hitungannya: users 1688 · portfolios 1723 · blocks 5939
#    (+1031 karantina) · likes 740 · follows 4341 · comments 79
#    · notifications 6946 · feedback 17 · skipped {}
#    Kalau ada yang tidak cocok, JANGAN lanjut.
$DC run --rm --entrypoint /migrate-old -e DB_NAME="$DB_NAME" api \
  --v1 "postgres://$DB_USER:$DB_PASSWORD@db:5432/v1_src?sslmode=disable" \
  --v2 "postgres://$DB_USER:$DB_PASSWORD@db:5432/$DB_NAME?sslmode=disable" \
  --execute --migrate-media

# 6. Verifikasi: jumlah baris v2 harus sama dengan v1 (kecuali blok yang
#    dikarantina). Cek juga `legacy_urls` terisi.
$DC exec -T db psql -U "$DB_USER" -d "$DB_NAME" -c \
  "select (select count(*) from users) users, (select count(*) from portfolios) portfolios,
          (select count(*) from portfolio_blocks) blocks, (select count(*) from follows) follows;"

# 7. Nyalakan sisa stack v2 + deploy (tag versi / workflow_dispatch).
$DC up -d
```

Kalau migrasi gagal di tengah jalan, ulangi dari langkah 3 — database v2
dibuang dan dibuat ulang. Data v1 tidak pernah hilang: ada di
`grafikarsa_postgres_data` **dan** di `/tmp/v1-pre-migrate.dump`.

Yang **tidak** boleh dilewati: langkah 5 (dry-run) dan langkah 6. Tool
`migrate-old` sekarang gagal keras kalau ada objek storage yang tidak
terdaftar sebagai media — jadi "migrasi sukses" tidak lagi bisa berarti
"sebagian data hilang diam-diam".

## Sunset plan (P6-09)

Rencana resmi membuang v1 setelah go-live. Tidak ada data v1 yang dihapus
sebelum 30 hari.

| Hari | Aksi | Pemeriksa |
|---|---|---|
| H-1 | Aktifkan maintenance mode v1 | ops |
| H-0 | Migrasi DB + mirror storage, verifikasi | ops + migration dry-run (`migrate-old --dry-run` di staging dengan dump prod) |
| H-0 | Go-live v2, redirect `/storage/*` → objek baru | ops |
| H-0 + 1 jam | Matikan maintenance mode | ops |
| H+1…H+6 | Monitor 404/500, latency p95, complain | ops + support |
| H+7 | Review monitoring. Kalau bersih → lanjutkan | ops |
| H+30 | **Hapus v1**: container, image, volume, dan backup lama di server | ops — butuh persetujuan tertulis |

Syarat H+30: monitoring H+7 bersih (tanpa insiden 5xx selama 7 hari berturut),
tidak ada request ke v1 yang tercatat di access log selama 7 hari, dan backup
v2 sudah terverifikasi lewat restore drill.

Sebelum dihapus, pastikan backup v2 (DB + storage) ada di **dua** tempat
(server + mesin terpisah) dan sudah pernah di-restore-drill.

## Integration test lokal

Integration test memakai Postgres sungguhan (bukan SQLite — skema v2 memakai
`citext`, `pg_trgm`, generated column `tsvector`, dan unique partial index).

```bash
# database test TERPISAH dari database dev
docker exec -i gk-pg-test psql -U grafikarsa -d postgres \
  -c 'CREATE DATABASE grafikarsa_test;'
cd apps/api
DB_HOST=127.0.0.1 DB_PORT=5439 DB_USER=grafikarsa DB_PASSWORD=test \
  DB_NAME=grafikarsa_test go run ./cmd/tools/migrate up

# jalankan test
TEST_DATABASE_URL="postgres://grafikarsa:test@127.0.0.1:5439/grafikarsa_test?sslmode=disable" \
  go test ./...
```

**Wajib pakai database terpisah.** Test ini menghapus isi tabel dengan
`TRUNCATE`. Helper `internal/dbtest` menolak DSN yang nama databasenya tidak
mengandung "test" — kalau tidak, satu `go test ./...` bisa menghapus seluruh
user, portfolio, dan komentar di database dev/produksi tanpa jejak. Kalau
memang disengaja, setel `ALLOW_DESTRUCTIVE_TEST_DB=1`.

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
  (`x-client-ip`); `X-Forwarded-For` mentah tidak dipercaya.
- **Status user** dicek via cache 60 detik tiap request: menonaktifkan atau
  menghapus user langsung memutus akses meski token belum expired.
- **Scope penilaian**: guru/admin_jurusan hanya melihat & menilai portfolio
  jurusannya (404 untuk di luar scope).
- **Media retire**: mengganti avatar/banner/thumbnail menghapus objek media lama
  (objek + varian) sehingga kuota tidak menumpuk.
