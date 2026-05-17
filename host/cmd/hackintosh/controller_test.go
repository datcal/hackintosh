package main

import (
	"errors"
	"testing"
)

func TestControllerQuitCallsSessionStop(t *testing.T) {
	s := &appSession{}
	stopped := false
	c := &hostController{
		session: s,
		openFn:  func(string) error { return nil },
		restartFn: func() error { return nil },
		quitFn: func() { stopped = true },
	}
	c.Quit()
	if !stopped {
		t.Fatal("Quit did not call quitFn")
	}
}

func TestControllerRestartCallsRestartFn(t *testing.T) {
	called := 0
	c := &hostController{
		session: &appSession{},
		openFn:  func(string) error { return nil },
		restartFn: func() error { called++; return nil },
		quitFn: func() {},
	}
	c.Restart()
	if called != 1 {
		t.Fatalf("Restart: got %d, want 1", called)
	}
}

func TestControllerRestartLogsErrorButDoesNotPanic(t *testing.T) {
	c := &hostController{
		session: &appSession{},
		openFn:  func(string) error { return nil },
		restartFn: func() error { return errors.New("could not spawn") },
		quitFn: func() {},
	}
	// Should not panic. Test passes by not panicking.
	c.Restart()
}

func TestControllerOpenSimulatorUsesURL(t *testing.T) {
	s := &appSession{}
	s.setSimulatorURL("http://localhost:8080")
	var got string
	c := &hostController{
		session: s,
		openFn: func(url string) error { got = url; return nil },
		restartFn: func() error { return nil },
		quitFn: func() {},
	}
	if err := c.OpenSimulator(); err != nil {
		t.Fatal(err)
	}
	if got != "http://localhost:8080" {
		t.Fatalf("OpenSimulator: opened %q, want http://localhost:8080", got)
	}
}

func TestControllerOpenSimulatorReturnsErrorWhenURLEmpty(t *testing.T) {
	c := &hostController{
		session: &appSession{},
		openFn: func(url string) error { return nil },
		restartFn: func() error { return nil },
		quitFn: func() {},
	}
	if err := c.OpenSimulator(); err == nil {
		t.Fatal("expected error when SimulatorURL is empty")
	}
}

func TestControllerSimulatorURLReadsFromSession(t *testing.T) {
	s := &appSession{}
	s.setSimulatorURL("http://localhost:9000")
	c := &hostController{session: s}
	if got := c.SimulatorURL(); got != "http://localhost:9000" {
		t.Fatalf("SimulatorURL: got %q, want http://localhost:9000", got)
	}
}
