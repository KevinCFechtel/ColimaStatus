package colima

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const fallbackSystemPath = "/usr/bin:/bin:/usr/sbin:/sbin"

type CommandOutput struct {
	Stdout string
	Stderr string
}

type Runner interface {
	Run(ctx context.Context, executable string, args ...string) (CommandOutput, error)
}

type ExecRunner struct {
	lookupPaths []string
}

func (runner ExecRunner) Run(ctx context.Context, executable string, args ...string) (CommandOutput, error) {
	command := exec.CommandContext(ctx, executable, args...)
	lookupPaths := append([]string{filepath.Dir(executable)}, runner.lookupPaths...)
	command.Env = append(environmentWithPath(os.Environ(), lookupPaths...), "NO_COLOR=1")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	output := CommandOutput{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return output, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return output, ctxErr
	}

	details := strings.TrimSpace(output.Stderr)
	if details == "" {
		details = strings.TrimSpace(output.Stdout)
	}
	if len(details) > 600 {
		details = details[:600] + "…"
	}
	if details == "" {
		return output, err
	}
	return output, fmt.Errorf("%w: %s", err, details)
}

func executableDirectories(paths ...string) []string {
	directories := make([]string, 0, len(paths))
	for _, path := range paths {
		if path != "" && filepath.IsAbs(path) {
			directories = append(directories, filepath.Dir(path))
		}
	}
	return directories
}

func environmentWithPath(environment []string, directories ...string) []string {
	result := make([]string, 0, len(environment)+1)
	currentPath := ""
	for _, variable := range environment {
		name, value, found := strings.Cut(variable, "=")
		if found && name == "PATH" {
			currentPath = value
			continue
		}
		result = append(result, variable)
	}
	if currentPath == "" {
		currentPath = fallbackSystemPath
	}

	pathEntries := make([]string, 0, len(directories)+len(filepath.SplitList(currentPath)))
	seen := make(map[string]struct{})
	add := func(path string) {
		if path == "" || !filepath.IsAbs(path) {
			return
		}
		path = filepath.Clean(path)
		if _, exists := seen[path]; exists {
			return
		}
		seen[path] = struct{}{}
		pathEntries = append(pathEntries, path)
	}
	for _, directory := range directories {
		add(directory)
	}
	for _, path := range filepath.SplitList(currentPath) {
		add(path)
	}

	return append(result, "PATH="+strings.Join(pathEntries, string(os.PathListSeparator)))
}
