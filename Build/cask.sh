#!/usr/bin/env bash
set -euo pipefail

# Generates the Homebrew cask for the current release archive.
#
# The cask itself lives in a separate tap repository (a GitHub repository named
# homebrew-tap). This script only produces the file so that version and
# checksum can never drift from the artifact that Build/release.sh produced.
# The tap's CONTRIBUTING.md is the contract this output has to meet; a generated
# file edited in the tap is overwritten by the next release without a conflict,
# so corrections belong here.
#
# Usage:
#   ./Build/release.sh          # produce and verify the release archive
#   ./Build/cask.sh             # print the cask
#   ./Build/cask.sh Casks/colimastatus.rb  # or write it somewhere
#   ./Build/cask.sh --verify-published     # compare the checksum against the
#                                          # asset already uploaded to the release

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
RELEASE_DIR="${REPOSITORY_DIR}/dist/release"
RELEASE_BASE_URL="https://github.com/KevinCFechtel/ColimaStatus/releases/download"

OUTPUT_PATH=""
VERIFY_PUBLISHED="no"
for argument in "$@"; do
  case "${argument}" in
    --verify-published) VERIFY_PUBLISHED="yes" ;;
    -*)
      echo "Unknown option: ${argument}" >&2
      exit 1
      ;;
    *) OUTPUT_PATH="${argument}" ;;
  esac
done

# shellcheck source=version.sh
source "${SCRIPT_DIR}/version.sh"

ARCHIVE_NAME="ColimaStatus-${APP_VERSION}-macos-universal.zip"
ARCHIVE="${RELEASE_DIR}/${ARCHIVE_NAME}"

if [[ ! -f "${ARCHIVE}" ]]; then
  echo "Release archive is missing: ${ARCHIVE}" >&2
  echo "Run ./Build/release.sh first." >&2
  exit 1
fi

ARCHIVE_SHA256="$(shasum -a 256 "${ARCHIVE}" | awk '{print $1}')"
if [[ -z "${ARCHIVE_SHA256}" ]]; then
  echo "Checksum for ${ARCHIVE} could not be computed." >&2
  exit 1
fi

# Homebrew identifies macOS releases by symbol, and `depends_on macos:` takes a
# bare symbol meaning "this release or newer". The symbol has to agree with
# LSMinimumSystemVersion in the built bundle, because `brew audit --online`
# reads that key out of the app and fails on a mismatch. Deriving it from
# APP_DEPLOYMENT_TARGET keeps the cask correct when the deployment target moves
# instead of leaving a stale literal behind. Table from Homebrew 7.0.6:
# brew ruby -e 'MacOSVersion::SYMBOLS.each { |s, v| puts "#{v} #{s}" }'
macos_symbol_for_target() {
  case "${1%%.*}" in
    11) printf 'big_sur' ;;
    12) printf 'monterey' ;;
    13) printf 'ventura' ;;
    14) printf 'sonoma' ;;
    15) printf 'sequoia' ;;
    26) printf 'tahoe' ;;
    27) printf 'golden_gate' ;;
    *) return 1 ;;
  esac
}

if ! MACOS_SYMBOL="$(macos_symbol_for_target "${APP_DEPLOYMENT_TARGET}")"; then
  echo "No Homebrew macOS symbol is known for deployment target ${APP_DEPLOYMENT_TARGET}." >&2
  echo "List the symbols with" >&2
  echo "  brew ruby -e 'MacOSVersion::SYMBOLS.each { |s, v| puts \"#{v} #{s}\" }'" >&2
  echo "and extend macos_symbol_for_target in ${BASH_SOURCE[0]}." >&2
  exit 1
fi

# brew audit --online downloads the published asset and compares it against
# sha256, so the checksum has to describe the bytes that were actually
# uploaded. Re-running release.sh produces a fresh archive with a different
# checksum, which is how a cask ends up failing audit in the shape of a
# tampered download.
if [[ "${VERIFY_PUBLISHED}" == "yes" ]]; then
  ASSET_URL="${RELEASE_BASE_URL}/v${APP_VERSION}/${ARCHIVE_NAME}"
  DOWNLOADED_ASSET="$(mktemp -t colimastatus-cask-asset)"
  trap 'rm -f "${DOWNLOADED_ASSET}"' EXIT

  echo "Downloading the published asset: ${ASSET_URL}" >&2
  if ! curl --fail --location --silent --show-error \
    --output "${DOWNLOADED_ASSET}" "${ASSET_URL}"; then
    echo "The published asset could not be downloaded." >&2
    echo "Publish the release and upload ${ARCHIVE_NAME} before generating the cask." >&2
    exit 1
  fi

  PUBLISHED_SHA256="$(shasum -a 256 "${DOWNLOADED_ASSET}" | awk '{print $1}')"
  if [[ "${PUBLISHED_SHA256}" != "${ARCHIVE_SHA256}" ]]; then
    echo "The published asset does not match the local archive." >&2
    echo "  local:     ${ARCHIVE_SHA256}" >&2
    echo "  published: ${PUBLISHED_SHA256}" >&2
    echo "Upload ${ARCHIVE} to the release, or regenerate the cask from the" >&2
    echo "archive that was uploaded. brew audit --online would fail otherwise." >&2
    exit 1
  fi
  echo "The published asset matches the local archive." >&2
fi

# zap has to name every path the app creates, because what is missing here is
# what `brew uninstall --cask --zap` leaves behind.
#   Application Support: config.json
#   Logs:                colimastatus.log and its one rotated generation
#   Preferences:         written by AppKit for the menu bar item position
# The login item is registered through SMAppService, which keeps its state in
# the system's background task database rather than in a file a cask could
# remove; `brew uninstall` unregisters nothing, so a reinstall may still launch
# at login.
ZAP_PATHS=(
  "~/Library/Application Support/ColimaStatus"
  "~/Library/Logs/ColimaStatus"
  "~/Library/Preferences/dev.kevincfechtel.ColimaStatus.plist"
)

# brew style's Cask/ArrayAlphabetization rejects a bracketed single-element
# array and requires a longer one to be sorted, so the stanza is generated
# instead of written out: adding a path here stays correct either way.
SORTED_ZAP_PATHS=()
while IFS= read -r zap_path; do
  SORTED_ZAP_PATHS+=("${zap_path}")
done < <(printf '%s\n' "${ZAP_PATHS[@]}" | sort)

if [[ ${#SORTED_ZAP_PATHS[@]} -eq 1 ]]; then
  ZAP_STANZA="  zap trash: \"${SORTED_ZAP_PATHS[0]}\""
else
  ZAP_STANZA="  zap trash: ["
  for zap_path in "${SORTED_ZAP_PATHS[@]}"; do
    ZAP_STANZA+=$'\n'"    \"${zap_path}\","
  done
  ZAP_STANZA+=$'\n'"  ]"
fi

CASK_CONTENTS="$(
  cat <<CASK
cask "colimastatus" do
  version "${APP_VERSION}"
  sha256 "${ARCHIVE_SHA256}"

  url "${RELEASE_BASE_URL}/v#{version}/ColimaStatus-#{version}-macos-universal.zip"
  name "ColimaStatus"
  desc "Menu bar app for the Colima container runtime"
  homepage "https://github.com/KevinCFechtel/ColimaStatus"

  livecheck do
    url :url
    strategy :github_latest
  end

  depends_on macos: :${MACOS_SYMBOL}

  app "ColimaStatus.app"

${ZAP_STANZA}
end
CASK
)"

if [[ -n "${OUTPUT_PATH}" ]]; then
  printf '%s\n' "${CASK_CONTENTS}" >"${OUTPUT_PATH}"
  echo "Cask for ${APP_VERSION} written: ${OUTPUT_PATH}"
  echo "SHA-256: ${ARCHIVE_SHA256}"
else
  printf '%s\n' "${CASK_CONTENTS}"
fi
