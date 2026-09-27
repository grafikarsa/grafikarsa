#!/bin/bash
# ==============================================
# Grafikarsa — Initial server setup (sekali jalan)
# ==============================================
# Dipakai untuk server BARU yang kosong (Ubuntu 22/24).
# Yang dilakukan:
#   1. Install Docker (jika belum ada)
#   2. Siapkan /opt/grafikarsa + .env (generate secret otomatis)
#   3. Pull & up semua container (Caddy di 80/443, sisanya internal)
#   4. Import db/db.sql jika database masih kosong
#   5. Buat bucket MinIO + public-read
#
# Cara pakai (di server, di dalam folder yang berisi repo/file ini):
#   DOMAIN=grafikarsa.com ./scripts/server-init.sh
#   DOMAIN=103.134.78.58 ./scripts/server-init.sh   # tanpa domain (HTTP)
#
# File yang wajib ada di direktori kerja:
#   docker-compose.deploy.yml, Caddyfile, .env.example, db/db.sql
# Hasil: .env + containers di /opt/grafikarsa
set -euo pipefail

APP_DIR="/opt/grafikarsa"
DOMAIN="${DOMAIN:-${1:-}}"

if [ -z "$DOMAIN" ]; then
  echo "❌ DOMAIN belum diisi."
  echo "   Contoh: DOMAIN=grafikarsa.com ./scripts/server-init.sh"
  echo "   Atau IP: DOMAIN=103.134.78.58 ./scripts/server-init.sh"
  exit 1
fi

for f in docker-compose.deploy.yml Caddyfile .env.example db/db.sql; do
  if [ ! -f "$f" ]; then
    echo "❌ File $f tidak ditemukan di $(pwd). Copy dulu repo/files ke sini."
    exit 1
  fi
done

# --- 1. Docker ---
if ! command -v docker >/dev/null 2>&1; then
  echo "📦 Install Docker..."
  curl -fsSL https://get.docker.com -o get-docker.sh
  sudo sh get-docker.sh
  rm -f get-docker.sh
else
  echo "✅ Docker sudah ada: $(docker --version)"
fi

# --- 2. Direktori + .env ---
sudo mkdir -p "$APP_DIR"
sudo chown -R "$(id -u):$(id -g)" "$APP_DIR"
cp docker-compose.deploy.yml Caddyfile "$APP_DIR/"
mkdir -p "$APP_DIR/db"
cp db/db.sql db/migrate-to-single-domain.sql "$APP_DIR/db/" 2>/dev/null || cp db/db.sql "$APP_DIR/db/"
cd "$APP_DIR"

# Skema http vs https: IP = http, domain = https
if [[ "$DOMAIN" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  SCHEME="http"; PRESIGN_SSL="false"
else
  SCHEME="https"; PRESIGN_SSL="true"
fi

if [ ! -f .env ]; then
  echo "📝 Membuat .env baru untuk DOMAIN=$DOMAIN ..."
  cp .env.example .env
  rand() { openssl rand -base64 32 | tr -d '\n'; }
  sed -i "s/^DOMAIN=.*/DOMAIN=$DOMAIN/" .env
  sed -i "s/^DB_PASSWORD=.*/DB_PASSWORD=$(rand)/" .env
  sed -i "s/^MINIO_SECRET_KEY=.*/MINIO_SECRET_KEY=$(rand)/" .env
  sed -i "s/^JWT_ACCESS_SECRET=.*/JWT_ACCESS_SECRET=$(rand)/" .env
  sed -i "s/^JWT_REFRESH_SECRET=.*/JWT_REFRESH_SECRET=$(rand)/" .env
  sed -i "s|^NEXT_PUBLIC_API_URL=.*|NEXT_PUBLIC_API_URL=$SCHEME://$DOMAIN/api/v1|" .env
  sed -i "s|^NEXT_PUBLIC_APP_URL=.*|NEXT_PUBLIC_APP_URL=$SCHEME://$DOMAIN|" .env
  sed -i "s|^NEXT_PUBLIC_STORAGE_URL=.*|NEXT_PUBLIC_STORAGE_URL=$SCHEME://$DOMAIN/storage/grafikarsa|" .env
  sed -i "s|^CORS_ORIGINS=.*|CORS_ORIGINS=$SCHEME://$DOMAIN|" .env
  sed -i "s/^MINIO_ENDPOINT=.*/MINIO_ENDPOINT=minio:9000/" .env
  sed -i "s/^MINIO_PRESIGN_HOST=.*/MINIO_PRESIGN_HOST=$DOMAIN/" .env
  sed -i "s/^MINIO_PRESIGN_USE_SSL=.*/MINIO_PRESIGN_USE_SSL=$PRESIGN_SSL/" .env
  sed -i "s|^MINIO_PRESIGN_PATH_PREFIX=.*|MINIO_PRESIGN_PATH_PREFIX=/storage|" .env
  sed -i "s|^STORAGE_PUBLIC_URL=.*|STORAGE_PUBLIC_URL=$SCHEME://$DOMAIN/storage/grafikarsa|" .env
  echo "✅ .env dibuat (password & JWT digenerate random)."
else
  echo "ℹ️  .env sudah ada — dipakai apa adanya (tidak dioverwrite)."
fi

# --- 3. Up ---
echo "🚀 Pull & up..."
export $(grep -v '^#' .env | xargs -0 2>/dev/null || grep -v '^#' .env | xargs)
docker compose -f docker-compose.deploy.yml pull
docker compose -f docker-compose.deploy.yml up -d
docker compose -f docker-compose.deploy.yml ps

# --- 4. Tunggu DB + import jika kosong ---
echo "⏳ Menunggu database..."
for i in $(seq 1 24); do
  if docker exec grafikarsa-db pg_isready -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1; then break; fi
  sleep 5
done
TABLES=$(docker exec grafikarsa-db psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';" 2>/dev/null | tr -d ' ' || echo 0)
if [ "${TABLES:-0}" = "0" ]; then
  echo "📥 Database kosong — import db/db.sql ..."
  docker exec -i grafikarsa-db psql -U "$DB_USER" -d "$DB_NAME" < db/db.sql
else
  echo "ℹ️  Database sudah berisi $TABLES tabel — import dilewati."
fi

# --- 5. Bucket MinIO ---
echo "⏳ Menunggu MinIO..."
for i in $(seq 1 24); do
  if docker exec grafikarsa-minio mc ready local >/dev/null 2>&1; then break; fi
  sleep 5
done
docker exec grafikarsa-minio sh -c "mc alias set local http://localhost:9000 \"\$MINIO_ROOT_USER\" \"\$MINIO_ROOT_PASSWORD\" >/dev/null 2>&1; mc mb local/$MINIO_BUCKET --ignore-existing >/dev/null 2>&1; mc anonymous set download local/$MINIO_BUCKET >/dev/null 2>&1; echo '✅ Bucket $MINIO_BUCKET public-read.'"

echo ""
echo "✅ SELESAI."
echo "   Frontend : $SCHEME://$DOMAIN/"
echo "   API      : $SCHEME://$DOMAIN/api/v1/health  (via proxy, tanpa subdomain)"
echo "   Storage  : $SCHEME://$DOMAIN/storage/$MINIO_BUCKET/"
echo ""
echo "   DNS yang dibutuhkan (cukup 1):  A  @  <IP-server>"
echo "   Cek: docker compose -f $APP_DIR/docker-compose.deploy.yml ps"
echo "   Log: docker compose -f $APP_DIR/docker-compose.deploy.yml logs -f"
