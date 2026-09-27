package tray

import "sync"

// fakeMenu records what the app did to the menu, so that render decisions can
// be asserted instead of looked at.
type fakeMenu struct {
	mutex          sync.Mutex
	title, tooltip string
	icon           []byte
	removalAllowed bool
	items          []*fakeItem
	separators     int
	quitCalls      int
}

func (menu *fakeMenu) SetTemplateIcon(icon []byte) {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	menu.icon = icon
}

func (menu *fakeMenu) SetTitle(title string) {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	menu.title = title
}

func (menu *fakeMenu) SetTooltip(tooltip string) {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	menu.tooltip = tooltip
}

func (menu *fakeMenu) SetRemovalAllowed(allowed bool) {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	menu.removalAllowed = allowed
}

func (menu *fakeMenu) AddItem(title, tooltip string) Item {
	return menu.add(title, tooltip, false)
}

func (menu *fakeMenu) AddCheckbox(title, tooltip string, checked bool) Item {
	return menu.add(title, tooltip, checked)
}

func (menu *fakeMenu) add(title, tooltip string, checked bool) Item {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	item := &fakeItem{
		title:   title,
		tooltip: tooltip,
		checked: checked,
		enabled: true,
		visible: true,
		clicked: make(chan struct{}),
	}
	menu.items = append(menu.items, item)
	return item
}

func (menu *fakeMenu) AddSeparator() {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	menu.separators++
}

func (menu *fakeMenu) Quit() {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	menu.quitCalls++
}

type fakeItem struct {
	mutex          sync.Mutex
	title, tooltip string
	enabled        bool
	visible        bool
	checked        bool
	clicked        chan struct{}
}

func (item *fakeItem) SetTitle(title string) {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.title = title
}

func (item *fakeItem) SetTooltip(tooltip string) {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.tooltip = tooltip
}

func (item *fakeItem) Enable()  { item.set(func() { item.enabled = true }) }
func (item *fakeItem) Disable() { item.set(func() { item.enabled = false }) }
func (item *fakeItem) Show()    { item.set(func() { item.visible = true }) }
func (item *fakeItem) Hide()    { item.set(func() { item.visible = false }) }
func (item *fakeItem) Check()   { item.set(func() { item.checked = true }) }
func (item *fakeItem) Uncheck() { item.set(func() { item.checked = false }) }

func (item *fakeItem) Clicked() <-chan struct{} { return item.clicked }

func (item *fakeItem) set(change func()) {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	change()
}

func (item *fakeItem) state() (title string, enabled, visible, checked bool) {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	return item.title, item.enabled, item.visible, item.checked
}
