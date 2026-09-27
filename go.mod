module github.com/KevinCFechtel/ColimaStatus

// The Go toolchain determines the macOS floor of every produced binary:
// Go 1.27 requires macOS 13 or later. APP_DEPLOYMENT_TARGET in
// Build/version.sh must match; Build/build.sh verifies the linked binary
// against it and fails the build if they drift apart.
go 1.27

toolchain go1.27.1

tool github.com/nicksnyder/go-i18n/v2/goi18n

require (
	fyne.io/systray v1.12.2
	github.com/nicksnyder/go-i18n/v2 v2.6.1
	golang.org/x/sys v0.15.0
	golang.org/x/text v0.32.0
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
)
