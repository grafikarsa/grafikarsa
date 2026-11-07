#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/src" -w /src sqlc/sqlc:1.31.1 "${@:-generate}"
