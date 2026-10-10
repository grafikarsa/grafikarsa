#!/usr/bin/env bash
# Uji kontrak: jalankan schemathesis terhadap API yang hidup memakai
# packages/contracts/openapi.yaml. Semua endpoint GET publik diuji; endpoint
# yang butuh auth di-skip otomatis karena tidak ada kredensial.
#
# Pakai:
#   ./tools/schemathesis.sh http://localhost:8091
#
# Tanpa argumen, memakai API dev default. Butuh `schemathesis` terpasang
# (pipx install schemathesis) — skrip ini sengaja tidak menarik dependensi
# Python ke dalam repo.
set -euo pipefail
BASE="${1:-http://localhost:8091}"
SPEC="$(dirname "$0")/../packages/contracts/openapi.yaml"

if ! command -v schemathesis >/dev/null 2>&1; then
  echo "SKIP: schemathesis belum terpasang (pipx install schemathesis)"
  exit 0
fi

echo "Menjalankan schemathesis terhadap $BASE (spec: $SPEC)"
schemathesis run "$SPEC" \
  --base-url "$BASE" \
  --checks all \
  --exclude-path-regex '/api/v1/admin' \
  --exclude-path-regex '/api/v1/me' \
  --exclude-path-regex '/api/v1/auth' \
  --exclude-method POST \
  --exclude-method PUT \
  --exclude-method PATCH \
  --exclude-method DELETE \
  --max-examples 20 \
  --hypothesis-seed 1
