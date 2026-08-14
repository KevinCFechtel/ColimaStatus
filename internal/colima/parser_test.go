package colima

import (
	"strings"
	"testing"
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
