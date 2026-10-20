#!/usr/bin/env bash
# ============================================================
# Uji kontrak + beban terhadap API yang hidup.
#
# Dipakai CI (job "contract") dan bisa dipanggil manual:
#   ./tools/contract-test.sh http://localhost:8091
#
# Butuh: migrasi + seed sudah dijalankan, dan API hidup di $BASE.
#   - schemathesis : pipx install schemathesis  (lewati kalau tak ada)
#   - k6           : image grafana/k6           (lewati kalau tak ada)
#
# Exit code non-nol kalau ada kegagalan — dipakai sebagai gate CI.
# ============================================================
set -uo pipefail

# BASE      = BFF (Next.js) — dipakai schemathesis, karena spec
#             mendokumentasikan path publik /api/v1/... yang dilayani BFF.
# API_BASE  = API Go langsung — dipakai k6, karena skrip k6 memanggil
#             path /v1/... (tanpa prefix /api).
BASE="${1:-http://localhost:3000}"
API_BASE="${2:-http://localhost:8091}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RC=0

log() { echo "[$(date +%H:%M:%S)] $*"; }

# ---------- 1. Contract test (schemathesis) ----------
#
# BASE di sini adalah BFF (Next.js), bukan API langsung: spec mendokumentasikan
# path publik (/api/v1/...) yang memang dilayani BFF. k6 di bawah memakai
# URL API secara langsung, karena skripnya memanggil path /v1/... .
if command -v schemathesis >/dev/null 2>&1; then
  log "schemathesis terhadap $BASE"
  if "$ROOT/tools/schemathesis.sh" "$BASE"; then
    log "schemathesis: LULUS"
  else
    log "schemathesis: GAGAL"
    RC=1
  fi
else
  log "SKIP: schemathesis belum terpasang (pipx install schemathesis)"
fi

# ---------- 2. Load test (k6) ----------
if command -v k6 >/dev/null 2>&1 || docker image inspect grafana/k6:latest >/dev/null 2>&1; then
  log "k6 smoke test terhadap $API_BASE"
  K6_ARGS=(-e "BASE=$API_BASE")
  [ -n "${USER:-}" ] && K6_ARGS+=(-e "USER=$USER")
  [ -n "${PASS:-}" ] && K6_ARGS+=(-e "PASS=$PASS")
  if command -v k6 >/dev/null 2>&1; then
    K6_CMD=(k6 run "${K6_ARGS[@]}" "$ROOT/tools/k6/smoke.js")
  else
    # Di dalam CI, path repo di-mount ke container k6 supaya skripnya terbaca.
    K6_CMD=(docker run --rm --network host \
      -v "$ROOT/tools/k6:/scripts:ro" \
      grafana/k6:latest run "${K6_ARGS[@]}" /scripts/smoke.js)
  fi
  if "${K6_CMD[@]}"; then
    log "k6: LULUS (threshold terpenuhi)"
  else
    log "k6: GAGAL (threshold dilanggar)"
    RC=1
  fi
else
  log "SKIP: k6 tidak tersedia"
fi

exit $RC
