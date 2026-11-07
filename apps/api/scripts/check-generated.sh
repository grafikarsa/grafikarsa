#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

go generate ./internal/http/apigen/

if git diff --exit-code -- internal/http/apigen; then
	echo "kode generate OpenAPI up to date"
else
	echo "kode generate OpenAPI tidak sinkron dengan packages/contracts/openapi.yaml" >&2
	echo "jalankan 'go generate ./internal/http/apigen/' di apps/api lalu commit hasilnya" >&2
	exit 1
fi
