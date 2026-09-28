package colima

import (
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestParseProfilesObjectStream(t *testing.T) {
	t.Parallel()

	input := `{"name":"default","status":"Running","arch":"aarch64","cpus":4,"memory":8589934592,"disk":107374182400,"runtime":"docker","address":"192.168.5.2"}
{"name":"kubernetes","status":"Stopped","arch":"aarch64","cpus":2,"memory":2147483648,"disk":64424509440,"runtime":"containerd"}`
	profiles, err := ParseProfiles(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseProfiles() error = %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("len(ParseProfiles()) = %d, want 2", len(profiles))
	}
	if profiles[0].Name != "default" || profiles[0].State != StateRunning {
		t.Fatalf("first profile = %#v", profiles[0])
	}
	if profiles[1].State != StateStopped {
		t.Fatalf("second state = %q, want %q", profiles[1].State, StateStopped)
	}
}

func TestParseProfilesArrayAndStates(t *testing.T) {
	t.Parallel()

	profiles, err := ParseProfiles(strings.NewReader(`[
{"name":"broken","status":"Broken"},
{"name":"future","status":"Pausing"}
]`))
	if err != nil {
		t.Fatalf("ParseProfiles() error = %v", err)
	}
	if profiles[0].State != StateBroken {
		t.Fatalf("broken state = %q", profiles[0].State)
	}
	if profiles[1].State != StateUnknown || profiles[1].RawStatus != "Pausing" {
		t.Fatalf("unknown profile = %#v", profiles[1])
	}
}

func TestParseProfilesEmpty(t *testing.T) {
	t.Parallel()

	profiles, err := ParseProfiles(strings.NewReader(" \n"))
	if err != nil {
		t.Fatalf("ParseProfiles() error = %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("len(ParseProfiles()) = %d, want 0", len(profiles))
	}
}

func TestParseProfilesRejectsMalformedJSON(t *testing.T) {
	t.Parallel()

	if _, err := ParseProfiles(strings.NewReader(`{"name":`)); err == nil {
		t.Fatal("ParseProfiles() error = nil, want malformed JSON error")
	}
}

// A future Colima that reports a number as a string must not take the whole
// status read down with it: the state is what the menu needs, the sizes are
// detail.
func TestParseProfilesAcceptsNumbersAsStrings(t *testing.T) {
	t.Parallel()

	profiles, err := ParseProfiles(strings.NewReader(
		`{"name":"default","status":"Running","cpus":"4","memory":"2147483648","disk":"107374182400"}`))
	if err != nil {
		t.Fatalf("ParseProfiles() error = %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("ParseProfiles() returned %d profiles, want 1", len(profiles))
	}
	got := profiles[0]
	if got.State != StateRunning || got.CPUs != 4 || got.Memory != 2147483648 || got.Disk != 107374182400 {
		t.Fatalf("ParseProfiles() = %#v", got)
	}
}

func TestParseProfilesKeepsGoingPastAnUnreadableNumber(t *testing.T) {
	t.Parallel()

	profiles, err := ParseProfiles(strings.NewReader(
		`{"name":"default","status":"Running","memory":"2 GiB"}`))
	if err != nil {
		t.Fatalf("ParseProfiles() error = %v", err)
	}
	if len(profiles) != 1 || profiles[0].State != StateRunning {
		t.Fatalf("ParseProfiles() = %#v, want the state despite the unreadable size", profiles)
	}
	if profiles[0].Memory != 0 {
		t.Fatalf("Memory = %d, want 0 for a value that cannot be interpreted", profiles[0].Memory)
	}
}

func TestParseProfilesSkipsOneBadEntry(t *testing.T) {
	t.Parallel()

	profiles, err := ParseProfiles(strings.NewReader(
		`[{"name":"default","status":"Running"},{"name":["not","a","name"]},{"name":"work","status":"Stopped"}]`))
	if err != nil {
		t.Fatalf("ParseProfiles() error = %v", err)
	}
	if len(profiles) != 2 || profiles[0].Name != "default" || profiles[1].Name != "work" {
		t.Fatalf("ParseProfiles() = %#v, want the two readable profiles", profiles)
	}
}

func TestParseProfilesReportsFullyUnreadableOutput(t *testing.T) {
	t.Parallel()

	if _, err := ParseProfiles(strings.NewReader(`[{"name":{"nested":true}}]`)); err == nil {
		t.Fatal("ParseProfiles() error = nil, want an error when nothing could be interpreted")
	}
	if _, err := ParseProfiles(strings.NewReader(`this is not JSON`)); err == nil {
		t.Fatal("ParseProfiles() error = nil, want an error for output that is not JSON")
	}
}

// RawStatus carries a status this version does not know, so the menu can name
// it instead of flattening it to "unknown".
func TestParseProfilesKeepsTheRawStatus(t *testing.T) {
	t.Parallel()

	profiles, err := ParseProfiles(strings.NewReader(`{"name":"default","status":" Starting "}`))
	if err != nil {
		t.Fatalf("ParseProfiles() error = %v", err)
	}
	if profiles[0].State != StateUnknown || profiles[0].RawStatus != "Starting" {
		t.Fatalf("ParseProfiles() = %#v, want StateUnknown with the trimmed raw status", profiles[0])
	}
}

func TestShortCommandOutputCutsOnARuneBoundary(t *testing.T) {
	t.Parallel()

	output := strings.Repeat("ä", maximumDetailLength)
	got := shortCommandOutput(output)
	if !utf8.ValidString(got) {
		t.Fatalf("shortCommandOutput() = %q, want valid UTF-8", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("shortCommandOutput() = %q, want the ellipsis marker", got)
	}
	if got := shortCommandOutput("short"); got != "short" {
		t.Fatalf("shortCommandOutput() = %q, want short input returned unchanged", got)
	}
}


// This fixture mirrors the public shape emitted by current `colima list
// --json`. It pins the external contract separately from the synthetic parser
// edge cases above so an upstream field/type change is obvious during review.
func TestParseProfilesAgainstColimaListContract(t *testing.T) {
	t.Parallel()

	file, err := os.Open("testdata/colima-list-contract.jsonl")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = file.Close() }()

	profiles, err := ParseProfiles(file)
	if err != nil {
		t.Fatalf("ParseProfiles() error = %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("ParseProfiles() returned %d profiles, want 2", len(profiles))
	}
	if profiles[0].Name != "default" || profiles[0].State != StateRunning ||
		profiles[0].CPUs != 4 || profiles[0].Memory != 8589934592 ||
		profiles[0].Disk != 107374182400 || profiles[0].Runtime != "docker" {
		t.Fatalf("default profile = %#v", profiles[0])
	}
	if profiles[1].Name != "work" || profiles[1].State != StateStopped ||
		profiles[1].Runtime != "containerd" {
		t.Fatalf("work profile = %#v", profiles[1])
	}
}
