package geom

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/echoflaresat/spacecam/colors"
	"github.com/echoflaresat/spacecam/vectors"
	"golang.org/x/exp/mmap"
)

// Texture represents an RGB image with sampling by ECEF position vectors.
type Texture struct {
	TiffHeader
	Data io.ReaderAt
}

type TiffHeader struct {
	ByteOrder       binary.ByteOrder
	Width, Height   int
	RowsPerStrip    int
	StripOffsets    []int
	StripByteCounts []int
	BitsPerSample   []int
	SamplesPerPixel int
	Photometric     int
	Compression     int
	PlanarConfig    int
}

// NewTexture constructs a Texture from a raw uint8 slice (H × W × 3).
// Data must be laid out row-major, tightly packed.
func LoadTexture(path string) (Texture, error) {
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

// https://www.loc.gov/preservation/digital/formats/content/tiff_tags.shtml
const (
	TagImageWidth                = 256
	TagImageLength               = 257
	TagBitsPerSample             = 258
	TagCompression               = 259
	TagPhotometricInterpretation = 262
	TagStripOffsets              = 273
	TagSamplesPerPixel           = 277
	TagStripByteCounts           = 279
)

func ParseTiffHeader(reader io.ReaderAt) (TiffHeader, error) {
	read := func(offset int64, size int) ([]byte, error) {
		buf := make([]byte, size)
		_, err := reader.ReadAt(buf, offset)
		return buf, err
	}

	// Read 8-byte header
	header, err := read(0, 8)
	if err != nil {
		return TiffHeader{}, err
	}

	var bo binary.ByteOrder
	switch string(header[0:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return TiffHeader{}, fmt.Errorf("invalid byte order")
	}
	if bo.Uint16(header[2:4]) != 42 {
		return TiffHeader{}, fmt.Errorf("invalid TIFF magic number")
	}
	ifdOffset := int64(bo.Uint32(header[4:8]))

	// Read number of entries
	entryCountRaw, err := read(ifdOffset, 2)
	if err != nil {
		return TiffHeader{}, err
	}
	numEntries := int(bo.Uint16(entryCountRaw))
	entriesRaw, err := read(ifdOffset+2, numEntries*12)
	if err != nil {
		return TiffHeader{}, err
	}

	hdr := TiffHeader{
		ByteOrder:       bo,
		BitsPerSample:   nil,
		SamplesPerPixel: -1,
		Photometric:     -1,
		Compression:     -1,
		PlanarConfig:    1, // default
	}

	for i := 0; i < numEntries; i++ {
		entry := entriesRaw[i*12 : (i+1)*12]
		tag := bo.Uint16(entry[0:2])
		// typ := bo.Uint16(entry[2:4])
		count := bo.Uint32(entry[4:8])
		valOffset := int64(bo.Uint32(entry[8:12]))

		readShortArray := func() ([]int, error) {
			if count == 1 {
				return []int{int(bo.Uint16(entry[8:10]))}, nil
			}
			buf, err := read(valOffset, int(count*2))
			if err != nil {
				return nil, err
			}
			out := make([]int, count)
			for i := uint32(0); i < count; i++ {
				out[i] = int(bo.Uint16(buf[i*2:]))
			}
			return out, nil
		}
		readLongArray := func() ([]int, error) {
			if count == 1 {
				return []int{int(valOffset)}, nil
			}
			buf, err := read(valOffset, int(count*4))
			if err != nil {
				return nil, err
			}
			out := make([]int, count)
			for i := uint32(0); i < count; i++ {
				out[i] = int(bo.Uint32(buf[i*4:]))
			}
			return out, nil
		}

		switch tag {
		case 256: // ImageWidth
			hdr.Width = int(valOffset)
		case 257: // ImageLength
			hdr.Height = int(valOffset)
		case 258: // BitsPerSample
			hdr.BitsPerSample, err = readShortArray()
			if err != nil {
				return TiffHeader{}, err
			}
		case 259: // Compression
			hdr.Compression = int(bo.Uint16(entry[8:10]))
		case 262: // PhotometricInterpretation
			hdr.Photometric = int(bo.Uint16(entry[8:10]))
		case 273: // StripOffsets
			hdr.StripOffsets, err = readLongArray()
			if err != nil {
				return TiffHeader{}, err
			}
		case 277: // SamplesPerPixel
			hdr.SamplesPerPixel = int(bo.Uint16(entry[8:10]))
		case 278: // RowsPerStrip
			hdr.RowsPerStrip = int(valOffset)
		case 279: // StripByteCounts
			hdr.StripByteCounts, err = readLongArray()
			if err != nil {
				return TiffHeader{}, err
			}
		case 284: // PlanarConfiguration
			hdr.PlanarConfig = int(bo.Uint16(entry[8:10]))
		}
	}

	// Validate
	if hdr.Width <= 0 || hdr.Height <= 0 {
		return TiffHeader{}, fmt.Errorf("invalid dimensions")
	}
	if hdr.Compression != 1 {
		return TiffHeader{}, fmt.Errorf("unsupported compression: %d", hdr.Compression)
	}
	if hdr.Photometric != 2 {
		return TiffHeader{}, fmt.Errorf("expected RGB photometric interpretation, got %d", hdr.Photometric)
	}
	if hdr.SamplesPerPixel != 3 {
		return TiffHeader{}, fmt.Errorf("expected 3 samples/pixel, got %d", hdr.SamplesPerPixel)
	}
	if len(hdr.BitsPerSample) != 3 || hdr.BitsPerSample[0] != 8 {
		return TiffHeader{}, fmt.Errorf("expected BitsPerSample = [8,8,8], got %v", hdr.BitsPerSample)
	}
	if len(hdr.StripOffsets) == 0 || len(hdr.StripOffsets) != len(hdr.StripByteCounts) {
		return TiffHeader{}, fmt.Errorf("invalid strip offset/length")
	}

	return hdr, nil
}

// Sample maps the 3D vector P (ECEF) to texture coordinates and returns a base.Color4.
func (t Texture) Sample(P vectors.Vec3) colors.Color4 {
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
		panic(fmt.Errorf("tiled read failed at index %d: %w", idx, err))
	}

	return colors.From8BitRgb(buf[0], buf[1], buf[2], 255)
}
