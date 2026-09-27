package tray

import "fyne.io/systray"

// Menu and Item are the slice of the menu bar library this app actually uses.
//
// They exist so that the render logic — which state enables which action, what
// each row says — can be tested. systray talks to a real menu bar and has no
// test mode, so without this seam the part most likely to break when Colima
// reports something new is the part with no coverage.
type Menu interface {
	SetTemplateIcon(icon []byte)
	SetTitle(title string)
	SetTooltip(tooltip string)
	SetRemovalAllowed(allowed bool)
	AddItem(title, tooltip string) Item
	AddCheckbox(title, tooltip string, checked bool) Item
	AddSeparator()
	Quit()
}

// Item is one row. Clicked is a receive-only channel so that an implementation
// is free to close it, which is how systray signals that the menu went away.
type Item interface {
	SetTitle(title string)
	SetTooltip(tooltip string)
	Enable()
	Disable()
	Show()
	Hide()
	Check()
	Uncheck()
	Clicked() <-chan struct{}
}

// SystrayMenu drives the real menu bar.
type SystrayMenu struct{}

func (SystrayMenu) SetTemplateIcon(icon []byte) {
	// systray takes a light and a dark variant; a template icon is tinted by
	// macOS, so the same bytes serve both.
	systray.SetTemplateIcon(icon, icon)
}

func (SystrayMenu) SetTitle(title string) { systray.SetTitle(title) }

func (SystrayMenu) SetTooltip(tooltip string) { systray.SetTooltip(tooltip) }

func (SystrayMenu) SetRemovalAllowed(allowed bool) { systray.SetRemovalAllowed(allowed) }

func (SystrayMenu) AddItem(title, tooltip string) Item {
	return systrayItem{item: systray.AddMenuItem(title, tooltip)}
}

func (SystrayMenu) AddCheckbox(title, tooltip string, checked bool) Item {
	return systrayItem{item: systray.AddMenuItemCheckbox(title, tooltip, checked)}
}

func (SystrayMenu) AddSeparator() { systray.AddSeparator() }

func (SystrayMenu) Quit() { systray.Quit() }

type systrayItem struct {
	item *systray.MenuItem
}

func (wrapper systrayItem) SetTitle(title string) { wrapper.item.SetTitle(title) }

func (wrapper systrayItem) SetTooltip(tooltip string) { wrapper.item.SetTooltip(tooltip) }

func (wrapper systrayItem) Enable() { wrapper.item.Enable() }

func (wrapper systrayItem) Disable() { wrapper.item.Disable() }

func (wrapper systrayItem) Show() { wrapper.item.Show() }

func (wrapper systrayItem) Hide() { wrapper.item.Hide() }

func (wrapper systrayItem) Check() { wrapper.item.Check() }

func (wrapper systrayItem) Uncheck() { wrapper.item.Uncheck() }

func (wrapper systrayItem) Clicked() <-chan struct{} { return wrapper.item.ClickedCh }
