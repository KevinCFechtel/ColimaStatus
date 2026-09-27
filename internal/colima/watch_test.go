package colima

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"
)

func TestRelevantLimaEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		json string
		want bool
	}{
		{
			name: "running",
			json: `{"instance":"colima","event":{"status":{"running":true}}}`,
			want: true,
		},
		{
			name: "exiting",
			json: `{"instance":"colima","event":{"status":{"exiting":true}}}`,
			want: true,
		},
		{
			name: "degraded",
			json: `{"instance":"colima","event":{"status":{"degraded":true}}}`,
			want: true,
		},
		{
			name: "runtime errors",
			json: `{"instance":"colima","event":{"status":{"running":true,"errors":["unhealthy"]}}}`,
			want: true,
		},
		{
			name: "instance lifecycle",
			json: `{"instance":"colima","event":{"created":true}}`,
			want: true,
		},
		{
			name: "port forwarding",
			json: `{"instance":"colima","event":{"status":{"running":true,"portForward":{"type":"forwarding"}}}}`,
			want: true,
		},
		{
			name: "port forwarding without lifecycle state",
			json: `{"instance":"colima","event":{"status":{"portForward":{"type":"forwarding"}}}}`,
			want: false,
		},
		{
			name: "different profile",
			json: `{"instance":"colima-work","event":{"status":{"running":true}}}`,
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := relevantLimaEvent([]byte(test.json), "colima")
			if err != nil {
				t.Fatalf("relevantLimaEvent() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("relevantLimaEvent() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestRelevantLimaEventRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	if _, err := relevantLimaEvent([]byte(`{"instance":`), "colima"); err == nil {
		t.Fatal("relevantLimaEvent() error = nil, want malformed JSON error")
	}
}

func TestWatchFiltersEventStream(t *testing.T) {
	t.Parallel()

	script := writeExecutable(t, `#!/bin/sh
printf '%s\n' \
  '{"instance":"colima","event":{"status":{"portForward":{"type":"forwarding"}}}}' \
  '{"instance":"colima-work","event":{"status":{"running":true}}}' \
  '{"instance":"colima","event":{"status":{"running":true}}}' \
  '{"instance":"colima","event":{"status":{"running":true,"portForward":{"type":"forwarding"}}}}' \
  '{"instance":"colima","event":{"status":{"running":true,"errors":["unhealthy"]}}}' \
  '{"instance":"colima","event":{"status":{"exiting":true}}}'
`)
	client := NewClient("colima", "default")
	client.limaPath = script
	client.limaHome = t.TempDir()

	var notifications atomic.Int32
	err := client.Watch(context.Background(), func() { notifications.Add(1) })
	if err == nil {
		t.Fatal("Watch() error = nil, want ended stream error")
	}
	if notifications.Load() != 3 {
		t.Fatalf("Watch() notifications = %d, want 3", notifications.Load())
	}
}

func TestWatchDetectsUnsupportedCommand(t *testing.T) {
	t.Parallel()

	script := writeExecutable(t, "#!/bin/sh\necho 'unknown command watch' >&2\nexit 1\n")
	client := NewClient("colima", "default")
	client.limaPath = script
	client.limaHome = t.TempDir()

	err := client.Watch(context.Background(), func() {})
	if !errors.Is(err, ErrWatchUnsupported) {
		t.Fatalf("Watch() error = %v, want ErrWatchUnsupported", err)
	}
}

func TestLimaInstanceName(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"":            "colima",
		"default":     "colima",
		"work":        "colima-work",
		"colima-work": "colima-work",
	}
	for profile, want := range tests {
		if got := limaInstanceName(profile); got != want {
			t.Errorf("limaInstanceName(%q) = %q, want %q", profile, got, want)
		}
	}
}

func TestLocateLimactlNextToColima(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	colimaPath := filepath.Join(directory, "colima")
	limactlPath := filepath.Join(directory, "limactl")
	for _, path := range []string{colimaPath, limactlPath} {
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}
	if got := locateLimactl(colimaPath); got != limactlPath {
		t.Fatalf("locateLimactl() = %q, want %q", got, limactlPath)
	}
}

func TestResolveLimaHomePrefersExplicitLimaHome(t *testing.T) {
	t.Setenv("LIMA_HOME", "relative-lima")
	t.Setenv("COLIMA_HOME", "relative-colima")

	want, err := filepath.Abs("relative-lima")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	if got := resolveLimaHome(); got != want {
		t.Fatalf("resolveLimaHome() = %q, want %q", got, want)
	}
}

func TestResolveLimaHomeUsesColimaHome(t *testing.T) {
	t.Setenv("LIMA_HOME", "")
	t.Setenv("COLIMA_HOME", "relative-colima")

	base, err := filepath.Abs("relative-colima")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	want := filepath.Join(base, "_lima")
	if got := resolveLimaHome(); got != want {
		t.Fatalf("resolveLimaHome() = %q, want %q", got, want)
	}
}

func writeExecutable(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "limactl")
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

// Recorded from `limactl watch --json --history` against Lima 2.2.0 with a
// Colima profile that started and stopped. It is the contract this package
// does not own: if a Lima release renames or restructures these fields, the
// app degrades to the safety interval without any error, so the shape is
// pinned here rather than trusted.
func TestLifecycleStateAgainstRecordedLimaEvents(t *testing.T) {
	t.Parallel()

	file, err := os.Open("testdata/lima-watch-2.2.0.jsonl")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = file.Close() }()

	var states []string
	var ignored int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			continue
		}
		state, relevant, err := limaLifecycleState(scanner.Bytes(), "colima")
		if err != nil {
			t.Fatalf("limaLifecycleState() error = %v for %s", err, scanner.Text())
		}
		if !relevant {
			ignored++
			continue
		}
		states = append(states, state)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	// Port forwarding, SSH and vsock noise must not wake the monitor: those
	// events arrive continuously while a profile runs.
	if ignored != 5 {
		t.Errorf("ignored %d events, want the 5 non-lifecycle ones", ignored)
	}
	want := []string{"running", "running", "running", "running", "exiting"}
	if !slices.Equal(states, want) {
		t.Fatalf("lifecycle states = %#v, want %#v", states, want)
	}
}
