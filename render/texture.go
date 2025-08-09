package render

import (
	"image"
	_ "image/jpeg" // register JPEG format with image.Decode
	_ "image/png"  // register PNG format with image.Decode
	"math"
	"os"

	"github.com/echoflaresat/spacecam/colors"
	"github.com/echoflaresat/spacecam/vectors"
	"github.com/echoflaresat/tiff"
)

// Texture represents an RGB image with sampling by ECEF position vectors.
type Texture struct {
	Width  int
	Height int
	img    image.Image
	file   *os.File
}

func LoadImage(f *os.File) (image.Image, error) {
	img, err := tiff.Decode(f)

	// fallback to image codecs
	if err != nil {
		img, _, err = image.Decode(f)
	}
	return img, err
}

// NewTexture constructs a Texture from a raw uint8 slice (H × W × 3).
// Data must be laid out row-major, tightly packed.
func LoadTexture(path string) (Texture, error) {
	f, err := os.Open(path)
	if err != nil {
		return Texture{}, err
	}

	img, err := tiff.Decode(f)

	// fallback to image codecs
	if err != nil {
		img, _, err = image.Decode(f)
	}

	if err != nil {
		f.Close()
		return Texture{}, err
	}

	return Texture{
		Width:  img.Bounds().Max.X,
		Height: img.Bounds().Max.Y,
		img:    img,
		file:   f,
	}, nil
}

func (t Texture) Close() error {
	if t.file != nil {
		return t.file.Close()
	}
	return nil
}

func (t Texture) Sample4(P vectors.Vec3) (colors.Color4, colors.Color4, colors.Color4, colors.Color4) {
	x, y := t.getXY(P)
	return t.getColorAtXY(x, y),
		t.getColorAtXY(x+1, y),
		t.getColorAtXY(x, y+1),
		t.getColorAtXY(x+1, y+1)
}

// Sample maps the 3D vector P (ECEF) to texture coordinates and returns a color.Color4.
// Equivalent to the Python version: lon-lat projection, no interpolation.
func (t Texture) Sample(P vectors.Vec3) colors.Color4 {
	return t.getColorAtXY(t.getXY(P))
}

func (t Texture) getColorAtXY(x, y int) colors.Color4 {
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

	c := t.img.At(x, y)
	return colors.FromStandardColor(c)
}

func (t Texture) getXY(P vectors.Vec3) (int, int) {
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
	return x, y
}
