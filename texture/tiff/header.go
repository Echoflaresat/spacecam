package tiff

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/echoflaresat/spacecam/texture/tiff/compression"
	"github.com/echoflaresat/spacecam/texture/tiff/photometric"
	"github.com/echoflaresat/spacecam/texture/tiff/planarconfig"
	"github.com/echoflaresat/spacecam/texture/tiff/tifftag"
)

type TiffHeader struct {
	ByteOrder       binary.ByteOrder
	Width, Height   int
	SamplesPerPixel int
	BitsPerSample   []int
	Photometric     photometric.Interpretation
	Compression     compression.Type
	PlanarConfig    planarconfig.Type

	// Strip layout
	RowsPerStrip    int
	StripOffsets    []int
	StripByteCounts []int

	// Tile layout
	TileWidth      int
	TileHeight     int
	TileOffsets    []int
	TileByteCounts []int
}

var ErrInvalidTiffHeader = errors.New("invalid TIFF header")

func parseTiffHeader(reader io.ReaderAt) (TiffHeader, error) {
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
		return TiffHeader{}, ErrInvalidTiffHeader
	}
	if bo.Uint16(header[2:4]) != 42 {
		return TiffHeader{}, ErrInvalidTiffHeader
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
		Photometric:     photometric.Unknown,
		Compression:     compression.Unknown,
		PlanarConfig:    planarconfig.Unknown,
	}

	for i := 0; i < numEntries; i++ {
		entry := entriesRaw[i*12 : (i+1)*12]
		tag := tifftag.Tag(bo.Uint16(entry[0:2]))
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
		case tifftag.ImageWidth:
			hdr.Width = int(valOffset)
		case tifftag.ImageLength:
			hdr.Height = int(valOffset)
		case tifftag.BitsPerSample:
			hdr.BitsPerSample, err = readShortArray()
			if err != nil {
				return TiffHeader{}, err
			}
		case tifftag.Compression:
			hdr.Compression = compression.Type(bo.Uint16(entry[8:10]))
		case tifftag.PhotometricInterpretation:
			hdr.Photometric = photometric.Interpretation(bo.Uint16(entry[8:10]))
		case tifftag.StripOffsets:
			hdr.StripOffsets, err = readLongArray()
			if err != nil {
				return TiffHeader{}, err
			}
		case tifftag.SamplesPerPixel:
			hdr.SamplesPerPixel = int(bo.Uint16(entry[8:10]))
		case tifftag.RowsPerStrip:
			hdr.RowsPerStrip = int(valOffset)
		case tifftag.StripByteCounts:
			hdr.StripByteCounts, err = readLongArray()
			if err != nil {
				return TiffHeader{}, err
			}
		case tifftag.PlanarConfiguration:
			hdr.PlanarConfig = planarconfig.Type(bo.Uint16(entry[8:10]))
		case tifftag.TileWidth:
			hdr.TileWidth = int(valOffset)
		case tifftag.TileLength:
			hdr.TileHeight = int(valOffset)
		case tifftag.TileOffsets:
			hdr.TileOffsets, err = readLongArray()
			if err != nil {
				return TiffHeader{}, err
			}
		case tifftag.TileByteCounts:
			hdr.TileByteCounts, err = readLongArray()
			if err != nil {
				return TiffHeader{}, err
			}
		}
	}

	return hdr, nil
}
