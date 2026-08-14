package colima

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type CommandOutput struct {
	Stdout string
	Stderr string
}

type Runner interface {
	Run(ctx context.Context, executable string, args ...string) (CommandOutput, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, executable string, args ...string) (CommandOutput, error) {
	command := exec.CommandContext(ctx, executable, args...)
	command.Env = append(os.Environ(), "NO_COLOR=1")

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
