#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
APP_DIR="${REPOSITORY_DIR}/dist/ColimaStatus.app"
CONTENTS_DIR="${APP_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"

if [[ "${APP_DIR}" != "${REPOSITORY_DIR}/dist/ColimaStatus.app" ]]; then
  echo "Unexpected app path: ${APP_DIR}" >&2
  exit 1
fi

rm -rf -- "${APP_DIR}"
mkdir -p "${MACOS_DIR}" "${RESOURCES_DIR}"
install -m 0644 "${SCRIPT_DIR}/Info.plist" "${CONTENTS_DIR}/Info.plist"
install -m 0644 "${SCRIPT_DIR}/AppIcon.icns" "${RESOURCES_DIR}/AppIcon.icns"

cd "${REPOSITORY_DIR}"
MACOSX_DEPLOYMENT_TARGET=11.0 \
  CGO_CFLAGS="${CGO_CFLAGS:-} -mmacosx-version-min=11.0" \
  CGO_LDFLAGS="${CGO_LDFLAGS:-} -mmacosx-version-min=11.0" \
  CGO_ENABLED=1 GOOS=darwin GOARCH="${GOARCH:-$(go env GOARCH)}" \
  go build -buildvcs=false -trimpath -o "${MACOS_DIR}/ColimaStatus" -ldflags "-s -w" ./cmd/colimastatus

if command -v codesign >/dev/null 2>&1; then
  codesign --force --deep --sign - "${APP_DIR}"
fi

echo "ColimaStatus app created: ${APP_DIR}"
