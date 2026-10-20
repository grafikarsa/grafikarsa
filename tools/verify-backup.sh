#!/usr/bin/env bash
# Verifikasi integritas backup Grafikarsa (lokal atau server).
#   ./verify-backup.sh /path/ke/<timestamp>
#
# Cek: checksum sha256, arsip storage terbaca, dump pg bisa dibaca-list.
# Tidak menyentuh data asli sama sekali (read-only).
set -uo pipefail

DIR="${1:-.}"
cd "$DIR" || { echo "FAIL: tidak bisa masuk $DIR"; exit 1; }
fail=0

echo "=== 1. checksum ==="
# MANIFEST.txt = daftar checksum (dipakai sha256sum -c).
# SHA256SUMS.check = LAPORAN hasil pengecekan (bukan daftar checksum).
if [ -f MANIFEST.txt ]; then
  if sha256sum -c MANIFEST.txt >/dev/null 2>&1; then
    echo "  OK semua file cocok ($(grep -c . MANIFEST.txt) file)"
  else
    echo "  FAIL: ada file yang tidak cocok:"
    sha256sum -c MANIFEST.txt 2>&1 | grep -v ': OK$' | head -10
    fail=1
  fi
  [ -f SHA256SUMS.check ] && echo "  (laporan: $(grep -c ': OK$' SHA256SUMS.check) file OK saat backup dibuat)"
else
  echo "  FAIL: MANIFEST.txt tidak ditemukan"; fail=1
fi

echo "=== 2. arsip object storage ==="
TAR=$(ls storage/*.tar.gz 2>/dev/null | head -1)
if [ -n "$TAR" ]; then
  n=$(tar -tzf "$TAR" 2>/dev/null | grep -c . )
  if [ "$n" -gt 0 ]; then echo "  OK: $n entri dalam $TAR"; else echo "  FAIL: arsip tidak terbaca"; fail=1; fi
else
  echo "  FAIL: arsip storage tidak ditemukan"; fail=1
fi

echo "=== 3. dump database ==="
DUMP=$(ls db/*.dump 2>/dev/null | head -1)
if [ -n "$DUMP" ]; then
  sz=$(stat -c%s "$DUMP")
  echo "  ukuran: $sz byte"
  if command -v pg_restore >/dev/null 2>&1; then
    pg_restore -l "$DUMP" >/dev/null 2>&1 && echo "  OK: pg_restore -l sukses" || { echo "  FAIL: pg_restore -l gagal"; fail=1; }
  else
    echo "  (pg_restore tidak ada di host — validasi lewat container:"
    echo "   docker cp $DUMP grafikarsa-db:/tmp/v.dump && docker exec grafikarsa-db pg_restore -l /tmp/v.dump)"
  fi
else
  echo "  FAIL: dump tidak ditemukan"; fail=1
fi

echo "=== 4. inventaris ==="
[ -f inventory/object-count-*.txt ] && echo "  object: $(cat inventory/object-count-*.txt)"
[ -f inventory/table-rowcounts-*.csv ] && echo "  tabel  : $(wc -l < inventory/table-rowcounts-*.csv)"
[ -f SUMMARY.txt ] && echo && cat SUMMARY.txt

echo
[ "$fail" -eq 0 ] && echo "HASIL: BACKUP SEHAT ✓" || echo "HASIL: ADA MASALAH ✗"
exit $fail
