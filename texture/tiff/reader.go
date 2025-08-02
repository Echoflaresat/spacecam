package tiff

import (
	"image"
	"io"
	"sync"

	stdtiff "golang.org/x/image/tiff"
)

const littleEndianHeader = "II\x2A\x00"
const bigEndianHeader = "MM\x00\x2A"

// DecodeConfig returns the color model and dimensions of a TIFF image without decoding the entire image.
func DecodeConfig(r io.Reader) (image.Config, error) {
	return stdtiff.DecodeConfig(r)
}

// Decode reads a TIFF image from r and returns it as an image.Image.
func Decode(r io.Reader) (image.Image, error) {
	var readerAt io.ReaderAt

	if ra, ok := r.(io.ReaderAt); ok {
		readerAt = ra
	} else if rs, ok := r.(io.ReadSeeker); ok {
		readerAt = &readerAtFromSeeker{rs: rs}
	}

	if readerAt != nil {
		if img, err := LoadStripedTiff(readerAt); err == nil {
			return img, nil
		}
		if img, err := LoadTiledTiff(readerAt); err == nil {
			return img, nil
		}
	}

	// Fallback to standard decoder
	return stdtiff.Decode(r)
}

type readerAtFromSeeker struct {
	rs io.ReadSeeker
	mu sync.Mutex
}

func (r *readerAtFromSeeker) ReadAt(p []byte, off int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.rs.Seek(off, io.SeekStart); err != nil {
		return 0, err
	}
	return r.rs.Read(p)
}

func init() {
	image.RegisterFormat("tiff", littleEndianHeader, Decode, DecodeConfig)
	image.RegisterFormat("tiff", bigEndianHeader, Decode, DecodeConfig)
}
