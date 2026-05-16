// Package render builds 1-bit 128x64 framebuffers in the SSD1306-native
// "page-major" layout so the byte buffer can be shipped to the MCU without
// any transposition.
//
// Each byte represents 8 vertical pixels in a single column of a "page".
// Pages are 8 rows tall; there are 8 pages stacked vertically.
//
//	offset(col, row) = (row / 8) * 128 + col
//	bit(row)         = row % 8        (bit 0 = top pixel of the page)
package render

import "github.com/datcal/hackintosh/host/internal/render/font"

const (
	Width      = 128
	Height     = 64
	Pages      = Height / 8
	FrameBytes = Width * Pages // 1024
)

// Canvas is a 1-bit framebuffer matching the SSD1306 layout.
type Canvas struct {
	Buf [FrameBytes]byte
}

// New returns a freshly-zeroed canvas (all pixels off).
func New() *Canvas { return &Canvas{} }

// Clear turns every pixel off.
func (c *Canvas) Clear() {
	for i := range c.Buf {
		c.Buf[i] = 0
	}
}

// Bytes returns the underlying frame as a fresh slice ready to ship to the MCU.
func (c *Canvas) Bytes() []byte {
	out := make([]byte, FrameBytes)
	copy(out, c.Buf[:])
	return out
}

// Inside reports whether the pixel is within the screen.
func (c *Canvas) Inside(x, y int) bool {
	return x >= 0 && x < Width && y >= 0 && y < Height
}

// Set turns a single pixel on.
func (c *Canvas) Set(x, y int) {
	if !c.Inside(x, y) {
		return
	}
	c.Buf[(y/8)*Width+x] |= 1 << uint(y%8)
}

// Unset turns a single pixel off.
func (c *Canvas) Unset(x, y int) {
	if !c.Inside(x, y) {
		return
	}
	c.Buf[(y/8)*Width+x] &^= 1 << uint(y%8)
}

// Toggle flips a single pixel.
func (c *Canvas) Toggle(x, y int) {
	if !c.Inside(x, y) {
		return
	}
	c.Buf[(y/8)*Width+x] ^= 1 << uint(y%8)
}

// Put writes the pixel to `on` (true = lit).
func (c *Canvas) Put(x, y int, on bool) {
	if on {
		c.Set(x, y)
	} else {
		c.Unset(x, y)
	}
}

// Get returns the current state of one pixel.
func (c *Canvas) Get(x, y int) bool {
	if !c.Inside(x, y) {
		return false
	}
	return c.Buf[(y/8)*Width+x]&(1<<uint(y%8)) != 0
}

// FillRect fills a rectangle (inclusive top-left, exclusive bottom-right).
func (c *Canvas) FillRect(x, y, w, h int, on bool) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.Put(xx, yy, on)
		}
	}
}

// DrawRect strokes a rectangle outline 1px wide.
func (c *Canvas) DrawRect(x, y, w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	c.DrawHLine(x, y, w)
	c.DrawHLine(x, y+h-1, w)
	c.DrawVLine(x, y, h)
	c.DrawVLine(x+w-1, y, h)
}

// DrawHLine draws a horizontal line.
func (c *Canvas) DrawHLine(x, y, w int) {
	for i := 0; i < w; i++ {
		c.Set(x+i, y)
	}
}

// DrawVLine draws a vertical line.
func (c *Canvas) DrawVLine(x, y, h int) {
	for i := 0; i < h; i++ {
		c.Set(x, y+i)
	}
}

// DrawLine — Bresenham.
func (c *Canvas) DrawLine(x0, y0, x1, y1 int) {
	dx := abs(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -abs(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		c.Set(x0, y0)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// DrawCircle — midpoint algorithm, outline only.
func (c *Canvas) DrawCircle(cx, cy, r int) {
	if r <= 0 {
		return
	}
	x := r
	y := 0
	err := 1 - r
	for x >= y {
		c.Set(cx+x, cy+y)
		c.Set(cx-x, cy+y)
		c.Set(cx+x, cy-y)
		c.Set(cx-x, cy-y)
		c.Set(cx+y, cy+x)
		c.Set(cx-y, cy+x)
		c.Set(cx+y, cy-x)
		c.Set(cx-y, cy-x)
		y++
		if err < 0 {
			err += 2*y + 1
		} else {
			x--
			err += 2*(y-x) + 1
		}
	}
}

// FillCircle — filled disc.
func (c *Canvas) FillCircle(cx, cy, r int) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				c.Set(cx+x, cy+y)
			}
		}
	}
}

// DrawBitmap draws an 8-bits-per-byte horizontally-packed bitmap. Each row of
// the bitmap is ceil(w/8) bytes, MSB-first; bit set = pixel on.
func (c *Canvas) DrawBitmap(x, y int, bmp []byte, w, h int) {
	rowBytes := (w + 7) / 8
	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			b := bmp[row*rowBytes+col/8]
			if b&(0x80>>uint(col%8)) != 0 {
				c.Set(x+col, y+row)
			}
		}
	}
}

// DrawText writes a string using the small built-in pixel font.
// Returns the x coordinate one pixel past the last drawn glyph.
func (c *Canvas) DrawText(x, y int, s string, f *font.Face) int {
	return f.Draw(c.pixelSetter(), x, y, s)
}

// TextWidth measures a string in the given font.
func (c *Canvas) TextWidth(s string, f *font.Face) int { return f.Width(s) }

// pixelSetter returns a closure matching the font.PixelSetter signature.
func (c *Canvas) pixelSetter() func(x, y int) {
	return func(x, y int) { c.Set(x, y) }
}

// Sub returns a writable view of a rectangular region. Pixels drawn outside
// the region are clipped. Useful for "draw inside this window".
type Sub struct {
	c             *Canvas
	x, y, w, h    int
}

func (c *Canvas) SubRect(x, y, w, h int) *Sub { return &Sub{c, x, y, w, h} }

func (s *Sub) Set(x, y int) {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return
	}
	s.c.Set(s.x+x, s.y+y)
}

// Origin returns absolute coords of (0,0) inside the sub-region.
func (s *Sub) Origin() (int, int) { return s.x, s.y }

// Size returns the sub-region dimensions.
func (s *Sub) Size() (int, int) { return s.w, s.h }

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
