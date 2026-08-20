//go:build !darwin && !windows

package localization

import "os"

func platformPreferredLanguages() []string {
	return environmentPreferredLanguages(os.Getenv)
}
