#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
SOURCE_ICON="${REPOSITORY_DIR}/assets/AppIcon.png"
SOURCE_ADAPTIVE_ICON="${REPOSITORY_DIR}/assets/AppIcon.icon"
ADAPTIVE_FOREGROUND="${SOURCE_ADAPTIVE_ICON}/Assets/Llama.png"
OUTPUT_ICON="${SCRIPT_DIR}/AppIcon.icns"
OUTPUT_ASSET_CATALOG="${SCRIPT_DIR}/Assets.car"
OUTPUT_PREVIEW_PNG="${REPOSITORY_DIR}/assets/AppIconPreview.png"
TEMP_DIR="$(mktemp -d /tmp/colimastatus-icon.XXXXXX)"
ICONSET_DIR="${TEMP_DIR}/ColimaStatus.iconset"
ASSET_OUTPUT_DIR="${TEMP_DIR}/asset-catalog"
ADAPTIVE_ICONSET_DIR="${TEMP_DIR}/AdaptiveAppIcon.iconset"
PARTIAL_INFO_PLIST="${TEMP_DIR}/asset-catalog-info.plist"

cleanup() {
  rm -rf -- "${TEMP_DIR}"
}
trap cleanup EXIT

if [[ ! -f "${SOURCE_ICON}" ]]; then
  echo "Source icon is missing: ${SOURCE_ICON}" >&2
  exit 1
fi

mkdir -p "${ICONSET_DIR}" "${ASSET_OUTPUT_DIR}" "$(dirname -- "${ADAPTIVE_FOREGROUND}")"

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

#go run ./tools/appiconforeground "${SOURCE_ICON}" "${ADAPTIVE_FOREGROUND}"

xcrun actool "${SOURCE_ADAPTIVE_ICON}" \
  --compile "${ASSET_OUTPUT_DIR}" \
  --platform macosx \
  --minimum-deployment-target 11.0 \
  --target-device mac \
  --app-icon AppIcon \
  --include-all-app-icons \
  --enable-on-demand-resources NO \
  --output-partial-info-plist "${PARTIAL_INFO_PLIST}"

iconutil --convert iconset \
  --output "${ADAPTIVE_ICONSET_DIR}" \
  "${ASSET_OUTPUT_DIR}/AppIcon.icns"

install -m 0644 "${ASSET_OUTPUT_DIR}/Assets.car" "${OUTPUT_ASSET_CATALOG}"
install -m 0644 \
  "${ADAPTIVE_ICONSET_DIR}/icon_128x128@2x.png" \
  "${OUTPUT_PREVIEW_PNG}"

echo "App icons created: ${OUTPUT_ICON}, ${OUTPUT_ASSET_CATALOG}, ${OUTPUT_PREVIEW_PNG}"
