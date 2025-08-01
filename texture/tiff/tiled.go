package tiff

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"

	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/exp/mmap"
)

type tiledTiff struct {
	header TiffHeader
	reader *mmap.ReaderAt
	cache  *lru.Cache // tileIndex -> []byte
}

func LoadTiledTiff(path string) (image.Image, error) {
	reader, err := mmap.Open(path)
	if err != nil {
		return nil, err
	}

	header, err := parseTiffHeader(reader)
	if err != nil {
		return nil, err
	}

	if header.Compression != 1 && header.Compression != 8 {
		return nil, fmt.Errorf("unsupported compression: %d", header.Compression)
	}
	if header.Photometric != 2 && header.Photometric != 1 {
		return nil, fmt.Errorf("unsupported photometric interpretation: %d", header.Photometric)
	}
	switch header.Photometric {
	case 1:
		if header.SamplesPerPixel != 1 || header.BitsPerSample[0] != 8 {
			return nil, fmt.Errorf("unsupported grayscale format")
		}
	case 2:
		if header.SamplesPerPixel != 3 || header.BitsPerSample[0] != 8 {
			return nil, fmt.Errorf("unsupported RGB format")
		}
	default:
		return nil, fmt.Errorf("unsupported photometric: %d", header.Photometric)
	}

	if len(header.TileOffsets) == 0 || len(header.TileOffsets) != len(header.TileByteCounts) {
		return nil, fmt.Errorf("invalid tile offset/length")
	}

	cache, _ := lru.New(200) // Keep last 20 tiles decompressed

	return &tiledTiff{
		header: header,
		reader: reader,
		cache:  cache,
	}, nil

}

func (t *tiledTiff) ColorModel() color.Model {
	return color.RGBAModel
}

func (t *tiledTiff) Bounds() image.Rectangle {
	return image.Rect(0, 0, t.header.Width, t.header.Height)
}

func (t *tiledTiff) At(x, y int) color.Color {
	h := t.header

	tileX := x / h.TileWidth
	tileY := y / h.TileHeight
	tilesAcross := int(math.Ceil(float64(h.Width) / float64(h.TileWidth)))
	tileIndex := tileY*tilesAcross + tileX

	var tile []byte
	if val, ok := t.cache.Get(tileIndex); ok {
		tile = val.([]byte)
	} else {
		tile = t.loadTile(tileIndex)
		t.cache.Add(tileIndex, tile)
	}

	localX := x % h.TileWidth
	localY := y % h.TileHeight
	rowStride := h.TileWidth * h.SamplesPerPixel
	pixOffset := localY*rowStride + localX*h.SamplesPerPixel

	switch h.Photometric {
	case 2: // RGB
		return color.RGBA{
			R: tile[pixOffset],
			G: tile[pixOffset+1],
			B: tile[pixOffset+2],
			A: 255,
		}
	case 1: // BlackIsZero grayscale
		v := tile[pixOffset]
		return color.RGBA{R: v, G: v, B: v, A: 255}
	default:
		panic(fmt.Sprintf("unsupported PhotometricInterpretation: %d", h.Photometric))
	}
}

func (t *tiledTiff) loadTile(index int) []byte {
	h := t.header
	offset := h.TileOffsets[index]
	byteCount := h.TileByteCounts[index]

	buf := make([]byte, byteCount)
	_, err := t.reader.ReadAt(buf, int64(offset))
	if err != nil {
		panic(fmt.Sprintf("failed to read tile %d: %v", index, err))
	}

	if h.Compression == 8 { // DEFLATE
		r, err := zlib.NewReader(io.NopCloser(bytes.NewReader(buf)))
		if err != nil {
			panic(fmt.Sprintf("zlib decompression error: %v", err))
		}
		defer r.Close()
		tile, err := io.ReadAll(r)
		if err != nil {
			panic(fmt.Sprintf("zlib read error: %v", err))
		}
		return tile
	}
	return buf
}
