package export

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// PDF page geometry (A4 portrait, points).
const (
	pdfPageW    = 595.0
	pdfPageH    = 842.0
	pdfMargin   = 36.0
	pdfBodySize = 8.5
	pdfLeading  = 11.0
	pdfTitleSz  = 14.0
	// Courier advance width is 0.6em; at pdfBodySize this is the per-char width.
	pdfCharW = pdfBodySize * 0.6
)

// EncodePDF renders the table as a paginated PDF using the standard Courier
// font (monospaced, so columns line up) plus Helvetica-Bold for the title.
// No fonts are embedded — the base-14 fonts are guaranteed available in every
// PDF viewer. Text is WinAnsi-encoded so Portuguese accents render correctly.
func EncodePDF(w io.Writer, t *Table) error {
	if err := t.Validate(); err != nil {
		return err
	}
	t.normalize()

	pages := layoutPDF(t)

	// Object plan: 1=Catalog, 2=Pages, 3=Font Courier, 4=Font Helvetica-Bold,
	// then for each page a Page object and a Contents stream object.
	var buf bytes.Buffer
	offsets := []int{} // offsets[i] = byte offset of object i+1
	buf.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")

	addObj := func(body string) {
		offsets = append(offsets, buf.Len())
		buf.WriteString(strconv.Itoa(len(offsets)))
		buf.WriteString(" 0 obj\n")
		buf.WriteString(body)
		buf.WriteString("\nendobj\n")
	}

	pageObjStart := 5 // first page object number
	kids := make([]string, len(pages))
	for i := range pages {
		pageNum := pageObjStart + i*2
		kids[i] = strconv.Itoa(pageNum) + " 0 R"
	}

	// 1: Catalog
	addObj("<< /Type /Catalog /Pages 2 0 R >>")
	// 2: Pages
	addObj(fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>",
		len(pages), strings.Join(kids, " ")))
	// 3: Courier
	addObj("<< /Type /Font /Subtype /Type1 /BaseFont /Courier /Encoding /WinAnsiEncoding >>")
	// 4: Helvetica-Bold
	addObj("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")

	for i, content := range pages {
		contentObjNum := pageObjStart + i*2 + 1
		// Page object
		addObj(fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %s %s] "+
				"/Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> /Contents %d 0 R >>",
			ftoa(pdfPageW), ftoa(pdfPageH), contentObjNum))
		// Contents stream
		stream := content
		addObj(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	}

	// Cross-reference table.
	xrefStart := buf.Len()
	n := len(offsets) + 1
	buf.WriteString("xref\n")
	buf.WriteString(fmt.Sprintf("0 %d\n", n))
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}
	buf.WriteString("trailer\n")
	buf.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", n))
	buf.WriteString("startxref\n")
	buf.WriteString(strconv.Itoa(xrefStart))
	buf.WriteString("\n%%EOF\n")

	_, err := w.Write(buf.Bytes())
	return err
}

// layoutPDF computes column widths, paginates rows, and returns one content
// stream per page.
func layoutPDF(t *Table) []string {
	widths := columnCharWidths(t)

	header := renderRow(t.Columns, widths)
	sep := strings.Repeat("-", len(header))

	// Rows that fit on the first page (after title block) and subsequent pages.
	firstPageTop := pdfPageH - pdfMargin - pdfTitleSz - 2*pdfLeading // room for title+subtitle
	bottom := pdfMargin + pdfLeading

	var pages []string
	i := 0
	pageIndex := 0
	for {
		var sb strings.Builder
		var y float64
		sb.WriteString("BT\n")

		if pageIndex == 0 {
			// Title (Helvetica-Bold) then subtitle/timestamp (Courier).
			y = pdfPageH - pdfMargin - pdfTitleSz
			sb.WriteString(fmt.Sprintf("/F2 %s Tf\n", ftoa(pdfTitleSz)))
			sb.WriteString(fmt.Sprintf("%s %s Td (%s) Tj\n", ftoa(pdfMargin), ftoa(y), pdfStr(t.Title)))
			sb.WriteString(fmt.Sprintf("/F1 %s Tf\n", ftoa(pdfBodySize)))
			meta := "Gerado em " + t.GeneratedAt.Format("02/01/2006 15:04")
			if t.Subtitle != "" {
				meta = t.Subtitle + "  •  " + meta
			}
			y = firstPageTop
			sb.WriteString(fmt.Sprintf("0 %s Td (%s) Tj\n", ftoa(-(pdfTitleSz)), pdfStr(meta)))
			// Move absolute to header start: reset by ending text and restarting.
			sb.WriteString("ET\nBT\n")
		}
		sb.WriteString(fmt.Sprintf("/F1 %s Tf\n", ftoa(pdfBodySize)))
		sb.WriteString(fmt.Sprintf("%s TL\n", ftoa(pdfLeading)))
		startY := firstPageTop - pdfLeading
		if pageIndex > 0 {
			startY = pdfPageH - pdfMargin
		}
		sb.WriteString(fmt.Sprintf("%s %s Td\n", ftoa(pdfMargin), ftoa(startY)))

		// Header + separator repeated on every page.
		sb.WriteString(fmt.Sprintf("(%s) Tj T*\n", pdfStr(header)))
		sb.WriteString(fmt.Sprintf("(%s) Tj T*\n", pdfStr(sep)))
		y = startY - 2*pdfLeading

		for i < len(t.Rows) && y > bottom {
			line := renderRow(t.Rows[i], widths)
			sb.WriteString(fmt.Sprintf("(%s) Tj T*\n", pdfStr(line)))
			y -= pdfLeading
			i++
		}
		sb.WriteString("ET")
		pages = append(pages, sb.String())
		pageIndex++

		if i >= len(t.Rows) {
			break
		}
	}
	return pages
}

// columnCharWidths returns the character width allotted to each column, capped
// so the whole row fits the usable page width.
func columnCharWidths(t *Table) []int {
	widths := make([]int, len(t.Columns))
	for c, col := range t.Columns {
		widths[c] = runeLen(col)
	}
	for _, row := range t.Rows {
		for c, cell := range row {
			if l := runeLen(cell); l > widths[c] {
				widths[c] = l
			}
		}
	}
	// Cap each column and enforce the total fits the page.
	const maxCol = 40
	for c := range widths {
		if widths[c] > maxCol {
			widths[c] = maxCol
		}
		if widths[c] < 3 {
			widths[c] = 3
		}
	}
	usableW := float64(pdfPageW - 2*pdfMargin)
	maxChars := int(usableW / pdfCharW)
	for {
		total := len(widths) - 1 // single-space separators
		for _, w := range widths {
			total += w
		}
		if total <= maxChars {
			break
		}
		// Shrink the widest column by one until it fits.
		widest := 0
		for c := 1; c < len(widths); c++ {
			if widths[c] > widths[widest] {
				widest = c
			}
		}
		if widths[widest] <= 3 {
			break
		}
		widths[widest]--
	}
	return widths
}

// renderRow pads/truncates each cell to its column width and joins with spaces.
func renderRow(cells []string, widths []int) string {
	parts := make([]string, len(cells))
	for c, cell := range cells {
		parts[c] = fit(cell, widths[c])
	}
	return strings.Join(parts, " ")
}

// fit pads (right) or truncates a string to exactly n runes, adding an ellipsis
// when truncated.
func fit(s string, n int) string {
	r := []rune(s)
	if len(r) == n {
		return s
	}
	if len(r) < n {
		return s + strings.Repeat(" ", n-len(r))
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

func runeLen(s string) int { return len([]rune(s)) }

func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', 2, 64) }

// pdfStr escapes a string for a PDF literal and re-encodes it as WinAnsi bytes
// so accented Portuguese characters render under WinAnsiEncoding.
func pdfStr(s string) string {
	var b strings.Builder
	for _, r := range s {
		c := toWinAnsi(r)
		switch c {
		case '\\', '(', ')':
			b.WriteByte('\\')
			b.WriteByte(c)
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
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
		// Latin-1 supplement coincides with WinAnsi in this range (covers
		// áàâãéêíóôõúçÁÀÂÃÉÊÍÓÔÕÚÇ and °ºª etc.).
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
		return 0x27 // ASCII apostrophe — safe everywhere
	case '“', '”':
		return 0x22 // ASCII quote
	case '–', '—':
		return 0x2D // hyphen
	case '•':
		return 0x95
	case '›':
		return 0x9B
	}
	return '?'
}
