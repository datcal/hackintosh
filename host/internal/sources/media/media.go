// Package media polls the OS's "now playing" state on a 2-second cadence.
// Windows uses an inline PowerShell snippet that calls the WinRT
// GlobalSystemMediaTransportControlsSessionManager. macOS shells out to the
// open-source `nowplaying-cli` tool if installed.
package media

import (
	"context"
	"log"
	"time"

	"github.com/datcal/hackintosh/host/internal/store"
)

// Worker periodically polls the OS and updates the media slot of the store.
type Worker struct {
	S      *store.Store
	Period time.Duration // default 2s
}

func (w *Worker) Run(ctx context.Context) {
	if w.Period == 0 {
		w.Period = 2 * time.Second
	}
	t := time.NewTicker(w.Period)
	defer t.Stop()

	w.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	m, err := fetchNowPlaying(ctx)
	if err != nil {
		// Most "errors" here are just "no app is currently providing media".
		// Don't spam logs — just publish an empty state.
		w.S.SetMedia(store.Media{Valid: false, UpdatedAt: time.Now()})
		// Log only unusual failures (e.g. tool not installed).
		if isUnusual(err) {
			log.Printf("media: %v", err)
		}
		return
	}
	m.UpdatedAt = time.Now()
	w.S.SetMedia(m)
}
