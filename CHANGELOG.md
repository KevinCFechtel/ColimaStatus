# Changelog

All notable changes to ColimaStatus are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-09-27

### Added

- First release: a macOS menu bar app that shows whether the selected Colima
  profile is running, stopped, missing or broken, displays its runtime,
  architecture, CPU count and memory, and starts or stops Colima without a
  terminal. A broken profile is stopped with `--force`.
- Lima lifecycle events through `limactl watch --json`, debounced, with bounded
  exponential retry. A 15-minute safety check remains as the fallback for Lima
  versions without the command. The last-check tooltip says which of the two is
  currently providing updates.
- Optional configuration file at
  `~/Library/Application Support/ColimaStatus/config.json` for the profile, the
  Colima executable path, the language and the safety interval. Out-of-range
  values are clamped, and a missing, unreadable or future-version file falls
  back to the defaults with the reason recorded in the log. The file exists
  because environment variables do not reach an app launched from Finder or as
  a login item; `COLIMASTATUS_PROFILE`, `COLIMASTATUS_COLIMA_PATH` and
  `COLIMASTATUS_LANGUAGE` still override it.
- File log at `~/Library/Logs/ColimaStatus/colimastatus.log`, rotated at 1 MiB
  with one generation kept. It records which Colima executable was found and
  which profile is monitored. Previously the log went to stderr only, which is
  discarded for a menu bar app. Both the log and the settings file can be
  revealed from the menu.
- Colima and `limactl` are discovered from `PATH`, Homebrew, MacPorts,
  `~/.local/bin` and Nix locations, because a GUI or login-item launch receives
  a minimal `PATH`. Every Colima child process is given a `PATH` containing
  both directories, since Colima invokes `limactl` internally.
- Native launch at login through `SMAppService`, with a direct link to the
  Login Items panel when macOS requires approval.
- English and German localization following the macOS language, with the
  monochrome Colima llama as a template menu bar icon and an adaptive app icon
  for the light, dark and tinted appearances.
- Universal release binaries covering Apple Silicon and Intel in one download,
  signed, notarized and stapled, distributed as a Homebrew cask generated from
  the published artifact.
- Continuous integration on macOS running formatting, `go vet`, localization
  catalog checks, race-enabled tests, a universal build with bundle
  verification, `golangci-lint` and `govulncheck`, plus Dependabot updates for
  Go modules and GitHub Actions. Tagged commits are built by a workflow that
  attests the build provenance of the unsigned archive.

### Notes on the supported macOS version

The bundle targets **macOS 13 or later**. An earlier draft declared macOS 11,
which could not be met: the Go toolchain used to build ColimaStatus requires
macOS 13, so the binary would not have run reliably on macOS 11 or 12 despite
the bundle advertising support. The toolchain is now pinned in `go.mod`,
`APP_DEPLOYMENT_TARGET` in `Build/version.sh` is the single source of truth for
the floor, and `Build/build.sh` verifies with `vtool` that every architecture
slice was actually linked against it. A toolchain upgrade that raises the real
floor now fails the build instead of shipping a bundle that promises more than
it supports.

[Unreleased]: https://github.com/KevinCFechtel/ColimaStatus/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/KevinCFechtel/ColimaStatus/releases/tag/v0.1.0
