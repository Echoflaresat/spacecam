package planarconfig

import "fmt"

// Type represents the TIFF PlanarConfiguration tag (284).
type Type int

const (
	Unknown  Type = -1
	Contig   Type = 1 // Chunky: interleaved samples (RGB RGB ...)
	Separate Type = 2 // Planar: separate planes for each sample (RRR... GGG... BBB...)
)

func (c Type) String() string {
	switch c {
	case Unknown:
		return "Unknown"
	case Contig:
		return "Contig"
	case Separate:
		return "Separate"
	default:
		return fmt.Sprintf("Config(%d)", int(c))
	}
}
