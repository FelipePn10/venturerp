package pdfkit

import "strings"

// pdfString escapes a string as a PDF literal and re-encodes it to WinAnsi bytes
// so accented Portuguese characters render under WinAnsiEncoding.
func pdfString(s string) string {
	var b strings.Builder
	for _, r := range s {
		c := toWinAnsi(r)
		switch c {
		case '\\', '(', ')':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// toWinAnsi maps a rune to its single Windows-1252 byte, falling back to '?'.
func toWinAnsi(r rune) byte {
	switch {
	case r < 0x80:
		return byte(r)
	case r >= 0xA0 && r <= 0xFF:
		return byte(r)
	}
	switch r {
	case '€':
		return 0x80
	case '‚':
		return 0x82
	case 'ƒ':
		return 0x83
	case '„':
		return 0x84
	case '…':
		return 0x85
	case '‹':
		return 0x8B
	case '‘', '’':
		return 0x27
	case '“', '”':
		return 0x22
	case '–', '—':
		return 0x2D
	case '•':
		return 0x95
	case '›':
		return 0x9B
	}
	return '?'
}
