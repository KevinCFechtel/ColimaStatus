// Package tray renders the ColimaStatus menu bar item: the menu state, the
// available actions, and the template icons for each Colima state.
package tray

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/KevinCFechtel/ColimaStatus/internal/autostart"
	"github.com/KevinCFechtel/ColimaStatus/internal/colima"
	"github.com/KevinCFechtel/ColimaStatus/internal/localization"
	"github.com/KevinCFechtel/ColimaStatus/internal/monitor"
	"github.com/KevinCFechtel/ColimaStatus/internal/power"
)

// autostartRefreshInterval only has to notice a change the user made in System
// Settings, which is rare. Every tick crosses into Objective-C to ask
// SMAppService, so a short interval would burn energy for a value that almost
// never moves; the toggle itself refreshes immediately.
const autostartRefreshInterval = 30 * time.Second

// Options carries everything the menu bar app needs. It is a struct rather
// than a parameter list because the paths are only used for the reveal actions
// and would otherwise make the call unreadable.
type Options struct {
	Controller monitor.Controller
	Interval   time.Duration
	Autostart  autostart.Controller
	Texts      *localization.Strings
	LogPath    string
	ConfigPath string
	// Menu defaults to the real menu bar; tests supply their own.
	Menu Menu
}

type App struct {
	controller monitor.Controller
	interval   time.Duration
	autostart  autostart.Controller
	texts      *localization.Strings
	logPath    string
	configPath string

	ctx           context.Context
	cancel        context.CancelFunc
	monitor       *monitor.Monitor
	wait          sync.WaitGroup
	stopWakeWatch func()
	menu          Menu

	statusItem            Item
	detailsItem           Item
	checkedItem           Item
	startItem             Item
	stopItem              Item
	refreshItem           Item
	autostartItem         Item
	autostartSettingsItem Item
	showLogItem           Item
	showConfigItem        Item
	quitItem              Item
}

func New(options Options) *App {
	ctx, cancel := context.WithCancel(context.Background())
	menu := options.Menu
	if menu == nil {
		menu = SystrayMenu{}
	}
	return &App{
		menu:       menu,
		controller: options.Controller,
		interval:   options.Interval,
		autostart:  options.Autostart,
		texts:      options.Texts,
		logPath:    options.LogPath,
		configPath: options.ConfigPath,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (app *App) OnReady() {
	app.buildMenu()
	app.monitor = monitor.New(app.controller, app.interval, app.render)
	app.stopWakeWatch = power.Watch(app.monitor.NotifyWake)
	app.refreshAutostart()
	app.startBackgroundTasks()
}

// buildMenu creates the rows and nothing else, so that the render logic can be
// exercised without starting background tasks.
func (app *App) buildMenu() {
	app.setIcon(false)
	app.menu.SetTitle("")
	app.menu.SetTooltip(app.texts.TrayTooltip())
	app.menu.SetRemovalAllowed(false)

	app.statusItem = app.menu.AddItem(app.texts.Checking(), app.texts.CurrentStatusTooltip())
	app.statusItem.Disable()
	app.detailsItem = app.menu.AddItem("", app.texts.ProfileDetailsTooltip())
	app.detailsItem.Disable()
	app.detailsItem.Hide()
	app.checkedItem = app.menu.AddItem("", app.texts.LastCheckTooltip())
	app.checkedItem.Disable()
	app.checkedItem.Hide()
	app.menu.AddSeparator()
	app.startItem = app.menu.AddItem(app.texts.Start(), app.texts.StartTooltip())
	app.stopItem = app.menu.AddItem(app.texts.Stop(), app.texts.StopTooltip())
	app.startItem.Disable()
	app.stopItem.Disable()
	app.refreshItem = app.menu.AddItem(app.texts.Refresh(), app.texts.RefreshTooltip())
	app.autostartItem = app.menu.AddCheckbox(app.texts.AutostartTitle(), app.texts.AutostartEnableTooltip(), false)
	app.autostartSettingsItem = app.menu.AddItem(
		app.texts.OpenLoginItems(),
		app.texts.OpenLoginItemsTooltip(),
	)
	app.autostartSettingsItem.Hide()
	app.menu.AddSeparator()
	app.showConfigItem = app.menu.AddItem(
		app.texts.ShowConfiguration(),
		app.texts.ShowConfigurationTooltip(),
	)
	app.showLogItem = app.menu.AddItem(app.texts.ShowLog(), app.texts.ShowLogTooltip())
	app.menu.AddSeparator()
	app.quitItem = app.menu.AddItem(app.texts.Quit(), app.texts.QuitTooltip())
}

func (app *App) OnExit() {
	if app.stopWakeWatch != nil {
		app.stopWakeWatch()
	}
	app.cancel()
	app.wait.Wait()
}

// goRun registers a background task with the wait group at the point where it
// is started. A hard-coded count has to be kept in sync by hand, and getting it
// wrong hangs the app on quit or panics.
func (app *App) goRun(task func()) {
	app.wait.Add(1)
	go func() {
		defer app.wait.Done()
		task()
	}()
}

func (app *App) startBackgroundTasks() {
	app.goRun(func() {
		app.monitor.Run(app.ctx)
	})
	app.goRun(func() {
		ticker := time.NewTicker(autostartRefreshInterval)
		defer ticker.Stop()
		// Every click channel is checked for closure. systray closes them all
		// when the menu goes away, and a receive from a closed channel succeeds
		// immediately, so a case that ignores the flag would spin.
		for {
			select {
			case <-app.ctx.Done():
				return
			case _, open := <-app.startItem.Clicked():
				if !open {
					return
				}
				app.monitor.Trigger(monitor.ActionStart)
			case _, open := <-app.stopItem.Clicked():
				if !open {
					return
				}
				app.monitor.Trigger(monitor.ActionStop)
			case _, open := <-app.refreshItem.Clicked():
				if !open {
					return
				}
				app.monitor.Trigger(monitor.ActionRefresh)
			case _, open := <-app.autostartItem.Clicked():
				if !open {
					return
				}
				app.toggleAutostart()
			case _, open := <-app.autostartSettingsItem.Clicked():
				if !open {
					return
				}
				app.openAutostartSettings()
			case _, open := <-app.showConfigItem.Clicked():
				if !open {
					return
				}
				app.reveal(app.configPath)
			case _, open := <-app.showLogItem.Clicked():
				if !open {
					return
				}
				app.reveal(app.logPath)
			case <-ticker.C:
				app.refreshAutostart()
			case _, open := <-app.quitItem.Clicked():
				if !open {
					return
				}
				app.cancel()
				app.menu.Quit()
				return
			}
		}
	})
}

func (app *App) render(state monitor.State) {
	if state.Busy != "" {
		app.renderBusy(state.Busy)
		return
	}
	app.refreshItem.Enable()

	if state.Profile == nil {
		app.setIcon(false)
		app.menu.SetTooltip(app.texts.UnavailableTooltip())
		app.statusItem.SetTitle(app.texts.Unavailable())
		app.renderError(state.Err)
		app.startItem.Disable()
		app.stopItem.Disable()
		return
	}

	app.renderProfile(*state.Profile, state.Watching)
	if state.Err != nil {
		app.renderError(state.Err)
	}
}

func (app *App) renderBusy(action monitor.Action) {
	title := app.texts.Checking()
	switch action {
	case monitor.ActionStart:
		title = app.texts.Starting()
	case monitor.ActionStop:
		title = app.texts.Stopping()
	}
	app.setIcon(false)
	app.menu.SetTooltip(app.texts.BusyTooltip())
	app.statusItem.SetTitle(title)
	app.detailsItem.Hide()
	app.checkedItem.Hide()
	app.startItem.Disable()
	app.stopItem.Disable()
	app.refreshItem.Disable()
}

func (app *App) renderProfile(profile colima.Profile, watching bool) {
	status := profilePresentation(app.texts, profile)
	app.setIcon(profile.State == colima.StateRunning)
	app.menu.SetTooltip("ColimaStatus – " + status)
	app.statusItem.SetTitle(status)

	if details := profileDetails(profile); details != "" {
		app.detailsItem.SetTitle(details)
		app.detailsItem.Show()
	} else {
		app.detailsItem.Hide()
	}
	app.checkedItem.SetTitle(app.texts.LastChecked(profile.CheckedAt))
	app.checkedItem.SetTooltip(app.checkedTooltip(profile.CheckedAt, watching))
	app.checkedItem.Show()

	app.startItem.Enable()
	app.stopItem.Enable()
	app.stopItem.SetTitle(app.texts.Stop())
	switch profile.State {
	case colima.StateRunning:
		app.startItem.Disable()
	case colima.StateStopped, colima.StateMissing:
		app.stopItem.Disable()
	case colima.StateBroken:
		app.startItem.Disable()
		app.stopItem.SetTitle(app.texts.ForceStop())
	default:
		// A status this version does not recognize is a Colima that moved on,
		// not a broken one. Both actions stay available: disabling them would
		// turn any future status value into a menu with no way to act.
	}
}

// renderError shows the failure in the user's language and keeps the technical
// cause out of the menu. The cause is English, often long, and only useful when
// debugging, so it goes to the log, which the menu can reveal.
func (app *App) renderError(err error) {
	if err == nil {
		return
	}
	log.Printf("Colima reported a failure: %v", err)
	app.checkedItem.SetTitle(failureMessage(app.texts, colima.KindOf(err)))
	app.checkedItem.SetTooltip(app.texts.DetailHint())
	app.checkedItem.Show()
}

func (app *App) setIcon(active bool) {
	app.menu.SetTemplateIcon(iconPNG(active))
}

type autostartMenuState struct {
	title        string
	tooltip      string
	checked      bool
	enabled      bool
	showSettings bool
}

func (app *App) refreshAutostart() {
	if app.autostart == nil {
		app.applyAutostartMenuState(autostartMenuStateFor(app.texts, autostart.Unsupported))
		return
	}
	status, err := app.autostart.Status()
	if err != nil {
		app.reportAutostartError(err)
		return
	}
	app.applyAutostartMenuState(autostartMenuStateFor(app.texts, status))
}

func (app *App) toggleAutostart() {
	if app.autostart == nil {
		return
	}
	status, err := app.autostart.Status()
	if err != nil {
		app.reportAutostartError(err)
		return
	}

	if status == autostart.RequiresApproval {
		app.openAutostartSettings()
		return
	}
	desiredEnabled, canToggle := autostartToggle(status)
	if !canToggle {
		app.applyAutostartMenuState(autostartMenuStateFor(app.texts, status))
		return
	}

	resultingStatus, err := app.autostart.SetEnabled(desiredEnabled)
	if err != nil {
		app.reportAutostartError(err)
		return
	}
	app.applyAutostartMenuState(autostartMenuStateFor(app.texts, resultingStatus))
	if resultingStatus == autostart.RequiresApproval {
		app.openAutostartSettings()
	}
}

func (app *App) openAutostartSettings() {
	if app.autostart == nil {
		return
	}
	if err := app.autostart.OpenSettings(); err != nil {
		app.reportAutostartError(err)
	}
}

func (app *App) reportAutostartError(err error) {
	log.Printf("launch at login could not be managed: %v", err)
	app.autostartItem.SetTitle(app.texts.AutostartManageFailed())
	app.autostartItem.SetTooltip(err.Error())
	app.autostartItem.Disable()
}

func (app *App) applyAutostartMenuState(menuState autostartMenuState) {
	app.autostartItem.SetTitle(menuState.title)
	app.autostartItem.SetTooltip(menuState.tooltip)
	if menuState.checked {
		app.autostartItem.Check()
	} else {
		app.autostartItem.Uncheck()
	}
	if menuState.enabled {
		app.autostartItem.Enable()
	} else {
		app.autostartItem.Disable()
	}
	if menuState.showSettings {
		app.autostartSettingsItem.Show()
	} else {
		app.autostartSettingsItem.Hide()
	}
}

func autostartMenuStateFor(texts *localization.Strings, status autostart.Status) autostartMenuState {
	switch status {
	case autostart.Disabled:
		return autostartMenuState{
			title:   texts.AutostartTitle(),
			tooltip: texts.AutostartEnableTooltip(),
			enabled: true,
		}
	case autostart.Enabled:
		return autostartMenuState{
			title:   texts.AutostartTitle(),
			tooltip: texts.AutostartDisableTooltip(),
			checked: true,
			enabled: true,
		}
	case autostart.RequiresApproval:
		return autostartMenuState{
			title:        texts.AutostartApprovalTitle(),
			tooltip:      texts.AutostartApprovalTooltip(),
			enabled:      true,
			showSettings: true,
		}
	case autostart.NotFound:
		return autostartMenuState{
			title:   texts.AutostartTitle(),
			tooltip: texts.AutostartRegisterTooltip(),
			enabled: true,
		}
	default:
		return autostartMenuState{
			title:   texts.AutostartUnsupportedTitle(),
			tooltip: texts.AutostartUnsupportedTooltip(),
		}
	}
}

func autostartToggle(status autostart.Status) (enabled bool, canToggle bool) {
	switch status {
	case autostart.Disabled, autostart.NotFound:
		return true, true
	case autostart.Enabled:
		return false, true
	default:
		return false, false
	}
}

// failureMessage maps a domain failure onto a menu row. Keeping the mapping
// here rather than in the localization package leaves that package free of any
// knowledge about Colima.
func failureMessage(texts *localization.Strings, kind colima.Kind) string {
	switch kind {
	case colima.KindUnavailable:
		return texts.ColimaNotFound()
	case colima.KindStatus:
		return texts.StatusFailed()
	case colima.KindStart:
		return texts.StartFailed()
	case colima.KindStop:
		return texts.StopFailed()
	case colima.KindTimeout:
		return texts.Timeout()
	default:
		return texts.UnknownFailure()
	}
}

func profilePresentation(texts *localization.Strings, profile colima.Profile) string {
	name := profile.Name
	if name == "" {
		name = "default"
	}
	switch profile.State {
	case colima.StateRunning:
		return texts.ProfileRunning(name)
	case colima.StateStopped:
		return texts.ProfileStopped(name)
	case colima.StateMissing:
		return texts.ProfileMissing(name)
	case colima.StateBroken:
		return texts.ProfileBroken(name)
	default:
		if status := strings.TrimSpace(profile.RawStatus); status != "" {
			return texts.ProfileUnknownWithStatus(name, status)
		}
		return texts.ProfileUnknown(name)
	}
}

func profileDetails(profile colima.Profile) string {
	parts := make([]string, 0, 5)
	if profile.Runtime != "" {
		parts = append(parts, profile.Runtime)
	}
	if profile.Arch != "" {
		parts = append(parts, profile.Arch)
	}
	if profile.CPUs > 0 {
		parts = append(parts, fmt.Sprintf("%d CPU", profile.CPUs))
	}
	if profile.Memory > 0 {
		parts = append(parts, formatBytes(profile.Memory)+" RAM")
	}
	if profile.Disk > 0 {
		parts = append(parts, formatBytes(profile.Disk)+" disk")
	}
	return strings.Join(parts, " · ")
}

func formatBytes(bytes int64) string {
	const gibibyte = int64(1024 * 1024 * 1024)
	if bytes%gibibyte == 0 {
		return fmt.Sprintf("%d GiB", bytes/gibibyte)
	}
	return fmt.Sprintf("%.1f GiB", float64(bytes)/float64(gibibyte))
}

// checkedTooltip explains where the next update will come from, so that a slow
// reaction to a Colima change is attributable instead of looking like a bug.
func (app *App) checkedTooltip(checkedAt time.Time, watching bool) string {
	availability := app.texts.WatchFallback()
	if watching {
		availability = app.texts.WatchActive()
	}
	return app.texts.FormatTimestamp(checkedAt) + " · " + availability
}

// reveal selects a file in Finder. A file that does not exist yet cannot be
// selected, so its directory is opened instead.
func (app *App) reveal(path string) {
	if path == "" {
		return
	}
	arguments := []string{"-R", path}
	if _, err := os.Stat(path); err != nil {
		arguments = []string{filepath.Dir(path)}
	}
	if err := exec.CommandContext(app.ctx, "open", arguments...).Run(); err != nil {
		log.Printf("%s could not be revealed in Finder: %v", path, err)
	}
}
