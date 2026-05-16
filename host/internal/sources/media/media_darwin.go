//go:build darwin

package media

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/datcal/hackintosh/host/internal/store"
)

// fetchNowPlaying shells out to https://github.com/kirtan-shah/nowplaying-cli.
// The tool reads from macOS's MediaRemote.framework — install via:
//
//	brew install nowplaying-cli
//
// We request a specific set of properties to keep parsing simple.
func fetchNowPlaying(ctx context.Context) (store.Media, error) {
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cctx, "nowplaying-cli", "get",
		"title", "artist", "playbackRate", "elapsedTime", "duration")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		// PathError on first run = tool not installed.
		if errors.Is(err, exec.ErrNotFound) {
			return store.Media{Valid: false}, errors.New("nowplaying-cli not installed: brew install nowplaying-cli")
		}
		return store.Media{Valid: false}, nil
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	get := func(i int) string {
		if i >= len(lines) {
			return ""
		}
		return strings.TrimSpace(lines[i])
	}

	title := get(0)
	artist := get(1)
	if title == "null" {
		title = ""
	}
	if artist == "null" {
		artist = ""
	}
	rate, _ := strconv.ParseFloat(get(2), 64)
	elapsed, _ := strconv.ParseFloat(get(3), 64)
	duration, _ := strconv.ParseFloat(get(4), 64)

	if title == "" && artist == "" {
		return store.Media{Valid: false}, nil
	}
	return store.Media{
		Valid:    true,
		Title:    title,
		Artist:   artist,
		Playing:  rate > 0,
		Position: time.Duration(elapsed * float64(time.Second)),
		Length:   time.Duration(duration * float64(time.Second)),
	}, nil
}

func isUnusual(err error) bool {
	return strings.Contains(err.Error(), "not installed")
}
