#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
SOURCE_ICON="${REPOSITORY_DIR}/assets/AppIcon.png"
OUTPUT_ICON="${SCRIPT_DIR}/AppIcon.icns"
TEMP_DIR="$(mktemp -d /tmp/colimastatus-icon.XXXXXX)"
ICONSET_DIR="${TEMP_DIR}/ColimaStatus.iconset"

cleanup() {
  rm -rf -- "${TEMP_DIR}"
}
trap cleanup EXIT

if [[ ! -f "${SOURCE_ICON}" ]]; then
  echo "Source icon is missing: ${SOURCE_ICON}" >&2
  exit 1
fi

mkdir -p "${ICONSET_DIR}"

sips -z 16 16 "${SOURCE_ICON}" --out "${ICONSET_DIR}/icon_16x16.png" >/dev/null
sips -z 32 32 "${SOURCE_ICON}" --out "${ICONSET_DIR}/icon_16x16@2x.png" >/dev/null
sips -z 32 32 "${SOURCE_ICON}" --out "${ICONSET_DIR}/icon_32x32.png" >/dev/null
sips -z 64 64 "${SOURCE_ICON}" --out "${ICONSET_DIR}/icon_32x32@2x.png" >/dev/null
sips -z 128 128 "${SOURCE_ICON}" --out "${ICONSET_DIR}/icon_128x128.png" >/dev/null
sips -z 256 256 "${SOURCE_ICON}" --out "${ICONSET_DIR}/icon_128x128@2x.png" >/dev/null
sips -z 256 256 "${SOURCE_ICON}" --out "${ICONSET_DIR}/icon_256x256.png" >/dev/null
sips -z 512 512 "${SOURCE_ICON}" --out "${ICONSET_DIR}/icon_256x256@2x.png" >/dev/null
sips -z 512 512 "${SOURCE_ICON}" --out "${ICONSET_DIR}/icon_512x512.png" >/dev/null
sips -z 1024 1024 "${SOURCE_ICON}" --out "${ICONSET_DIR}/icon_512x512@2x.png" >/dev/null

cd "${REPOSITORY_DIR}"
go run ./tools/icnspack "${ICONSET_DIR}" "${OUTPUT_ICON}"

echo "App icon created: ${OUTPUT_ICON}"
