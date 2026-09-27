package colima

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// defaultDirectories are searched in order when PATH does not contain the
// executable, which is the normal case for a menu bar app: Finder and login
// items start it with a minimal PATH. Homebrew and MacPorts come first because
// they are how most installations arrive; the Nix and per-user locations exist
// because those installations never appear in a GUI PATH either.
var defaultDirectories = []string{
	"/opt/homebrew/bin",
	"/usr/local/bin",
	"/opt/local/bin",
	"~/.local/bin",
	"~/.nix-profile/bin",
	"/run/current-system/sw/bin",
	"/nix/var/nix/profiles/default/bin",
}

// candidatePaths expands the search directories for one executable name.
func candidatePaths(name string) []string {
	home, homeErr := os.UserHomeDir()
	paths := make([]string, 0, len(defaultDirectories))
	for _, directory := range defaultDirectories {
		if strings.HasPrefix(directory, "~/") {
			if homeErr != nil {
				continue
			}
			directory = filepath.Join(home, directory[2:])
		}
		paths = append(paths, filepath.Join(directory, name))
	}
	return paths
}

// Locate finds Colima even when a macOS app receives a reduced PATH.
func Locate(configuredPath string) (string, error) {
	if configuredPath != "" {
		path, err := validateExecutable(configuredPath)
		if err != nil {
			return "", fmt.Errorf("configured Colima path is invalid: %w", err)
		}
		return path, nil
	}

	if path, err := exec.LookPath("colima"); err == nil {
		if validated, validateErr := validateExecutable(path); validateErr == nil {
			return validated, nil
		}
	}

	for _, path := range candidatePaths("colima") {
		if validated, err := validateExecutable(path); err == nil {
			return validated, nil
		}
	}

	return "", errors.New("not found in PATH or at a standard location")
}

func validateExecutable(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absolutePath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", errors.New("path names a directory")
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "", errors.New("file is not executable")
	}
	return absolutePath, nil
}
