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

// System monitor — CPU EQ meter, RAM brick fill, disk %, network traffic dots.
type System struct {
	t       float64
	cpu     *anim.Spring
	ram     *anim.Spring
	disk    *anim.Spring
	netUp   *anim.Spring
	netDown *anim.Spring
	dotsUp  []dot
	dotsDn  []dot
}

type dot struct {
	x      float64
	active bool
}

func NewSystem() *System {
	return &System{
		cpu: anim.NewSpring(0), ram: anim.NewSpring(0), disk: anim.NewSpring(0),
		netUp: anim.NewSpring(0), netDown: anim.NewSpring(0),
		dotsUp: make([]dot, 12), dotsDn: make([]dot, 12),
	}
}

func (s *System) Name() string { return "SYSTEM" }

func (s *System) Tick(dt float64) {
	s.t += dt
	s.cpu.Step(dt)
	s.ram.Step(dt)
	s.disk.Step(dt)
	s.netUp.Step(dt)
	s.netDown.Step(dt)

	// Net traffic dots: spawn rate proportional to throughput, move sideways.
	maybeSpawn := func(slot []dot, throughput, dir float64) {
		spawnPerSec := math.Min(throughput/40.0, 8.0) // 1 dot per 40 KB/s, capped
		// each frame, spawn with probability = spawnPerSec*dt
		if math.Sin(s.t*spawnPerSec*math.Pi*2) > 0.6 {
			for i := range slot {
				if !slot[i].active {
					if dir > 0 {
						slot[i].x = 0
					} else {
						slot[i].x = float64(render.Width / 2)
					}
					slot[i].active = true
					break
				}
			}
		}
		for i := range slot {
			if slot[i].active {
				slot[i].x += dir * 20 * dt
				if slot[i].x < 0 || slot[i].x > float64(render.Width/2) {
					slot[i].active = false
				}
			}
		}
	}
	maybeSpawn(s.dotsUp, s.netUp.Value, 1)
	maybeSpawn(s.dotsDn, s.netDown.Value, -1)
}

func (s *System) Render(c *render.Canvas, snap store.Snapshot, _ time.Time) {
	h := snap.HW
	if h.Valid {
		s.cpu.Target = h.CPUPct
		s.ram.Target = h.RAMPct
		s.disk.Target = h.DiskPct
		s.netUp.Target = h.NetUpKBs
		s.netDown.Target = h.NetDownKBs
	}

	// --- Layout: 2 columns. Left: CPU, RAM. Right: DISK, NET. ---
	const yT = render.ContentTop + 2

	// CPU label + EQ meter
	font.Small.Draw(setPx(c), 4, yT, fmt.Sprintf("CPU %d", int(math.Round(s.cpu.Value))))
	drawEQMeter(c, 4, yT+9, 54, 7, s.cpu.Value/100, s.t)

	// RAM label + brick fill
	font.Small.Draw(setPx(c), 4, yT+22, fmt.Sprintf("RAM %d", int(math.Round(s.ram.Value))))
	drawBrickFill(c, 4, yT+31, 54, 6, s.ram.Value/100)

	// DISK label + brick fill (right col)
	font.Small.Draw(setPx(c), 64, yT, fmt.Sprintf("D %d", int(math.Round(s.disk.Value))))
	drawBrickFill(c, 64, yT+9, 58, 6, s.disk.Value/100)

	// NET label + traffic dots
	font.Small.Draw(setPx(c), 64, yT+19,
		fmt.Sprintf("U%d D%d", int(math.Round(s.netUp.Value)), int(math.Round(s.netDown.Value))))
	// up channel
	yUp := yT + 29
	c.DrawHLine(64, yUp, 58)
	for _, d := range s.dotsUp {
		if d.active {
			c.Set(64+int(d.x), yUp)
		}
	}
	// down channel
	yDn := yT + 32
	c.DrawHLine(64, yDn, 58)
	for _, d := range s.dotsDn {
		if d.active {
			c.Set(64+int(d.x), yDn)
		}
	}

	// --- Uptime line at the bottom ---
	upStr := fmt.Sprintf("up %s", fmtUptime(h.Uptime))
	font.Small.Draw(setPx(c), 4, render.ContentBottom-9, upStr)
}

// drawEQMeter renders a horizontal bar whose "floor" is the smoothed value and
// whose top has small dancing peaks (band-limited noise).
func drawEQMeter(c *render.Canvas, x, y, w, h int, frac float64, t float64) {
	if frac < 0 { frac = 0 }
	if frac > 1 { frac = 1 }
	c.DrawRect(x, y, w, h)
	floorW := int(float64(w-2) * frac)
	// solid floor
	c.FillRect(x+1, y+1, floorW, h-2, true)
	// dancing peaks above the floor
	bands := 8
	bandW := (w - 2) / bands
	for i := 0; i < bands; i++ {
		bx := x + 1 + i*bandW
		wob := anim.Wobble(t, 6+float64(i)*0.3, i*17)
		// scale: higher CPU = wilder
		amp := frac * 2.0 // up to 2 px above the floor
		peakY := y + 1 + (h - 2) - floorW
		_ = peakY
		// Compute a vertical strip width — clamp by bandW
		hAbove := int(amp * (0.5 + wob*0.5) * float64(h-2))
		if hAbove < 0 { hAbove = 0 }
		if hAbove > h-2 { hAbove = h - 2 }
		// draw small vertical bar of height hAbove at bx, sitting on the floor
		topY := y + 1 + (h-2) - floorW - hAbove
		for yy := topY; yy < y+1+(h-2)-floorW; yy++ {
			c.Set(bx, yy)
			if bandW > 2 {
				c.Set(bx+1, yy)
			}
		}
	}
}

// drawBrickFill renders a horizontal bar filled in 4-px-wide bricks.
func drawBrickFill(c *render.Canvas, x, y, w, h int, frac float64) {
	if frac < 0 { frac = 0 }
	if frac > 1 { frac = 1 }
	c.DrawRect(x, y, w, h)
	fillW := int(float64(w-2) * frac)
	brick := 4
	for bx := 0; bx < fillW; bx += brick {
		bw := brick - 1
		if bx+bw > fillW {
			bw = fillW - bx
		}
		c.FillRect(x+1+bx, y+1, bw, h-2, true)
	}
}

func fmtUptime(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
