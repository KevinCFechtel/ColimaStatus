#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
LOCALE_DIR="${REPOSITORY_DIR}/internal/localization/locales"
CHECK_DIR="$(mktemp -d /tmp/colimastatus-localization.XXXXXX)"
EXTRACT_DIR="${CHECK_DIR}/extract"
MERGE_DIR="${CHECK_DIR}/merge"

cleanup() {
  rm -rf -- "${CHECK_DIR}"
}
trap cleanup EXIT

mkdir -p "${EXTRACT_DIR}" "${MERGE_DIR}"
cd "${REPOSITORY_DIR}"

go tool goi18n extract \
  -sourceLanguage en \
  -format json \
  -outdir "${EXTRACT_DIR}" \
  internal/localization

if ! cmp -s "${LOCALE_DIR}/active.en.json" "${EXTRACT_DIR}/active.en.json"; then
  echo "The English localization catalog is out of date." >&2
  diff -u "${LOCALE_DIR}/active.en.json" "${EXTRACT_DIR}/active.en.json" || true
  echo "Regenerate and translate the catalogs before continuing." >&2
  exit 1
fi

go tool goi18n merge \
  -sourceLanguage en \
  -format json \
  -outdir "${MERGE_DIR}" \
  "${LOCALE_DIR}/active.en.json" \
  "${LOCALE_DIR}/active.de.json"

if [[ -f "${MERGE_DIR}/translate.de.json" ]]; then
  echo "The German localization catalog contains missing or outdated translations." >&2
  exit 1
fi

for language_code in en de; do
  committed="${LOCALE_DIR}/active.${language_code}.json"
  generated="${MERGE_DIR}/active.${language_code}.json"
  if ! cmp -s "${committed}" "${generated}"; then
    echo "The ${language_code} localization catalog is not normalized." >&2
    diff -u "${committed}" "${generated}" || true
    exit 1
  fi
done

echo "Localization catalogs are complete and up to date."
