package pdfkit

// Font is one of the base-14 Helvetica variants — always available in every PDF
// viewer, so nothing has to be embedded.
type Font int

const (
	FontRegular Font = iota
	FontBold
	FontOblique
)

// baseFont maps a Font to its PDF /BaseFont name and the resource alias used in
// page content streams.
func (f Font) baseName() string {
	switch f {
	case FontBold:
		return "Helvetica-Bold"
	case FontOblique:
		return "Helvetica-Oblique"
	default:
		return "Helvetica"
	}
}

func (f Font) resAlias() string {
	switch f {
	case FontBold:
		return "F2"
	case FontOblique:
		return "F3"
	default:
		return "F1"
	}
}

// Advance widths (1000-unit em) for ASCII 0x20..0x7E, taken from the Adobe AFM
// metrics. Helvetica-Oblique shares Helvetica's widths.
var helvWidths = [95]int{
	278, 278, 355, 556, 556, 889, 667, 191, 333, 333, // 0x20-0x29
	389, 584, 278, 333, 278, 278, 556, 556, 556, 556, // 0x2A-0x33
	556, 556, 556, 556, 556, 556, 278, 278, 584, 584, // 0x34-0x3D
	584, 556, 1015, 667, 667, 722, 722, 667, 611, 778, // 0x3E-0x47
	722, 278, 500, 667, 556, 833, 722, 778, 667, 778, // 0x48-0x51
	722, 667, 611, 722, 667, 944, 667, 667, 611, 278, // 0x52-0x5B
	278, 278, 469, 556, 333, 556, 556, 500, 556, 556, // 0x5C-0x65
	278, 556, 556, 222, 222, 500, 222, 833, 556, 556, // 0x66-0x6F
	556, 556, 333, 500, 278, 556, 500, 722, 500, 500, // 0x70-0x79
	500, 334, 260, 334, 584, // 0x7A-0x7E
}

var helvBoldWidths = [95]int{
	278, 333, 474, 556, 556, 889, 722, 238, 333, 333, // 0x20-0x29
	389, 584, 278, 333, 278, 278, 556, 556, 556, 556, // 0x2A-0x33
	556, 556, 556, 556, 556, 556, 333, 333, 584, 584, // 0x34-0x3D
	584, 611, 975, 722, 722, 722, 722, 667, 611, 778, // 0x3E-0x47
	722, 278, 556, 722, 611, 833, 722, 778, 667, 778, // 0x48-0x51
	722, 667, 611, 722, 667, 944, 667, 667, 611, 333, // 0x52-0x5B
	278, 333, 584, 556, 333, 556, 611, 556, 611, 556, // 0x5C-0x65
	333, 611, 611, 278, 278, 556, 278, 889, 611, 611, // 0x66-0x6F
	611, 611, 389, 556, 333, 611, 556, 778, 556, 556, // 0x70-0x79
	500, 389, 280, 389, 584, // 0x7A-0x7E
}

// runeWidth returns the advance width (1000-unit em) of r in the given font.
// Accented Latin characters fold to their base letter, whose advance is the same
// in Helvetica; unknown runes fall back to the average lowercase width.
func runeWidth(f Font, r rune) int {
	r = foldAccent(r)
	if r >= 0x20 && r <= 0x7E {
		if f == FontBold {
			return helvBoldWidths[r-0x20]
		}
		return helvWidths[r-0x20]
	}
	return 556
}

// TextWidth returns the rendered width of s in points at the given size.
func TextWidth(f Font, size float64, s string) float64 {
	total := 0
	for _, r := range s {
		total += runeWidth(f, r)
	}
	return float64(total) / 1000.0 * size
}

// foldAccent maps a Latin-1/Latin accented rune to its unaccented base letter so
// width lookups stay correct for Portuguese text without a full Unicode table.
func foldAccent(r rune) rune {
	switch r {
	case 'á', 'à', 'â', 'ã', 'ä', 'å':
		return 'a'
	case 'é', 'è', 'ê', 'ë':
		return 'e'
	case 'í', 'ì', 'î', 'ï':
		return 'i'
	case 'ó', 'ò', 'ô', 'õ', 'ö':
		return 'o'
	case 'ú', 'ù', 'û', 'ü':
		return 'u'
	case 'ç':
		return 'c'
	case 'ñ':
		return 'n'
	case 'ý', 'ÿ':
		return 'y'
	case 'Á', 'À', 'Â', 'Ã', 'Ä', 'Å':
		return 'A'
	case 'É', 'È', 'Ê', 'Ë':
		return 'E'
	case 'Í', 'Ì', 'Î', 'Ï':
		return 'I'
	case 'Ó', 'Ò', 'Ô', 'Õ', 'Ö':
		return 'O'
	case 'Ú', 'Ù', 'Û', 'Ü':
		return 'U'
	case 'Ç':
		return 'C'
	case 'Ñ':
		return 'N'
	case 'ª':
		return 'a'
	case 'º':
		return 'o'
	}
	return r
}
