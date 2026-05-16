package render

import "github.com/datcal/hackintosh/host/internal/render/anim"

// Transition composites two screens during a slide-up animation when the user
// presses button B. The outgoing screen slides up, the incoming slides up
// from below — both at the same offset.
//
// Duration is 250 ms with ease-out-cubic.
const TransitionDuration = 0.25 // seconds

// CompositeSlideUp paints `out` and `in` into `dst` at progress t∈[0,1].
// At t=0 the outgoing fills the screen; at t=1 the incoming fills the screen.
func CompositeSlideUp(dst, out, in *Canvas, t float64) {
	t = anim.EaseOutCubic(t)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	contentH := ContentBottom - ContentTop
	offset := int(float64(contentH) * t)

	// Copy chrome from `in` (so the title bar animation flows continuously
	// with the new screen's name).
	for y := 0; y < ContentTop; y++ {
		for x := 0; x < Width; x++ {
			dst.Put(x, y, in.Get(x, y))
		}
	}

	// Compose content: outgoing shifted up by `offset`, incoming entering
	// from below.
	for y := ContentTop; y < ContentBottom; y++ {
		yOut := y + offset
		yIn := y - (contentH - offset)
		for x := 0; x < Width; x++ {
			var px bool
			if yOut < ContentBottom {
				px = out.Get(x, yOut)
			} else if yIn >= ContentTop {
				px = in.Get(x, yIn)
			}
			dst.Put(x, y, px)
		}
	}

	// Restore the border (chrome already drew its top edge above).
	dst.DrawRect(0, TitleBarHeight, Width, Height-TitleBarHeight)
}
