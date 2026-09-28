package tray

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/KevinCFechtel/ColimaStatus/internal/colima"
	"github.com/KevinCFechtel/ColimaStatus/internal/localization"
	"github.com/KevinCFechtel/ColimaStatus/internal/monitor"
)

func newRenderTestApp(t *testing.T) (*App, *fakeMenu) {
	t.Helper()

	menu := &fakeMenu{}
	app := New(Options{
		Interval: time.Hour,
		Texts:    localization.MustNew("en"),
		Menu:     menu,
	})
	app.buildMenu()
	return app, menu
}

// The action a state allows is the part most likely to break when Colima
// reports something new, so every state is pinned down here.
func TestRenderEnablesTheActionsThatFitTheState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		state      colima.State
		raw        string
		wantStart  bool
		wantStop   bool
		wantActive bool
	}{
		{name: "running", state: colima.StateRunning, wantStart: false, wantStop: true, wantActive: true},
		{name: "stopped", state: colima.StateStopped, wantStart: true, wantStop: false},
		{name: "missing", state: colima.StateMissing, wantStart: true, wantStop: false},
		{name: "broken", state: colima.StateBroken, wantStart: false, wantStop: true},
		// A status this version does not know must leave both actions usable.
		{name: "unknown", state: colima.StateUnknown, raw: "Starting", wantStart: true, wantStop: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			app, menu := newRenderTestApp(t)
			profile := colima.Profile{
				Name: "default", State: test.state, RawStatus: test.raw,
				CheckedAt: time.Date(2026, 9, 27, 14, 0, 0, 0, time.UTC),
			}
			app.render(monitor.State{Profile: &profile})

			if _, startEnabled, _, _ := app.startItem.(*fakeItem).state(); startEnabled != test.wantStart {
				t.Errorf("start enabled = %v, want %v", startEnabled, test.wantStart)
			}
			if _, stopEnabled, _, _ := app.stopItem.(*fakeItem).state(); stopEnabled != test.wantStop {
				t.Errorf("stop enabled = %v, want %v", stopEnabled, test.wantStop)
			}
			if _, refreshEnabled, _, _ := app.refreshItem.(*fakeItem).state(); !refreshEnabled {
				t.Error("refresh is disabled, want it always available")
			}

			active := string(menu.icon) == string(iconPNG(true))
			if active != test.wantActive {
				t.Errorf("bright icon = %v, want %v", active, test.wantActive)
			}
		})
	}
}

func TestRenderNamesABrokenProfilesForceStop(t *testing.T) {
	t.Parallel()

	app, _ := newRenderTestApp(t)
	texts := localization.MustNew("en")

	app.render(monitor.State{Profile: &colima.Profile{Name: "default", State: colima.StateBroken}})
	if title, _, _, _ := app.stopItem.(*fakeItem).state(); title != texts.ForceStop() {
		t.Fatalf("stop title = %q, want %q", title, texts.ForceStop())
	}

	// Recovering must put the ordinary label back.
	app.render(monitor.State{Profile: &colima.Profile{Name: "default", State: colima.StateRunning}})
	if title, _, _, _ := app.stopItem.(*fakeItem).state(); title != texts.Stop() {
		t.Fatalf("stop title = %q, want %q after recovery", title, texts.Stop())
	}
}

func TestRenderWhileBusyDisablesEveryAction(t *testing.T) {
	t.Parallel()

	for _, action := range []monitor.Action{monitor.ActionStart, monitor.ActionStop, monitor.ActionRefresh} {
		app, _ := newRenderTestApp(t)
		app.render(monitor.State{Busy: action})

		for name, item := range map[string]Item{
			"start": app.startItem, "stop": app.stopItem, "refresh": app.refreshItem,
		} {
			if _, enabled, _, _ := item.(*fakeItem).state(); enabled {
				t.Errorf("%s is enabled while %s is running", name, action)
			}
		}
	}
}

func TestRenderWithoutAProfileReportsTheFailure(t *testing.T) {
	t.Parallel()

	app, _ := newRenderTestApp(t)
	failure := &colima.Error{Kind: colima.KindUnavailable, Cause: errors.New("not found in PATH")}
	app.render(monitor.State{Err: failure})

	title, _, _, _ := app.statusItem.(*fakeItem).state()
	if title != localization.MustNew("en").Unavailable() {
		t.Fatalf("status title = %q", title)
	}
	checked, _, visible, _ := app.checkedItem.(*fakeItem).state()
	if !visible {
		t.Fatal("the failure row is hidden")
	}
	if strings.Contains(checked, "not found in PATH") {
		t.Fatalf("failure row = %q, want the technical cause left out", checked)
	}
	if _, startEnabled, _, _ := app.startItem.(*fakeItem).state(); startEnabled {
		t.Error("start is enabled although Colima is unavailable")
	}
}

func TestRenderShowsTheProfileDetails(t *testing.T) {
	t.Parallel()

	app, _ := newRenderTestApp(t)
	app.render(monitor.State{Profile: &colima.Profile{
		Name: "default", State: colima.StateRunning, Runtime: "docker", Arch: "aarch64",
		CPUs: 2, Memory: 2 << 30, Disk: 100 << 30,
	}})

	details, _, visible, _ := app.detailsItem.(*fakeItem).state()
	if !visible {
		t.Fatal("the details row is hidden although the profile has details")
	}
	for _, want := range []string{"docker", "aarch64", "2 CPU", "2 GiB RAM", "100 GiB disk"} {
		if !strings.Contains(details, want) {
			t.Errorf("details = %q, want it to contain %q", details, want)
		}
	}

	// A profile without details must not leave an empty row behind.
	app.render(monitor.State{Profile: &colima.Profile{Name: "default", State: colima.StateMissing}})
	if _, _, stillVisible, _ := app.detailsItem.(*fakeItem).state(); stillVisible {
		t.Error("the details row stays visible for a profile with no details")
	}
}

func TestRenderReportsWhereUpdatesComeFrom(t *testing.T) {
	t.Parallel()

	app, _ := newRenderTestApp(t)
	profile := colima.Profile{Name: "default", State: colima.StateRunning}

	app.render(monitor.State{Profile: &profile, Watch: monitor.WatchActive})
	_, _, _, _ = app.checkedItem.(*fakeItem).state()
	withEvents := app.checkedItem.(*fakeItem).tooltipText()

	app.render(monitor.State{Profile: &profile, Watch: monitor.WatchUnavailable})
	withoutEvents := app.checkedItem.(*fakeItem).tooltipText()

	if withEvents == withoutEvents {
		t.Fatal("the tooltip does not distinguish live events from the periodic fallback")
	}
}

func (item *fakeItem) tooltipText() string {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	return item.tooltip
}
