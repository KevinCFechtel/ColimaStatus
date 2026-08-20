package localization

import (
	"time"

	"golang.org/x/text/language"
)

func (strings *Strings) TrayTooltip() string { return strings.localize(messageTrayTooltip, nil) }
func (strings *Strings) Checking() string    { return strings.localize(messageTrayChecking, nil) }
func (strings *Strings) CurrentStatusTooltip() string {
	return strings.localize(messageTrayCurrentStatusTooltip, nil)
}
func (strings *Strings) ProfileDetailsTooltip() string {
	return strings.localize(messageTrayProfileDetailsTooltip, nil)
}
func (strings *Strings) LastCheckTooltip() string {
	return strings.localize(messageTrayLastCheckTooltip, nil)
}
func (strings *Strings) Start() string        { return strings.localize(messageTrayStart, nil) }
func (strings *Strings) StartTooltip() string { return strings.localize(messageTrayStartTooltip, nil) }
func (strings *Strings) Stop() string         { return strings.localize(messageTrayStop, nil) }
func (strings *Strings) StopTooltip() string  { return strings.localize(messageTrayStopTooltip, nil) }
func (strings *Strings) ForceStop() string    { return strings.localize(messageTrayForceStop, nil) }
func (strings *Strings) Refresh() string      { return strings.localize(messageTrayRefresh, nil) }
func (strings *Strings) RefreshTooltip() string {
	return strings.localize(messageTrayRefreshTooltip, nil)
}
func (strings *Strings) OpenLoginItems() string {
	return strings.localize(messageTrayOpenLoginItems, nil)
}
func (strings *Strings) OpenLoginItemsTooltip() string {
	return strings.localize(messageTrayOpenLoginItemsTooltip, nil)
}
func (strings *Strings) Quit() string        { return strings.localize(messageTrayQuit, nil) }
func (strings *Strings) QuitTooltip() string { return strings.localize(messageTrayQuitTooltip, nil) }
func (strings *Strings) UnavailableTooltip() string {
	return strings.localize(messageTrayUnavailableTooltip, nil)
}
func (strings *Strings) Unavailable() string { return strings.localize(messageTrayUnavailable, nil) }
func (strings *Strings) BusyTooltip() string { return strings.localize(messageTrayBusyTooltip, nil) }
func (strings *Strings) Starting() string    { return strings.localize(messageTrayStarting, nil) }
func (strings *Strings) Stopping() string    { return strings.localize(messageTrayStopping, nil) }
func (strings *Strings) AutostartManageFailed() string {
	return strings.localize(messageTrayAutostartManageFailed, nil)
}
func (strings *Strings) AutostartTitle() string {
	return strings.localize(messageTrayAutostartTitle, nil)
}
func (strings *Strings) AutostartEnableTooltip() string {
	return strings.localize(messageTrayAutostartEnableTooltip, nil)
}
func (strings *Strings) AutostartDisableTooltip() string {
	return strings.localize(messageTrayAutostartDisableTooltip, nil)
}
func (strings *Strings) AutostartApprovalTitle() string {
	return strings.localize(messageTrayAutostartApprovalTitle, nil)
}
func (strings *Strings) AutostartApprovalTooltip() string {
	return strings.localize(messageTrayAutostartApprovalTooltip, nil)
}
func (strings *Strings) AutostartRegisterTooltip() string {
	return strings.localize(messageTrayAutostartRegisterTooltip, nil)
}
func (strings *Strings) AutostartUnsupportedTitle() string {
	return strings.localize(messageTrayAutostartUnsupportedTitle, nil)
}
func (strings *Strings) AutostartUnsupportedTooltip() string {
	return strings.localize(messageTrayAutostartUnsupportedTooltip, nil)
}
func (strings *Strings) ColimaNotFound() string {
	return strings.localize(messageErrorColimaNotFound, nil)
}

func (strings *Strings) LastChecked(checkedAt time.Time) string {
	return strings.localize(messageTrayLastChecked, map[string]any{"Time": strings.FormatTime(checkedAt)})
}

func (strings *Strings) FormatTimestamp(value time.Time) string {
	if strings.isGerman() {
		return value.Format("02.01.2006, 15:04:05")
	}
	return value.Format("Jan 2, 2006, 3:04:05 PM")
}

func (strings *Strings) FormatTime(value time.Time) string {
	if strings.isGerman() {
		return value.Format("15:04:05")
	}
	return value.Format("3:04:05 PM")
}

func (strings *Strings) ProfileRunning(name string) string {
	return strings.localize(messageProfileRunning, map[string]any{"Name": name})
}
func (strings *Strings) ProfileStopped(name string) string {
	return strings.localize(messageProfileStopped, map[string]any{"Name": name})
}
func (strings *Strings) ProfileMissing(name string) string {
	return strings.localize(messageProfileMissing, map[string]any{"Name": name})
}
func (strings *Strings) ProfileBroken(name string) string {
	return strings.localize(messageProfileBroken, map[string]any{"Name": name})
}
func (strings *Strings) ProfileUnknown(name string) string {
	return strings.localize(messageProfileUnknown, map[string]any{"Name": name})
}

func (strings *Strings) isGerman() bool {
	base, _ := strings.language.Base()
	german, _ := language.German.Base()
	return base == german
}
