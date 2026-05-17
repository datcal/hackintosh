package main

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/datcal/hackintosh/host/internal/openbrowser"
)

// hostController implements tray.Controller for the running host app.
// openFn / restartFn / quitFn are injected for testability -- production
// uses openbrowser.Open, reExec, and session.stop respectively.
type hostController struct {
	session   *appSession
	openFn    func(url string) error
	restartFn func() error
	quitFn    func()
}

func newHostController(s *appSession) *hostController {
	return &hostController{
		session:   s,
		openFn:    openbrowser.Open,
		restartFn: reExec,
		quitFn:    s.stop,
	}
}

func (c *hostController) Quit() { c.quitFn() }

func (c *hostController) Restart() {
	// 1. Capture the done channel before cancelling so we can wait on it.
	done := c.session.doneCh()
	// 2. Cancel the active run so deferred cleanup (e.g. dev.Close()) executes.
	//    os.Exit later would skip those defers entirely.
	c.session.stop()
	// 3. Wait for the session to actually return, or give up after a short
	//    grace period if something is stuck.
	if done != nil {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
	// 4. Spawn the new process and exit ours.
	if err := c.restartFn(); err != nil {
		log.Printf("tray: restart failed (keeping current process running): %v", err)
	}
}

func (c *hostController) OpenSimulator() error {
	url := c.session.getSimulatorURL()
	if url == "" {
		return errors.New("simulator is not running")
	}
	return c.openFn(url)
}

func (c *hostController) SimulatorURL() string {
	return c.session.getSimulatorURL()
}

// reExec spawns a fresh copy of the running binary with the same arguments,
// then exits the current process. Used by the tray's Restart menu item.
func reExec() error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil // unreachable
}
