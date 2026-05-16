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

// AirQuality screen — big AQI band, PM2.5/PM10, sway-animated leaf, and a
// drifting haze pattern when AQI is poor.
type AirQuality struct {
	t      float64
	pm25   *anim.Spring
	pm10   *anim.Spring
	fill   float64 // 0..1 reveal of the 5-bar indicator
	haze   *rand.Rand
}

func NewAirQuality() *AirQuality {
	return &AirQuality{
		pm25: anim.NewSpring(0),
		pm10: anim.NewSpring(0),
		haze: rand.New(rand.NewPCG(0xABCD, 0x1234)),
	}
}

func (s *AirQuality) Name() string { return "AIR" }

func (s *AirQuality) Tick(dt float64) {
	s.t += dt
	s.pm25.Step(dt)
	s.pm10.Step(dt)
	if s.fill < 1 {
		s.fill += dt / 0.3 // reveal over 300ms
		if s.fill > 1 {
			s.fill = 1
		}
	}
}

func (s *AirQuality) Render(c *render.Canvas, snap store.Snapshot, _ time.Time) {
	a := snap.AQ
	if a.Valid {
		s.pm25.Target = a.PM25
		s.pm10.Target = a.PM10
	}

	// --- Drifting haze for poor air quality (drawn behind everything) ---
	if a.Valid && a.AQI >= 4 {
		off := int(s.t * 6)
		for y := render.ContentTop + 1; y < render.ContentBottom-1; y++ {
			for x := 1; x < render.Width-1; x++ {
				if ((x+off+y*7)*131%29) < 3 {
					c.Set(x, y)
				}
			}
		}
	}

	// --- Big AQI number top-left (Big = 15x21) ---
	aqiStr := "-"
	if a.Valid {
		aqiStr = fmt.Sprintf("%d", a.AQI)
	}
	font.Big.Draw(setPx(c), 6, 14, aqiStr)

	// --- Label below big number ---
	lbl := a.Label()
	font.Small.Draw(setPx(c), 4, 40, lbl)

	// --- 5-bar indicator middle, between number and PM values ---
	bx, by := 32, 16
	bw, bh, bgap := 4, 22, 2
	level := 0
	if a.Valid {
		level = a.AQI
	}
	revealCount := int(s.fill * 5)
	if revealCount > level {
		revealCount = level
	}
	for i := 0; i < 5; i++ {
		x := bx + i*(bw+bgap)
		h := bh - (4-i)*3
		y := by + (bh - h)
		c.DrawRect(x, y, bw, h)
		if i < revealCount {
			c.FillRect(x+1, y+1, bw-2, h-2, true)
		}
	}

	// --- PM values right side stacked ---
	pmStr := fmt.Sprintf("PM2.5 %d", int(math.Round(s.pm25.Value)))
	font.Small.Draw(setPx(c), 66, 14, pmStr)
	pm10Str := fmt.Sprintf("PM10  %d", int(math.Round(s.pm10.Value)))
	font.Small.Draw(setPx(c), 66, 24, pm10Str)

	// --- Sway-animated leaf bottom-right corner ---
	swayAmp := 1 + float64(level)*0.8
	swayX := int(swayAmp * math.Sin(s.t*2.0))
	c.DrawBitmap(render.Width-13+swayX, 38, icons.Leaf.Data, icons.Leaf.W, icons.Leaf.H)
}
