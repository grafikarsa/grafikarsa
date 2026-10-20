#!/usr/bin/env bash
# ============================================================
# Backup harian Grafikarsa v2 (server prod)
#   - PostgreSQL (compose service "db") -> pg_dump -Fc
#   - Object storage (RustFS)          -> tar.gz volume (read-only)
#   - Dragonfly (cache)                -> snapshot (opsional, best effort)
#   - Konfigurasi deploy               -> .env, compose, Caddyfile
#   - Inventaris + manifest             -> row counts, daftar object, sha256
#
# Cron: 0 2 * * * /opt/grafikarsa/gk-backup.sh >> /var/log/grafikarsa-backup.log 2>&1
#
# Env (semua opsional):
#   APP_DIR        direktori aplikasi      (default /opt/grafikarsa)
#   COMPOSE_FILE   file compose             (default docker-compose.deploy.yml)
#   BACKUP_ROOT    direktori tujuan         (default $APP_DIR/backup)
#   KEEP           jumlah backup disimpan    (default 1 — tiap backup ~5 GB)
#   DB_SERVICE     service postgres         (default db)
#   STORAGE_CONTAINER container object store (default grafikarsa-storage)
#   STORAGE_MOUNT  titik mount di container (default /data)
#   CACHE_CONTAINER container cache         (default grafikarsa-dragonfly)
#
# Catatan: script ini TIDAK memakai rclone (tidak terpasang di server) dan
# TIDAK menyentuh volume lewat nama — titik mount diambil dari container
# yang benar-benar jalan, jadi tetap benar walau nama volume berubah.
# ============================================================
set -uo pipefail

APP_DIR="${APP_DIR:-/opt/grafikarsa}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.deploy.yml}"
BACKUP_ROOT="${BACKUP_ROOT:-$APP_DIR/backup}"
KEEP="${KEEP:-1}"
DB_SERVICE="${DB_SERVICE:-db}"
STORAGE_CONTAINER="${STORAGE_CONTAINER:-grafikarsa-storage}"
STORAGE_MOUNT="${STORAGE_MOUNT:-/data}"
CACHE_CONTAINER="${CACHE_CONTAINER:-grafikarsa-dragonfly}"

TS="$(date +%Y%m%d-%H%M%S)"
DEST="$BACKUP_ROOT/$TS"

cd "$APP_DIR" || { echo "ERROR: tidak bisa masuk $APP_DIR" >&2; exit 1; }
DC="docker compose -f $COMPOSE_FILE"

log() { echo "[$(date -Is)] $*"; }
fail() { echo "[$(date -Is)] ERROR: $*" >&2; exit 1; }

# Ambil nilai dari .env server (compose membacanya juga).
env_value() {
  local key="$1" default="${2:-}"
  local val
  val=$(grep -E "^${key}=" .env 2>/dev/null | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'")
  [ -n "$val" ] && printf '%s' "$val" || printf '%s' "$default"
}

DB_USER="$(env_value DB_USER grafikarsa)"
DB_NAME="$(env_value DB_NAME grafikarsa)"
S3_BUCKET="$(env_value S3_BUCKET grafikarsa)"

# ---- retensi: hapus backup lama, tapi SETELAH backup baru selesai & terverifikasi ----
# Kalau backup lama dihapus lebih dulu dan backup baru ternyata gagal, server
# berakhir tanpa backup sama sekali — jaring pengaman terakhir ikut hilang
# justru di saat paling dibutuhkan. Jadi retensi baru dijalankan setelah
# backup baru dinyatakan sehat.
apply_retention() {
  [ -d "$BACKUP_ROOT" ] || return 0
  local old
  old=$(ls -1d "$BACKUP_ROOT"/[0-9]* 2>/dev/null | sort | head -n -"$KEEP" 2>/dev/null)
  for o in $old; do
    [ "$o" = "$DEST" ] && continue
    log "retensi: hapus backup lama $o"
    rm -rf "$o"
  done
}

mkdir -p "$DEST"/{db,storage,cache,config,inventory} || fail "gagal membuat $DEST"
chmod 700 "$DEST" "$DEST"/*
log "Mulai backup -> $DEST"

# ---------- 1. PostgreSQL ----------
log "1/7 pg_dump database '$DB_NAME' ..."
# Coba beberapa kali: kalau DB sedang restart (mis. jadwal cron kebetulan
# menabrak restart container), backup tetap bisa jalan begitu DB siap.
DUMP_OK=0
for attempt in 1 2 3; do
  if $DC exec -T "$DB_SERVICE" pg_dump -U "$DB_USER" -Fc "$DB_NAME" \
      > "$DEST/db/$DB_NAME-$TS.dump" 2>"$DEST/db/pg_dump.err"; then
    # pg_dump keluar 0 walau menulis 0 byte ke stdout? cek ukurannya.
    if [ -s "$DEST/db/$DB_NAME-$TS.dump" ]; then
      DUMP_OK=1
      break
    fi
  fi
  log "    percobaan $attempt gagal ($(tail -1 "$DEST/db/pg_dump.err" 2>/dev/null | cut -c1-120))"
  [ "$attempt" = "3" ] || sleep 10
done
if [ "$DUMP_OK" = "1" ]; then
  log "    dump OK ($(du -h "$DEST/db/$DB_NAME-$TS.dump" | cut -f1))"
else
  fail "pg_dump gagal setelah 3 percobaan (lihat $DEST/db/pg_dump.err)"
fi

# ---------- 2. Object storage (RustFS) ----------
# Volume dibaca dari DALAM container helper, bukan lewat path host.
# Alasannya: direktori volume Docker (/var/lib/docker/volumes/<n>/_data)
# mode-nya 700 root — script yang jalan sebagai user non-root tidak bisa
# membacanya, padahal crontab server memang jalan sebagai root. Dengan
# container helper, script tetap jalan sebagai user mana pun yang punya
# akses docker, dan image yang dipakai adalah image milik service storage
# itu sendiri (jadi tidak perlu menarik image tambahan).
STORAGE_VOL=""
STORAGE_IMG=""
if docker ps --format '{{.Names}}' | grep -qx "$STORAGE_CONTAINER"; then
  STORAGE_VOL=$(docker inspect -f "{{range .Mounts}}{{if eq .Destination \"$STORAGE_MOUNT\"}}{{.Name}}{{end}}{{end}}" \
    "$STORAGE_CONTAINER" 2>/dev/null)
  STORAGE_IMG=$(docker inspect -f '{{.Config.Image}}' "$STORAGE_CONTAINER" 2>/dev/null)
fi
[ -n "$STORAGE_VOL" ] || fail "volume object storage tidak ditemukan (container $STORAGE_CONTAINER belum jalan?)"
[ -n "$STORAGE_IMG" ] || fail "image service storage tidak bisa dibaca"
log "2/7 archive object storage (volume $STORAGE_VOL) ..."
# Kompresi: pigz kalau ada (multi-thread, lebih cepat), gzip kalau tidak.
COMPRESS="gzip -1"
command -v pigz >/dev/null 2>&1 && COMPRESS="pigz -1 -p 4"
# Tar seluruh volume: selain data bucket, metadata store RustFS juga ikut —
# tanpa itu, restore hanya bisa lewat mc mirror, bukan file biasa.
docker run --rm --entrypoint sh -v "$STORAGE_VOL:/src:ro" "$STORAGE_IMG" \
  -c 'cd /src && tar -cf - .' 2>"$DEST/storage/tar.err" \
  | $COMPRESS > "$DEST/storage/storage-data-$TS.tar.gz" || fail "tar storage gagal (lihat $DEST/storage/tar.err)"
log "    arsip OK ($(du -h "$DEST/storage/storage-data-$TS.tar.gz" | cut -f1))"

# ---------- 3. Dragonfly (cache, best effort) ----------
log "3/7 dump cache (dragonfly) ..."
CACHE_OK=0
if docker ps --format '{{.Names}}' | grep -qx "$CACHE_CONTAINER"; then
  # Paksa snapshot dulu (selalu ada, walau cron-nya belum jatuh tempo).
  # Dragonfly menyediakan antarmuka Redis, jadi redis-cli bisa dipakai.
  if docker exec "$CACHE_CONTAINER" redis-cli BGSAVE >/dev/null 2>&1; then
    sleep 3
    # Dragonfly TIDAK menulis dump.rdb seperti Redis: format snapshot-nya
    # sendiri (dump-0000.dfs + dump-summary.dfs). Karena itu seluruh isi
    # /data disalin, bukan satu file.
    if docker cp "$CACHE_CONTAINER":/data/. "$DEST/cache/" >/dev/null 2>&1 \
       && [ -n "$(ls -A "$DEST/cache" 2>/dev/null)" ]; then
      CACHE_OK=1
      log "    snapshot cache OK"
    fi
  fi
fi
if [ "$CACHE_OK" != "1" ]; then
  # Cache itu disposable — kegagalan di sini TIDAK boleh menggagalkan backup.
  echo "Cache tidak dibackup (dragonfly belum jalan atau snapshot tidak tersedia)." \
    > "$DEST/cache/NOTE.txt"
  log "    cache dilewati (tidak wajib)"
fi

# ---------- 4. Konfigurasi deploy ----------
log "4/7 salin konfigurasi deploy ..."
cp .env                      "$DEST/config/env.backup" 2>/dev/null
cp "$COMPOSE_FILE"           "$DEST/config/" 2>/dev/null
cp Caddyfile*                "$DEST/config/" 2>/dev/null
cp gk-backup.sh              "$DEST/config/gk-backup.sh" 2>/dev/null
chmod 600 "$DEST/config/env.backup" 2>/dev/null
docker ps --format '{{.Names}} {{.Image}}' > "$DEST/config/running-images.txt" 2>/dev/null

# ---------- 5. Inventaris DB ----------
log "5/7 inventaris database (row counts per tabel) ..."
$DC exec -T "$DB_SERVICE" psql -U "$DB_USER" -d "$DB_NAME" -tAF',' -c "
  SELECT table_name,
         (xpath('/row/c/text()', query_to_xml(format('SELECT count(*) AS c FROM %I.%I', table_schema, table_name), false, true, '')))[1]::text::bigint AS rows
  FROM information_schema.tables
  WHERE table_schema='public' AND table_type='BASE TABLE'
  ORDER BY table_name;" > "$DEST/inventory/table-rowcounts-$TS.csv" 2>/dev/null

# ---------- 6. Inventaris object storage ----------
log "6/7 inventaris object storage ..."
# Inventaris diambil dari dalam volume (sumber yang sama dengan arsip),
# jadi selalu konsisten dengan isi arsip.
docker run --rm --entrypoint sh -v "$STORAGE_VOL:/src:ro" "$STORAGE_IMG" \
  -c "find /src/$S3_BUCKET -type f -exec stat -c '%s %n' {} + 2>/dev/null | sed 's| /src/| |' | sort -k2" \
  > "$DEST/inventory/object-list-$TS.txt" 2>/dev/null
OBJ_COUNT=$(grep -c . "$DEST/inventory/object-list-$TS.txt" 2>/dev/null || echo 0)
echo "$OBJ_COUNT" > "$DEST/inventory/object-count-$TS.txt"
if [ "$OBJ_COUNT" = "0" ]; then
  echo "WARN: bucket '$S3_BUCKET' kosong atau tidak ada di volume" > "$DEST/inventory/NOTE.txt"
fi

# ---------- 7. Manifest + verifikasi ----------
log "7/7 manifest sha256 + verifikasi ..."
VERIFY_FAIL=0
( cd "$DEST" && find . -type f ! -name 'MANIFEST.txt' -print0 | sort -z | xargs -0 sha256sum > MANIFEST.txt )
if ( cd "$DEST" && sha256sum -c MANIFEST.txt > SHA256SUMS.check 2>&1 ); then
  echo "SHA256: OK semua checksum cocok" > "$DEST/VERIFY.txt"
else
  echo "SHA256: GAGAL" > "$DEST/VERIFY.txt"
  VERIFY_FAIL=1
fi
# Validasi dump: pg_restore hanya ada di dalam container postgres.
if $DC exec -T "$DB_SERVICE" pg_restore -l < "$DEST/db/$DB_NAME-$TS.dump" >/dev/null 2>&1; then
  echo "pg_dump: VALID (pg_restore -l OK)" >> "$DEST/VERIFY.txt"
else
  echo "pg_dump: TIDAK valid" >> "$DEST/VERIFY.txt"
  VERIFY_FAIL=1
fi
if tar -tzf "$DEST/storage/storage-data-$TS.tar.gz" >/dev/null 2>&1; then
  echo "tar storage: valid" >> "$DEST/VERIFY.txt"
else
  echo "tar storage: TIDAK valid" >> "$DEST/VERIFY.txt"
  VERIFY_FAIL=1
fi

{
  echo "Backup Grafikarsa v2"
  echo "Waktu        : $(date -Is)"
  echo "Host         : $(hostname)"
  echo "Database     : $DB_NAME (user $DB_USER)"
  echo "Object store : $STORAGE_CONTAINER (bucket $S3_BUCKET)"
  echo "Object count : $(cat "$DEST/inventory/object-count-$TS.txt" 2>/dev/null)"
  echo
  du -sh "$DEST"/db "$DEST"/storage "$DEST"/cache "$DEST"/config "$DEST"/inventory 2>/dev/null
  echo "TOTAL: $(du -sh "$DEST" | cut -f1)"
} > "$DEST/SUMMARY.txt"

chmod -R go-rwx "$DEST"
cat "$DEST/SUMMARY.txt"
cat "$DEST/VERIFY.txt" 2>/dev/null
df -h / | tail -1

# Verifikasi yang gagal harus membuat script keluar non-zero. Kalau tidak,
# cron menganggap backup sukses padahal arsipnya rusak atau checksumnya
# tidak cocok — jaring pengaman terakhir justru يقول "aman".
if [ "$VERIFY_FAIL" != "0" ]; then
  log "VERIFIKASI GAGAL — backup ini TIDAK bisa dipercaya (lihat $DEST/VERIFY.txt)"
  log "Backup lama sengaja TIDAK dihapus, jadi masih ada cadangan."
  exit 1
fi

# Baru sekarang aman retiring backup lama: ada yang baru DAN sudah terverifikasi.
apply_retention
log "SELESAI -> $DEST"
