#!/usr/bin/env bash
set -euo pipefail

# Drives a complete release from a clean main checkout to a pushed cask.
#
# The steps themselves live in the scripts this one calls; the value here is the
# order and the preconditions. Two orderings are not interchangeable:
#
#   * The release asset has to exist before the cask is generated, because
#     brew audit --online downloads it and compares it against sha256. A cask
#     that lands first turns the tap's audit CI red.
#   * The signed archive comes from the tag's workflow run, not from a local
#     build, so the maintainer's working tree is out of the shipped binary.
#     The archive carries a GitHub build provenance attestation that binds
#     those exact bytes to the repository, the workflow and the commit.
#
# Signing and notarization need the private Developer ID and a notarytool
# keychain profile, so this cannot run in CI and deliberately refuses to.
#
# Usage:
#   ./Build/local-release.sh --dry-run   # run every check, change nothing
#   ./Build/local-release.sh             # release, asking before each push
#   ./Build/local-release.sh --yes       # release without asking
#
# Options:
#   --dry-run            Check only. Does not sign, tag, publish, or push.
#   --yes                Do not ask before the steps that change remote state.
#   --tap PATH           The homebrew-tap checkout (default ../homebrew-tap).
#   --skip-ci-check      Do not require a green CI run on the release commit.
#   --skip-tap           Stop after the GitHub release; do not touch the tap.
#   --allow-unattested   Accept a workflow build without a provenance
#                        attestation. Only for the first release after the
#                        workflow gained attestation; it drops the proof that
#                        the signed bundle came from the tagged commit.

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
RELEASE_DIR="${REPOSITORY_DIR}/dist/release"

DRY_RUN="no"
ASSUME_YES="no"
SKIP_CI_CHECK="no"
SKIP_TAP="no"
ALLOW_UNATTESTED="no"
TAP_DIR="${REPOSITORY_DIR}/../homebrew-tap"
TAP_NAME="kevincfechtel/tap"
# Only used to run brew style and brew audit against the working copy; never
# tapped for real, never pushed to.
VERIFY_TAP_NAME="colimastatus-release-check/tap"
RELEASE_WORKFLOW="Release artifacts"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN="yes" ;;
    --yes) ASSUME_YES="yes" ;;
    --skip-ci-check) SKIP_CI_CHECK="yes" ;;
    --skip-tap) SKIP_TAP="yes" ;;
    --allow-unattested) ALLOW_UNATTESTED="yes" ;;
    --tap)
      if [[ $# -lt 2 ]]; then
        echo "--tap needs a path." >&2
        exit 1
      fi
      TAP_DIR="$2"
      shift
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
  shift
done

if [[ -n "${CI:-}" ]]; then
  echo "This script signs and notarizes and must not run in CI." >&2
  exit 1
fi

# shellcheck source=version.sh
source "${SCRIPT_DIR}/version.sh"

RELEASE_TAG="v${APP_VERSION}"
ARCHIVE_NAME="ColimaStatus-${APP_VERSION}-macos-universal.zip"
ARCHIVE="${RELEASE_DIR}/${ARCHIVE_NAME}"
APP_DIR="${REPOSITORY_DIR}/dist/ColimaStatus.app"
CI_DIR="${REPOSITORY_DIR}/dist/ci"
UNSIGNED_NAME="ColimaStatus-${APP_VERSION}-unsigned.zip"
UNSIGNED_ARCHIVE="${CI_DIR}/${UNSIGNED_NAME}"
ARTIFACT_NAME="ColimaStatus-${RELEASE_TAG}-unsigned"
DIGEST_RECORD="${ARCHIVE}.unsigned-sha256"
RELEASE_RUN_ID=""
UNSIGNED_DIGEST=""
NOTES_FILE=""
VERIFY_TAP_LINK=""

step() { printf '\n== %s\n' "$1"; }
fail() {
  echo "$1" >&2
  exit 1
}

# brew style and brew audit only load casks from a tap inside the Homebrew
# prefix, so the working copy has to be reachable from there.
#
# The verification deliberately does not go through kevincfechtel/tap. That tap
# is usually installed as a real clone, and brew then reads the clone instead of
# the working copy the cask was just generated into: both checks pass without
# ever seeing the new cask, which makes this gate a placebo exactly when it
# matters. Untapping to work around it would tear down the maintainer's own
# installation and break their brew upgrade until they tap again.
#
# A throwaway tap name that nothing else uses avoids both problems: the checks
# read the generated file, and the installed tap is left untouched.
link_tap() {
  local taps_dir
  taps_dir="$(brew --repository)/Library/Taps"
  VERIFY_TAP_LINK="${taps_dir}/${VERIFY_TAP_NAME%%/*}/homebrew-${VERIFY_TAP_NAME##*/}"

  # ln -sfn into an existing directory does not fail, it drops the link inside
  # it, so whatever sits at the path is cleared first.
  unlink_tap

  mkdir -p -- "$(dirname -- "${VERIFY_TAP_LINK}")"
  ln -sfn "$(cd -- "${TAP_DIR}" && pwd)" "${VERIFY_TAP_LINK}"

  # The gate has to prove it is looking at the file that was just generated.
  cmp -s "${TAP_DIR}/Casks/colimastatus.rb" "${VERIFY_TAP_LINK}/Casks/colimastatus.rb" ||
    fail "The verification tap does not expose the generated cask: ${VERIFY_TAP_LINK}"
}

# unlink_tap only ever removes the throwaway path, never a real tap: a symlink
# is deleted, a directory is only removed if it is empty.
unlink_tap() {
  [[ -n "${VERIFY_TAP_LINK}" ]] || return 0

  if [[ -L "${VERIFY_TAP_LINK}" ]]; then
    rm -f -- "${VERIFY_TAP_LINK}"
  elif [[ -d "${VERIFY_TAP_LINK}" ]]; then
    rmdir -- "${VERIFY_TAP_LINK}" 2>/dev/null ||
      fail "Refusing to remove ${VERIFY_TAP_LINK}: it is a non-empty directory."
  fi
  rmdir -- "$(dirname -- "${VERIFY_TAP_LINK}")" 2>/dev/null || true
}

cleanup() {
  [[ -n "${NOTES_FILE}" ]] && rm -f -- "${NOTES_FILE}"
  unlink_tap
}
trap cleanup EXIT

# confirm gates everything that changes state outside this machine. A release is
# hard to take back: a pushed tag, a published release, and a pushed cask are all
# visible to users immediately.
confirm() {
  local prompt="$1"
  [[ "${ASSUME_YES}" == "yes" ]] && return 0
  if [[ ! -t 0 ]]; then
    fail "Cannot ask for confirmation without a terminal. Pass --yes to proceed."
  fi
  local answer=""
  read -r -p "${prompt} [y/N] " answer
  [[ "${answer}" == "y" || "${answer}" == "Y" ]]
}

# ---------------------------------------------------------------------------
step "1/10 Checking the working copy and the tools"

for command_name in git gh brew shasum; do
  command -v "${command_name}" >/dev/null 2>&1 ||
    fail "Required program is missing: ${command_name}"
done

gh auth status >/dev/null 2>&1 || fail "gh is not authenticated. Run: gh auth login"

REPOSITORY_SLUG="$(gh repo view --json nameWithOwner --jq '.nameWithOwner')"
[[ -n "${REPOSITORY_SLUG}" ]] || fail "Could not determine the GitHub repository."

cd "${REPOSITORY_DIR}"

# A dirty tree would be stamped into the binary as a -dirty commit and would
# ship sources nobody can reconstruct from the tag.
[[ -z "$(git status --porcelain)" ]] ||
  fail "The working copy has uncommitted changes. Commit or stash them first."

CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
[[ "${CURRENT_BRANCH}" == "main" ]] ||
  fail "Releases are cut from main, not from ${CURRENT_BRANCH}."

git fetch --quiet origin main
RELEASE_COMMIT="$(git rev-parse HEAD)"
[[ "${RELEASE_COMMIT}" == "$(git rev-parse origin/main)" ]] ||
  fail "main and origin/main differ. Push or pull first."

echo "Version:    ${APP_VERSION} (build ${APP_BUILD_NUMBER})"
echo "Commit:     ${RELEASE_COMMIT}"
echo "Deployment: macOS ${APP_DEPLOYMENT_TARGET}"

# ---------------------------------------------------------------------------
step "2/10 Checking the changelog"

# A release whose notes still sit under "Unreleased" ships without a readable
# description, and the compare links at the bottom of the file stay wrong.
grep -q "^## \[${APP_VERSION}\]" CHANGELOG.md ||
  fail "CHANGELOG.md has no '## [${APP_VERSION}]' section. Move the Unreleased entries into it."

# Entries still sitting under Unreleased would be missing from the release
# notes, which is what happens when VERSION was not bumped for them.
UNRELEASED_ENTRIES="$(
  awk '
    /^## \[Unreleased\]/ { collecting = 1; next }
    collecting && /^## \[/ { exit }
    collecting && /^- / { print }
  ' CHANGELOG.md | grep -c '' || true
)"
if [[ "${UNRELEASED_ENTRIES}" -gt 0 ]]; then
  fail "CHANGELOG.md lists ${UNRELEASED_ENTRIES} entries under [Unreleased]. Move them into '## [${APP_VERSION}]', or bump VERSION; otherwise they are missing from the release notes."
fi

NOTES_FILE="$(mktemp -t colimastatus-release-notes)"
awk -v version="## [${APP_VERSION}]" '
  index($0, version) == 1 { collecting = 1; next }
  collecting && /^## \[/ { exit }
  collecting { print }
' CHANGELOG.md >"${NOTES_FILE}"

[[ -s "${NOTES_FILE}" ]] || fail "The '## [${APP_VERSION}]' section in CHANGELOG.md is empty."
echo "Release notes: $(grep -c '' "${NOTES_FILE}") lines from the changelog"

# ---------------------------------------------------------------------------
step "3/10 Checking CI on the release commit"

if [[ "${SKIP_CI_CHECK}" == "yes" ]]; then
  echo "Skipped on request."
else
  CI_CONCLUSIONS="$(gh run list --commit "${RELEASE_COMMIT}" --workflow CI \
    --json conclusion --jq '.[].conclusion' 2>/dev/null || true)"
  if [[ -z "${CI_CONCLUSIONS}" ]]; then
    fail "No CI run found for ${RELEASE_COMMIT}. Wait for it, or pass --skip-ci-check."
  fi
  if grep -qvx 'success' <<<"${CI_CONCLUSIONS}"; then
    echo "${CI_CONCLUSIONS}" >&2
    fail "CI on the release commit is not green."
  fi
  echo "CI is green."
fi

# ---------------------------------------------------------------------------
step "4/10 Tagging ${RELEASE_TAG}"

EXISTING_TAG_COMMIT="$(git rev-list -n 1 "${RELEASE_TAG}" 2>/dev/null || true)"
if [[ -n "${EXISTING_TAG_COMMIT}" ]]; then
  [[ "${EXISTING_TAG_COMMIT}" == "${RELEASE_COMMIT}" ]] ||
    fail "${RELEASE_TAG} already exists on ${EXISTING_TAG_COMMIT}, not on HEAD. Bump VERSION."
  echo "${RELEASE_TAG} already points at HEAD."
elif [[ "${DRY_RUN}" == "yes" ]]; then
  echo "Would create and push ${RELEASE_TAG}."
else
  confirm "Create and push ${RELEASE_TAG}?" || fail "Stopped before tagging."
  git tag -a "${RELEASE_TAG}" -m "ColimaStatus ${APP_VERSION}"
  git push origin "${RELEASE_TAG}"
  echo "Pushed ${RELEASE_TAG}; .github/workflows/release.yml verifies the tagged tree."
fi

# ---------------------------------------------------------------------------
step "5/10 Waiting for the tag's workflow build"

if [[ "${DRY_RUN}" == "yes" ]]; then
  echo "Would wait for '${RELEASE_WORKFLOW}' on ${RELEASE_COMMIT}, then download"
  echo "and verify ${ARTIFACT_NAME}."
else
  # The run is created by the tag push and may not be listed immediately.
  WAITED=0
  while :; do
    RELEASE_RUN_ID="$(gh run list --workflow "${RELEASE_WORKFLOW}" \
      --commit "${RELEASE_COMMIT}" --limit 1 \
      --json databaseId --jq '.[0].databaseId' 2>/dev/null || true)"
    [[ -n "${RELEASE_RUN_ID}" ]] && break
    if (( WAITED >= 300 )); then
      fail "No '${RELEASE_WORKFLOW}' run appeared for ${RELEASE_COMMIT}."
    fi
    sleep 10
    WAITED=$((WAITED + 10))
  done

  echo "Run ${RELEASE_RUN_ID}; waiting for it to finish."
  gh run watch "${RELEASE_RUN_ID}" --exit-status >/dev/null ||
    fail "The release workflow failed. Fix the tagged commit before signing."
  echo "The workflow build passed."
fi

# ---------------------------------------------------------------------------
step "6/10 Downloading and verifying the attested build"

if [[ "${DRY_RUN}" == "yes" ]]; then
  echo "Would verify the attestation with:"
  echo "  gh attestation verify ${UNSIGNED_NAME} --repo ${REPOSITORY_SLUG}"
else
  rm -rf -- "${CI_DIR}"
  mkdir -p -- "${CI_DIR}"
  gh run download "${RELEASE_RUN_ID}" --name "${ARTIFACT_NAME}" --dir "${CI_DIR}" ||
    fail "Could not download ${ARTIFACT_NAME} from run ${RELEASE_RUN_ID}."
  [[ -f "${UNSIGNED_ARCHIVE}" ]] ||
    fail "The artifact does not contain ${UNSIGNED_NAME}."

  UNSIGNED_DIGEST="$(shasum -a 256 "${UNSIGNED_ARCHIVE}" | awk '{print $1}')"

  # This is the link that makes the release traceable: the attestation proves
  # these bytes were produced by this repository's workflow, and the run they
  # came from is the one for the tagged commit. Everything after this only
  # signs; it does not rebuild.
  if gh attestation verify "${UNSIGNED_ARCHIVE}" --repo "${REPOSITORY_SLUG}" >/dev/null 2>&1; then
    echo "Provenance attestation verified."
  elif [[ "${ALLOW_UNATTESTED}" == "yes" ]]; then
    echo "WARNING: no valid attestation; continuing because --allow-unattested was given." >&2
  else
    fail "No valid provenance attestation for ${UNSIGNED_NAME}. Pass --allow-unattested to override."
  fi

  echo "Unsigned SHA-256: ${UNSIGNED_DIGEST}"
  rm -rf -- "${APP_DIR}"
  ditto -x -k "${UNSIGNED_ARCHIVE}" "${REPOSITORY_DIR}/dist"
  [[ -d "${APP_DIR}" ]] || fail "The artifact did not extract to ${APP_DIR}."
fi

# ---------------------------------------------------------------------------
step "7/10 Signing, notarizing and stapling"

if [[ "${DRY_RUN}" == "yes" ]]; then
  echo "Would run ./Build/release.sh against the downloaded bundle, which signs"
  echo "and submits an external notarization request. Skipped: not a test target."
elif [[ -f "${ARCHIVE}" ]] &&
  [[ -f "${DIGEST_RECORD}" ]] &&
  [[ "$(cat "${DIGEST_RECORD}")" == "${UNSIGNED_DIGEST}" ]] &&
  ! confirm "${ARCHIVE_NAME} is already signed from this build. Do it again?"; then
  # Notarization is a round trip to Apple, so a resumed run should not spend it
  # again. The recorded digest is what makes skipping safe: an archive left over
  # from an earlier tag or an earlier workflow run does not match, and is
  # re-signed rather than published under this run's provenance.
  echo "Keeping the existing archive."
else
  rm -f -- "${DIGEST_RECORD}"
  COLIMASTATUS_PREBUILT_APP="${APP_DIR}" "${SCRIPT_DIR}/release.sh"
  printf '%s\n' "${UNSIGNED_DIGEST}" >"${DIGEST_RECORD}"
fi

# ---------------------------------------------------------------------------
step "8/10 Publishing the release and uploading the asset"

if [[ "${DRY_RUN}" == "yes" ]]; then
  echo "Would publish ${RELEASE_TAG} with ${ARCHIVE_NAME} as its only asset,"
  echo "with the provenance of the workflow build appended to the notes."
else
  [[ -f "${ARCHIVE}" ]] || fail "The release archive is missing: ${ARCHIVE}"

  # Signing rewrote the bundle, so the published asset is not the attested
  # object. Recording both digests is what lets a reader follow the chain:
  # GitHub attests the unsigned archive against the tag, Apple's stapled ticket
  # covers the shipped CDHash, and these two lines name both.
  #
  # The CDHash is read from the archive that is actually published, not from
  # dist/, which holds an unsigned bundle whenever the signing step was skipped
  # on a resumed run.
  #
  # awk must not exit early: that closes the pipe, codesign dies from SIGPIPE,
  # and pipefail plus errexit abort the script with no message at all.
  CDHASH_DIR="$(mktemp -d /tmp/colimastatus-cdhash.XXXXXX)"
  ditto -x -k "${ARCHIVE}" "${CDHASH_DIR}"
  CDHASH="$(codesign --display --verbose=4 "${CDHASH_DIR}/ColimaStatus.app" 2>&1 |
    awk -F= '/^CDHash=/ && !seen { print $2; seen = 1 }')"
  rm -rf -- "${CDHASH_DIR}"
  [[ -n "${CDHASH}" ]] || fail "Could not read the CDHash from ${ARCHIVE_NAME}."

  {
    echo
    echo "## Provenance"
    echo
    echo "Built by [workflow run ${RELEASE_RUN_ID}](https://github.com/${REPOSITORY_SLUG}/actions/runs/${RELEASE_RUN_ID})"
    echo "from \`${RELEASE_TAG}\` (\`${RELEASE_COMMIT}\`) and signed locally."
    echo
    echo "The attestation covers the **unsigned** build this release was signed"
    echo "from, not the asset below: signing and stapling change the bytes."
    echo
    echo "- Unsigned build SHA-256: \`${UNSIGNED_DIGEST}\`"
    echo "- Shipped bundle CDHash: \`${CDHASH}\`"
    echo
    echo "Verify the unsigned build after downloading it from the run:"
    echo
    echo '```sh'
    echo "gh attestation verify ${UNSIGNED_NAME} --repo ${REPOSITORY_SLUG}"
    echo '```'
  } >>"${NOTES_FILE}"

  if gh release view "${RELEASE_TAG}" >/dev/null 2>&1; then
    echo "Release ${RELEASE_TAG} already exists."
    # Collected into a variable first: grep -q would close the pipe early,
    # gh would fail on SIGPIPE, and pipefail would report the asset as absent.
    RELEASE_ASSETS="$(gh release view "${RELEASE_TAG}" --json assets --jq '.assets[].name')"
    if grep -qx "${ARCHIVE_NAME}" <<<"${RELEASE_ASSETS}"; then
      echo "Asset ${ARCHIVE_NAME} is already attached; the checksum is verified in step 9."
    else
      confirm "Upload ${ARCHIVE_NAME} to the existing release?" ||
        fail "Stopped before uploading."
      gh release upload "${RELEASE_TAG}" "${ARCHIVE}"
    fi
  else
    confirm "Publish release ${RELEASE_TAG}?" || fail "Stopped before publishing."
    # Exactly one universal asset: brew install --cask does not report an
    # architecture mismatch, it installs whatever the URL returns.
    gh release create "${RELEASE_TAG}" "${ARCHIVE}" \
      --title "${RELEASE_TAG}" \
      --notes-file "${NOTES_FILE}"
  fi
fi

# ---------------------------------------------------------------------------
step "9/10 Generating and verifying the cask"

if [[ "${SKIP_TAP}" == "yes" ]]; then
  echo "Skipped on request. The cask is not updated, so brew still serves the"
  echo "previous version."
elif [[ "${DRY_RUN}" == "yes" ]]; then
  [[ -d "${TAP_DIR}/.git" ]] || fail "No tap checkout at ${TAP_DIR}. Pass --tap PATH."
  echo "Would generate ${TAP_DIR}/Casks/colimastatus.rb with --verify-published,"
  echo "then run brew style and brew audit --online against ${VERIFY_TAP_NAME}."
else
  [[ -d "${TAP_DIR}/.git" ]] || fail "No tap checkout at ${TAP_DIR}. Pass --tap PATH."

  # --verify-published downloads the asset and refuses to emit a cask whose
  # sha256 does not describe the uploaded bytes.
  "${SCRIPT_DIR}/cask.sh" --verify-published "${TAP_DIR}/Casks/colimastatus.rb"

  link_tap
  brew style --cask "${VERIFY_TAP_NAME}"
  brew audit --cask --online --strict --tap="${VERIFY_TAP_NAME}"
  unlink_tap
  echo "brew style and brew audit --online passed."
fi

# ---------------------------------------------------------------------------
step "10/10 Pushing the cask"

if [[ "${SKIP_TAP}" == "yes" ]]; then
  echo "Skipped on request."
elif [[ "${DRY_RUN}" == "yes" ]]; then
  echo "Would commit Casks/colimastatus.rb as 'colimastatus ${APP_VERSION}' and push,"
  echo "which starts the tap's audit workflow."
else
  if [[ -z "$(git -C "${TAP_DIR}" status --porcelain Casks/colimastatus.rb)" ]]; then
    echo "The generated cask is identical to the committed one; nothing to push."
  else
    confirm "Commit and push the cask to ${TAP_NAME}?" || fail "Stopped before pushing the cask."
    git -C "${TAP_DIR}" add Casks/colimastatus.rb
    git -C "${TAP_DIR}" commit -m "colimastatus ${APP_VERSION}"
    git -C "${TAP_DIR}" push
    echo "Pushed; the tap's audit workflow re-runs brew audit --online."
  fi
fi

printf '\n'
if [[ "${DRY_RUN}" == "yes" ]]; then
  echo "Dry run complete. Nothing was signed, tagged, published, or pushed."
else
  echo "Release ${APP_VERSION} (build ${APP_BUILD_NUMBER}) done."
  echo "Verify the user path:"
  echo "  brew update && brew install --cask ${TAP_NAME}/colimastatus"
fi
