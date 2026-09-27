package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingFileWritesDefaults(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "config.json")
	configuration, notes := Load(path)
	if len(notes) != 0 {
		t.Fatalf("Load() notes = %#v, want none", notes)
	}
	if configuration != Default() {
		t.Fatalf("Load() = %#v, want %#v", configuration, Default())
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the defaults were not written to disk: %v", err)
	}

	// The written file has to load back unchanged, otherwise the settings the
	// user discovers on disk are not the ones in effect.
	reloaded, notes := Load(path)
	if len(notes) != 0 || reloaded != configuration {
		t.Fatalf("reload = %#v (notes %#v), want %#v", reloaded, notes, configuration)
	}
}

func TestLoadAppliesStoredValues(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, `{
		"version": 1,
		"profile": "work",
		"colimaPath": "/opt/colima",
		"language": "de",
		"checkIntervalMinutes": 42
	}`)

	configuration, notes := Load(path)
	if len(notes) != 0 {
		t.Fatalf("Load() notes = %#v, want none", notes)
	}
	want := Config{
		Profile:       "work",
		ColimaPath:    "/opt/colima",
		Language:      "de",
		CheckInterval: 42 * time.Minute,
	}
	if configuration != want {
		t.Fatalf("Load() = %#v, want %#v", configuration, want)
	}
}

func TestLoadFallsBackWithoutFailing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		contents string
	}{
		{name: "invalid JSON", contents: `{"profile":`},
		{name: "unknown version", contents: `{"version": 99, "profile": "work"}`},
		{name: "wrong field type", contents: `{"version": 1, "profile": 7}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			configuration, notes := Load(writeConfig(t, test.contents))
			if configuration != Default() {
				t.Fatalf("Load() = %#v, want the defaults", configuration)
			}
			if len(notes) == 0 {
				t.Fatal("Load() notes = none, want an explanation for the fallback")
			}
		})
	}
}

func TestLoadClampsTheCheckInterval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		minutes int
		want    time.Duration
	}{
		{name: "below the minimum", minutes: 0, want: MinimumCheckInterval},
		{name: "negative", minutes: -5, want: MinimumCheckInterval},
		{name: "above the maximum", minutes: 100000, want: MaximumCheckInterval},
		{name: "inside the range", minutes: 30, want: 30 * time.Minute},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			contents, err := json.Marshal(map[string]any{"version": Version, "checkIntervalMinutes": test.minutes})
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			configuration, notes := Load(writeConfig(t, string(contents)))
			if configuration.CheckInterval != test.want {
				t.Fatalf("CheckInterval = %s, want %s", configuration.CheckInterval, test.want)
			}
			clamped := test.want != time.Duration(test.minutes)*time.Minute
			if clamped != (len(notes) == 1) {
				t.Fatalf("notes = %#v, want exactly one note only when clamping", notes)
			}
		})
	}
}

func TestApplyEnvironmentOverridesStoredValues(t *testing.T) {
	t.Parallel()

	environment := map[string]string{
		"COLIMASTATUS_PROFILE":     "from-env",
		"COLIMASTATUS_COLIMA_PATH": "/env/colima",
		"COLIMASTATUS_LANGUAGE":    "en",
	}
	stored := Config{Profile: "stored", ColimaPath: "/stored", Language: "de", CheckInterval: time.Hour}

	got := ApplyEnvironment(stored, func(name string) string { return environment[name] })
	want := Config{Profile: "from-env", ColimaPath: "/env/colima", Language: "en", CheckInterval: time.Hour}
	if got != want {
		t.Fatalf("ApplyEnvironment() = %#v, want %#v", got, want)
	}

	// An unset variable must not blank a stored value.
	unchanged := ApplyEnvironment(stored, func(string) string { return "" })
	if unchanged != stored {
		t.Fatalf("ApplyEnvironment() = %#v, want the stored settings unchanged", unchanged)
	}
}

func TestSaveIsAtomicAndLeavesNoTemporaryFiles(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "config.json")
	configuration := Config{Profile: "work", ColimaPath: "/opt/colima", CheckInterval: time.Hour}
	if err := Save(path, configuration); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, notes := Load(path)
	if len(notes) != 0 || loaded != configuration {
		t.Fatalf("Load() = %#v (notes %#v), want %#v", loaded, notes, configuration)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "config.json" {
		t.Fatalf("directory contains %d entries, want only config.json", len(entries))
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
