# Changelog

All notable changes to ColimaStatus are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-09-27

First public release. ColimaStatus keeps the state of a local
[Colima](https://github.com/abiosoft/colima) profile visible in the macOS menu
bar and lets you start or stop it without opening a terminal. Everything runs
locally: no account, no telemetry, and no background service of its own.

The **Changed** and **Fixed** entries below refer to the pre-release builds that
were in daily use before this version; there is no earlier published release.

### Added

- Menu bar item showing whether the selected Colima profile is running,
  stopped, missing or broken, with its runtime, architecture, CPU count, memory
  and disk size. A broken profile is stopped with `--force`.
- Start, stop and immediate refresh from the menu. The monochrome Colima llama
  is used as a macOS template icon, bright while Colima runs and dimmed
  otherwise, next to an adaptive app icon for the light, dark and tinted
  appearances.
- Lima lifecycle events through `limactl watch --json`, debounced, with bounded
  exponential retry. A 15-minute safety check remains as the fallback for Lima
  versions without the command, and the tooltip on the last-check row says
  which of the two is currently providing updates.
- The status is re-read when the Mac wakes from sleep. Go's timers read
  `mach_absolute_time`, which is suspended along with the system, so a closed
  laptop never ran the check it was scheduled for. Waking also releases the
  event watcher from its retry backoff, since the stream most likely died with
  the machine.
- Optional settings file at
  `~/Library/Application Support/ColimaStatus/config.json` for the profile, the
  Colima executable path, the language and the safety interval. Out-of-range
  values are clamped, and a missing, unreadable or future-version file falls
  back to the defaults with the reason recorded in the log.
- File log at `~/Library/Logs/ColimaStatus/colimastatus.log`, rotated at 1 MiB
  with one generation kept. It records which Colima executable was found and
  which profile is monitored. Both the log and the settings file can be
  revealed from the menu.
- Colima and `limactl` are discovered from `PATH`, Homebrew, MacPorts,
  `~/.local/bin` and Nix locations, because an app started from Finder or as a
  login item receives a minimal `PATH`. Every Colima child process is given a
  `PATH` containing both directories, since Colima invokes `limactl`
  internally.
- Native launch at login through `SMAppService`, with a direct link to the
  Login Items panel when macOS requires approval.
- English and German localization following the macOS language. Date and time
  layouts live in the message catalogs, so a further language needs no Go
  change.
- `--help` describing the flags, the file locations and the environment
  variables, and `--version`.
- Universal binaries covering Apple Silicon and Intel in one download, signed,
  notarized and stapled, distributed as a Homebrew cask:
  `brew install --cask kevincfechtel/tap/colimastatus`.
- Continuous integration on macOS running formatting, `go vet`, localization
  catalog checks, race-enabled tests, a universal build with bundle
  verification, `golangci-lint`, `govulncheck` and `shellcheck`, plus
  Dependabot updates. Tagged commits are built by a workflow that attests the
  build provenance of the unsigned archive, which the local signing step
  verifies before it signs.

### Changed

- **The minimum supported macOS version is 13.0.** An earlier draft declared
  macOS 11, which could not be met: the Go toolchain used to build ColimaStatus
  requires macOS 13, so the binary would not have run reliably on macOS 11 or
  12 despite the bundle advertising support. The toolchain is now pinned in
  `go.mod`, `APP_DEPLOYMENT_TARGET` in `Build/version.sh` is the single source
  of truth for the floor, and `Build/build.sh` verifies with `vtool` that every
  architecture slice was actually linked against it. A toolchain upgrade that
  raises the real floor now fails the build instead of shipping a bundle that
  promises more than it supports.
- Settings moved from environment variables into a file. The variables never
  reached the app in its normal form: macOS starts it from Finder or as a login
  item, neither of which inherits a shell environment, so the profile could not
  actually be changed for the intended use.
  `COLIMASTATUS_PROFILE`, `COLIMASTATUS_COLIMA_PATH` and
  `COLIMASTATUS_LANGUAGE` still override the file when set.
- Colima failures are shown in the selected language. Previously the raw
  English error text appeared underneath an otherwise localized menu. Failures
  are now classified in the domain and mapped to localized messages by the
  menu, while the technical cause goes to the log.
- The status parser tolerates output it does not fully understand: a number may
  arrive as a string, a single unreadable entry is skipped, and only output
  that yields nothing at all is an error. A status value this version does not
  know keeps both actions available instead of leaving a menu with no way to
  act, and is shown verbatim.
- Profiles are selected with Colima's documented global `-p` flag rather than
  positionally.
- The menu bar icons are generated at build time and embedded. They were
  previously computed on every start, with three panic paths that ran before
  logging existed.
- Ad-hoc signing no longer passes `--deep`, which Apple discourages for signing
  and which this bundle does not need.
- `Info.plist` declares English as the development region, lists the available
  localizations so a per-app language can be chosen in System Settings, and
  carries a copyright and an application category.

### Fixed

- A click arriving while an operation is running is queued instead of dropped.
  Pressing refresh during a long start previously did nothing.
- Long text is truncated on rune boundaries, so a menu row or tooltip can no
  longer end in a broken multi-byte character.
- Every menu click channel is checked for closure. When the menu went away, the
  cases that ignored it could spin.
- Background tasks register with the wait group individually instead of relying
  on a hard-coded count, which had to be kept in sync by hand and would have
  hung the app on quit or panicked on the next added task.
- The launch-at-login state is polled every 30 seconds instead of every 5. Each
  poll crosses into Objective-C for a value that only changes when the user
  changes it in System Settings.
- A message that cannot be rendered falls back to its English source text and
  is logged, rather than panicking inside a menu callback and taking the app
  down over a catalog problem.

[Unreleased]: https://github.com/KevinCFechtel/ColimaStatus/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/KevinCFechtel/ColimaStatus/releases/tag/v1.0.0
