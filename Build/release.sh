#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
APP_DIR="${REPOSITORY_DIR}/dist/ColimaStatus.app"
RELEASE_DIR="${REPOSITORY_DIR}/dist/release"
RELEASE_ENV_FILE="${COLIMASTATUS_RELEASE_ENV_FILE:-${SCRIPT_DIR}/.env}"
EXPECTED_BUNDLE_ID="dev.kevincfechtel.ColimaStatus"

# shellcheck source=version.sh
source "${SCRIPT_DIR}/version.sh"

if [[ -f "${RELEASE_ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${RELEASE_ENV_FILE}"
  set +a
fi

SIGNING_IDENTITY="${SIGNING_IDENTITY:-}"
NOTARY_PROFILE="${NOTARY_PROFILE:-}"
SIGNING_TIMESTAMP_URL="${SIGNING_TIMESTAMP_URL:-}"
NOTARY_TIMEOUT="${NOTARY_TIMEOUT:-30m}"
# Releases are always universal so that Apple Silicon and Intel Macs share one
# download. Build/build.sh produces both slices and lipo merges them. The
# Homebrew tap rejects per-architecture releases.
# COLIMASTATUS_PREBUILT_APP adopts an already built bundle instead of building
# one. Build/local-release.sh points it at the attested archive from the tag's
# workflow run, which keeps the maintainer's working tree out of the shipped
# binary. Empty means build locally, which stays the default.
PREBUILT_APP="${COLIMASTATUS_PREBUILT_APP:-}"

RELEASE_ARCHS="arm64 amd64"
RELEASE_ARCH_LABEL="universal"
REQUIRED_SLICES=(arm64 x86_64)
RELEASE_VERSION="${APP_VERSION}"
RELEASE_BUILD_NUMBER="${APP_BUILD_NUMBER}"
BUNDLE_ID="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "${SCRIPT_DIR}/Info.plist")"

if [[ -z "${SIGNING_IDENTITY}" ]]; then
  echo "SIGNING_IDENTITY is missing (Developer ID Application)." >&2
  exit 1
fi

if [[ -z "${NOTARY_PROFILE}" ]]; then
  echo "NOTARY_PROFILE is missing (the name of a notarytool keychain profile)." >&2
  exit 1
fi

if [[ "${BUNDLE_ID}" != "${EXPECTED_BUNDLE_ID}" ]]; then
  echo "Unexpected bundle identifier: ${BUNDLE_ID}" >&2
  exit 1
fi

if command -v git >/dev/null 2>&1 && git -C "${REPOSITORY_DIR}" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  EXPECTED_RELEASE_TAG="v${RELEASE_VERSION}"
  RELEASE_TAGS="$(git -C "${REPOSITORY_DIR}" tag --points-at HEAD --list 'v*')"
  if [[ -n "${RELEASE_TAGS}" ]] && ! grep -Fx -- "${EXPECTED_RELEASE_TAG}" <<<"${RELEASE_TAGS}" >/dev/null; then
    echo "The release tag on the current commit does not match VERSION." >&2
    echo "Expected: ${EXPECTED_RELEASE_TAG}" >&2
    echo "Found:    ${RELEASE_TAGS}" >&2
    exit 1
  fi
fi

for command_name in awk codesign dscacheutil ditto go grep lipo security spctl xcrun; do
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "Required program is missing: ${command_name}" >&2
    exit 1
  fi
done

if ! security find-identity -v -p codesigning | grep -F -- "${SIGNING_IDENTITY}" >/dev/null; then
  echo "SIGNING_IDENTITY was not found as a valid code signing identity." >&2
  exit 1
fi

if [[ -z "${SIGNING_TIMESTAMP_URL}" ]]; then
  timestamp_ipv4="$({
    dscacheutil -q host -a name timestamp.apple.com || true
  } | awk '/ip_address:/ && $2 ~ /^[0-9.]+$/ {print $2; exit}')"

  if [[ -z "${timestamp_ipv4}" ]]; then
    echo "No IPv4 address found for timestamp.apple.com." >&2
    echo "Set SIGNING_TIMESTAMP_URL explicitly instead." >&2
    exit 1
  fi

  SIGNING_TIMESTAMP_URL="http://${timestamp_ipv4}/ts01"
fi

SUBMISSION_ARCHIVE="${RELEASE_DIR}/ColimaStatus-${RELEASE_VERSION}-notarization.zip"
FINAL_ARCHIVE="${RELEASE_DIR}/ColimaStatus-${RELEASE_VERSION}-macos-${RELEASE_ARCH_LABEL}.zip"
CHECK_DIR="$(mktemp -d /tmp/colimastatus-release-check.XXXXXX)"

cleanup() {
  rm -rf -- "${CHECK_DIR}"
}
trap cleanup EXIT

verify_bundle_version() {
  local bundle_path="$1"
  local bundle_info_plist="${bundle_path}/Contents/Info.plist"
  local actual_version
  local actual_build_number

  actual_version="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "${bundle_info_plist}")"
  actual_build_number="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleVersion' "${bundle_info_plist}")"
  if [[ "${actual_version}" != "${RELEASE_VERSION}" || "${actual_build_number}" != "${RELEASE_BUILD_NUMBER}" ]]; then
    echo "Version metadata does not match VERSION and BUILD_NUMBER: ${bundle_path}" >&2
    echo "Expected: ${RELEASE_VERSION} (${RELEASE_BUILD_NUMBER})" >&2
    echo "Found:    ${actual_version} (${actual_build_number})" >&2
    exit 1
  fi
}

# verify_bundle_slices fails the release if a slice is missing, so that an
# Intel Mac can never be handed an Apple-Silicon-only build.
verify_bundle_slices() {
  local bundle_path="$1"
  local actual_slices
  actual_slices="$(lipo -archs "${bundle_path}/Contents/MacOS/ColimaStatus")"

  local required
  for required in "${REQUIRED_SLICES[@]}"; do
    if [[ " ${actual_slices} " != *" ${required} "* ]]; then
      echo "The build is missing the ${required} slice: ${bundle_path}" >&2
      echo "Found: ${actual_slices}" >&2
      exit 1
    fi
  done
}

if [[ -n "${PREBUILT_APP}" ]]; then
  echo "1/8 Adopting the prebuilt app bundle"
  if [[ ! -d "${PREBUILT_APP}" ]]; then
    echo "COLIMASTATUS_PREBUILT_APP is not a bundle directory: ${PREBUILT_APP}" >&2
    exit 1
  fi

  # The remaining steps all operate on APP_DIR, so a bundle from elsewhere is
  # copied in. ditto is used rather than cp because it keeps permissions and
  # the bundle structure intact.
  PREBUILT_APP="$(cd -- "${PREBUILT_APP}" && pwd)"
  if [[ "${PREBUILT_APP}" != "${APP_DIR}" ]]; then
    rm -rf -- "${APP_DIR}"
    mkdir -p -- "$(dirname -- "${APP_DIR}")"
    ditto "${PREBUILT_APP}" "${APP_DIR}"
  fi
else
  echo "1/8 Building the ColimaStatus app"
  COLIMASTATUS_ARCHS="${RELEASE_ARCHS}" "${SCRIPT_DIR}/build.sh"
fi

# Both paths are verified the same way: a prebuilt bundle is not trusted more
# than a local build.
verify_bundle_version "${APP_DIR}"
verify_bundle_slices "${APP_DIR}"

echo "2/8 Signing with Developer ID and hardened runtime"
codesign \
  --force \
  --options runtime \
  --timestamp="${SIGNING_TIMESTAMP_URL}" \
  --sign "${SIGNING_IDENTITY}" \
  "${APP_DIR}"

codesign --verify --deep --strict --verbose=4 "${APP_DIR}"

mkdir -p "${RELEASE_DIR}"

echo "3/8 Creating the notarization archive"
rm -f -- "${SUBMISSION_ARCHIVE}"
COPYFILE_DISABLE=1 ditto \
  -c -k \
  --keepParent \
  --norsrc \
  --noextattr \
  "${APP_DIR}" \
  "${SUBMISSION_ARCHIVE}"

echo "4/8 Submitting to Apple and waiting for the result"
xcrun notarytool submit \
  "${SUBMISSION_ARCHIVE}" \
  --keychain-profile "${NOTARY_PROFILE}" \
  --wait \
  --timeout "${NOTARY_TIMEOUT}"

echo "5/8 Stapling the notarization ticket to the app"
xcrun stapler staple "${APP_DIR}"
xcrun stapler validate "${APP_DIR}"

echo "6/8 Verifying the signature and Gatekeeper assessment"
codesign --verify --deep --strict --verbose=4 "${APP_DIR}"
spctl --assess --type execute --verbose=4 "${APP_DIR}"

echo "7/8 Creating a clean release archive without AppleDouble files"
rm -f -- "${FINAL_ARCHIVE}"
COPYFILE_DISABLE=1 ditto \
  -c -k \
  --keepParent \
  --norsrc \
  --noextattr \
  "${APP_DIR}" \
  "${FINAL_ARCHIVE}"

echo "8/8 Extracting the release archive again and verifying it in full"
ditto -x -k "${FINAL_ARCHIVE}" "${CHECK_DIR}"
verify_bundle_version "${CHECK_DIR}/ColimaStatus.app"
verify_bundle_slices "${CHECK_DIR}/ColimaStatus.app"
xcrun stapler validate "${CHECK_DIR}/ColimaStatus.app"
codesign --verify --deep --strict --verbose=4 "${CHECK_DIR}/ColimaStatus.app"
spctl --assess --type execute --verbose=4 "${CHECK_DIR}/ColimaStatus.app"

echo "Release ${RELEASE_VERSION} (build ${RELEASE_BUILD_NUMBER}) created: ${FINAL_ARCHIVE}"
