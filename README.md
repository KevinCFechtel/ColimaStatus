# ColimaStatus

ColimaStatus is a small macOS menu bar application for the local
[Colima](https://github.com/abiosoft/colima) installation. It is written in Go
and uses the standalone [`fyne.io/systray`](https://fyne.io/systray) module.

The official Colima llama appears as a monochrome silhouette in the menu bar.
It uses full system foreground brightness while Colima is running and appears
dimmed while Colima is stopped, unavailable, or changing state. As a macOS
template icon it automatically remains legible in light and dark appearances.

The menu shows the selected profile and its runtime, architecture, CPUs, and
memory. Colima can be started and stopped directly from there. A broken profile
is stopped with Colima's `--force` option, as recommended by the Colima FAQ.
The status is refreshed immediately after launch, after each action, manually,
and every minute.

## Requirements

- macOS 11 or later
- Go 1.25 or later
- Xcode Command Line Tools
- Colima

Colima can be installed with Homebrew:

```sh
brew install colima
```

## Build and run

```sh
./Build/build.sh
open dist/ColimaStatus.app
```

The build script creates an ad-hoc signed app bundle in `dist/`. The app is an
agent application (`LSUIElement`) and therefore does not appear in the Dock.
The app icon can be regenerated from `assets/AppIcon.png` with
`./Build/generate-icons.sh`.

For local development:

```sh
./Build/format.sh
./Build/test.sh
./Build/vet.sh
./Build/run.sh
```

## Create a release

The bundle identifier is `dev.kevincfechtel.ColimaStatus`.

A signed and notarized release requires an Apple Developer ID Application
certificate and a `notarytool` profile stored in the Keychain:

```sh
xcrun notarytool store-credentials macos-notary
cp Build/.env.example Build/.env
# Edit SIGNING_IDENTITY in Build/.env.
./Build/release.sh
```

`Build/.env` is ignored by Git. The release script builds the app with the
hardened runtime, signs it, submits it to Apple, staples the notarization
ticket, verifies it with Gatekeeper, and writes the final archive to
`dist/release/`.

## Configuration

The default Colima profile is monitored. The following environment variables
can be set before starting the executable:

| Variable | Purpose |
| --- | --- |
| `COLIMASTATUS_PROFILE` | Monitor a profile other than `default` |
| `COLIMASTATUS_COLIMA_PATH` | Use a Colima executable in a non-standard location |

For GUI launches, macOS supplies a reduced `PATH`. ColimaStatus therefore also
checks the common Homebrew and MacPorts paths automatically.
