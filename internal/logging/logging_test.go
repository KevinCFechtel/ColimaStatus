package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathIsUnderTheUserLogDirectory(t *testing.T) {
	t.Parallel()

	path, err := Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join("Library", "Logs", directory, logFileName)) {
		t.Fatalf("Path() = %q, want a path under ~/Library/Logs/%s", path, directory)
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("Path() = %q, want an absolute path", path)
	}
}

func TestDescribePathAlwaysNamesALocation(t *testing.T) {
	t.Parallel()

	if got := DescribePath(); !strings.Contains(got, logFileName) {
		t.Fatalf("DescribePath() = %q, want it to name the log file", got)
	}
}

func TestRotateKeepsOneGeneration(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), logFileName)
	if err := os.WriteFile(path, make([]byte, maximumSize+1), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	rotate(path)

	if _, err := os.Stat(path); err == nil {
		t.Fatal("the oversized log is still in place, want it rotated aside")
	}
	information, err := os.Stat(path + ".1")
	if err != nil {
		t.Fatalf("the rotated generation is missing: %v", err)
	}
	if information.Size() != maximumSize+1 {
		t.Fatalf("rotated size = %d, want the original %d", information.Size(), maximumSize+1)
	}
}

func TestRotateLeavesASmallLogAlone(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), logFileName)
	if err := os.WriteFile(path, []byte("still small"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	rotate(path)

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the log was rotated although it is below the limit: %v", err)
	}
	if _, err := os.Stat(path + ".1"); err == nil {
		t.Fatal("a rotated generation was created for a small log")
	}
}

// A missing log must not make rotation fail: the first start has no file yet.
func TestRotateToleratesAMissingLog(t *testing.T) {
	t.Parallel()

	rotate(filepath.Join(t.TempDir(), "absent.log"))
}
