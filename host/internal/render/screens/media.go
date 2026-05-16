package screens

import (
	"fmt"
	"math"
	"time"

	"github.com/datcal/hackintosh/host/internal/render"
	"github.com/datcal/hackintosh/host/internal/render/anim"
	"github.com/datcal/hackintosh/host/internal/render/font"
	"github.com/datcal/hackintosh/host/internal/render/icons"
	"github.com/datcal/hackintosh/host/internal/store"
)

// Media — playing title + artist + position + an EQ-visualizer bottom strip,
// or an idle splash when nothing is playing.
type Media struct {
	t           float64
	titleMq     *anim.Marquee
	lastTitle   string
	noteY       float64 // y of the rising note glyph (idle splash)
	noteCycle   float64 // seconds since last note spawn
	peaks       [16]float64 // EQ peak-hold
}

func NewMedia() *Media {
	return &Media{
		titleMq: anim.NewMarquee(0, render.Width-8),
		noteY:   float64(render.ContentBottom),
	}
}

func (s *Media) Name() string { return "MEDIA" }

func (s *Media) Tick(dt float64) {
	s.t += dt
	s.titleMq.Tick(dt)

	s.noteCycle += dt
	if s.noteY > float64(render.ContentTop+8) {
		s.noteY -= 8 * dt
	}
	if s.noteCycle > 10 {
		s.noteCycle = 0
		s.noteY = float64(render.ContentBottom)
	}

	// Peak-hold decay
	for i := range s.peaks {
		s.peaks[i] -= dt * 0.4
		if s.peaks[i] < 0 {
			s.peaks[i] = 0
		}
	}
}

func (s *Media) Render(c *render.Canvas, snap store.Snapshot, _ time.Time) {
	m := snap.Media

	if !m.Valid {
		s.renderIdle(c)
		return
	}

	// Title (marquee if too wide)
	title := "♪ " + m.Title
	if m.Title == "" {
		title = "♪ ..."
	}
	tw := font.Small.Width(title)
	s.titleMq.SetText(tw)
	if title != s.lastTitle {
		s.lastTitle = title
	}
	off := s.titleMq.Offset()
	// Clip drawing to the visible region.
	// We render the string twice when marquee wraps — but simpler: just one copy.
	sub := func(x, y int) {
		if x < 4 || x >= render.Width-4 {
			return
		}
		c.Set(x, y)
	}
	font.Small.Draw(sub, 4+off, render.ContentTop+3, title)

	// Artist
	if m.Artist != "" {
		font.Small.Draw(setPx(c), 4, render.ContentTop+12, truncate(m.Artist, 19))
	}

	// Play/pause + position
	if m.Playing {
		c.DrawBitmap(4, render.ContentTop+23, icons.Play.Data, icons.Play.W, icons.Play.H)
	} else {
		c.DrawBitmap(4, render.ContentTop+23, icons.Pause.Data, icons.Pause.W, icons.Pause.H)
	}

	pos := m.Position
	if m.Playing {
		// Smoothly advance between samples — assume 1s/s drift.
		pos += time.Since(m.UpdatedAt)
		if pos > m.Length && m.Length > 0 {
			pos = m.Length
		}
	}
	posStr := fmt.Sprintf("%s / %s", fmtDuration(pos), fmtDuration(m.Length))
	font.Small.Draw(setPx(c), 16, render.ContentTop+25, posStr)

	// Position bar
	barY := render.ContentTop + 36
	c.DrawHLine(4, barY, render.Width-8)
	if m.Length > 0 {
		w := int(float64(render.Width-10) * float64(pos) / float64(m.Length))
		for x := 4; x < 4+w; x++ {
			c.Set(x, barY-1)
			c.Set(x, barY)
		}
	}

	// EQ visualizer along the bottom
	s.renderEQ(c, m.Playing)
}

func (s *Media) renderIdle(c *render.Canvas) {
	// Three breathing dots in the middle, "No media" below
	pulse := 0.5 + 0.5*anim.EaseInOutSine(anim.Loop(s.t, 1.5))
	dotY := 28
	cxs := []int{render.Width/2 - 8, render.Width / 2, render.Width/2 + 8}
	for _, cx := range cxs {
		if pulse > 0.4 {
			c.FillCircle(cx, dotY, 1)
		}
		if pulse > 0.7 {
			c.FillCircle(cx, dotY, 2)
		}
	}
	msg := "No media"
	w := font.Small.Width(msg)
	font.Small.Draw(setPx(c), (render.Width-w)/2, 42, msg)

	// Rising note glyph
	if int(s.noteY) < render.ContentBottom-1 {
		c.DrawBitmap(render.Width/2-3, int(s.noteY), icons.Note.Data, icons.Note.W, icons.Note.H)
	}
}

func (s *Media) renderEQ(c *render.Canvas, playing bool) {
	bands := 16
	bw := (render.Width - 8) / bands
	baseY := render.ContentBottom - 3
	maxH := 8
	for i := 0; i < bands; i++ {
		x := 4 + i*bw
		var h float64
		if playing {
			h = float64(maxH) * (0.3 + 0.7*math.Abs(anim.Wobble(s.t, 5+float64(i)*0.4, i*23)))
		}
		if h > s.peaks[i] {
			s.peaks[i] = h
		}
		// bar
		for yy := baseY; yy > baseY-int(h); yy-- {
			c.Set(x, yy)
		}
		// peak-hold dot
		px := x
		py := baseY - int(s.peaks[i])
		if py >= render.ContentTop {
			c.Set(px, py)
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return ""
	}
	return s[:max-1] + "."
}

func fmtDuration(d time.Duration) string {
	if d < 0 || d == 0 {
		return "0:00"
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", mins, secs)
}
