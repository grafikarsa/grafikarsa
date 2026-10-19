#!/usr/bin/env bash
# Backup harian: pg_dump + mirror storage S3 + retensi.
# Dipasang di crontab server:  0 2 * * * /opt/grafikarsa/backup.sh >> /var/log/grafikarsa-backup.log 2>&1
# Retensi: 7 harian (default). Set BACKUP_RETENTION untuk mengubah.
set -euo pipefail
cd "$(dirname "$0")"

BACKUP_DIR="${BACKUP_DIR:-./backup}"
RETENTION="${BACKUP_RETENTION:-7}"
STAMP="$(date +%F)"
DC="docker compose -f docker-compose.deploy.yml"

mkdir -p "$BACKUP_DIR"

# 1. Database
if $DC ps --status running db >/dev/null 2>&1; then
  $DC exec -T db pg_dump -Fc -U "${DB_USER:-grafikarsa}" "${DB_NAME:-grafikarsa}" \
    > "$BACKUP_DIR/db-$STAMP.dump"
  echo "OK db -> $BACKUP_DIR/db-$STAMP.dump"
else
  echo "SKIP db: container tidak berjalan"
fi

# 2. Object storage (rclone; konfigurasi remote dari env S3_*)
if command -v rclone >/dev/null 2>&1 && [ -n "${S3_ENDPOINT:-}" ]; then
  rclone sync ":s3,provider=Other,endpoint=${S3_ENDPOINT},env_auth=false,access_key_id=${S3_ACCESS_KEY},secret_access_key=${S3_SECRET_KEY}:${S3_BUCKET}" \
    "$BACKUP_DIR/storage" --checksum --transfers 4 2>&1 | tail -3
  echo "OK storage -> $BACKUP_DIR/storage"
else
  echo "SKIP storage: rclone/S3 env tidak tersedia"
fi

# 3. Retensi
find "$BACKUP_DIR" -name 'db-*.dump' -mtime "+$RETENTION" -delete 2>/dev/null || true
echo "OK retensi: hapus dump lebih dari $RETENTION hari"

# 4. Peringatan disk
USED=$(df --output=pcent / | tail -1 | tr -d ' %')
if [ "$USED" -ge 80 ]; then
  echo "WARN: disk ${USED}% terpakai — bersihkan atau tambah kapasitas"
fi
