package screens

import (
	"fmt"
	"math"
	"time"

	"github.com/datcal/hackintosh/host/internal/render"
	"github.com/datcal/hackintosh/host/internal/render/anim"
	"github.com/datcal/hackintosh/host/internal/render/font"
	"github.com/datcal/hackintosh/host/internal/store"
)

// Clock screen — big HH:MM with a swinging pendulum and pulsing colon.
type Clock struct {
	t           float64
	pendulumX   float64 // x offset of the pendulum bob from its center
	lastMinute  int
	flipPhase   float64 // 0..1 during a flip animation; 0 = idle
	ripplePhase float64 // 0..1 — rings expand outward on each second
	lastSecond  int
}

func NewClock() *Clock { return &Clock{lastMinute: -1, lastSecond: -1} }

func (s *Clock) Name() string { return "CLOCK" }

func (s *Clock) Tick(dt float64) {
	s.t += dt

	// Pendulum: sinusoid 1 Hz with amplitude 6 px
	s.pendulumX = 6 * math.Sin(2*math.Pi*1.0*s.t)

	if s.flipPhase > 0 {
		s.flipPhase -= dt / 0.2 // 200 ms
		if s.flipPhase < 0 {
			s.flipPhase = 0
		}
	}
	if s.ripplePhase > 0 {
		s.ripplePhase -= dt / 0.6 // 600 ms ring lifetime
		if s.ripplePhase < 0 {
			s.ripplePhase = 0
		}
	}
}

func (s *Clock) Render(c *render.Canvas, _ store.Snapshot, now time.Time) {
	hh := now.Hour()
	mm := now.Minute()
	ss := now.Second()

	// Only trigger animations on actual changes, not on the first render
	// (when last* is the sentinel -1).
	if mm != s.lastMinute {
		if s.lastMinute >= 0 {
			s.flipPhase = 1.0
		}
		s.lastMinute = mm
	}
	if ss != s.lastSecond {
		if s.lastSecond >= 0 {
			s.ripplePhase = 1.0
		}
		s.lastSecond = ss
	}

	// Big HH:MM centered (15x21 per char + 3 px gap = 18 px advance,
	// "HH:MM" = 5 chars × 18 = 90 px wide).
	timeStr := fmt.Sprintf("%02d:%02d", hh, mm)
	tw := font.Big.Width(timeStr)
	tx := (render.Width - tw) / 2
	ty := 14

	if s.flipPhase > 0 {
		// During the flip, draw the MM digits shifted up easing back to rest.
		hhPrefix := fmt.Sprintf("%02d:", hh)
		font.Big.Draw(setPx(c), tx, ty, hhPrefix)
		yOff := int(10 * (1 - anim.EaseOutCubic(1-s.flipPhase)))
		font.Big.Draw(setPx(c), tx+font.Big.Width(hhPrefix), ty-yOff, fmt.Sprintf("%02d", mm))
	} else {
		font.Big.Draw(setPx(c), tx, ty, timeStr)
	}

	// Pulsing colon halo — small cross of dots above and below the colon
	// glyphs that brightens with the second tick.
	colonX := tx + font.Big.Width(fmt.Sprintf("%02d", hh)) + 6
	colonY := ty + 10
	pulse := anim.EaseInOutSine(anim.Loop(s.t, 1.0))
	if pulse > 0.5 {
		c.Set(colonX-3, colonY-1)
		c.Set(colonX-3, colonY+1)
		c.Set(colonX+5, colonY-1)
		c.Set(colonX+5, colonY+1)
	}

	// Ripple rings emanating from the colon on each second tick.
	if s.ripplePhase > 0 {
		r := int(4 + (1-s.ripplePhase)*22)
		drawRingDotted(c, colonX, colonY, r, int(8*s.ripplePhase))
	}

	// Date line: "Sat May 16"
	dayStr := now.Format("Mon Jan _2")
	dw := font.Small.Width(dayStr)
	font.Small.Draw(setPx(c), (render.Width-dw)/2, 42, dayStr)

	// Pendulum — small dot at (cx + sin*amp, cy), with a 1px line to anchor.
	pxCx := render.Width / 2
	pyAnchor := 52
	pxBob := pxCx + int(s.pendulumX)
	pyBob := pyAnchor + 7
	c.DrawLine(pxCx, pyAnchor, pxBob, pyBob)
	c.FillCircle(pxBob, pyBob, 1)
}

// drawRingDotted draws a ring at (cx,cy) radius r where only some pixels light
// — used for fading ring animations.
func drawRingDotted(c *render.Canvas, cx, cy, r, density int) {
	if density < 1 {
		return
	}
	steps := 32
	for i := 0; i < steps; i++ {
		if i%(steps/density+1) != 0 {
			continue
		}
		a := float64(i) / float64(steps) * 2 * math.Pi
		c.Set(cx+int(math.Cos(a)*float64(r)), cy+int(math.Sin(a)*float64(r)))
	}
}

// helper so screens don't repeat the closure boilerplate
func setPx(c *render.Canvas) func(int, int) {
	return func(x, y int) { c.Set(x, y) }
}
