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
	"github.com/KevinCFechtel/ColimaStatus/internal/monitor"
)

const autostartRefreshInterval = 5 * time.Second

type App struct {
	controller monitor.Controller
	interval   time.Duration
	autostart  autostart.Controller

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

func New(controller monitor.Controller, interval time.Duration, autostartController autostart.Controller) *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		controller: controller,
		interval:   interval,
		autostart:  autostartController,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (app *App) OnReady() {
	app.setIcon(false)
	systray.SetTitle("")
	systray.SetTooltip("ColimaStatus")
	systray.SetRemovalAllowed(false)

	app.statusItem = systray.AddMenuItem("Colima-Status wird geprüft …", "Aktueller Colima-Status")
	app.statusItem.Disable()
	app.detailsItem = systray.AddMenuItem("", "Colima-Konfiguration")
	app.detailsItem.Disable()
	app.detailsItem.Hide()
	app.checkedItem = systray.AddMenuItem("", "Zeitpunkt der letzten Statusabfrage")
	app.checkedItem.Disable()
	app.checkedItem.Hide()
	systray.AddSeparator()
	app.startItem = systray.AddMenuItem("Starten", "Colima starten")
	app.stopItem = systray.AddMenuItem("Stoppen", "Colima stoppen")
	app.startItem.Disable()
	app.stopItem.Disable()
	app.refreshItem = systray.AddMenuItem("Status aktualisieren", "Colima-Status sofort prüfen")
	app.autostartItem = systray.AddMenuItemCheckbox("Bei Anmeldung starten", "ColimaStatus automatisch starten", false)
	app.autostartSettingsItem = systray.AddMenuItem(
		"Anmeldeobjekte in Systemeinstellungen öffnen …",
		"Autostart für ColimaStatus in macOS freigeben",
	)
	app.autostartSettingsItem.Hide()
	systray.AddSeparator()
	app.quitItem = systray.AddMenuItem("Beenden", "ColimaStatus beenden")

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
		systray.SetTooltip("ColimaStatus – Fehler")
		app.statusItem.SetTitle("Colima-Status nicht verfügbar")
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
	title := "Colima-Status wird geprüft …"
	if action == monitor.ActionStart {
		title = "Colima wird gestartet …"
	} else if action == monitor.ActionStop {
		title = "Colima wird gestoppt …"
	}
	app.setIcon(false)
	systray.SetTooltip("ColimaStatus – Vorgang läuft")
	app.statusItem.SetTitle(title)
	app.detailsItem.Hide()
	app.checkedItem.Hide()
	app.startItem.Disable()
	app.stopItem.Disable()
	app.refreshItem.Disable()
}

func (app *App) renderProfile(profile colima.Profile) {
	status := profilePresentation(profile)
	app.setIcon(profile.State == colima.StateRunning)
	systray.SetTooltip("ColimaStatus – " + status)
	app.statusItem.SetTitle(status)

	if details := profileDetails(profile); details != "" {
		app.detailsItem.SetTitle(details)
		app.detailsItem.Show()
	} else {
		app.detailsItem.Hide()
	}
	app.checkedItem.SetTitle("Zuletzt geprüft: " + profile.CheckedAt.Format("15:04:05"))
	app.checkedItem.SetTooltip(profile.CheckedAt.Format("02.01.2006, 15:04:05"))
	app.checkedItem.Show()

	app.startItem.Enable()
	app.stopItem.Enable()
	app.stopItem.SetTitle("Stoppen")
	switch profile.State {
	case colima.StateRunning:
		app.startItem.Disable()
	case colima.StateStopped, colima.StateMissing:
		app.stopItem.Disable()
	case colima.StateBroken:
		app.startItem.Disable()
		app.stopItem.SetTitle("Erzwungen stoppen")
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
		app.applyAutostartMenuState(autostartMenuStateFor(autostart.Unsupported))
		return
	}
	status, err := app.autostart.Status()
	if err != nil {
		app.reportAutostartError(err)
		return
	}
	app.applyAutostartMenuState(autostartMenuStateFor(status))
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
		app.applyAutostartMenuState(autostartMenuStateFor(status))
		return
	}

	resultingStatus, err := app.autostart.SetEnabled(desiredEnabled)
	if err != nil {
		app.reportAutostartError(err)
		return
	}
	app.applyAutostartMenuState(autostartMenuStateFor(resultingStatus))
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
	log.Printf("Autostart konnte nicht verwaltet werden: %v", err)
	app.autostartItem.SetTitle("Autostart konnte nicht geändert werden")
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

func autostartMenuStateFor(status autostart.Status) autostartMenuState {
	switch status {
	case autostart.Disabled:
		return autostartMenuState{
			title:   "Bei Anmeldung starten",
			tooltip: "ColimaStatus automatisch nach der Anmeldung starten",
			enabled: true,
		}
	case autostart.Enabled:
		return autostartMenuState{
			title:   "Bei Anmeldung starten",
			tooltip: "Autostart für ColimaStatus deaktivieren",
			checked: true,
			enabled: true,
		}
	case autostart.RequiresApproval:
		return autostartMenuState{
			title:        "Bei Anmeldung starten (Freigabe erforderlich)",
			tooltip:      "In den macOS-Systemeinstellungen freigeben",
			enabled:      true,
			showSettings: true,
		}
	case autostart.NotFound:
		return autostartMenuState{
			title:   "Bei Anmeldung starten",
			tooltip: "ColimaStatus als Anmeldeobjekt registrieren",
			enabled: true,
		}
	default:
		return autostartMenuState{
			title:   "Bei Anmeldung starten (ab macOS 13)",
			tooltip: "Diese Funktion benötigt macOS 13 oder neuer",
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

func profilePresentation(profile colima.Profile) string {
	name := profile.Name
	if name == "" {
		name = "default"
	}
	switch profile.State {
	case colima.StateRunning:
		return fmt.Sprintf("Colima läuft (%s)", name)
	case colima.StateStopped:
		return fmt.Sprintf("Colima ist gestoppt (%s)", name)
	case colima.StateMissing:
		return fmt.Sprintf("Colima ist noch nicht eingerichtet (%s)", name)
	case colima.StateBroken:
		return fmt.Sprintf("Colima ist defekt (%s)", name)
	default:
		return fmt.Sprintf("Unbekannter Colima-Status (%s)", name)
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
