package render

import (
	"github.com/datcal/hackintosh/host/internal/tea"
	"github.com/datcal/hackintosh/host/internal/render/anim"
	"github.com/datcal/hackintosh/host/internal/render/font"
)

// TitleBarHeight is the height (in pixels) of the striped title bar at the top
// of every screen. Content area is below this.
const (
	TitleBarHeight = 10
	BorderInset    = 0 // 0 means the border is on rows TitleBarHeight..Height-1
	ContentTop     = TitleBarHeight
	ContentBottom  = Height
)

// ChromeState holds animation state for the always-on chrome elements.
type ChromeState struct {
	StripeOffset float64 // pixels, drifts continuously
	ScanlineY    float64 // 0..Height, moves slowly down then resets
	timeSec      float64
}

// NewChromeState returns a fresh chrome animation state.
func NewChromeState() *ChromeState { return &ChromeState{ScanlineY: -10} }

// Tick advances chrome animations.
func (cs *ChromeState) Tick(dt float64) {
	cs.timeSec += dt
	// Stripes drift at ~2 px/s, wrapping every 8 px.
	cs.StripeOffset += 2 * dt
	for cs.StripeOffset >= 8 {
		cs.StripeOffset -= 8
	}
	// Scanline sweep: travels Height+20 pixels over 8 seconds, then teleports
	// off-screen above and loops.
	cs.ScanlineY += float64(Height+20) / 8 * dt
	if cs.ScanlineY > float64(Height+20) {
		cs.ScanlineY = -10
	}
}

// DrawChrome paints the title bar and border.
//
// `title` is the screen name (centered in the title bar).
// `timer` is the current tea-timer snapshot; the right edge of the title bar
//  shows the countdown when the timer is active.
func DrawChrome(c *Canvas, cs *ChromeState, title string, timer tea.Snapshot) {
	// --- Title bar background: diagonal stripes pattern ---
	off := int(cs.StripeOffset)
	for y := 0; y < TitleBarHeight-1; y++ {
		for x := 0; x < Width; x++ {
			if ((x + y + off) & 0x07) < 4 {
				c.Set(x, y)
			}
		}
	}
	// Solid 1-px line under the title bar.
	c.DrawHLine(0, TitleBarHeight-1, Width)

	// --- Title text (inverted: cleared rect with text drawn back in) ---
	tw := font.Small.Width(title)
	textPad := 3
	rectW := tw + textPad*2
	rectX := (Width - rectW) / 2

	// Clear a stripe so text sits on black.
	c.FillRect(rectX, 0, rectW, TitleBarHeight-1, false)
	font.Small.Draw(c.pixelSetter(), rectX+textPad, 1, title)

	// --- Tea-timer countdown in right edge ---
	if timer.Active() {
		var label string
		switch timer.State {
		case tea.Paused:
			label = fmtMMSS(timer.Remaining) + "P"
		case tea.Done:
			label = "TEA"
		default:
			label = fmtMMSS(timer.Remaining)
		}
		lw := font.Small.Width(label)
		px := Width - lw - 3
		c.FillRect(px-1, 0, lw+2, TitleBarHeight-1, false)
		font.Small.Draw(c.pixelSetter(), px, 1, label)
	}

	// --- Content area border ---
	c.DrawRect(0, TitleBarHeight, Width, Height-TitleBarHeight)

	// (scanline removed by request — the diagonal-stripe title bar drift
	// is enough ambient motion for the chrome.)
}

// DrawTimerStrip paints the 1-px progress strip under the title bar while the
// tea timer is running. Draws nothing when idle.
func DrawTimerStrip(c *Canvas, timer tea.Snapshot, t float64) {
	if !timer.Active() {
		return
	}
	y := TitleBarHeight
	w := int(float64(Width-2) * clamp01(timer.Progress))
	// Subtle breathing brightness via dithered density.
	breathing := 0.5 + 0.5*anim.EaseInOutSine(anim.Loop(t, 4))
	density := int(breathing*3) + 1 // 1..4 pixels lit per "block"
	for x := 1; x < 1+w; x++ {
		if (x % 4) < density {
			c.Set(x, y)
		}
	}
}

// DrawTimerDone draws the celebratory "TEA!" overlay shown when the timer
// reaches zero (during the Done state, which auto-dismisses after 10 sec).
func DrawTimerDone(c *Canvas, timer tea.Snapshot, t float64) {
	if timer.State != tea.Done {
		return
	}
	// Big "TEA!" panel centered, over-painting whatever's behind it.
	c.FillRect(8, 18, Width-16, Height-26, false)
	c.DrawRect(8, 18, Width-16, Height-26)

	label := "TEA!"
	w := font.Medium.Width(label)
	font.Medium.Draw(c.pixelSetter(), (Width-w)/2, 26, label)

	// Three bouncing dots beneath the label so the panel feels alive.
	for i := 0; i < 3; i++ {
		base := 50
		phase := anim.Loop(t+float64(i)*0.2, 1.0)
		off := int(4 * anim.EaseInOutSine(phase*2))
		if phase > 0.5 {
			off = int(4 * anim.EaseInOutSine((1-phase)*2))
		}
		cx := (Width / 2) - 8 + i*8
		c.FillCircle(cx, base-off, 1)
	}
}

func fmtMMSS(d interface{ Seconds() float64 }) string {
	total := int(d.Seconds())
	if total < 0 {
		total = 0
	}
	mm := total / 60
	ss := total % 60
	return twoDigit(mm) + ":" + twoDigit(ss)
}

func twoDigit(n int) string {
	if n < 0 {
		n = 0
	}
	if n > 99 {
		n = 99
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

func clamp01(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}
