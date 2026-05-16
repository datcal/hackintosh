// Package font provides a 5x7 bitmap pixel font with full printable ASCII
// coverage, plus two scaled-up faces built from the same source data.
//
// Glyph encoding (the "Adafruit GLCD" packing):
//   - Each glyph is 5 columns wide x 7 rows tall.
//   - Stored as [5]byte: one byte per column.
//   - In each byte, bit 0 = top pixel of the column, bit 6 = bottom.
//
// All faces share a 6-px advance (5-px glyph + 1-px gap). Medium and Big
// nearest-neighbor scale every pixel into an NxN block.
package font

// Face renders bitmap text via a pixel-set callback. The callback insulates
// us from the underlying canvas — the font just emits coordinates.
type Face struct {
	scale int
}

// Predefined faces — the same 5x7 source, scaled 1x / 2x / 3x.
var (
	Small  = &Face{scale: 1} // 5x7,  advance 6,  one line ~= 7 px
	Medium = &Face{scale: 2} // 10x14, advance 12, one line ~= 14 px
	Big    = &Face{scale: 3} // 15x21, advance 18, one line ~= 21 px
)

// glyphW / glyphH / advance are the base (unscaled) dimensions.
const (
	glyphW  = 5
	glyphH  = 7
	advance = 6
)

// Height returns the pixel height of one line of text in this face.
func (f *Face) Height() int { return glyphH * f.scale }

// Width returns how wide the rendered string will be in pixels.
func (f *Face) Width(s string) int { return len(s) * advance * f.scale }

// Scale returns the integer scale factor (1, 2, 3).
func (f *Face) Scale() int { return f.scale }

// Draw renders s at (x, y) where y is the TOP of the cell (not baseline).
// Returns the x coordinate one pixel past the last glyph.
func (f *Face) Draw(setPixel func(x, y int), x, y int, s string) int {
	for _, r := range s {
		idx := -1
		if r >= 0x20 && r <= 0x7E {
			idx = int(r - 0x20)
		}
		if idx < 0 {
			// Unknown rune — render a small square placeholder.
			for dy := 0; dy < f.scale*2; dy++ {
				for dx := 0; dx < f.scale*2; dx++ {
					setPixel(x+dx, y+dy)
				}
			}
			x += advance * f.scale
			continue
		}
		g := small5x7[idx]
		if f.scale == 1 {
			for col := 0; col < glyphW; col++ {
				colBits := g[col]
				for row := 0; row < glyphH; row++ {
					if colBits&(1<<uint(row)) != 0 {
						setPixel(x+col, y+row)
					}
				}
			}
		} else {
			s := f.scale
			for col := 0; col < glyphW; col++ {
				colBits := g[col]
				for row := 0; row < glyphH; row++ {
					if colBits&(1<<uint(row)) == 0 {
						continue
					}
					for dy := 0; dy < s; dy++ {
						for dx := 0; dx < s; dx++ {
							setPixel(x+col*s+dx, y+row*s+dy)
						}
					}
				}
			}
		}
		x += advance * f.scale
	}
	return x
}
