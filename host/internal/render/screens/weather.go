package screens

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/datcal/hackintosh/host/internal/render"
	"github.com/datcal/hackintosh/host/internal/render/anim"
	"github.com/datcal/hackintosh/host/internal/render/font"
	"github.com/datcal/hackintosh/host/internal/render/icons"
	"github.com/datcal/hackintosh/host/internal/store"
)

// Weather screen — temperature, condition, and a condition-driven particle
// system (rain, snow, sun rays, drifting clouds, thunder flash).
type Weather struct {
	t           float64
	temp        *anim.Spring
	wind        *anim.Spring
	rain        *anim.ParticleSystem
	snow        *anim.ParticleSystem
	clouds      [2]float64 // x positions of two clouds
	lastFlash   float64
}

func NewWeather() *Weather {
	w := &Weather{
		temp: anim.NewSpring(0),
		wind: anim.NewSpring(0),
	}
	w.rain = anim.NewParticleSystem(14, 0xCAFE_F00D, func(p *anim.Particle, r *rand.Rand) {
		p.X = float64(r.IntN(render.Width-4)) + 1
		p.Y = float64(render.ContentTop) + r.Float64()*4
		p.VX = -2
		p.VY = 40 + r.Float64()*30
	})
	w.snow = anim.NewParticleSystem(18, 0xC0FF_FEEE, func(p *anim.Particle, r *rand.Rand) {
		p.X = float64(r.IntN(render.Width-4)) + 1
		p.Y = float64(render.ContentTop) + r.Float64()*4
		p.VX = math.Sin(r.Float64()*math.Pi*2) * 6
		p.VY = 14 + r.Float64()*10
	})
	w.clouds[0] = 16
	w.clouds[1] = 70
	return w
}

func (s *Weather) Name() string { return "WEATHER" }

func (s *Weather) Tick(dt float64) {
	s.t += dt
	s.temp.Step(dt)
	s.wind.Step(dt)

	// Always step the particle systems; we choose which to draw in Render.
	s.rain.Step(dt)
	s.snow.Step(dt)

	// Drift clouds
	s.clouds[0] += 4 * dt
	s.clouds[1] += 6.5 * dt
	if s.clouds[0] > render.Width+20 {
		s.clouds[0] = -16
	}
	if s.clouds[1] > render.Width+20 {
		s.clouds[1] = -16
	}
}

func (s *Weather) Render(c *render.Canvas, snap store.Snapshot, _ time.Time) {
	w := snap.Weather
	if w.Valid {
		s.temp.Target = w.TempC
		s.wind.Target = w.WindKMH
	}

	cond := w.Condition

	// --- Ambient particles / decoration (drawn first so they sit behind text) ---
	switch cond {
	case store.CondRain, store.CondThunder:
		drawParticles(c, s.rain.P, icons.Raindrop)
	case store.CondSnow:
		drawParticles(c, s.snow.P, icons.Snowflake)
	case store.CondCloudy, store.CondFog:
		c.DrawBitmap(int(s.clouds[0]), 18, icons.Cloud.Data, icons.Cloud.W, icons.Cloud.H)
		c.DrawBitmap(int(s.clouds[1]), 30, icons.Cloud.Data, icons.Cloud.W, icons.Cloud.H)
	case store.CondSunny:
		drawSun(c, render.Width-22, 28, s.t)
	}

	// Thunder flash every ~6 seconds
	if cond == store.CondThunder {
		if s.t-s.lastFlash > 6.0 {
			s.lastFlash = s.t
		}
		if s.t-s.lastFlash < 0.08 {
			// invert the content area for one frame
			for y := render.ContentTop + 1; y < render.ContentBottom-1; y++ {
				for x := 1; x < render.Width-1; x++ {
					c.Toggle(x, y)
				}
			}
		}
	}

	if !w.Valid {
		msg := "fetching..."
		mw := font.Small.Width(msg)
		font.Small.Draw(setPx(c), (render.Width-mw)/2, 30, msg)
		return
	}

	// --- Big temperature top-left (Medium = 10x14) ---
	tempStr := fmt.Sprintf("%dC", int(math.Round(s.temp.Value)))
	font.Medium.Draw(setPx(c), 4, 14, tempStr)

	// --- Condition + feels-like stacked below ---
	font.Small.Draw(setPx(c), 4, 32, cond.String())
	fl := fmt.Sprintf("feels %d", int(math.Round(w.FeelsLikeC)))
	font.Small.Draw(setPx(c), 4, 42, fl)

	// --- Wind line at the bottom (only if there's a non-zero gust) ---
	wlen := int(math.Min(s.wind.Value*1.5, 36))
	wy := 56
	wx0 := 4
	if wlen >= 4 {
		c.DrawHLine(wx0, wy, wlen)
		c.Set(wx0+wlen-1, wy-1)
		c.Set(wx0+wlen-1, wy+1)
		c.Set(wx0+wlen-2, wy-2)
		c.Set(wx0+wlen-2, wy+2)
	}
	windStr := fmt.Sprintf("%dkmh", int(math.Round(s.wind.Value)))
	textX := wx0
	if wlen >= 4 {
		textX = wx0 + wlen + 5
	}
	font.Small.Draw(setPx(c), textX, wy-3, windStr)

	// --- Not connected fallback ---
	if !w.Valid {
		msg := "fetching..."
		mw := font.Small.Width(msg)
		font.Small.Draw(setPx(c), (render.Width-mw)/2, 30, msg)
	}
}

func drawParticles(c *render.Canvas, parts []anim.Particle, ic icons.Icon) {
	for _, p := range parts {
		if !p.Active {
			continue
		}
		x, y := int(p.X), int(p.Y)
		if y > render.ContentBottom-2 {
			continue
		}
		if y < render.ContentTop+1 {
			continue
		}
		c.DrawBitmap(x, y, ic.Data, ic.W, ic.H)
	}
}

// drawSun draws a small disc with 8 rotating rays.
func drawSun(c *render.Canvas, cx, cy int, t float64) {
	c.DrawBitmap(cx-6, cy-6, icons.SunDisc.Data, icons.SunDisc.W, icons.SunDisc.H)
	const innerR = 8
	const outerR = 12
	rays := 8
	rot := t * (math.Pi * 2 / 15) // ~4 rpm
	for i := 0; i < rays; i++ {
		a := rot + float64(i)*math.Pi*2/float64(rays)
		x0 := cx + int(math.Cos(a)*innerR)
		y0 := cy + int(math.Sin(a)*innerR)
		x1 := cx + int(math.Cos(a)*outerR)
		y1 := cy + int(math.Sin(a)*outerR)
		c.DrawLine(x0, y0, x1, y1)
	}
}
