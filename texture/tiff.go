package texture

import (
	"encoding/binary"
	"fmt"
	"io"
)

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
	TagRowsPerStrip              = 278
	TagPlanarConfiguration       = 284
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
		case TagImageWidth:
			hdr.Width = int(valOffset)
		case TagImageLength:
			hdr.Height = int(valOffset)
		case TagBitsPerSample:
			hdr.BitsPerSample, err = readShortArray()
			if err != nil {
				return TiffHeader{}, err
			}
		case TagCompression:
			hdr.Compression = int(bo.Uint16(entry[8:10]))
		case TagPhotometricInterpretation:
			hdr.Photometric = int(bo.Uint16(entry[8:10]))
		case TagStripOffsets:
			hdr.StripOffsets, err = readLongArray()
			if err != nil {
				return TiffHeader{}, err
			}
		case TagSamplesPerPixel:
			hdr.SamplesPerPixel = int(bo.Uint16(entry[8:10]))
		case TagRowsPerStrip:
			hdr.RowsPerStrip = int(valOffset)
		case TagStripByteCounts:
			hdr.StripByteCounts, err = readLongArray()
			if err != nil {
				return TiffHeader{}, err
			}
		case TagPlanarConfiguration:
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
