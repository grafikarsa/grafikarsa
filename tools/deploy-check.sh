#!/usr/bin/env bash
# Validasi secret tidak default sebelum deploy. Gagal = stop.
# Usage: ./tools/deploy-check.sh
set -euo pipefail
cd "$(dirname "$0")/.."
test -f .env || { echo "FATAL: .env tidak ada (cp .env.example .env)"; exit 1; }
fail=0
check() { # check VAR_NAME
  v=$(grep -E "^$1=" .env | cut -d= -f2-)
  case "$v" in
    ""|*"<rand"*|*"change-me"*|*"your_"*|"password"|"secret")
      echo "FATAL: $1 belum diisi ($v)"; fail=1;;
  esac
}
check DOMAIN
check DB_PASSWORD
check S3_ACCESS_KEY
check S3_SECRET_KEY
check JWT_ACCESS_SECRET
check JWT_REFRESH_SECRET
check CSRF_SECRET
check ADMIN_SEED_PASSWORD
check NEXT_PUBLIC_APP_URL
check INTERNAL_API_URL
# Origin check BFF memakai NEXT_PUBLIC_APP_URL; kalau masih localhost semua
# mutasi (POST/PATCH) dibalas 403 CSRF_INVALID.
if grep -qE '^NEXT_PUBLIC_APP_URL=.*localhost' .env; then
  echo "FATAL: NEXT_PUBLIC_APP_URL masih localhost (mutasi akan 403 CSRF_INVALID)"; fail=1
fi
if [ "${DOMAIN:-localhost}" = "localhost" ]; then
  echo "FATAL: DOMAIN masih localhost (deploy butuh domain/IP publik)"; fail=1
fi
exit $fail
