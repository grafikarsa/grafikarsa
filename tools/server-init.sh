#!/usr/bin/env bash
# Setup awal server Ubuntu baru (Fase 0: secret + firewall + info).
# Usage: ./tools/server-init.sh   (jalankan di /opt/grafikarsa, sebagai root/deploysudo)
set -euo pipefail
cd "$(dirname "$0")/.."

test -f .env || cp .env.example .env
gen() { # gen VAR_NAME
  v=$(grep -E "^$1=" .env | cut -d= -f2-)
  case "$v" in ""|*"<rand"*) echo "$1=$(openssl rand -base64 32)" ;;
  *) echo "$1=$v" ;; esac
}
for k in DB_PASSWORD S3_ACCESS_KEY S3_SECRET_KEY JWT_ACCESS_SECRET JWT_REFRESH_SECRET CSRF_SECRET ADMIN_SEED_PASSWORD; do
  gen "$k"
done > /tmp/grafikarsa-env-new
python3 - <<'PY'
import re
base = open('.env').read()
new = dict(l.split('=', 1) for l in open('/tmp/grafikarsa-env-new').read().splitlines())
def sub(m):
    k = m.group(1)
    return f"{k}={new[k]}" if k in new else m.group(0)
open('.env', 'w').write(re.sub(r'^([A-Z_]+)=.*$', sub, base, flags=re.M))
PY
chmod 600 .env
echo "secret OK (.env 600)"

if command -v ufw >/dev/null 2>&1; then
  ufw allow 80/tcp >/dev/null; ufw allow 443/tcp >/dev/null; ufw allow 443/udp >/dev/null
  echo "ufw OK (80, 443/tcp+udp)"
fi
echo "next: isi DOMAIN di .env, lalu: docker compose -f docker-compose.deploy.yml up -d"
