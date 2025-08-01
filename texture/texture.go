package texture

import (
	"errors"
	"image"
	"log/slog"
	"math"
	"os"

	"github.com/echoflaresat/spacecam/base"
	"github.com/echoflaresat/spacecam/texture/tiff"

	_ "image/jpeg" // register JPEG format with image.Decode
	_ "image/png"  // register PNG format with image.Decode
)

// Texture represents an RGB image with sampling by ECEF position vectors.
type Texture struct {
	Width  int
	Height int
	img    image.Image
}

// NewTexture constructs a Texture from a raw uint8 slice (H × W × 3).
// Data must be laid out row-major, tightly packed.
func Load(path string) (Texture, error) {
	img, err := loadImage(path)
	if err != nil {
		return Texture{}, err
	}

	return Texture{
		Width:  img.Bounds().Max.X,
		Height: img.Bounds().Max.Y,
		img:    img,
	}, nil
}

func loadImage(path string) (image.Image, error) {
	img, err := tiff.LoadStripedTiff(path)
	if err == nil {
		return img, nil
	}
	if !errors.Is(err, tiff.ErrInvalidTiffHeader) {
		slog.Warn("failed to load striped TIFF", "path", path, "error", err)
	}

	img, err = tiff.LoadTiledTiff(path)
	if err == nil {
		return img, nil
	}
	if !errors.Is(err, tiff.ErrInvalidTiffHeader) {
		slog.Warn("failed to load tiled TIFF", "path", path, "error", err)
	}

	// fallback to image codecs
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err = image.Decode(f)
	return img, err

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

	c := t.img.At(x, y)
	return base.Color4FromStandardColor(c)
}
