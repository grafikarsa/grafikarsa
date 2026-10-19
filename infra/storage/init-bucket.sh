#!/usr/bin/env bash
# Buat bucket S3 secara idempoten (agnostik vendor: RustFS/MinIO/R2/S3).
# Usage: S3_ENDPOINT=... S3_ACCESS_KEY=... S3_SECRET_KEY=... ./init-bucket.sh [bucket]
set -euo pipefail

BUCKET="${1:-${S3_BUCKET:-grafikarsa}}"
ENDPOINT="${S3_ENDPOINT:?S3_ENDPOINT wajib diisi}"
export AWS_ACCESS_KEY_ID="${S3_ACCESS_KEY:?S3_ACCESS_KEY wajib diisi}"
export AWS_SECRET_ACCESS_KEY="${S3_SECRET_KEY:?S3_SECRET_KEY wajib diisi}"

if command -v mc >/dev/null 2>&1; then
  mc alias set init "${ENDPOINT}" "${AWS_ACCESS_KEY_ID}" "${AWS_SECRET_ACCESS_KEY}" >/dev/null
  mc mb --ignore-existing "init/${BUCKET}"
else
  aws --endpoint-url "${ENDPOINT}" s3 mb "s3://${BUCKET}" 2>&1 | grep -v AlreadyExists || true
fi
echo "bucket OK: ${BUCKET}"
