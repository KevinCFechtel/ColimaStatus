// Package logging directs ColimaStatus's log to a file the user can find.
//
// ColimaStatus is a menu bar app that is normally started from Finder or as a
// login item. In both cases nothing is attached to stderr, so log output is
// lost exactly when it is needed: a Colima executable that cannot be located
// under the restricted PATH of a login item is reported only through the log.
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

const (
	// maximumSize is when the current log is rotated. The log is diagnostic
	// only, so one generation of history is enough.
	maximumSize = 1 << 20 // 1 MiB
	logFileName = "colimastatus.log"
	directory   = "ColimaStatus"
)

// Setup points the standard logger at the log file and returns a function that
// closes it. It never fails: if the file cannot be opened, logging falls back
// to stderr alone, because losing the log must never keep the app from running.
func Setup() (closeLog func()) {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)

	path, err := Path()
	if err != nil {
		log.Printf("log file location could not be determined, logging to stderr only: %v", err)
		return func() {}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		log.Printf("log directory could not be created, logging to stderr only: %v", err)
		return func() {}
	}
	rotate(path)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		log.Printf("log file could not be opened, logging to stderr only: %v", err)
		return func() {}
	}

	// Keep stderr as well, so that running from a terminal during development
	// still shows output without having to tail the file.
	log.SetOutput(io.MultiWriter(os.Stderr, file))
	return func() {
		log.SetOutput(os.Stderr)
		_ = file.Close()
	}
}

// Path returns the log file location.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory could not be determined: %w", err)
	}
	return filepath.Join(home, "Library", "Logs", directory, logFileName), nil
}

// DescribePath returns the log location for help output, falling back to the
// canonical path when the home directory is unknown.
func DescribePath() string {
	path, err := Path()
	if err != nil {
		return filepath.Join("~", "Library", "Logs", directory, logFileName)
	}
	return path
}

// rotate moves an oversized log aside, keeping exactly one previous generation.
func rotate(path string) {
	information, err := os.Stat(path)
	if err != nil || information.Size() < maximumSize {
		return
	}
	// A failed rotation is not worth reporting: the log is simply appended to
	// and will be rotated again on the next start.
	_ = os.Rename(path, path+".1")
}
