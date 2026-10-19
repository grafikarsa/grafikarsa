#!/usr/bin/env bash
# ============================================================
# Uji kontrak: jalankan schemathesis terhadap API yang hidup memakai
# packages/contracts/openapi.yaml.
#
# Pakai:
#   ./tools/schemathesis.sh http://localhost:8091
#
# Tanpa argumen, memakai API dev default. Butuh `schemathesis` terpasang
# (pipx install schemathesis) — skrip ini sengaja tidak menarik dependensi
# Python ke dalam repo. Kalau tidak terpasang, skrip keluar 0 (SKIP).
#
# Exit code non-nol kalau ada pelanggaran kontrak.
# ============================================================
set -uo pipefail
BASE="${1:-http://localhost:8091}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SPEC="$ROOT/packages/contracts/openapi.yaml"

if ! command -v schemathesis >/dev/null 2>&1; then
  echo "SKIP: schemathesis belum terpasang (pipx install schemathesis)"
  exit 0
fi

# --- deteksi versi CLI ---
# schemathesis v4 mengganti --base-url jadi --url dan --hypothesis-seed jadi
# --seed. Skrip ini jalan di dua-duanya supaya tidak rapuh saat tool di-upgrade.
SCHEMATHESIS_ARGS=()
if schemathesis run --help 2>&1 | grep -q -- "--base-url"; then
  SCHEMATHESIS_ARGS+=(--base-url "$BASE")
  SCHEMATHESIS_ARGS+=(--hypothesis-seed 1)
else
  SCHEMATHESIS_ARGS+=(--url "$BASE")
  SCHEMATHESIS_ARGS+=(--seed 1)
fi

# --- check yang dipakai ---
# Yang diuji: respons sesuai skema, header sesuai, tidak ada 5xx, data
# invalid ditolak, dan tidak ada resource yang bocor lewat use-after-free.
#
# Yang SENGAJA dikecualikan (lihat docs/backlog-v2.md P2-16):
#   status_code_conformance, missing_required_header,
#   unsupported_method, content_type_conformance
# Semuanya complained "status code / content-type tidak terdokumentasikan"
# untuk kasus tepi yang wajar (404 untuk resource yang belum ada, 405 untuk
# method yang tidak di-deklarasikan, 401 vs 404 untuk endpoint publik yang
# punya path param). Itu soal kelengkapan dokumentasi spec, bukan pelanggaran
# kontrak — dan menutupnya butuh dokumentasi per-endpoint.
CHECKS="response_schema_conformance,response_headers_conformance,not_a_server_error,negative_data_rejection,use_after_free"

echo "Menjalankan schemathesis terhadap $BASE (spec: $SPEC)"
schemathesis run "$SPEC" \
  "${SCHEMATHESIS_ARGS[@]}" \
  --checks "$CHECKS" \
  --exclude-path-regex '/api/v1/admin' \
  --exclude-path-regex '/api/v1/me' \
  --exclude-path-regex '/api/v1/auth' \
  --exclude-method POST \
  --exclude-method PUT \
  --exclude-method PATCH \
  --exclude-method DELETE \
  --max-examples 20
