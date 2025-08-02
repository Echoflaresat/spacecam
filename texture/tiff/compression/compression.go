package compression

import "fmt"

// Type represents a TIFF compression method.
type Type int

const (
	Unknown    Type = -1
	None       Type = 1
	CCITT      Type = 2
	G3         Type = 3 // Group 3 Fax
	G4         Type = 4 // Group 4 Fax
	LZW        Type = 5
	JPEGOld    Type = 6 // Superseded by JPEG
	JPEG       Type = 7
	Deflate    Type = 8 // zlib compression
	PackBits   Type = 32773
	DeflateOld Type = 32946 // Superseded by Deflate
)

// String returns a readable name for the compression type.
func (c Type) String() string {
	switch c {
	case Unknown:
		return "Unknown"
	case None:
		return "None"
	case CCITT:
		return "CCITT"
	case G3:
		return "G3Fax"
	case G4:
		return "G4Fax"
	case LZW:
		return "LZW"
	case JPEGOld:
		return "JPEGOld"
	case JPEG:
		return "JPEG"
	case Deflate:
		return "Deflate"
	case PackBits:
		return "PackBits"
	case DeflateOld:
		return "DeflateOld"
	default:
		return fmt.Sprintf("CompressionType(%d)", int(c))
	}
}
