// Package screens contains the per-screen renderers. Each screen owns its
// own animation state and is invoked once per frame from the app's render
// loop.
package screens

import (
	"time"

	"github.com/datcal/hackintosh/host/internal/render"
	"github.com/datcal/hackintosh/host/internal/store"
)

// Screen is implemented by every renderable screen.
type Screen interface {
	Name() string // chrome title (UPPERCASE, short — fits in ~60 px)
	Tick(dt float64)
	Render(c *render.Canvas, s store.Snapshot, now time.Time)
}

// All returns the cycling order of screens. The first entry is the default
// boot screen.
func All() []Screen {
	return []Screen{
		NewClock(),
		NewWeather(),
		NewAirQuality(),
		NewCurrency(),
		NewSystem(),
		NewMedia(),
		NewPokemon(),
		NewGermanWord(),
		NewGermanMeaning(),
	}
}
