package screens

import (
	"time"

	"github.com/datcal/hackintosh/host/internal/render"
	"github.com/datcal/hackintosh/host/internal/render/font"
	"github.com/datcal/hackintosh/host/internal/store"
)

// GermanMeaningScreen shows the Turkish and English translations of today's word.
type GermanMeaningScreen struct{ t float64 }

func NewGermanMeaning() *GermanMeaningScreen { return &GermanMeaningScreen{} }

func (s *GermanMeaningScreen) Name() string    { return "ANLAM" }
func (s *GermanMeaningScreen) Tick(dt float64) { s.t += dt }

func (s *GermanMeaningScreen) Render(c *render.Canvas, snap store.Snapshot, _ time.Time) {
	g := snap.GermanWord
	if !g.Valid {
		msg := "fetching..."
		font.Small.Draw(setPx(c), (render.Width-font.Small.Width(msg))/2, 30, msg)
		return
	}

	// German word header (ASCII-folded so font can render it)
	font.Small.Draw(setPx(c), 4, 12, truncate(asciiFold(g.German), 20))

	// Separator
	c.DrawHLine(2, 22, render.Width-4)

	// Turkish meaning (ASCII-folded)
	trLine := truncate("TR: "+asciiFold(g.Turkish), 20)
	font.Small.Draw(setPx(c), 4, 27, trLine)

	// English meaning
	enLine := truncate("EN: "+g.English, 20)
	font.Small.Draw(setPx(c), 4, 38, enLine)
}
