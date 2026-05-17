package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSessionStopCancelsContext(t *testing.T) {
	s := &appSession{}
	stopped := make(chan struct{})
	go func() {
		s.run(func(ctx context.Context) error {
			<-ctx.Done()
			close(stopped)
			return ctx.Err()
		})
	}()

	// Give the goroutine time to enter run().
	time.Sleep(20 * time.Millisecond)
	s.stop()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("session.stop() did not cancel the context in time")
	}
}

func TestSessionRunReturnsCallbackError(t *testing.T) {
	s := &appSession{}
	want := errors.New("boom")
	got := s.run(func(ctx context.Context) error { return want })
	if !errors.Is(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSessionStopMarksQuit(t *testing.T) {
	s := &appSession{}
	if s.getQuit() {
		t.Fatal("fresh session should not be marked quit")
	}
	s.stop()
	if !s.getQuit() {
		t.Fatal("after stop(), session should be marked quit")
	}
}
