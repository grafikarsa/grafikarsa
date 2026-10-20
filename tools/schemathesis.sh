#!/usr/bin/env bash
# ============================================================
# Uji kontrak: jalankan schemathesis terhadap API yang hidup memakai
# packages/contracts/openapi.yaml.
#
# Contoh:
#   ./tools/schemathesis.sh http://localhost:3000
#
# TARGET: BFF (Next.js), bukan API langsung.
#
# Spec mendokumentasikan path PUBLIK (/api/v1/...) karena itu yang dipakai
# browser. Kalau test diarahkan ke API langsung, prefix /api tidak ada di
# sana dan setiap endpoint membalas 404 — test jadi tidak menguji apa pun.
# BFF meneruskan ke API, jadi menguji BFF sekaligus API-nya.
#
# Env:
#   Jalankan dengan env berikut agar hasilnya tidak diganggu rate limit:
#     RATE_LIMIT_REQUESTS=100000
#
# Exclude:
#   - /api/v1/admin, /api/v1/me, /api/v1/auth : butuh kredensial khusus.
#   - /api/v1/storage/*                       : route redirect URL lama v1.
#     Endpoint ini sengaja membalas 410 Gone kalau file lamanya sudah tidak
#     tersedia — itu jawaban yang benar secara semantik (dan lebih baik buat
#     SEO daripada 404: mesin pencari tahu URL-nya memang hilang). Tapi
#     check ST202 menganggap 410 sebagai "permintaan ditolak", jadi selama
#     check itu ikut jalan, route ini selalu dilaporkan gagal walau
#     kontraknya sudah benar.
#   - POSTset -uo pipefail
BASE="${1:-http://localhost:3000}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SPEC="$ROOT/packages/contracts/openapi.yaml"

if ! command -v schemathesis >/dev/null 2>&1; then
  echo "SKIP: schemathesis belum terpasang (pipx install schemathesis)"
  exit 0
fi

# schemathesis v4 memakai --url; v3 memakai --base-url. Dukung keduanya
# supaya skrip tidak rapuh saat tool di-upgrade.
SCHEMATHESIS_ARGS=()
if schemathesis run --help 2>&1 | grep -q -- "--base-url"; then
  SCHEMATHESIS_ARGS+=(--base-url "$BASE")
  SCHEMATHESIS_ARGS+=(--hypothesis-seed 1)
else
  SCHEMATHESIS_ARGS+=(--url "$BASE")
  SCHEMATHESIS_ARGS+=(--seed 1)
fi

echo "Menjalankan schemathesis terhadap $BASE (spec: $SPEC)"
schemathesis run "$SPEC" \
  "${SCHEMATHESIS_ARGS[@]}" \
  --checks all \
  --exclude-path-regex '/api/v1/admin' \
  --exclude-path-regex '/api/v1/me' \
  --exclude-path-regex '/api/v1/auth' \
  --exclude-path-regex '/api/v1/storage/.*' \
  --exclude-method POST \
  --exclude-method PUT \
  --exclude-method PATCH \
  --exclude-method DELETE \
  --exclude-method TRACE \
  --exclude-method QUERY \
  --max-examples 20
