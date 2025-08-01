package base

import "image/color"

// Color4 is a linear RGBA color with float64 components in [0,1].
// (No gamma conversion; identical semantics to your Python class.)
type Color4 struct {
	R, G, B, A float64
}

func NewColor(r, g, b, a float64) Color4 {
	return Color4{R: r, G: g, B: b, A: a}
}

func White() Color4 {
	return Color4{R: 1, G: 1, B: 1, A: 1}
}

func Black() Color4 {
	return Color4{R: 1, G: 1, B: 1, A: 1}
}

// Add returns c + o (component-wise).
func (c Color4) Add(o Color4) Color4 {
	return Color4{c.R + o.R, c.G + o.G, c.B + o.B, c.A + o.A}
}

// Mul returns c * o (component-wise).
func (c Color4) Mul(o Color4) Color4 {
	return Color4{c.R * o.R, c.G * o.G, c.B * o.B, c.A * o.A}
}

// Scale returns c * s (scalar).
func (c Color4) Scale(s float64) Color4 {
	return Color4{c.R * s, c.G * s, c.B * s, c.A * s}
}

// Mix returns lerp(c, o, t) = c*(1-t) + o*t.
func (c Color4) Mix(o Color4, t float64) Color4 {
	return c.Scale(1.0 - t).Add(o.Scale(t))
}

// Clamp01 clamps each component into [0,1].
func (c Color4) Clamp01() Color4 {
	return Color4{
		R: clamp01(c.R),
		G: clamp01(c.G),
		B: clamp01(c.B),
		A: clamp01(c.A),
	}
}

// To8bitRGB returns (R,G,B) as 0..255 integers (truncates like Python's int()).
func (c Color4) ToNRGBA() color.NRGBA {
	return color.NRGBA{
		to8bit(c.R),
		to8bit(c.G),
		to8bit(c.B),
		to8bit(c.A),
	}
}

// --- helpers ---

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// to8bit mimics Python's: int(255 * clamp01(x)) with truncation toward zero.
func to8bit(x float64) uint8 {
	y := 255.0 * clamp01(x)
	// Truncate like Python's int(); avoid rounding to match semantics.
	if y < 0 {
		y = 0
	}
	if y > 255 {
		y = 255
	}
	return uint8(y)
}
