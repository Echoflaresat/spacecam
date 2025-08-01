package texture

import (
	"io"
	"math"

	"github.com/echoflaresat/spacecam/base"
	"golang.org/x/exp/mmap"
)

// Texture represents an RGB image with sampling by ECEF position vectors.
type Texture struct {
	TiffHeader
	Data io.ReaderAt
}

// NewTexture constructs a Texture from a raw uint8 slice (H × W × 3).
// Data must be laid out row-major, tightly packed.
func Load(path string) (Texture, error) {
	reader, err := mmap.Open(path)
	if err != nil {
		return Texture{}, err
	}

	header, err := ParseTiffHeader(reader)
	if err != nil {
		return Texture{}, err
	}

	return Texture{
		TiffHeader: header,
		Data:       reader,
	}, nil
}

// Sample maps the 3D vector P (ECEF) to texture coordinates and returns a base.Color4.
// Equivalent to the Python version: lon-lat projection, no interpolation.
func (t Texture) Sample(P base.Vec3) base.Color4 {
	px, py, pz := P.X, P.Y, P.Z

	lat := math.Atan2(pz, math.Sqrt(px*px+py*py))
	lon := math.Atan2(py, px)
	if lon < 0 {
		lon += 2 * math.Pi
	}

	u := float64(t.Width)/2.0 + lon/(2*math.Pi)*float64(t.Width-1)
	u = math.Mod(u, float64(t.Width))
	if u < 0 {
		u += float64(t.Width)
	}
	v := (0.5 - (lat / math.Pi)) * float64(t.Height-1)

	x := int(u)
	y := int(v)

	if x < 0 {
		x = 0
	} else if x >= t.Width {
		x = t.Width - 1
	}
	if y < 0 {
		y = 0
	} else if y >= t.Height {
		y = t.Height - 1
	}

	strip := y / t.RowsPerStrip
	localY := y % t.RowsPerStrip
	idx := t.StripOffsets[strip] + (localY*t.Width+x)*3
	buf := make([]byte, 3)

	_, err := t.Data.ReadAt(buf, int64(idx))
	if err != nil {
		return base.Black() // fallback on out-of-bounds
	}

	r := float64(buf[0]) / 255.0
	g := float64(buf[1]) / 255.0
	b := float64(buf[2]) / 255.0

	return base.Color4{R: r, G: g, B: b, A: 1.0}
}
