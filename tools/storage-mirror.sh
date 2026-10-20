#!/usr/bin/env bash
# ============================================================
# Mirror object storage v1 -> v2 (PRD §16.3).
#
# Dipakai saat go-live: bucket lama (MinIO v1) disalin ke bucket baru
# (RustFS/S3 v2). Layout key dipertahankan 100% — hanya host yang ganti,
# bukan key — supaya mapping DB sesederhana mungkin (Cuma ganti host).
#
# Env (wajib):
#   SRC_ENDPOINT  mis. http://127.0.0.1:9000   (bucket lama)
#   SRC_KEY       access key bucket lama
#   SRC_SECRET    secret key bucket lama
#   DST_ENDPOINT  mis. http://127.0.0.1:9020   (bucket baru)
#   DST_KEY       access key bucket baru
#   DST_SECRET    secret key bucket baru
#   BUCKET        nama bucket (default: grafikarsa)
#
# Env (opsional):
#   MIRROR_DRY_RUN=1     hanya-plan, jangan eksekusi mirror
#   MIRROR_SAMPLE=200    jumlah objek yang dicek checksum (default 200)
#   MIRROR_ALIAS_PREFIX  prefix nama alias mc (default gk)
#
# CATATAN: skrip ini tidak menyentuh bucket sumber selain membacanya.
# Jalankan saat maintenance mode aktif supaya tidak ada tulisan baru
# yang masuk setelah mirror selesai.
# ============================================================
set -euo pipefail

BUCKET="${BUCKET:-grafikarsa}"
MIRROR_DRY_RUN="${MIRROR_DRY_RUN:-0}"
MIRROR_SAMPLE="${MIRROR_SAMPLE:-200}"
PREFIX="${MIRROR_ALIAS_PREFIX:-gk}"

need() {
  [ -n "${!1:-}" ] || { echo "ERROR: env $1 wajib diisi"; exit 1; }
}
need SRC_ENDPOINT; need SRC_KEY; need SRC_SECRET
need DST_ENDPOINT; need DST_KEY; need DST_SECRET

command -v mc >/dev/null || { echo "ERROR: mc (MinIO Client) belum terpasang"; exit 1; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
LIST="$WORK/source.txt"
DIFF="$WORK/diff.txt"

log() { echo "[$(date +%H:%M:%S)] $*"; }

# ---------- 1. alias ----------
# `mc alias set` menyimpan kredensial ke konfigurasi mc milik operator
# (~/.mc/config.json) — di luar direktori sementara, dan tidak ikut terhapus
# kalau skrip gagal di tengah jalan. Arahkan ke direktori privat di bawah
# $WORK supaya seluruh kredensial ikut hilang saat exit trap berjalan.
export MC_CONFIG_DIR="$WORK/mc"
mkdir -p "$MC_CONFIG_DIR"
chmod 700 "$MC_CONFIG_DIR"
log "set up alias mc"
# --api S3v4 WAJIB: mc secara default memakai S3v2, yang ditolak MinIO
# modern & RustFS ("The Access Key Id you provided does not exist in our
# records"). Tanpa flag ini, mirror selalu gagal di langkah paling awal —
# dan pesan errornya tidak menyiratkannya soal versi API.
mc alias set --api S3v4 "${PREFIX}src" "$SRC_ENDPOINT" "$SRC_KEY" "$SRC_SECRET" >/dev/null
mc alias set --api S3v4 "${PREFIX}dst" "$DST_ENDPOINT" "$DST_KEY" "$DST_SECRET" >/dev/null

# ---------- 2. inventaris sumber ----------
log "inventaris objek di ${PREFIX}src/$BUCKET"
mc ls --recursive "${PREFIX}src/${BUCKET}" > "$LIST" || {
  echo "ERROR: gagal membaca bucket sumber"; exit 1;
}
TOTAL=$(wc -l < "$LIST")
log "objek di bucket sumber: $TOTAL"
if [ "$TOTAL" -eq 0 ]; then
  echo "WARNING: bucket sumber kosong — tidak ada yang dimirror"; exit 1
fi

# ---------- 3. pastikan bucket tujuan ada ----------
if ! mc stat "${PREFIX}dst/${BUCKET}" >/dev/null 2>&1; then
  log "bucket tujuan belum ada — dibuat"
  mc mb "${PREFIX}dst/${BUCKET}" >/dev/null
fi

# ---------- 4. mirror ----------
if [ "$MIRROR_DRY_RUN" = "1" ]; then
  log "DRY RUN: mc mirror --dry-run"
  mc mirror --dry-run --overwrite --preserve "${PREFIX}src/${BUCKET}" "${PREFIX}dst/${BUCKET}" 2>&1 | tail -20
  log "dry run selesai — tidak ada byte yang ditulis"
  exit 0
fi

log "mirror: ${PREFIX}src/$BUCKET -> ${PREFIX}dst/$BUCKET (15-30 mnt untuk 5 GB)"
mc mirror --overwrite --preserve "${PREFIX}src/${BUCKET}" "${PREFIX}dst/${BUCKET}"
log "mirror selesai"

# ---------- 5. verifikasi: mc diff ----------
log "verifikasi: mc diff (nama & ukuran objek)"
if mc diff "${PREFIX}src/${BUCKET}" "${PREFIX}dst/${BUCKET}" > "$DIFF" 2>&1; then
  log "mc diff: tidak ada perbedaan ✓"
else
  log "mc diff menemukan perbedaan:"
  head -20 "$DIFF"
  echo "  total baris beda: $(wc -l < "$DIFF")"
  # Difference tidak selalu berarti kegagalan: kalau ada perubahan di
  # sumber yang terjadi setelah mirror, itu hal wajar.
  exit 1
fi

# ---------- 6. verifikasi: spot-check checksum ----------
log "verifikasi: checksum $MIRROR_SAMPLE objek (spot-check)"
# Ambil sampel spread out (bukan hanya 200 pertama) supaya objek dari
# folder yang berbeda ikut teruji.
SAMPLE_FILE="$WORK/sample.txt"
awk -v n="$MIRROR_SAMPLE" 'NR % int((NR==0)?1:1) == 0' "$LIST" | shuf -n "$MIRROR_SAMPLE" --random-source=<(yes) > "$SAMPLE_FILE" 2>/dev/null || \
  shuf -n "$MIRROR_SAMPLE" "$LIST" > "$SAMPLE_FILE"
SAMPLED=$(wc -l < "$SAMPLE_FILE")
log "objek dicek: $SAMPLED"

CHECKED=0
MISMATCH=0
while IFS= read -r line; do
  [ -n "$line" ] || continue
  # Format mc ls: "[2024-01-01 10:00:00 UTC]     1234B  path/to/obj"
  path=$(echo "$line" | awk '{print $NF}')
  [ -n "$path" ] || continue
  src_sum=$(mc cat "${PREFIX}src/${BUCKET}/${path}" 2>/dev/null | md5sum | cut -d' ' -f1)
  dst_sum=$(mc cat "${PREFIX}dst/${BUCKET}/${path}" 2>/dev/null | md5sum | cut -d' ' -f1)
  if [ -z "$src_sum" ] || [ "$src_sum" != "$dst_sum" ]; then
    MISMATCH=$((MISMATCH + 1))
    echo "  BEDA: $path"
  fi
  CHECKED=$((CHECKED + 1))
done < "$SAMPLE_FILE"

log "checksum dicek: $CHECKED, berbeda: $MISMATCH"
if [ "$MISMATCH" -ne 0 ]; then
  echo "ERROR: ada $MISMATCH objek yang berbeda isinya"
  exit 1
fi

log "MIRROR SELESAI & TERVERIFIKASI"
log "objek: $TOTAL, checksum dicek: $CHECKED"
