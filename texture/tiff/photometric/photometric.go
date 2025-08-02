package photometric

import "fmt"

// Interpretation represents TIFF photometric interpretation modes.
type Interpretation int

const (
	Unknown     Interpretation = -1
	WhiteIsZero Interpretation = 0
	BlackIsZero Interpretation = 1
	RGB         Interpretation = 2
	Paletted    Interpretation = 3
	TransMask   Interpretation = 4 // transparency mask
	CMYK        Interpretation = 5
	YCbCr       Interpretation = 6
	CIELab      Interpretation = 8
)

func (p Interpretation) String() string {
	switch p {
	case Unknown:
		return "WhiteIsZero"
	case WhiteIsZero:
		return "WhiteIsZero"
	case BlackIsZero:
		return "BlackIsZero"
	case RGB:
		return "RGB"
	case Paletted:
		return "Paletted"
	case TransMask:
		return "TransMask"
	case CMYK:
		return "CMYK"
	case YCbCr:
		return "YCbCr"
	case CIELab:
		return "CIELab"
	default:
		return fmt.Sprintf("Interpretation(%d)", int(p))
	}
}
