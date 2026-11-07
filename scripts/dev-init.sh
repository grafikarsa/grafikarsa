#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

if [ -f .env ] && [ "${1:-}" != "--force" ]; then
  echo ".env sudah ada, lewati (pakai --force untuk menimpa)"
  exit 0
fi

POSTGRES_PASSWORD="$(openssl rand -hex 24)"
VALKEY_PASSWORD="$(openssl rand -hex 24)"
RUSTFS_SECRET_KEY="$(openssl rand -hex 24)"
RUSTFS_ACCESS_KEY="$(openssl rand -hex 10)"
GK_SECRET_KEY="$(openssl rand -hex 32)"

cp .env.example .env

# Ganti token referensi di dalam value (mis. DATABASE_URL).
sed -i \
  -e "s|__POSTGRES_PASSWORD__|${POSTGRES_PASSWORD}|g" \
  -e "s|__VALKEY_PASSWORD__|${VALKEY_PASSWORD}|g" \
  -e "s|__RUSTFS_ACCESS_KEY__|${RUSTFS_ACCESS_KEY}|g" \
  -e "s|__RUSTFS_SECRET_KEY__|${RUSTFS_SECRET_KEY}|g" \
  .env

# Ganti placeholder __RANDOM__ pada tiap key dengan nilai yang digenerate.
sed -i \
  -e "s|^POSTGRES_PASSWORD=__RANDOM__$|POSTGRES_PASSWORD=${POSTGRES_PASSWORD}|" \
  -e "s|^VALKEY_PASSWORD=__RANDOM__$|VALKEY_PASSWORD=${VALKEY_PASSWORD}|" \
  -e "s|^RUSTFS_ACCESS_KEY=__RANDOM__$|RUSTFS_ACCESS_KEY=${RUSTFS_ACCESS_KEY}|" \
  -e "s|^RUSTFS_SECRET_KEY=__RANDOM__$|RUSTFS_SECRET_KEY=${RUSTFS_SECRET_KEY}|" \
  -e "s|^GK_SECRET_KEY=__RANDOM__$|GK_SECRET_KEY=${GK_SECRET_KEY}|" \
  .env

chmod 600 .env
echo ".env dibuat."
