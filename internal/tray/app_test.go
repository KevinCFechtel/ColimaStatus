package tray

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/KevinCFechtel/ColimaStatus/internal/autostart"
	"github.com/KevinCFechtel/ColimaStatus/internal/colima"
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
	for _, test := range tests {
		got := profilePresentation(colima.Profile{Name: "work", State: test.state})
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

func TestShortError(t *testing.T) {
	t.Parallel()

	message := "1234567890"
	if got := shortError(errors.New(message)); got != message {
		t.Fatalf("shortError() = %q, want %q", got, message)
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
			state := autostartMenuStateFor(test.status)
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
