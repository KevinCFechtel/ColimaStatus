#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"

cd "${REPOSITORY_DIR}"
# Pin the scanner, not its advisory database: scheduled runs must still find
# newly disclosed vulnerabilities without a source change.
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
