#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
APP_DIR="${REPOSITORY_DIR}/dist/ColimaStatus.app"
CONTENTS_DIR="${APP_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"
INFO_PLIST="${CONTENTS_DIR}/Info.plist"
BUILDINFO_PACKAGE="github.com/KevinCFechtel/ColimaStatus/internal/buildinfo"

# shellcheck source=version.sh
source "${SCRIPT_DIR}/version.sh"

LDFLAGS=(
  -s -w
  "-X=${BUILDINFO_PACKAGE}.Version=${APP_VERSION}"
  "-X=${BUILDINFO_PACKAGE}.Build=${APP_BUILD_NUMBER}"
  "-X=${BUILDINFO_PACKAGE}.Commit=${APP_COMMIT}"
)

if [[ "${APP_DIR}" != "${REPOSITORY_DIR}/dist/ColimaStatus.app" ]]; then
  echo "Unexpected app path: ${APP_DIR}" >&2
  exit 1
fi

rm -rf -- "${APP_DIR}"
mkdir -p "${MACOS_DIR}" "${RESOURCES_DIR}"
install -m 0644 "${SCRIPT_DIR}/Info.plist" "${INFO_PLIST}"
install -m 0644 "${SCRIPT_DIR}/AppIcon.icns" "${RESOURCES_DIR}/AppIcon.icns"
install -m 0644 "${SCRIPT_DIR}/Assets.car" "${RESOURCES_DIR}/Assets.car"

/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString ${APP_VERSION}" "${INFO_PLIST}"
/usr/libexec/PlistBuddy -c "Set :CFBundleVersion ${APP_BUILD_NUMBER}" "${INFO_PLIST}"

cd "${REPOSITORY_DIR}"
MACOSX_DEPLOYMENT_TARGET=11.0 \
  CGO_CFLAGS="${CGO_CFLAGS:-} -mmacosx-version-min=11.0" \
  CGO_LDFLAGS="${CGO_LDFLAGS:-} -mmacosx-version-min=11.0" \
  CGO_ENABLED=1 GOOS=darwin GOARCH="${GOARCH:-$(go env GOARCH)}" \
  go build -buildvcs=false -trimpath -o "${MACOS_DIR}/ColimaStatus" -ldflags "${LDFLAGS[*]}" ./cmd/colimastatus

if command -v codesign >/dev/null 2>&1; then
  codesign --force --deep --sign - "${APP_DIR}"
fi

BUILT_VERSION="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "${INFO_PLIST}")"
BUILT_NUMBER="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleVersion' "${INFO_PLIST}")"
if [[ "${BUILT_VERSION}" != "${APP_VERSION}" || "${BUILT_NUMBER}" != "${APP_BUILD_NUMBER}" ]]; then
  echo "Versionsdaten im App-Bundle stimmen nicht mit VERSION und BUILD_NUMBER überein." >&2
  exit 1
fi

echo "ColimaStatus ${APP_VERSION} (Build ${APP_BUILD_NUMBER}) erstellt: ${APP_DIR}"
