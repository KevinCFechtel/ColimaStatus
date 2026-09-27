// Package autostart manages whether macOS launches ColimaStatus at login.
// The native controller bridges to Apple's Service Management API; the
// fallback on other platforms reports the feature as unsupported.
package autostart

// Status describes whether macOS can and may launch ColimaStatus at login.
type Status uint8

const (
	Unsupported Status = iota
	Disabled
	Enabled
	RequiresApproval
	NotFound
)

type Controller interface {
	Status() (Status, error)
	SetEnabled(enabled bool) (Status, error)
	OpenSettings() error
}
