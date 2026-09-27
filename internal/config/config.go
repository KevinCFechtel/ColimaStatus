// Package config owns ColimaStatus's user-editable settings. The file is
// optional: a missing, unreadable, or unknown-version file always yields
// working defaults so that a bad or future config can never keep the app from
// starting.
//
// The file exists because environment variables do not reach a menu bar app.
// macOS starts it from Finder or as a login item, neither of which inherits a
// shell environment, so COLIMASTATUS_* can only be set by a developer running
// the binary by hand. Those variables still win when they are present.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Version is the schema version written to disk. Bump it only for a change that
// an older ColimaStatus cannot interpret safely; additive fields do not need it.
const Version = 1

// Defaults. CheckInterval is deliberately long: it is only the safety net for
// the case where Lima lifecycle events are unavailable, not the primary way
// status is discovered.
const (
	DefaultProfile       = "default"
	DefaultCheckInterval = 15 * time.Minute
)

// Accepted range for the safety interval. Values outside it are clamped rather
// than rejected, so a typo degrades the setting instead of the app.
const (
	MinimumCheckInterval = time.Minute
	MaximumCheckInterval = 24 * time.Hour
)

// Config holds the effective settings for one ColimaStatus process.
type Config struct {
	// Profile is the Colima profile being monitored.
	Profile string
	// ColimaPath overrides executable autodetection when not empty.
	ColimaPath string
	// Language overrides the detected macOS language when not empty.
	Language string
	// CheckInterval is the wall-clock time between safety-net status checks.
	CheckInterval time.Duration
}

// file mirrors the on-disk schema. Every setting is a pointer so that an absent
// field falls back to its default instead of to the Go zero value.
type file struct {
	Version              *int    `json:"version"`
	Profile              *string `json:"profile"`
	ColimaPath           *string `json:"colimaPath"`
	Language             *string `json:"language"`
	CheckIntervalMinutes *int    `json:"checkIntervalMinutes"`
}

// Default returns the settings used when no config file exists.
func Default() Config {
	return Config{
		Profile:       DefaultProfile,
		CheckInterval: DefaultCheckInterval,
	}
}

// DefaultPath returns the canonical config location.
func DefaultPath() (string, error) {
	configurationDirectory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user configuration directory could not be determined: %w", err)
	}
	return filepath.Join(configurationDirectory, "ColimaStatus", "config.json"), nil
}

// DescribePath returns the config location for help output, falling back to the
// canonical path when the configuration directory is unknown.
func DescribePath() string {
	path, err := DefaultPath()
	if err != nil {
		return filepath.Join("~", "Library", "Application Support", "ColimaStatus", "config.json")
	}
	return path
}

// Load reads the config file at path. It never fails on content: anything that
// cannot be interpreted falls back to the default for that setting and is
// reported through notes, which the caller is expected to log. A missing file
// is written back with the defaults so that the settings are discoverable.
func Load(path string) (Config, []string) {
	configuration := Default()

	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if writeErr := Save(path, configuration); writeErr != nil {
			return configuration, []string{fmt.Sprintf(
				"default configuration could not be written to %s: %v", path, writeErr)}
		}
		return configuration, nil
	}
	if err != nil {
		return configuration, []string{fmt.Sprintf(
			"configuration could not be read, using defaults: %v", err)}
	}

	var stored file
	if err := json.Unmarshal(contents, &stored); err != nil {
		return configuration, []string{fmt.Sprintf(
			"configuration is not valid JSON, using defaults: %v", err)}
	}

	// An unknown version is treated as a cache miss, not an error. This keeps a
	// downgrade working after a newer ColimaStatus has written the file.
	if stored.Version != nil && *stored.Version != Version {
		return configuration, []string{fmt.Sprintf(
			"configuration version %d is not supported, using defaults", *stored.Version)}
	}

	var notes []string
	if stored.Profile != nil && *stored.Profile != "" {
		configuration.Profile = *stored.Profile
	}
	if stored.ColimaPath != nil {
		configuration.ColimaPath = *stored.ColimaPath
	}
	if stored.Language != nil {
		configuration.Language = *stored.Language
	}
	if stored.CheckIntervalMinutes != nil {
		interval := time.Duration(*stored.CheckIntervalMinutes) * time.Minute
		clamped, note := clampDuration(
			"checkIntervalMinutes", interval, MinimumCheckInterval, MaximumCheckInterval)
		configuration.CheckInterval = clamped
		notes = appendNote(notes, note)
	}
	return configuration, notes
}

// ApplyEnvironment lets the documented COLIMASTATUS_* variables override the
// stored settings. They keep working for a developer running the binary from a
// shell, where they are the quickest way to try a different profile.
func ApplyEnvironment(configuration Config, getenv func(string) string) Config {
	if profile := getenv("COLIMASTATUS_PROFILE"); profile != "" {
		configuration.Profile = profile
	}
	if path := getenv("COLIMASTATUS_COLIMA_PATH"); path != "" {
		configuration.ColimaPath = path
	}
	if language := getenv("COLIMASTATUS_LANGUAGE"); language != "" {
		configuration.Language = language
	}
	return configuration
}

// Save writes the configuration atomically so that a crash or a full disk can
// never leave a half-written file behind.
func Save(path string, configuration Config) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("configuration directory could not be created: %w", err)
	}

	version := Version
	profile := configuration.Profile
	colimaPath := configuration.ColimaPath
	language := configuration.Language
	minutes := int(configuration.CheckInterval / time.Minute)
	contents, err := json.MarshalIndent(file{
		Version:              &version,
		Profile:              &profile,
		ColimaPath:           &colimaPath,
		Language:             &language,
		CheckIntervalMinutes: &minutes,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("configuration could not be encoded: %w", err)
	}
	contents = append(contents, '\n')

	temporary, err := os.CreateTemp(directory, ".config-*")
	if err != nil {
		return fmt.Errorf("temporary configuration could not be created: %w", err)
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := writeAndSync(temporary, contents); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("configuration could not be saved: %w", err)
	}
	removeTemporary = false
	return nil
}

func writeAndSync(target *os.File, contents []byte) error {
	if _, err := target.Write(contents); err != nil {
		_ = target.Close()
		return fmt.Errorf("configuration could not be written: %w", err)
	}
	if err := target.Sync(); err != nil {
		_ = target.Close()
		return fmt.Errorf("configuration could not be synchronized: %w", err)
	}
	if err := target.Close(); err != nil {
		return fmt.Errorf("configuration could not be closed: %w", err)
	}
	return nil
}

func clampDuration(name string, value, minimum, maximum time.Duration) (time.Duration, string) {
	switch {
	case value < minimum:
		return minimum, fmt.Sprintf("%s was raised to the minimum of %s", name, minimum)
	case value > maximum:
		return maximum, fmt.Sprintf("%s was lowered to the maximum of %s", name, maximum)
	default:
		return value, ""
	}
}

func appendNote(notes []string, note string) []string {
	if note == "" {
		return notes
	}
	return append(notes, note)
}
