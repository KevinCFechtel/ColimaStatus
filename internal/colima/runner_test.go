package colima

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecRunnerAddsExecutableAndToolDirectoriesToPath(t *testing.T) {
	colimaDirectory := t.TempDir()
	limaDirectory := t.TempDir()
	colimaPath := writeTestExecutable(t, colimaDirectory, "colima", "#!/bin/sh\nexec limactl\n")
	writeTestExecutable(t, limaDirectory, "limactl", "#!/bin/sh\nprintf 'found limactl'\n")
	t.Setenv("PATH", "/usr/bin:/bin")

	runner := ExecRunner{lookupPaths: []string{limaDirectory}}
	output, err := runner.Run(context.Background(), colimaPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if output.Stdout != "found limactl" {
		t.Fatalf("Run() stdout = %q", output.Stdout)
	}
}

func TestEnvironmentWithPathReplacesAndDeduplicatesPath(t *testing.T) {
	t.Parallel()

	environment := environmentWithPath(
		[]string{"HOME=/tmp/home", "PATH=/usr/bin:/opt/homebrew/bin", "LANG=de_DE.UTF-8"},
		"/opt/homebrew/bin",
		"/custom/bin",
	)

	var paths []string
	for _, variable := range environment {
		if value, found := strings.CutPrefix(variable, "PATH="); found {
			paths = append(paths, value)
		}
	}
	if len(paths) != 1 {
		t.Fatalf("PATH entries = %v, want exactly one", paths)
	}
	want := strings.Join([]string{"/opt/homebrew/bin", "/custom/bin", "/usr/bin"}, string(os.PathListSeparator))
	if paths[0] != want {
		t.Fatalf("PATH = %q, want %q", paths[0], want)
	}
}

func writeTestExecutable(t *testing.T, directory, name, content string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
	return path
}
