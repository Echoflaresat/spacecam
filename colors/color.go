package colors

import (
	"image/color"
	"math"
)

// Color4 is a linear RGBA color with float64 components in [0,1].
type Color4 struct {
	R, G, B, A float64
}

func New(r, g, b, a float64) Color4 {
	return Color4{R: r, G: g, B: b, A: a}
}

func (c Color4) RGBA() (r, g, b, a uint32) {
	// Clamp to [0.0, 1.0]
	clamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}

	rf := clamp(c.R)
	gf := clamp(c.G)
	bf := clamp(c.B)
	af := clamp(c.A)

	// Convert to pre-multiplied 16-bit values
	return uint32(rf * af * 65535),
		uint32(gf * af * 65535),
		uint32(bf * af * 65535),
		uint32(af * 65535)
}

func FromStandardColor(c color.Color) Color4 {
	// Fast path: already a Color4
	if c4, ok := c.(Color4); ok {
		return c4
	}

	r16, g16, b16, a16 := c.RGBA()
	if a16 == 0 {
		return Color4{R: 0, G: 0, B: 0, A: 0}
	}

	// De-premultiply and normalize to [0,1]
	invA := float64(0xFFFF) / float64(a16)
	return Color4{
		R: float64(r16) * invA / 65535.0,
		G: float64(g16) * invA / 65535.0,
		B: float64(b16) * invA / 65535.0,
		A: float64(a16) / 65535.0,
	}
}

func From8BitRgb(r, g, b, a byte) Color4 {
	return Color4{
		R: float64(r) / 255.0,
		G: float64(g) / 255.0,
		B: float64(b) / 255.0,
		A: float64(a) / 255.0,
	}
}

func White() Color4 {
	return Color4{R: 1, G: 1, B: 1, A: 1}
}

func Black() Color4 {
	return Color4{R: 0, G: 0, B: 0, A: 1}
}

// Add returns c + o (component-wise).
func (c Color4) Add(o Color4) Color4 {
	return Color4{c.R + o.R, c.G + o.G, c.B + o.B, c.A + o.A}
}

// Mul returns c * o (component-wise).
func (c Color4) Mul(o Color4) Color4 {
	return Color4{c.R * o.R, c.G * o.G, c.B * o.B, c.A * o.A}
}

func (c Color4) BoostSaturation(factor float64) Color4 {
	avg := (c.R + c.G + c.B) / 3
	return Color4{
		R: avg + (c.R-avg)*factor,
		G: avg + (c.G-avg)*factor,
		B: avg + (c.B-avg)*factor,
		A: c.A,
	}
}

func (c Color4) CompositeOverBlack() Color4 {
	return Color4{c.R * c.A, c.G * c.A, c.B * c.A, 1.0}
}

func (c Color4) Pow(gamma float64) Color4 {
	return Color4{
		R: math.Pow(c.R, gamma),
		G: math.Pow(c.G, gamma),
		B: math.Pow(c.B, gamma),
		A: c.A, // leave alpha untouched
	}
}

// Scale returns c * s (scalar).
func (c Color4) Scale(s float64) Color4 {
	return Color4{c.R * s, c.G * s, c.B * s, c.A * s}
}

// Mix returns lerp(c, o, t) = c*(1-t) + o*t.
func (c Color4) Mix(o Color4, t float64) Color4 {
	return Color4{
		R: c.R*(1-t) + o.R*t,
		G: c.G*(1-t) + o.G*t,
		B: c.B*(1-t) + o.B*t,
		A: c.A*(1-t) + o.A*t,
	}
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
