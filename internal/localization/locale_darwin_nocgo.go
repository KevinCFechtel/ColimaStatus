//go:build darwin && !cgo

package localization

import "os"

func platformPreferredLanguages() []string {
	return environmentPreferredLanguages(os.Getenv)
}
