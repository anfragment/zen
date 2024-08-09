package systray

/*
	This file contains code from the systray project (https://github.com/getlantern/systray), licensed under the Apache License.
	See more in the COPYING.md file in the root directory of this project.
*/

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
)

var (
	systrayReady  func()
	systrayExit   func()
	menuItems     = make(map[uint32]*menuItem)
	menuItemsLock sync.RWMutex

	currentID = uint32(0)
	quitOnce  sync.Once
)

func init() {
	runtime.LockOSThread()
}

// menuItem is used to keep track each menu item of systray.
// Don't create it directly, use the one systray.AddMenuItem() returned
type menuItem struct {
	// ClickedCh is the channel which will be notified when the menu item is clicked
	ClickedCh chan struct{}

	// id uniquely identify a menu item, not supposed to be modified
	id uint32
	// title is the text shown on menu item
	title string
	// tooltip is the text shown when pointing to menu item
	tooltip string
	// disabled menu item is grayed out and has no effect when clicked
	disabled bool
	// checked menu item has a tick before the title
	checked bool
	// has the menu item a checkbox (Linux)
	isCheckable bool
	// parent item, for sub menus
	parent *menuItem
}

func (item *menuItem) String() string {
	if item.parent == nil {
		return fmt.Sprintf("MenuItem[%d, %q]", item.id, item.title)
	}
	return fmt.Sprintf("MenuItem[%d, parent %d, %q]", item.id, item.parent.id, item.title)
}

// newMenuItem returns a populated MenuItem object
func newMenuItem(title string, tooltip string, parent *menuItem) *menuItem {
	return &menuItem{
		ClickedCh:   make(chan struct{}),
		id:          atomic.AddUint32(&currentID, 1),
		title:       title,
		tooltip:     tooltip,
		disabled:    false,
		checked:     false,
		isCheckable: false,
		parent:      parent,
	}
}

// run initializes GUI and starts the event loop, then invokes the onReady callback. It blocks until
// systray.Quit() is called. It must be run from the main thread on macOS.
func run(onReady func(), onExit func()) {
	if onReady == nil {
		systrayReady = func() {}
	} else {
		// Run onReady on separate goroutine to avoid blocking event loop
		readyCh := make(chan interface{})
		go func() {
			<-readyCh
			onReady()
		}()
		systrayReady = func() {
			close(readyCh)
		}
	}
	// unlike onReady, onExit runs in the event loop to make sure it has time to
	// finish before the process terminates
	if onExit == nil {
		onExit = func() {}
	}
	systrayExit = onExit
	registerSystray()
	nativeLoop()
}

// setTooltip sets the systray tooltip to display on mouse hover of the tray icon.
func setTooltip(tooltip string) {
	if err := wt.setTooltip(tooltip); err != nil {
		log.Printf("Unable to set tooltip: %v", err)
		return
	}
}

// setIcon sets the systray icon.
// iconBytes should be the content of a .ico file.
func setIcon(iconBytes []byte) {
	iconFilePath, err := iconBytesToFilePath(iconBytes)
	if err != nil {
		log.Printf("Unable to write icon data to temp file: %v", err)
		return
	}
	if err := wt.setIcon(iconFilePath); err != nil {
		log.Printf("Unable to set icon: %v", err)
		return
	}
}

// quit the systray. This can be called from any goroutine.
func quit() {
	quitOnce.Do(quitInternal)
}

// addMenuItem adds a menu item with the designated title and tooltip.
// It can be safely invoked from different goroutines.
// Created menu items are checkable on Windows and OSX by default. For Linux you have to use AddMenuItemCheckbox
func addMenuItem(title string, tooltip string) *menuItem {
	item := newMenuItem(title, tooltip, nil)
	item.update()
	return item
}

// addSeparator adds a separator bar to the menu
func addSeparator() {
	id := atomic.AddUint32(&currentID, 1)
	err := wt.addSeparatorMenuItem(id, 0)
	if err != nil {
		log.Printf("Unable to addSeparator: %v", err)
		return
	}
}

// AddSubMenuItem adds a nested sub-menu item with the designated title and tooltip.
// It can be safely invoked from different goroutines.
// Created menu items are checkable on Windows and OSX by default. For Linux you have to use AddSubMenuItemCheckbox
func (item *menuItem) AddSubMenuItem(title string, tooltip string) *menuItem {
	child := newMenuItem(title, tooltip, item)
	child.update()
	return child
}

// AddSubMenuItemCheckbox adds a nested sub-menu item with the designated title and tooltip and a checkbox for Linux.
// It can be safely invoked from different goroutines.
// On Windows and OSX this is the same as calling AddSubMenuItem
func (item *menuItem) AddSubMenuItemCheckbox(title string, tooltip string, checked bool) *menuItem {
	child := newMenuItem(title, tooltip, item)
	child.isCheckable = true
	child.checked = checked
	child.update()
	return child
}

// SetTitle set the text to display on a menu item
func (item *menuItem) SetTitle(title string) {
	item.title = title
	item.update()
}

// SetTooltip set the tooltip to show when mouse hover
func (item *menuItem) SetTooltip(tooltip string) {
	item.tooltip = tooltip
	item.update()
}

// Disabled checks if the menu item is disabled
func (item *menuItem) Disabled() bool {
	return item.disabled
}

// Enable a menu item regardless if it's previously enabled or not
func (item *menuItem) Enable() {
	item.disabled = false
	item.update()
}

// Disable a menu item regardless if it's previously disabled or not
func (item *menuItem) Disable() {
	item.disabled = true
	item.update()
}

// Hide hides a menu item
func (item *menuItem) Hide() {
	hideMenuItem(item)
}

// Show shows a previously hidden menu item
func (item *menuItem) Show() {
	showMenuItem(item)
}

// Checked returns if the menu item has a check mark
func (item *menuItem) Checked() bool {
	return item.checked
}

// Check a menu item regardless if it's previously checked or not
func (item *menuItem) Check() {
	item.checked = true
	item.update()
}

// Uncheck a menu item regardless if it's previously unchecked or not
func (item *menuItem) Uncheck() {
	item.checked = false
	item.update()
}

// update propagates changes on a menu item to systray
func (item *menuItem) update() {
	menuItemsLock.Lock()
	menuItems[item.id] = item
	menuItemsLock.Unlock()
	addOrUpdateMenuItem(item)
}

func systrayMenuItemSelected(id uint32) {
	menuItemsLock.RLock()
	item, ok := menuItems[id]
	menuItemsLock.RUnlock()
	if !ok {
		log.Printf("no menu item with ID %v", id)
		return
	}
	select {
	case item.ClickedCh <- struct{}{}:
	// in case no one waiting for the channel
	default:
	}
}
