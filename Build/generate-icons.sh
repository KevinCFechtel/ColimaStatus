#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
SOURCE_ADAPTIVE_ICON="${REPOSITORY_DIR}/assets/AppIcon.icon"
OUTPUT_ASSET_CATALOG="${SCRIPT_DIR}/Assets.car"
OUTPUT_PREVIEW_PNG="${REPOSITORY_DIR}/assets/AppIconPreview.png"
TEMP_DIR="$(mktemp -d /tmp/colimastatus-icon.XXXXXX)"
ASSET_OUTPUT_DIR="${TEMP_DIR}/asset-catalog"
ADAPTIVE_ICONSET_DIR="${TEMP_DIR}/AdaptiveAppIcon.iconset"
PARTIAL_INFO_PLIST="${TEMP_DIR}/asset-catalog-info.plist"

# shellcheck source=version.sh
source "${SCRIPT_DIR}/version.sh"

cleanup() {
  rm -rf -- "${TEMP_DIR}"
}
trap cleanup EXIT

if [[ ! -d "${SOURCE_ADAPTIVE_ICON}" ]]; then
  echo "Source icon is missing: ${SOURCE_ADAPTIVE_ICON}" >&2
  exit 1
fi

mkdir -p "${ASSET_OUTPUT_DIR}"

xcrun actool "${SOURCE_ADAPTIVE_ICON}" \
  --compile "${ASSET_OUTPUT_DIR}" \
  --platform macosx \
  --minimum-deployment-target "${APP_DEPLOYMENT_TARGET}" \
  --target-device mac \
  --app-icon AppIcon \
  --include-all-app-icons \
  --enable-on-demand-resources NO \
  --output-partial-info-plist "${PARTIAL_INFO_PLIST}"

# The preview image in the README is one rendering of the compiled icon, so it
# always matches what macOS shows.
iconutil --convert iconset \
  --output "${ADAPTIVE_ICONSET_DIR}" \
  "${ASSET_OUTPUT_DIR}/AppIcon.icns"

install -m 0644 "${ASSET_OUTPUT_DIR}/Assets.car" "${OUTPUT_ASSET_CATALOG}"
install -m 0644 \
  "${ADAPTIVE_ICONSET_DIR}/icon_128x128@2x.png" \
  "${OUTPUT_PREVIEW_PNG}"

echo "App icon created: ${OUTPUT_ASSET_CATALOG}, ${OUTPUT_PREVIEW_PNG}"
