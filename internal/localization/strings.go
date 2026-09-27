package localization

import "time"

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

// FormatTimestamp and FormatTime take their layout from the catalog, so a new
// language is a translation rather than a change to this file.
func (strings *Strings) FormatTimestamp(value time.Time) string {
	return value.Format(strings.localize(messageFormatTimestamp, nil))
}

func (strings *Strings) FormatTime(value time.Time) string {
	return value.Format(strings.localize(messageFormatTime, nil))
}

// There is one method per failure rather than one taking a domain type, so
// that this package stays a provider of strings and does not need to know the
// domain's error taxonomy. The mapping lives in the presentation layer.

// StatusFailed names a failed status read.
func (strings *Strings) StatusFailed() string {
	return strings.localize(messageErrorStatusFailed, nil)
}

// StartFailed names a failed start.
func (strings *Strings) StartFailed() string {
	return strings.localize(messageErrorStartFailed, nil)
}

// StopFailed names a failed stop.
func (strings *Strings) StopFailed() string {
	return strings.localize(messageErrorStopFailed, nil)
}

// Timeout names a command that exceeded its deadline.
func (strings *Strings) Timeout() string { return strings.localize(messageErrorTimeout, nil) }

// UnknownFailure names a failure that fits no other category.
func (strings *Strings) UnknownFailure() string {
	return strings.localize(messageErrorUnknown, nil)
}

func (strings *Strings) DetailHint() string {
	return strings.localize(messageErrorDetailHint, nil)
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

// ProfileUnknownWithStatus names the status Colima reported, so that a value
// this version does not know about still reaches the user instead of being
// flattened to "unknown".
func (strings *Strings) ProfileUnknownWithStatus(name, status string) string {
	return strings.localize(messageProfileUnknownWithStatus, map[string]any{"Name": name, "Status": status})
}

func (strings *Strings) ShowLog() string { return strings.localize(messageTrayShowLog, nil) }
func (strings *Strings) ShowLogTooltip() string {
	return strings.localize(messageTrayShowLogTooltip, nil)
}
func (strings *Strings) ShowConfiguration() string {
	return strings.localize(messageTrayShowConfiguration, nil)
}
func (strings *Strings) ShowConfigurationTooltip() string {
	return strings.localize(messageTrayShowConfigurationTooltip, nil)
}
func (strings *Strings) WatchActive() string { return strings.localize(messageTrayWatchActive, nil) }
func (strings *Strings) WatchFallback() string {
	return strings.localize(messageTrayWatchFallback, nil)
}
