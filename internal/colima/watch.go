package colima

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var ErrWatchUnsupported = errors.New("Lima events are not supported")

const maximumWatchEventSize = 1024 * 1024

type limaWatchEvent struct {
	Instance string `json:"instance"`
	Event    struct {
		Status *struct {
			Running  *bool           `json:"running"`
			Degraded *bool           `json:"degraded"`
			Exiting  *bool           `json:"exiting"`
			Errors   json.RawMessage `json:"errors"`
		} `json:"status"`
	} `json:"event"`
}

// Watch streams Lima lifecycle events and calls notify when the configured
// Colima instance may have changed state. Port-forwarding and SSH events are
// deliberately ignored.
func (client *Client) Watch(ctx context.Context, notify func()) error {
	if client.limaPath == "" || client.limaHome == "" {
		return ErrWatchUnsupported
	}

	command := exec.CommandContext(ctx, client.limaPath, "watch", "--json")
	command.Env = append(os.Environ(), "LIMA_HOME="+client.limaHome, "NO_COLOR=1")
	stdout, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Lima event stream could not be opened: %w", err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrWatchUnsupported
		}
		return fmt.Errorf("Lima event stream could not be started: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), maximumWatchEventSize)
	lastLifecycleState := ""
	for scanner.Scan() {
		lifecycleState, relevant, parseErr := limaLifecycleState(scanner.Bytes(), client.limaInstance)
		if parseErr == nil && relevant && lifecycleState != lastLifecycleState {
			lastLifecycleState = lifecycleState
			notify()
		}
	}
	scanErr := scanner.Err()
	waitErr := command.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if scanErr != nil {
		return fmt.Errorf("Lima event stream could not be read: %w", scanErr)
	}
	if waitErr != nil {
		details := strings.TrimSpace(stderr.String())
		if watchCommandUnsupported(details) {
			return ErrWatchUnsupported
		}
		if details != "" {
			return fmt.Errorf("Lima event stream exited: %w: %s", waitErr, shortCommandOutput(details))
		}
		return fmt.Errorf("Lima event stream exited: %w", waitErr)
	}
	return errors.New("Lima event stream exited unexpectedly")
}

func relevantLimaEvent(data []byte, instance string) (bool, error) {
	_, relevant, err := limaLifecycleState(data, instance)
	return relevant, err
}

func limaLifecycleState(data []byte, instance string) (string, bool, error) {
	var event limaWatchEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return "", false, err
	}
	if event.Instance != instance {
		return "", false, nil
	}
	if event.Event.Status == nil {
		return "instance-lifecycle", true, nil
	}
	status := event.Event.Status
	if status.Exiting != nil && *status.Exiting {
		return "exiting", true, nil
	}
	if status.Degraded != nil {
		if *status.Degraded {
			return "degraded", true, nil
		}
		return "healthy", true, nil
	}
	if hasJSONValue(status.Errors) {
		return "errors", true, nil
	}
	if status.Running != nil {
		if *status.Running {
			return "running", true, nil
		}
		return "not-running", true, nil
	}
	return "", false, nil
}

func hasJSONValue(value json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(value))
	return trimmed != "" && trimmed != "null" && trimmed != "[]"
}

func watchCommandUnsupported(stderr string) bool {
	message := strings.ToLower(stderr)
	return strings.Contains(message, "unknown command") ||
		strings.Contains(message, "unknown flag") ||
		strings.Contains(message, "no help topic")
}

func shortCommandOutput(output string) string {
	const maximumLength = 600
	if len(output) <= maximumLength {
		return output
	}
	return output[:maximumLength] + "…"
}

func locateLimactl(colimaPath string) string {
	if colimaPath != "" {
		candidate := filepath.Join(filepath.Dir(colimaPath), "limactl")
		if path, err := validateExecutable(candidate); err == nil {
			return path
		}
	}
	if path, err := exec.LookPath("limactl"); err == nil {
		if validated, validateErr := validateExecutable(path); validateErr == nil {
			return validated
		}
	}
	for _, candidate := range []string{
		"/opt/homebrew/bin/limactl",
		"/usr/local/bin/limactl",
		"/opt/local/bin/limactl",
	} {
		if path, err := validateExecutable(candidate); err == nil {
			return path
		}
	}
	return ""
}

func resolveLimaHome() string {
	if path := os.Getenv("LIMA_HOME"); path != "" {
		return absolutePath(path)
	}
	if path := os.Getenv("COLIMA_HOME"); path != "" {
		return filepath.Join(absolutePath(path), "_lima")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	legacyHome := filepath.Join(home, ".colima")
	if _, err := os.Stat(legacyHome); err == nil {
		return filepath.Join(legacyHome, "_lima")
	}
	if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
		return filepath.Join(absolutePath(xdgHome), "colima", "_lima")
	}
	xdgDefault := filepath.Join(home, ".config", "colima")
	if _, err := os.Stat(xdgDefault); err == nil {
		return filepath.Join(xdgDefault, "_lima")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(legacyHome, "_lima")
	}
	configHome, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configHome, "colima", "_lima")
}

func absolutePath(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return absolute
}

func limaInstanceName(profile string) string {
	if profile == "" || profile == defaultProfile || profile == "colima" {
		return "colima"
	}
	return "colima-" + strings.TrimPrefix(profile, "colima-")
}
