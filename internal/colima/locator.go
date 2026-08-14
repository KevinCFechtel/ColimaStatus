package colima

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var defaultPaths = []string{
	"/opt/homebrew/bin/colima",
	"/usr/local/bin/colima",
	"/opt/local/bin/colima",
}

// Locate finds Colima even when a macOS app receives a reduced PATH.
func Locate(configuredPath string) (string, error) {
	if configuredPath != "" {
		path, err := validateExecutable(configuredPath)
		if err != nil {
			return "", fmt.Errorf("konfigurierter Colima-Pfad ist ungültig: %w", err)
		}
		return path, nil
	}

	if path, err := exec.LookPath("colima"); err == nil {
		if validated, validateErr := validateExecutable(path); validateErr == nil {
			return validated, nil
		}
	}

	for _, path := range defaultPaths {
		if validated, err := validateExecutable(path); err == nil {
			return validated, nil
		}
	}

	return "", errors.New("weder im PATH noch an einem Standardpfad gefunden")
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
		return "", errors.New("Pfad bezeichnet ein Verzeichnis")
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "", errors.New("Datei ist nicht ausführbar")
	}
	return absolutePath, nil
}
