#!/usr/bin/env bash
# Isi .env dengan secret acak — idempotent.
#
# Nilai yang masih berupa placeholder "<rand...>" diganti dengan random
# base64; nilai yang sudah diisi TIDAK disentuh. Jadi script ini aman
# dijalankan berulang kali.
#
# Dipakai oleh:
#   tools/server-init.sh   (server produksi)
#   make dev-init          (lokal)
#
# Kenapa perlu: .env.example sengaja berisi placeholder supaya tidak ada
# secret nyatan di repo. Tapi kalau .env dipakai apa adanya, DB & API
# memakai password literal "<rand: openssl rand -base64 32>" — dan begitu
# .env diganti (atau volume DB dibuat lebih dulu), passwordnya tidak
# cocok dan yang muncul hanya "password authentication failed" tanpa
# penjelasan.
set -euo pipefail
cd "$(dirname "$0")/.."

test -f .env || cp .env.example .env

gen() { # gen VAR_NAME
  v=$(grep -E "^$1=" .env | cut -d= -f2-)
  case "$v" in
    ""|*"<rand"*|*"<password"*|*"change-me"*) echo "$1=$(openssl rand -base64 32 | tr -d '\n=')" ;;
    *) echo "$1=$v" ;;
  esac
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
# PENTING: pola nama variabel harus menerima ANGKA juga. Pola lama
# "^([A-Z_]+)=" tidak cocok dengan S3_ACCESS_KEY / S3_SECRET_KEY (mengandung
# angka), sehingga dua kredensial itu TIDAK PERNAH di-randomize — di
# produksi pun kredensial storage memakai placeholder "<rand>".
open('.env', 'w').write(re.sub(r'^([A-Z0-9_]+)=.*$', sub, base, flags=re.M))
PY

rm -f /tmp/grafikarsa-env-new
chmod 600 .env
echo "secret OK (.env 600)"
