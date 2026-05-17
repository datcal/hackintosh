package screens

import (
	"strings"
	"time"

	"github.com/datcal/hackintosh/host/internal/render"
	"github.com/datcal/hackintosh/host/internal/render/font"
	"github.com/datcal/hackintosh/host/internal/store"
)

// PokemonScreen shows today's daily Pokémon with its sprite, name, and types.
type PokemonScreen struct{ t float64 }

func NewPokemon() *PokemonScreen { return &PokemonScreen{} }

func (s *PokemonScreen) Name() string    { return "POKEMON" }
func (s *PokemonScreen) Tick(dt float64) { s.t += dt }

func (s *PokemonScreen) Render(c *render.Canvas, snap store.Snapshot, _ time.Time) {
	p := snap.Pokemon
	if !p.Valid {
		msg := "fetching..."
		font.Small.Draw(setPx(c), (render.Width-font.Small.Width(msg))/2, 30, msg)
		return
	}

	// Sprite: left side, starting at (2, 11), max 44×44.
	if len(p.Sprite) > 0 {
		c.DrawBitmap(2, 11, p.Sprite, p.SpriteW, p.SpriteH)
	}

	// Name: Medium font right of sprite. Truncate to 6 chars (72px ≤ 76px available).
	name := truncate(p.Name, 6)
	font.Medium.Draw(setPx(c), 50, 14, name)

	// Type badges with outline box.
	if p.Type1 != "" {
		drawTypeBadge(c, 50, 31, strings.ToUpper(p.Type1))
	}
	if p.Type2 != "" {
		drawTypeBadge(c, 50, 43, strings.ToUpper(p.Type2))
	}
}

// drawTypeBadge draws a type label inside a 1px outline box.
func drawTypeBadge(c *render.Canvas, x, y int, label string) {
	w := font.Small.Width(label) + 4
	h := font.Small.Height() + 2
	c.DrawRect(x, y, w, h)
	font.Small.Draw(setPx(c), x+2, y+1, label)
}
