package localization

import "github.com/nicksnyder/go-i18n/v2/i18n"

var (
	messageTrayTooltip                     = &i18n.Message{ID: "Tray.Tooltip", Description: "Tooltip for the ColimaStatus menu bar icon.", Other: "ColimaStatus – Colima status"}
	messageTrayChecking                    = &i18n.Message{ID: "Tray.Checking", Description: "Status shown while Colima is being checked.", Other: "Checking Colima status …"}
	messageTrayCurrentStatusTooltip        = &i18n.Message{ID: "Tray.CurrentStatusTooltip", Description: "Tooltip for the current Colima status row.", Other: "Current Colima status"}
	messageTrayProfileDetailsTooltip       = &i18n.Message{ID: "Tray.ProfileDetailsTooltip", Description: "Tooltip for the Colima profile configuration row.", Other: "Colima configuration"}
	messageTrayLastCheckTooltip            = &i18n.Message{ID: "Tray.LastCheckTooltip", Description: "Tooltip for the time of the last Colima status check.", Other: "Time of the last status check"}
	messageTrayStart                       = &i18n.Message{ID: "Tray.Start", Description: "Menu action that starts Colima.", Other: "Start"}
	messageTrayStartTooltip                = &i18n.Message{ID: "Tray.StartTooltip", Description: "Tooltip for the action that starts Colima.", Other: "Start Colima"}
	messageTrayStop                        = &i18n.Message{ID: "Tray.Stop", Description: "Menu action that stops Colima.", Other: "Stop"}
	messageTrayStopTooltip                 = &i18n.Message{ID: "Tray.StopTooltip", Description: "Tooltip for the action that stops Colima.", Other: "Stop Colima"}
	messageTrayForceStop                   = &i18n.Message{ID: "Tray.ForceStop", Description: "Menu action that force stops a broken Colima profile.", Other: "Force stop"}
	messageTrayRefresh                     = &i18n.Message{ID: "Tray.Refresh", Description: "Menu action that checks Colima immediately.", Other: "Refresh status"}
	messageTrayRefreshTooltip              = &i18n.Message{ID: "Tray.RefreshTooltip", Description: "Tooltip for the immediate Colima status check action.", Other: "Check Colima status now"}
	messageTrayOpenLoginItems              = &i18n.Message{ID: "Tray.OpenLoginItems", Description: "Menu action that opens the macOS Login Items settings.", Other: "Open Login Items in System Settings …"}
	messageTrayOpenLoginItemsTooltip       = &i18n.Message{ID: "Tray.OpenLoginItemsTooltip", Description: "Tooltip for opening Login Items settings to approve ColimaStatus.", Other: "Allow ColimaStatus to launch at login in macOS"}
	messageTrayQuit                        = &i18n.Message{ID: "Tray.Quit", Description: "Menu action that quits ColimaStatus.", Other: "Quit"}
	messageTrayQuitTooltip                 = &i18n.Message{ID: "Tray.QuitTooltip", Description: "Tooltip for the quit action.", Other: "Quit ColimaStatus"}
	messageTrayUnavailableTooltip          = &i18n.Message{ID: "Tray.UnavailableTooltip", Description: "Menu bar tooltip when the Colima status is unavailable.", Other: "ColimaStatus – Error"}
	messageTrayUnavailable                 = &i18n.Message{ID: "Tray.Unavailable", Description: "Status shown when the Colima status is unavailable.", Other: "Colima status unavailable"}
	messageTrayBusyTooltip                 = &i18n.Message{ID: "Tray.BusyTooltip", Description: "Menu bar tooltip while an operation is running.", Other: "ColimaStatus – Operation in progress"}
	messageTrayStarting                    = &i18n.Message{ID: "Tray.Starting", Description: "Status shown while Colima is starting.", Other: "Starting Colima …"}
	messageTrayStopping                    = &i18n.Message{ID: "Tray.Stopping", Description: "Status shown while Colima is stopping.", Other: "Stopping Colima …"}
	messageTrayLastChecked                 = &i18n.Message{ID: "Tray.LastChecked", Description: "Status containing the localized time of the last Colima check.", Other: "Last checked: {{.Time}}"}
	messageTrayAutostartManageFailed       = &i18n.Message{ID: "Tray.AutostartManageFailed", Description: "Status shown when launch at login could not be changed.", Other: "Launch at login could not be changed"}
	messageTrayAutostartTitle              = &i18n.Message{ID: "Tray.AutostartTitle", Description: "Menu checkbox for launching ColimaStatus when the user logs in.", Other: "Launch at login"}
	messageTrayAutostartEnableTooltip      = &i18n.Message{ID: "Tray.AutostartEnableTooltip", Description: "Tooltip when launch at login can be enabled.", Other: "Launch ColimaStatus automatically after login"}
	messageTrayAutostartDisableTooltip     = &i18n.Message{ID: "Tray.AutostartDisableTooltip", Description: "Tooltip when launch at login can be disabled.", Other: "Disable launch at login for ColimaStatus"}
	messageTrayAutostartApprovalTitle      = &i18n.Message{ID: "Tray.AutostartApprovalTitle", Description: "Launch at login menu title when macOS approval is required.", Other: "Launch at login (approval required)"}
	messageTrayAutostartApprovalTooltip    = &i18n.Message{ID: "Tray.AutostartApprovalTooltip", Description: "Tooltip explaining that launch at login must be approved in macOS.", Other: "Allow ColimaStatus in macOS System Settings"}
	messageTrayAutostartRegisterTooltip    = &i18n.Message{ID: "Tray.AutostartRegisterTooltip", Description: "Tooltip when ColimaStatus can be registered as a login item.", Other: "Register ColimaStatus as a login item"}
	messageTrayAutostartUnsupportedTitle   = &i18n.Message{ID: "Tray.AutostartUnsupportedTitle", Description: "Launch at login menu title on unsupported macOS versions.", Other: "Launch at login (macOS 13 or later)"}
	messageTrayAutostartUnsupportedTooltip = &i18n.Message{ID: "Tray.AutostartUnsupportedTooltip", Description: "Tooltip explaining the minimum macOS version for launch at login.", Other: "This feature requires macOS 13 or later"}
	messageProfileRunning                  = &i18n.Message{ID: "Profile.Running", Description: "Status for a running Colima profile.", Other: "Colima is running ({{.Name}})"}
	messageProfileStopped                  = &i18n.Message{ID: "Profile.Stopped", Description: "Status for a stopped Colima profile.", Other: "Colima is stopped ({{.Name}})"}
	messageProfileMissing                  = &i18n.Message{ID: "Profile.Missing", Description: "Status for a Colima profile that has not been created.", Other: "Colima is not set up yet ({{.Name}})"}
	messageProfileBroken                   = &i18n.Message{ID: "Profile.Broken", Description: "Status for a broken Colima profile.", Other: "Colima is broken ({{.Name}})"}
	messageProfileUnknown                  = &i18n.Message{ID: "Profile.Unknown", Description: "Status for an unknown Colima profile state.", Other: "Unknown Colima status ({{.Name}})"}
	messageErrorColimaNotFound             = &i18n.Message{ID: "Error.ColimaNotFound", Description: "Error shown when no Colima executable can be found.", Other: "Colima was not found"}
)
