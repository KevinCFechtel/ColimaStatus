#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"

cd "${REPOSITORY_DIR}"
"${SCRIPT_DIR}/version.sh" >/dev/null
"${SCRIPT_DIR}/localization.sh"
go test -race ./...
