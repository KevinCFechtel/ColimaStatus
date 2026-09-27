package tray

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/KevinCFechtel/ColimaStatus/internal/autostart"
	"github.com/KevinCFechtel/ColimaStatus/internal/colima"
	"github.com/KevinCFechtel/ColimaStatus/internal/localization"
)

func TestIconIsPNG(t *testing.T) {
	t.Parallel()

	active := decodeTestIcon(t, iconPNG(true))
	inactive := decodeTestIcon(t, iconPNG(false))
	if active.Bounds().Dx() != 26 || active.Bounds().Dy() != 36 {
		t.Fatalf("icon bounds = %v, want 26x36", active.Bounds())
	}
	for _, corner := range []image.Point{
		active.Bounds().Min,
		{X: active.Bounds().Max.X - 1, Y: active.Bounds().Min.Y},
		{X: active.Bounds().Min.X, Y: active.Bounds().Max.Y - 1},
		{X: active.Bounds().Max.X - 1, Y: active.Bounds().Max.Y - 1},
	} {
		if _, _, _, alpha := active.At(corner.X, corner.Y).RGBA(); alpha != 0 {
			t.Fatalf("corner %v alpha = %d, want transparent background", corner, alpha)
		}
	}

	hasVisiblePixel := false
	hasDimmerPixel := false
	for y := active.Bounds().Min.Y; y < active.Bounds().Max.Y; y++ {
		for x := active.Bounds().Min.X; x < active.Bounds().Max.X; x++ {
			activeColor := color.NRGBAModel.Convert(active.At(x, y)).(color.NRGBA)
			inactiveColor := color.NRGBAModel.Convert(inactive.At(x, y)).(color.NRGBA)
			if activeColor.R != 0 || activeColor.G != 0 || activeColor.B != 0 {
				t.Fatalf("active icon contains a non-template color at %d,%d", x, y)
			}
			if activeColor.A > 0 {
				hasVisiblePixel = true
			}
			if inactiveColor.A < activeColor.A {
				hasDimmerPixel = true
			}
		}
	}
	if !hasVisiblePixel || !hasDimmerPixel {
		t.Fatalf("silhouette visibility = %v, dimmed variant = %v", hasVisiblePixel, hasDimmerPixel)
	}
}

func decodeTestIcon(t *testing.T, data []byte) image.Image {
	t.Helper()
	if len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("iconPNG() did not return a PNG image")
	}
	icon, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode icon: %v", err)
	}
	return icon
}

func TestProfilePresentation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state colima.State
		want  string
	}{
		{state: colima.StateRunning, want: "Colima läuft (work)"},
		{state: colima.StateStopped, want: "Colima ist gestoppt (work)"},
		{state: colima.StateMissing, want: "Colima ist noch nicht eingerichtet (work)"},
		{state: colima.StateBroken, want: "Colima ist defekt (work)"},
	}
	texts := localization.MustNew("de")
	for _, test := range tests {
		got := profilePresentation(texts, colima.Profile{Name: "work", State: test.state})
		if got != test.want {
			t.Errorf("profilePresentation(%q) = %q, want %q", test.state, got, test.want)
		}
	}
}

func TestProfileDetails(t *testing.T) {
	t.Parallel()

	got := profileDetails(colima.Profile{
		Runtime: "docker",
		Arch:    "aarch64",
		CPUs:    4,
		Memory:  8 * 1024 * 1024 * 1024,
	})
	if got != "docker · aarch64 · 4 CPU · 8 GiB RAM" {
		t.Fatalf("profileDetails() = %q", got)
	}
}

func TestAutostartMenuState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		status       autostart.Status
		checked      bool
		enabled      bool
		showSettings bool
	}{
		{name: "unsupported", status: autostart.Unsupported},
		{name: "disabled", status: autostart.Disabled, enabled: true},
		{name: "enabled", status: autostart.Enabled, checked: true, enabled: true},
		{
			name:         "requires approval",
			status:       autostart.RequiresApproval,
			enabled:      true,
			showSettings: true,
		},
		{name: "not found", status: autostart.NotFound, enabled: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			state := autostartMenuStateFor(localization.MustNew("de"), test.status)
			if state.checked != test.checked || state.enabled != test.enabled || state.showSettings != test.showSettings {
				t.Fatalf(
					"state = {checked:%t enabled:%t showSettings:%t}",
					state.checked,
					state.enabled,
					state.showSettings,
				)
			}
		})
	}
}

func TestAutostartToggle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status        autostart.Status
		wantEnabled   bool
		wantCanToggle bool
	}{
		{status: autostart.Disabled, wantEnabled: true, wantCanToggle: true},
		{status: autostart.Enabled, wantEnabled: false, wantCanToggle: true},
		{status: autostart.RequiresApproval},
		{status: autostart.Unsupported},
		{status: autostart.NotFound, wantEnabled: true, wantCanToggle: true},
	}

	for _, test := range tests {
		enabled, canToggle := autostartToggle(test.status)
		if enabled != test.wantEnabled || canToggle != test.wantCanToggle {
			t.Fatalf(
				"autostartToggle(%d) = (%t, %t), want (%t, %t)",
				test.status,
				enabled,
				canToggle,
				test.wantEnabled,
				test.wantCanToggle,
			)
		}
	}
}

// A status this version does not recognize has to reach the user verbatim,
// because that is the only clue about what Colima actually reported.
func TestProfilePresentationNamesAnUnknownStatus(t *testing.T) {
	t.Parallel()

	texts := localization.MustNew("en")
	profile := colima.Profile{Name: "default", State: colima.StateUnknown, RawStatus: "Starting"}
	if got := profilePresentation(texts, profile); !strings.Contains(got, "Starting") {
		t.Fatalf("profilePresentation() = %q, want it to name the reported status", got)
	}

	withoutStatus := colima.Profile{Name: "default", State: colima.StateUnknown}
	if got := profilePresentation(texts, withoutStatus); got == "" {
		t.Fatal("profilePresentation() = empty for an unknown state without a raw status")
	}
}

func TestCheckedTooltipStatesWhereUpdatesComeFrom(t *testing.T) {
	t.Parallel()

	app := &App{texts: localization.MustNew("en")}
	checkedAt := time.Date(2026, 9, 27, 14, 30, 5, 0, time.UTC)

	active := app.checkedTooltip(checkedAt, true)
	fallback := app.checkedTooltip(checkedAt, false)
	if active == fallback {
		t.Fatal("checkedTooltip() does not distinguish live updates from the periodic fallback")
	}
	for _, tooltip := range []string{active, fallback} {
		if !strings.Contains(tooltip, "2026") {
			t.Fatalf("checkedTooltip() = %q, want it to contain the timestamp", tooltip)
		}
	}
}

// A Colima failure must be named in the selected language, and the technical
// cause must not leak into the menu row.
func TestRenderErrorIsLocalizedAndHidesTheCause(t *testing.T) {
	t.Parallel()

	german := localization.MustNew("de")
	cause := fmt.Errorf("exec: %q: executable file not found", "colima")

	for _, test := range []struct {
		name string
		kind colima.Kind
	}{
		{name: "unavailable", kind: colima.KindUnavailable},
		{name: "status", kind: colima.KindStatus},
		{name: "start", kind: colima.KindStart},
		{name: "stop", kind: colima.KindStop},
		{name: "timeout", kind: colima.KindTimeout},
		{name: "unknown", kind: colima.KindUnknown},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			failure := &colima.Error{Kind: test.kind, Cause: cause}
			message := failureMessage(german, colima.KindOf(failure))
			if message == "" {
				t.Fatal("FailureMessage() = empty")
			}
			if strings.Contains(message, "executable file not found") {
				t.Fatalf("FailureMessage() = %q, want the technical cause left out", message)
			}
			if english := failureMessage(localization.MustNew("en"), test.kind); english == message {
				t.Fatalf("FailureMessage() = %q in both languages, want a translation", message)
			}
		})
	}
}

// An error from outside the package must still produce a usable row rather
// than an empty one.
func TestFailureMessageHandlesForeignErrors(t *testing.T) {
	t.Parallel()

	if got := failureMessage(localization.MustNew("en"), colima.KindOf(errors.New("boom"))); got == "" {
		t.Fatal("FailureMessage() = empty for an error from another package")
	}
}
