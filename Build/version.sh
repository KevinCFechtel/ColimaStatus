#!/usr/bin/env bash
set -euo pipefail

VERSION_SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
VERSION_REPOSITORY_DIR="$(cd -- "${VERSION_SCRIPT_DIR}/.." && pwd)"
VERSION_FILE="${VERSION_REPOSITORY_DIR}/VERSION"
BUILD_NUMBER_FILE="${VERSION_REPOSITORY_DIR}/BUILD_NUMBER"

read_version_value() {
  local file_path="$1"
  local value=""

  if [[ ! -f "${file_path}" ]]; then
    echo "Versionsdatei fehlt: ${file_path}" >&2
    return 1
  fi
  IFS= read -r value < "${file_path}" || true
  if [[ -z "${value}" ]]; then
    echo "Versionsdatei ist leer: ${file_path}" >&2
    return 1
  fi
  printf '%s' "${value}"
}

APP_VERSION="$(read_version_value "${VERSION_FILE}")"
APP_BUILD_NUMBER="$(read_version_value "${BUILD_NUMBER_FILE}")"

if [[ ! "${APP_VERSION}" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "Ungültige App-Version in VERSION: ${APP_VERSION} (erwartet: MAJOR.MINOR.PATCH)" >&2
  return 1 2>/dev/null || exit 1
fi

if [[ ! "${APP_BUILD_NUMBER}" =~ ^[1-9][0-9]*$ ]]; then
  echo "Ungültige Build-Nummer in BUILD_NUMBER: ${APP_BUILD_NUMBER} (erwartet: positive Ganzzahl)" >&2
  return 1 2>/dev/null || exit 1
fi

APP_COMMIT="unknown"
if command -v git >/dev/null 2>&1 && git -C "${VERSION_REPOSITORY_DIR}" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  APP_COMMIT="$(git -C "${VERSION_REPOSITORY_DIR}" rev-parse --short=12 HEAD)"
  if [[ -n "$(git -C "${VERSION_REPOSITORY_DIR}" status --porcelain --untracked-files=normal)" ]]; then
    APP_COMMIT="${APP_COMMIT}-dirty"
  fi
fi

export APP_VERSION APP_BUILD_NUMBER APP_COMMIT

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  printf 'Version: %s\nBuild: %s\nCommit: %s\n' \
    "${APP_VERSION}" \
    "${APP_BUILD_NUMBER}" \
    "${APP_COMMIT}"
fi
