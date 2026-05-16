//go:build !windows && !darwin

package media

import (
	"context"

	"github.com/datcal/hackintosh/host/internal/store"
)

// Unsupported OS — always returns "no media" so the screen falls back to its
// idle splash.
func fetchNowPlaying(_ context.Context) (store.Media, error) {
	return store.Media{Valid: false}, nil
}

func isUnusual(error) bool { return false }
