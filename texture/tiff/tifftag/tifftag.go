package tifftag

import "fmt"

// Tag represents a TIFF field tag number.
// https://www.loc.gov/preservation/digital/formats/content/tiff_tags.shtml

type Tag uint16

const (
	ImageWidth                Tag = 256
	ImageLength               Tag = 257
	BitsPerSample             Tag = 258
	Compression               Tag = 259
	PhotometricInterpretation Tag = 262
	StripOffsets              Tag = 273
	SamplesPerPixel           Tag = 277
	RowsPerStrip              Tag = 278
	StripByteCounts           Tag = 279
	PlanarConfiguration       Tag = 284
	TileWidth                 Tag = 322
	TileLength                Tag = 323
	TileOffsets               Tag = 324
	TileByteCounts            Tag = 325
)

func (t Tag) String() string {
	switch t {
	case ImageWidth:
		return "ImageWidth"
	case ImageLength:
		return "ImageLength"
	case BitsPerSample:
		return "BitsPerSample"
	case Compression:
		return "Compression"
	case PhotometricInterpretation:
		return "PhotometricInterpretation"
	case StripOffsets:
		return "StripOffsets"
	case SamplesPerPixel:
		return "SamplesPerPixel"
	case RowsPerStrip:
		return "RowsPerStrip"
	case StripByteCounts:
		return "StripByteCounts"
	case PlanarConfiguration:
		return "PlanarConfiguration"
	case TileWidth:
		return "TileWidth"
	case TileLength:
		return "TileLength"
	case TileOffsets:
		return "TileOffsets"
	case TileByteCounts:
		return "TileByteCounts"
	default:
		return fmt.Sprintf("Tag(%d)", t)
	}
}
