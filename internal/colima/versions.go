package colima

import (
	"context"
	"strings"
	"time"
)

const diagnosticVersionTimeout = 3 * time.Second

// ToolVersions records the versions of the external tools ColimaStatus relies
// on. They are diagnostic only: a version command that fails must never keep
// status monitoring from starting.
type ToolVersions struct {
	Colima string
	Lima   string
}

// Versions queries the discovered Colima and Lima executables with their
// documented --version flags. Empty fields mean the version could not be read.
func (client *Client) Versions(ctx context.Context) ToolVersions {
	return ToolVersions{
		Colima: client.commandVersion(ctx, client.path),
		Lima:   client.commandVersion(ctx, client.limaPath),
	}
}

func (client *Client) commandVersion(ctx context.Context, executable string) string {
	if executable == "" {
		return ""
	}
	versionContext, cancel := context.WithTimeout(ctx, diagnosticVersionTimeout)
	defer cancel()

	output, err := client.runner.Run(versionContext, executable, "--version")
	if err != nil {
		return ""
	}
	text := strings.TrimSpace(output.Stdout)
	if text == "" {
		text = strings.TrimSpace(output.Stderr)
	}
	if line, _, found := strings.Cut(text, "\n"); found {
		text = line
	}
	return strings.TrimSpace(text)
}
