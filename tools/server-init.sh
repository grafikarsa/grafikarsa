#!/usr/bin/env bash
# Setup awal server Ubuntu baru (Fase 0: secret + firewall + info).
# Usage: ./tools/server-init.sh   (jalankan di /opt/grafikarsa, sebagai root/deploysudo)
set -euo pipefail
cd "$(dirname "$0")/.."

# Secret di-generate oleh tools/gen-env.sh (dipakai juga oleh `make dev-init`).
bash "$(dirname "$0")/gen-env.sh"

if command -v ufw >/dev/null 2>&1; then
  ufw allow 80/tcp >/dev/null; ufw allow 443/tcp >/dev/null; ufw allow 443/udp >/dev/null
  echo "ufw OK (80, 443/tcp+udp)"
fi
echo "next: isi DOMAIN di .env, lalu: docker compose -f docker-compose.deploy.yml up -d"
