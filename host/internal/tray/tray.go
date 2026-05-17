// Package tray runs the menu-bar / system-tray icon for the host app.
//
// It owns the main thread on macOS (a Cocoa requirement) and exposes a tiny
// Controller interface that the caller wires to the running application.
package tray

import (
	_ "embed"
	"log"

	"fyne.io/systray"
)

//go:embed icon.png
var iconPNG []byte

// IconBytes returns the PNG bytes of the application icon. Exported so the
// installer package can write a copy to disk without duplicating the asset.
func IconBytes() []byte { return iconPNG }

// Controller is the small surface the tray uses to talk to the rest of the
// app. The concrete implementation lives in cmd/hackintosh.
type Controller interface {
	// Restart drops the current run state and re-execs the binary.
	Restart()
	// Quit cancels the run context and triggers a clean shutdown.
	Quit()
	// OpenSimulator launches the browser at the running simulator URL.
	// Should not be called when SimulatorURL returns "".
	OpenSimulator() error
	// SimulatorURL is the address the in-process virtual OLED is served on,
	// or "" if simulator is not running.
	SimulatorURL() string
}

// menuItem is the subset of *systray.MenuItem behavior the tray uses.
// Defined as an interface so tests can substitute a fake.
type menuItem interface {
	ClickedCh() <-chan struct{}
	Enable()
	Disable()
	Show()
	Hide()
}

type menuItems struct {
	open    menuItem
	restart menuItem
	quit    menuItem
}

// Run starts the tray icon and blocks until the user picks Quit (or the
// controller's Quit is otherwise triggered). MUST be called from the main
// goroutine -- systray.Run is not safe to call elsewhere on macOS.
func Run(c Controller) {
	done := make(chan struct{})
	systray.Run(func() {
		systray.SetIcon(iconPNG)
		systray.SetTitle("")
		systray.SetTooltip("Hackintosh")

		open := wrapItem(systray.AddMenuItem("Open Simulator", "Open the virtual OLED preview in your browser"))
		systray.AddSeparator()
		restart := wrapItem(systray.AddMenuItem("Restart", "Restart the host process"))
		quit := wrapItem(systray.AddMenuItem("Quit", "Stop and exit Hackintosh"))

		items := &menuItems{open: open, restart: restart, quit: quit}
		refresh(c, items)
		go wireMenu(c, items, done)
	}, func() {
		close(done)
	})
}

// systrayQuit is called when the user clicks Quit in the tray menu.
// It is a variable so tests can replace it with a no-op.
var systrayQuit = systray.Quit

// wireMenu listens for clicks on each menu item and dispatches to the
// Controller. Returns when done is closed.
func wireMenu(c Controller, items *menuItems, done chan struct{}) {
	for {
		select {
		case <-done:
			return
		case <-items.open.ClickedCh():
			if err := c.OpenSimulator(); err != nil {
				log.Printf("tray: OpenSimulator failed: %v", err)
			}
		case <-items.restart.ClickedCh():
			c.Restart()
		case <-items.quit.ClickedCh():
			c.Quit()
			systrayQuit()
			return
		}
		refresh(c, items)
	}
}

// refresh updates per-item visibility/enable state based on Controller state.
func refresh(c Controller, items *menuItems) {
	if c.SimulatorURL() == "" {
		items.open.Hide()
	} else {
		items.open.Show()
	}
}

// systrayItem is a small adapter so *systray.MenuItem matches the menuItem
// interface (the systray library returns its own concrete type).
type systrayItem struct {
	inner *systray.MenuItem
}

func (s systrayItem) ClickedCh() <-chan struct{} { return s.inner.ClickedCh }
func (s systrayItem) Enable()                    { s.inner.Enable() }
func (s systrayItem) Disable()                   { s.inner.Disable() }
func (s systrayItem) Show()                      { s.inner.Show() }
func (s systrayItem) Hide()                      { s.inner.Hide() }

func wrapItem(mi *systray.MenuItem) menuItem { return systrayItem{inner: mi} }
