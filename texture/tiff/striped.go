package tiff

import (
	"fmt"
	"image"
	"image/color"
	"io"

	"github.com/echoflaresat/spacecam/texture/tiff/compression"
	"github.com/echoflaresat/spacecam/texture/tiff/photometric"
)

type stripedTiff struct {
	header TiffHeader
	reader io.ReaderAt
}

func LoadStripedTiff(reader io.ReaderAt) (image.Image, error) {

	header, err := parseTiffHeader(reader)
	if err != nil {
		return nil, err
	}

	if header.Compression != compression.None {
		return nil, fmt.Errorf("unsupported compression: %d", header.Compression)
	}
	if header.Photometric != photometric.RGB && header.Photometric != photometric.BlackIsZero {
		return nil, fmt.Errorf("expected RGB photometric interpretation, got %d", header.Photometric)
	}

	switch header.Photometric {
	case photometric.BlackIsZero:
		if header.SamplesPerPixel != 1 || header.BitsPerSample[0] != 8 {
			return nil, fmt.Errorf("unsupported grayscale format")
		}
	case photometric.RGB:
		if header.SamplesPerPixel != 3 || header.BitsPerSample[0] != 8 {
			return nil, fmt.Errorf("unsupported RGB format")
		}
	default:
		return nil, fmt.Errorf("unsupported photometric: %d", header.Photometric)
	}

	if len(header.StripOffsets) == 0 || len(header.StripOffsets) != len(header.StripByteCounts) {
		return nil, fmt.Errorf("invalid strip offset/length")
	}

	return &stripedTiff{header: header, reader: reader}, nil
}

func (t *stripedTiff) ColorModel() color.Model {
	return color.RGBAModel
}

func (t *stripedTiff) Bounds() image.Rectangle {
	return image.Rect(0, 0, t.header.Width, t.header.Height)
}

func (t *stripedTiff) At(x, y int) color.Color {
	h := t.header

	strip := y / h.RowsPerStrip
	localY := y % h.RowsPerStrip
	bytesPerPixel := h.SamplesPerPixel

	idx := h.StripOffsets[strip] + (localY*h.Width+x)*bytesPerPixel

	switch h.Photometric {
	case photometric.RGB:
		var buf [3]byte
		_, err := t.reader.ReadAt(buf[:], int64(idx))
		if err != nil {
			panic(fmt.Sprintf("could not read RGB pixel at (%d,%d): %v", x, y, err))
		}
		return color.RGBA{
			R: buf[0],
			G: buf[1],
			B: buf[2],
			A: 255,
		}

	case photometric.BlackIsZero:
		var b [1]byte
		_, err := t.reader.ReadAt(b[:], int64(idx))
		if err != nil {
			panic(fmt.Sprintf("could not read grayscale pixel at (%d,%d): %v", x, y, err))
		}
		return color.RGBA{
			R: b[0],
			G: b[0],
			B: b[0],
			A: 255,
		}
	default:
		panic(fmt.Sprintf("unsupported PhotometricInterpretation: %d", h.Photometric))
	}
}
