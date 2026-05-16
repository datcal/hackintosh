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

// Currency screen — two rows (EUR->TRY, USD->TRY) with springs, arrow icons
// that bounce, and mini sparklines below each rate.
type Currency struct {
	t       float64
	eur     *anim.Spring
	usd     *anim.Spring
	highlight float64 // 0..1, fades after a fresh fetch
	prevUpd time.Time
}

func NewCurrency() *Currency {
	return &Currency{
		eur: anim.NewSpring(0),
		usd: anim.NewSpring(0),
	}
}

func (s *Currency) Name() string { return "CURRENCY" }

func (s *Currency) Tick(dt float64) {
	s.t += dt
	s.eur.Step(dt)
	s.usd.Step(dt)
	if s.highlight > 0 {
		s.highlight -= dt / 0.3
		if s.highlight < 0 {
			s.highlight = 0
		}
	}
}

func (s *Currency) Render(c *render.Canvas, snap store.Snapshot, _ time.Time) {
	cur := snap.Currency
	if cur.Valid {
		s.eur.Target = cur.EURTRY
		s.usd.Target = cur.USDTRY
		if !cur.UpdatedAt.Equal(s.prevUpd) {
			s.highlight = 1.0
			s.prevUpd = cur.UpdatedAt
		}
	}

	if !cur.Valid {
		msg := "fetching..."
		mw := font.Small.Width(msg)
		font.Small.Draw(setPx(c), (render.Width-mw)/2, 30, msg)
		return
	}

	// Two rows. Each row is 22 px tall: 7 px header + 14 px Medium rate +
	// 1 px gap. Sparkline (when present) is tucked at the right of the rate.
	drawRow(c, "EUR>TRY", s.eur.Value, cur.EURTRYPct, 12, cur.SparkEUR, s.t, s.highlight)
	drawRow(c, "USD>TRY", s.usd.Value, cur.USDTRYPct, 37, cur.SparkUSD, s.t, s.highlight)
}

// Row layout (y = top of header line):
//
//	y..y+6    "EUR>TRY"  +  arrow + "+0.18%" right-aligned
//	y+9..y+22 "53.06" in Medium font
//	y+15..y+22 24-px sparkline tucked to the right of the rate
func drawRow(c *render.Canvas, label string, value, pct float64, y int, spark []float64, t, highlight float64) {
	// --- Header line ---
	font.Small.Draw(setPx(c), 4, y, label)

	// Arrow + pct on the right side of the header
	bounceFreq := 2.0
	if math.Abs(pct) > 0.5 {
		bounceFreq = 0.4 // faster bounce on bigger changes
	}
	bounce := int(anim.EaseInOutSine(anim.Loop(t, bounceFreq)) * 1.5)

	arrow := icons.ArrowUp
	if pct < 0 {
		arrow = icons.ArrowDown
	}
	pctStr := fmt.Sprintf("%+.2f%%", pct)
	pctW := font.Small.Width(pctStr)
	arrowX := render.Width - 4 - pctW - arrow.W - 1
	c.DrawBitmap(arrowX, y-bounce, arrow.Data, arrow.W, arrow.H)
	font.Small.Draw(setPx(c), arrowX+arrow.W+1, y, pctStr)

	// --- Big rate, medium font ---
	rateStr := fmt.Sprintf("%.2f", value)
	font.Medium.Draw(setPx(c), 4, y+9, rateStr)

	// --- Sparkline tucked at top-right of the rate row ---
	rateW := font.Medium.Width(rateStr)
	sparkX := 4 + rateW + 4
	sparkW := render.Width - sparkX - 4
	if sparkW > 10 {
		drawSparkline(c, sparkX, y+15, sparkW, 7, spark)
	}

	// --- Fresh-data highlight wipe along the bottom of the row ---
	if highlight > 0 {
		w := int(float64(render.Width-8) * (1 - highlight))
		highlightY := y + 22
		for x := 4; x < 4+w; x++ {
			c.Toggle(x, highlightY)
		}
	}
}

func drawSparkline(c *render.Canvas, x, y, w, h int, data []float64) {
	if len(data) < 2 {
		return
	}
	// rescale to fit
	mn, mx := data[0], data[0]
	for _, v := range data {
		if v < mn { mn = v }
		if v > mx { mx = v }
	}
	if mx == mn {
		mx = mn + 1
	}
	for i := 1; i < len(data); i++ {
		x0 := x + (i-1)*w/(len(data)-1)
		x1 := x + i*w/(len(data)-1)
		y0 := y + h - 1 - int((data[i-1]-mn)/(mx-mn)*float64(h-1))
		y1 := y + h - 1 - int((data[i]-mn)/(mx-mn)*float64(h-1))
		c.DrawLine(x0, y0, x1, y1)
	}
}
