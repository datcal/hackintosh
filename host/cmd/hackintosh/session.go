package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// appSession wraps a context that the tray can cancel, and tracks the
// simulator URL (if any) for the "Open Simulator" menu item.
type appSession struct {
	mu           sync.Mutex
	cancel       context.CancelFunc
	simulatorURL string
	quit         bool
}

// run invokes work with a fresh cancellable context that also responds to
// SIGINT/SIGTERM. Blocks until work returns. Subsequent calls to stop() (from
// any goroutine) cancel the context.
func (s *appSession) run(work func(ctx context.Context) error) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.cancel = nil
		s.mu.Unlock()
		cancel()
	}()

	return work(ctx)
}

// stop cancels the active run AND marks the session as quit so the tray's
// run loop knows not to re-enter run().
func (s *appSession) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quit = true
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *appSession) getQuit() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.quit
}

// setSimulatorURL records the URL the in-process simulator is serving on,
// for the tray's "Open Simulator" menu item. "" disables that item.
func (s *appSession) setSimulatorURL(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.simulatorURL = url
}

func (s *appSession) getSimulatorURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.simulatorURL
}
