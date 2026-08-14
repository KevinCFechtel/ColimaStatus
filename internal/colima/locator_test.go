package colima

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocatePrefersConfiguredPath(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "custom-colima")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := Locate(path)
	if err != nil {
		t.Fatalf("Locate() error = %v", err)
	}
	if got != path {
		t.Fatalf("Locate() = %q, want %q", got, path)
	}
}

func TestLocateRejectsInvalidConfiguredPath(t *testing.T) {
	t.Parallel()

	if _, err := Locate(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("Locate() error = nil, want invalid path error")
	}
}
