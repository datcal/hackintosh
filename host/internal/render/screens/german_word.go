package screens

import (
	"fmt"
	"strings"
	"time"

	"github.com/datcal/hackintosh/host/internal/render"
	"github.com/datcal/hackintosh/host/internal/render/font"
	"github.com/datcal/hackintosh/host/internal/store"
)

// GermanWordScreen shows today's German word large and centered.
type GermanWordScreen struct{ t float64 }

func NewGermanWord() *GermanWordScreen { return &GermanWordScreen{} }

func (s *GermanWordScreen) Name() string    { return "WORT" }
func (s *GermanWordScreen) Tick(dt float64) { s.t += dt }

func (s *GermanWordScreen) Render(c *render.Canvas, snap store.Snapshot, _ time.Time) {
	g := snap.GermanWord
	if !g.Valid {
		msg := "fetching..."
		font.Small.Draw(setPx(c), (render.Width-font.Small.Width(msg))/2, 30, msg)
		return
	}

	// Header row: "WORT" left, "#N" right
	font.Small.Draw(setPx(c), 4, 12, "WORT")
	dayLabel := fmt.Sprintf("#%d", g.DayIdx+1)
	font.Small.Draw(setPx(c), render.Width-font.Small.Width(dayLabel)-4, 12, dayLabel)

	// Separator
	c.DrawHLine(2, 22, render.Width-4)

	// German word centered — Medium if it fits, Small otherwise
	word := asciiFold(g.German)
	if font.Medium.Width(word) <= render.Width-8 {
		x := (render.Width - font.Medium.Width(word)) / 2
		font.Medium.Draw(setPx(c), x, 29, word)
	} else {
		x := (render.Width - font.Small.Width(word)) / 2
		font.Small.Draw(setPx(c), x, 33, word)
	}
}

// asciiFold replaces umlauts and diacritics with ASCII equivalents so the
// 0x20–0x7E bitmap font can render them without showing placeholder squares.
func asciiFold(s string) string {
	return strings.NewReplacer(
		"ä", "a", "Ä", "A",
		"ö", "o", "Ö", "O",
		"ü", "u", "Ü", "U",
		"ß", "ss",
		"ş", "s", "Ş", "S",
		"ğ", "g", "Ğ", "G",
		"ı", "i", "İ", "I",
		"ç", "c", "Ç", "C",
	).Replace(s)
}

