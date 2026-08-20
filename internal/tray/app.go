package tray

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"

	"github.com/KevinCFechtel/ColimaStatus/internal/autostart"
	"github.com/KevinCFechtel/ColimaStatus/internal/colima"
	"github.com/KevinCFechtel/ColimaStatus/internal/localization"
	"github.com/KevinCFechtel/ColimaStatus/internal/monitor"
)

const autostartRefreshInterval = 5 * time.Second

type App struct {
	controller monitor.Controller
	interval   time.Duration
	autostart  autostart.Controller
	texts      *localization.Strings

	ctx     context.Context
	cancel  context.CancelFunc
	monitor *monitor.Monitor
	wait    sync.WaitGroup

	statusItem            *systray.MenuItem
	detailsItem           *systray.MenuItem
	checkedItem           *systray.MenuItem
	startItem             *systray.MenuItem
	stopItem              *systray.MenuItem
	refreshItem           *systray.MenuItem
	autostartItem         *systray.MenuItem
	autostartSettingsItem *systray.MenuItem
	quitItem              *systray.MenuItem
}

func New(
	controller monitor.Controller,
	interval time.Duration,
	autostartController autostart.Controller,
	texts *localization.Strings,
) *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		controller: controller,
		interval:   interval,
		autostart:  autostartController,
		texts:      texts,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (app *App) OnReady() {
	app.setIcon(false)
	systray.SetTitle("")
	systray.SetTooltip(app.texts.TrayTooltip())
	systray.SetRemovalAllowed(false)

	app.statusItem = systray.AddMenuItem(app.texts.Checking(), app.texts.CurrentStatusTooltip())
	app.statusItem.Disable()
	app.detailsItem = systray.AddMenuItem("", app.texts.ProfileDetailsTooltip())
	app.detailsItem.Disable()
	app.detailsItem.Hide()
	app.checkedItem = systray.AddMenuItem("", app.texts.LastCheckTooltip())
	app.checkedItem.Disable()
	app.checkedItem.Hide()
	systray.AddSeparator()
	app.startItem = systray.AddMenuItem(app.texts.Start(), app.texts.StartTooltip())
	app.stopItem = systray.AddMenuItem(app.texts.Stop(), app.texts.StopTooltip())
	app.startItem.Disable()
	app.stopItem.Disable()
	app.refreshItem = systray.AddMenuItem(app.texts.Refresh(), app.texts.RefreshTooltip())
	app.autostartItem = systray.AddMenuItemCheckbox(app.texts.AutostartTitle(), app.texts.AutostartEnableTooltip(), false)
	app.autostartSettingsItem = systray.AddMenuItem(
		app.texts.OpenLoginItems(),
		app.texts.OpenLoginItemsTooltip(),
	)
	app.autostartSettingsItem.Hide()
	systray.AddSeparator()
	app.quitItem = systray.AddMenuItem(app.texts.Quit(), app.texts.QuitTooltip())

	app.monitor = monitor.New(app.controller, app.interval, app.render)
	app.refreshAutostart()
	app.startBackgroundTasks()
}

func (app *App) OnExit() {
	app.cancel()
	app.wait.Wait()
}

func (app *App) startBackgroundTasks() {
	app.wait.Add(2)
	go func() {
		defer app.wait.Done()
		app.monitor.Run(app.ctx)
	}()
	go func() {
		defer app.wait.Done()
		ticker := time.NewTicker(autostartRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-app.ctx.Done():
				return
			case <-app.startItem.ClickedCh:
				app.monitor.Trigger(monitor.ActionStart)
			case <-app.stopItem.ClickedCh:
				app.monitor.Trigger(monitor.ActionStop)
			case <-app.refreshItem.ClickedCh:
				app.monitor.Trigger(monitor.ActionRefresh)
			case _, open := <-app.autostartItem.ClickedCh:
				if !open {
					return
				}
				app.toggleAutostart()
			case _, open := <-app.autostartSettingsItem.ClickedCh:
				if !open {
					return
				}
				app.openAutostartSettings()
			case <-ticker.C:
				app.refreshAutostart()
			case <-app.quitItem.ClickedCh:
				app.cancel()
				systray.Quit()
				return
			}
		}
	}()
}

func (app *App) render(state monitor.State) {
	if state.Busy != "" {
		app.renderBusy(state.Busy)
		return
	}
	app.refreshItem.Enable()

	if state.Profile == nil {
		app.setIcon(false)
		systray.SetTooltip(app.texts.UnavailableTooltip())
		app.statusItem.SetTitle(app.texts.Unavailable())
		app.renderError(state.Err)
		app.startItem.Disable()
		app.stopItem.Disable()
		return
	}

	app.renderProfile(*state.Profile)
	if state.Err != nil {
		app.renderError(state.Err)
	}
}

func (app *App) renderBusy(action monitor.Action) {
	title := app.texts.Checking()
	if action == monitor.ActionStart {
		title = app.texts.Starting()
	} else if action == monitor.ActionStop {
		title = app.texts.Stopping()
	}
	app.setIcon(false)
	systray.SetTooltip(app.texts.BusyTooltip())
	app.statusItem.SetTitle(title)
	app.detailsItem.Hide()
	app.checkedItem.Hide()
	app.startItem.Disable()
	app.stopItem.Disable()
	app.refreshItem.Disable()
}

func (app *App) renderProfile(profile colima.Profile) {
	status := profilePresentation(app.texts, profile)
	app.setIcon(profile.State == colima.StateRunning)
	systray.SetTooltip("ColimaStatus – " + status)
	app.statusItem.SetTitle(status)

	if details := profileDetails(profile); details != "" {
		app.detailsItem.SetTitle(details)
		app.detailsItem.Show()
	} else {
		app.detailsItem.Hide()
	}
	app.checkedItem.SetTitle(app.texts.LastChecked(profile.CheckedAt))
	app.checkedItem.SetTooltip(app.texts.FormatTimestamp(profile.CheckedAt))
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
		app.startItem.Disable()
		app.stopItem.Disable()
	}
}

func (app *App) renderError(err error) {
	if err == nil {
		return
	}
	app.checkedItem.SetTitle(shortError(err))
	app.checkedItem.SetTooltip(err.Error())
	app.checkedItem.Show()
}

func (app *App) setIcon(active bool) {
	icon := iconPNG(active)
	systray.SetTemplateIcon(icon, icon)
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
		return texts.ProfileUnknown(name)
	}
}

func profileDetails(profile colima.Profile) string {
	parts := make([]string, 0, 4)
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
	return strings.Join(parts, " · ")
}

func formatBytes(bytes int64) string {
	const gibibyte = int64(1024 * 1024 * 1024)
	if bytes%gibibyte == 0 {
		return fmt.Sprintf("%d GiB", bytes/gibibyte)
	}
	return fmt.Sprintf("%.1f GiB", float64(bytes)/float64(gibibyte))
}

func shortError(err error) string {
	message := err.Error()
	const maximumLength = 90
	if len(message) <= maximumLength {
		return message
	}
	return message[:maximumLength] + "…"
}
