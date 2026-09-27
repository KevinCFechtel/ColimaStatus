#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
APP_DIR="${REPOSITORY_DIR}/dist/ColimaStatus.app"
CONTENTS_DIR="${APP_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"
INFO_PLIST="${CONTENTS_DIR}/Info.plist"
APP_EXECUTABLE="${MACOS_DIR}/ColimaStatus"
BUILDINFO_PACKAGE="github.com/KevinCFechtel/ColimaStatus/internal/buildinfo"

# Releases ship a universal binary so that Intel Macs are covered too. The
# Homebrew tap rejects per-architecture releases, because `brew install --cask`
# does not detect an architecture mismatch and would hand an Intel Mac an app
# that cannot launch.
# Set COLIMASTATUS_ARCHS=arm64 for a faster single-architecture development build.
#
# The default is substituted only when the variable is unset, not when it is
# empty, so that an explicitly empty value reaches the check below instead of
# silently producing a full universal build.
read -r -a TARGET_ARCHS <<<"${COLIMASTATUS_ARCHS-arm64 amd64}"

# shellcheck source=version.sh
source "${SCRIPT_DIR}/version.sh"

LDFLAGS=(
  -s -w
  "-X=${BUILDINFO_PACKAGE}.Version=${APP_VERSION}"
  "-X=${BUILDINFO_PACKAGE}.Build=${APP_BUILD_NUMBER}"
  "-X=${BUILDINFO_PACKAGE}.Commit=${APP_COMMIT}"
)

if [[ ${#TARGET_ARCHS[@]} -eq 0 ]]; then
  echo "COLIMASTATUS_ARCHS is empty; expected at least one of arm64 or amd64." >&2
  exit 1
fi

if [[ "${APP_DIR}" != "${REPOSITORY_DIR}/dist/ColimaStatus.app" ]]; then
  echo "Unexpected app path: ${APP_DIR}" >&2
  exit 1
fi

rm -rf -- "${APP_DIR}"
mkdir -p "${MACOS_DIR}" "${RESOURCES_DIR}"
install -m 0644 "${SCRIPT_DIR}/Info.plist" "${INFO_PLIST}"
install -m 0644 "${SCRIPT_DIR}/Assets.car" "${RESOURCES_DIR}/Assets.car"

/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString ${APP_VERSION}" "${INFO_PLIST}"
/usr/libexec/PlistBuddy -c "Set :CFBundleVersion ${APP_BUILD_NUMBER}" "${INFO_PLIST}"
/usr/libexec/PlistBuddy -c "Set :LSMinimumSystemVersion ${APP_DEPLOYMENT_TARGET}" "${INFO_PLIST}"

BUILD_DIR="$(mktemp -d /tmp/colimastatus-build.XXXXXX)"
cleanup() {
  rm -rf -- "${BUILD_DIR}"
}
trap cleanup EXIT

cd "${REPOSITORY_DIR}"

# clang_arch translates a Go architecture into the clang -arch value that cgo
# needs when cross-compiling the Objective-C bridges.
clang_arch() {
  case "$1" in
    arm64) printf 'arm64' ;;
    amd64) printf 'x86_64' ;;
    *)
      echo "Unsupported architecture: $1 (expected arm64 or amd64)" >&2
      return 1
      ;;
  esac
}

SLICES=()
for arch in "${TARGET_ARCHS[@]}"; do
  clang_target="$(clang_arch "${arch}")"
  slice="${BUILD_DIR}/ColimaStatus-${arch}"
  echo "Building ${arch} slice"
  MACOSX_DEPLOYMENT_TARGET="${APP_DEPLOYMENT_TARGET}" \
    CC="clang -arch ${clang_target}" \
    CXX="clang++ -arch ${clang_target}" \
    CGO_CFLAGS="${CGO_CFLAGS:-} -mmacosx-version-min=${APP_DEPLOYMENT_TARGET}" \
    CGO_LDFLAGS="${CGO_LDFLAGS:-} -mmacosx-version-min=${APP_DEPLOYMENT_TARGET}" \
    CGO_ENABLED=1 GOOS=darwin GOARCH="${arch}" \
    go build -buildvcs=false -trimpath -o "${slice}" -ldflags "${LDFLAGS[*]}" ./cmd/colimastatus
  SLICES+=("${slice}")
done

if [[ ${#SLICES[@]} -eq 1 ]]; then
  install -m 0755 "${SLICES[0]}" "${APP_EXECUTABLE}"
else
  lipo -create -output "${APP_EXECUTABLE}" "${SLICES[@]}"
  chmod 0755 "${APP_EXECUTABLE}"
fi

# Ad-hoc signature for local development. Releases are re-signed with a
# Developer ID by Build/release.sh. --deep is deliberately not used: Apple
# discourages it for signing, and the bundle has no nested code anyway.
if command -v codesign >/dev/null 2>&1; then
  codesign --force --sign - "${APP_DIR}"
fi

BUILT_VERSION="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "${INFO_PLIST}")"
BUILT_NUMBER="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleVersion' "${INFO_PLIST}")"
if [[ "${BUILT_VERSION}" != "${APP_VERSION}" || "${BUILT_NUMBER}" != "${APP_BUILD_NUMBER}" ]]; then
  echo "Version metadata in the app bundle does not match VERSION and BUILD_NUMBER." >&2
  echo "Expected: ${APP_VERSION} (${APP_BUILD_NUMBER})" >&2
  echo "Found:    ${BUILT_VERSION} (${BUILT_NUMBER})" >&2
  exit 1
fi

# The declared minimum system version is only credible if the linked binary
# agrees. Without this check a Go toolchain upgrade can silently raise the real
# floor while LSMinimumSystemVersion keeps promising an older macOS release.
BUILT_MINIMUM_VERSION="$(/usr/libexec/PlistBuddy -c 'Print :LSMinimumSystemVersion' "${INFO_PLIST}")"
if [[ "${BUILT_MINIMUM_VERSION}" != "${APP_DEPLOYMENT_TARGET}" ]]; then
  echo "LSMinimumSystemVersion does not match the deployment target." >&2
  echo "Expected: ${APP_DEPLOYMENT_TARGET}" >&2
  echo "Found:    ${BUILT_MINIMUM_VERSION}" >&2
  exit 1
fi

# vtool prints one minos line per architecture, so every slice of a universal
# binary is verified rather than just the first.
SLICE_MINIMUMS=()
while IFS= read -r slice_minimum; do
  SLICE_MINIMUMS+=("${slice_minimum}")
done < <(xcrun vtool -show-build-version "${APP_EXECUTABLE}" | awk '/minos/ {print $2}')

if [[ ${#SLICE_MINIMUMS[@]} -ne ${#SLICES[@]} ]]; then
  echo "Expected ${#SLICES[@]} architecture slices to report a minimum macOS version," >&2
  echo "but vtool reported ${#SLICE_MINIMUMS[@]}." >&2
  exit 1
fi

for slice_minimum in "${SLICE_MINIMUMS[@]}"; do
  if [[ "${slice_minimum}" != "${APP_DEPLOYMENT_TARGET}" ]]; then
    echo "A binary slice targets macOS ${slice_minimum}, expected ${APP_DEPLOYMENT_TARGET}." >&2
    xcrun vtool -show-build-version "${APP_EXECUTABLE}" >&2
    exit 1
  fi
done

BUILT_ARCHS="$(lipo -archs "${APP_EXECUTABLE}")"
echo "ColimaStatus ${APP_VERSION} (build ${APP_BUILD_NUMBER}, macOS ${APP_DEPLOYMENT_TARGET}+) created: ${APP_DIR}"
echo "Architectures: ${BUILT_ARCHS}"
