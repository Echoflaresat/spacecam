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

func Red() Color4 {
	return Color4{R: 1, G: 0, B: 0, A: 1}
}

func Blue() Color4 {
	return Color4{R: 0, G: 0, B: 1, A: 1}
}

func Green() Color4 {
	return Color4{R: 0, G: 1, B: 0, A: 1}
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

func (c Color4) MulAlpha(a float64) Color4 {
	return Color4{
		R: c.R,
		G: c.G,
		B: c.B,
		A: c.A * a,
	}
}

func (c Color4) WithAlpha(a float64) Color4 {
	return Color4{
		R: c.R,
		G: c.G,
		B: c.B,
		A: a,
	}
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

// MixAlpha returns the mix of c and o with weight t,
// taking o.A (alpha) into account. If o is fully transparent,
// the result is just c. If o is fully opaque, it's a normal
// linear interpolation between c and o.
func (c Color4) MixAlpha(o Color4, t float64) Color4 {
	w := t * o.A // effective weight of o
	return Color4{
		R: c.R*(1-w) + o.R*w,
		G: c.G*(1-w) + o.G*w,
		B: c.B*(1-w) + o.B*w,
		A: c.A*(1-w) + o.A*w,
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

// FilterPreserveBrightness applies a colored filter f with strength t, modulated by f.A,
// while preserving the original perceived brightness (Rec.709 luminance).
// Steps:
//  1. convert to linear light
//  2. multiply by filter transmission (1 → no filter, f → full filter) with weight w = t*f.A
//  3. rescale so output luminance matches the original
//  4. convert back to sRGB
func (c Color4) FilterPreserveBrightness(f Color4, t float64) Color4 {
	w := clamp01(t * clamp01(f.A))
	if w == 0 {
		return c
	}

	// linearize
	cr, cg, cb := srgbToLinear(c.R), srgbToLinear(c.G), srgbToLinear(c.B)
	fr, fg, fb := srgbToLinear(f.R), srgbToLinear(f.G), srgbToLinear(f.B)

	// transmission (interpolate between 1 and filter color in linear light)
	tr := 1 + w*(fr-1)
	tg := 1 + w*(fg-1)
	tb := 1 + w*(fb-1)

	// apply multiplicative filter
	or := cr * tr
	og := cg * tg
	ob := cb * tb

	// preserve luminance (Rec.709 / sRGB primaries, in linear light)
	const rY, gY, bY = 0.2126, 0.7152, 0.0722
	Yin := rY*cr + gY*cg + bY*cb
	Yout := rY*or + gY*og + bY*ob

	// rescale to match original luminance
	scale := 1.0
	if Yout > 1e-12 {
		scale = Yin / Yout
	}
	or *= scale
	og *= scale
	ob *= scale

	// clamp to [0,1] after rescale (optional but practical)
	or = clamp01(or)
	og = clamp01(og)
	ob = clamp01(ob)

	// back to sRGB
	return Color4{
		R: linearToSrgb(or),
		G: linearToSrgb(og),
		B: linearToSrgb(ob),
		A: c.A, // viewing through glass doesn't change subject opacity
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

func lerp(a, b, t float64) float64 { return a*(1-t) + b*t }

// hueLerp interpolates angles (degrees) the short way around the circle.
func hueLerp(h1, h2, t float64) float64 {
	// Normalize to [0,360)
	h1 = math.Mod(h1+360.0, 360.0)
	h2 = math.Mod(h2+360.0, 360.0)
	d := h2 - h1
	// Wrap to [-180,180]
	if d > 180 {
		d -= 360
	} else if d < -180 {
		d += 360
	}
	h := h1 + t*d
	// Normalize result
	h = math.Mod(h+360.0, 360.0)
	return h
}

func rgbToHSV(r, g, b float64) (h, s, v float64) {
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	v = max
	d := max - min

	if max <= 0 {
		// black
		return 0, 0, 0
	}
	if d <= 0 {
		// gray
		return 0, 0, v
	}

	s = d / max

	var hh float64
	switch max {
	case r:
		hh = (g - b) / d
		if g < b {
			hh += 6
		}
	case g:
		hh = (b-r)/d + 2
	case b:
		hh = (r-g)/d + 4
	}
	h = (hh / 6.0) * 360.0
	return
}

func hsvToRGB(h, s, v float64) (r, g, b float64) {
	if s <= 0 {
		return v, v, v
	}
	h = math.Mod(h, 360.0)
	if h < 0 {
		h += 360.0
	}
	h /= 60.0
	i := math.Floor(h)
	f := h - i
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))

	switch int(i) % 6 {
	case 0:
		return v, t, p
	case 1:
		return q, v, p
	case 2:
		return p, v, t
	case 3:
		return p, q, v
	case 4:
		return t, p, v
	default:
		return v, p, q
	}
}

// IEC 61966-2-1 sRGB <-> linear
func srgbToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}
func linearToSrgb(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1.0/2.4) - 0.055
}
