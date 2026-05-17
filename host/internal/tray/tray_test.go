package tray

import (
	"sync"
	"testing"
	"time"
)

// fakeController records calls so the test can assert on them.
type fakeController struct {
	mu            sync.Mutex
	restartCalled int
	quitCalled    int
	openCalled    int
	openErr       error
	simulatorURL  string
}

func (f *fakeController) Restart() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.restartCalled++
}

func (f *fakeController) Quit() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.quitCalled++
}

func (f *fakeController) OpenSimulator() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.openCalled++
	return f.openErr
}

func (f *fakeController) SimulatorURL() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.simulatorURL
}

// fakeItem captures Enable/Disable/Show/Hide calls and exposes a click chan.
type fakeItem struct {
	enabled bool
	visible bool
	clicks  chan struct{}
}

func newFakeItem() *fakeItem {
	return &fakeItem{enabled: true, visible: true, clicks: make(chan struct{}, 1)}
}

func (i *fakeItem) ClickedCh() <-chan struct{} { return i.clicks }
func (i *fakeItem) Enable()                    { i.enabled = true }
func (i *fakeItem) Disable()                   { i.enabled = false }
func (i *fakeItem) Show()                      { i.visible = true }
func (i *fakeItem) Hide()                      { i.visible = false }

func (i *fakeItem) click() { i.clicks <- struct{}{} }

func TestWireMenuOpenSimulatorCallsController(t *testing.T) {
	c := &fakeController{simulatorURL: "http://localhost:8080"}
	items := &menuItems{open: newFakeItem(), restart: newFakeItem(), quit: newFakeItem()}
	done := make(chan struct{})
	go func() {
		wireMenu(c, items, done)
	}()
	items.open.(*fakeItem).click()
	time.Sleep(20 * time.Millisecond)
	close(done)

	if c.openCalled != 1 {
		t.Fatalf("OpenSimulator: got %d calls, want 1", c.openCalled)
	}
}

func TestWireMenuRestartCallsController(t *testing.T) {
	c := &fakeController{simulatorURL: "http://localhost:8080"}
	items := &menuItems{open: newFakeItem(), restart: newFakeItem(), quit: newFakeItem()}
	done := make(chan struct{})
	go func() {
		wireMenu(c, items, done)
	}()
	items.restart.(*fakeItem).click()
	time.Sleep(20 * time.Millisecond)
	close(done)

	if c.restartCalled != 1 {
		t.Fatalf("Restart: got %d calls, want 1", c.restartCalled)
	}
}

func TestWireMenuQuitCallsController(t *testing.T) {
	c := &fakeController{simulatorURL: "http://localhost:8080"}
	items := &menuItems{open: newFakeItem(), restart: newFakeItem(), quit: newFakeItem()}
	done := make(chan struct{})
	go func() {
		wireMenu(c, items, done)
	}()
	items.quit.(*fakeItem).click()
	time.Sleep(20 * time.Millisecond)
	close(done)

	if c.quitCalled != 1 {
		t.Fatalf("Quit: got %d calls, want 1", c.quitCalled)
	}
}

func TestRefreshHidesOpenSimulatorWhenNoURL(t *testing.T) {
	c := &fakeController{simulatorURL: ""}
	items := &menuItems{open: newFakeItem(), restart: newFakeItem(), quit: newFakeItem()}
	refresh(c, items)
	if items.open.(*fakeItem).visible {
		t.Fatalf("Open Simulator should be hidden when SimulatorURL is empty")
	}
}

func TestRefreshShowsOpenSimulatorWhenURLPresent(t *testing.T) {
	c := &fakeController{simulatorURL: "http://localhost:8080"}
	items := &menuItems{open: newFakeItem(), restart: newFakeItem(), quit: newFakeItem()}
	items.open.(*fakeItem).Hide()
	refresh(c, items)
	if !items.open.(*fakeItem).visible {
		t.Fatalf("Open Simulator should be visible when SimulatorURL is set")
	}
}

// Compile-time check that fakeController satisfies the Controller interface.
var _ Controller = (*fakeController)(nil)

// Compile-time check that fakeItem satisfies the menuItem interface.
var _ menuItem = (*fakeItem)(nil)
