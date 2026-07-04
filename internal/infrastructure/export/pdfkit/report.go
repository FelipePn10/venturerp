package pdfkit

import (
	"strconv"
	"strings"
	"time"
)

// Align is the horizontal alignment of a table column.
type Align int

const (
	AlignLeft Align = iota
	AlignRight
	AlignCenter
)

// Theme is the colour palette for a document. DefaultTheme is a corporate
// navy/grey scheme; callers override Brand with the company's colour.
type Theme struct {
	Brand     Color // header band & accents
	BrandText Color // text drawn on the brand band
	Title     Color // report title colour
	Text      Color // primary body text
	Muted     Color // secondary text (subtitle, footer)
	Zebra     Color // alternating row background
	Rule      Color // thin separators / borders
	TotalBg   Color // totals row background (light brand tint)
}

// DefaultTheme returns the corporate navy palette.
func DefaultTheme() Theme {
	return Theme{
		Brand:     Color{27, 58, 91}, // #1B3A5B navy
		BrandText: White,
		Title:     Color{27, 58, 91},
		Text:      Color{33, 37, 41},    // near-black
		Muted:     Color{108, 117, 125}, // grey
		Zebra:     Color{244, 246, 248}, // very light grey-blue
		Rule:      Color{222, 226, 230}, // light grey
		TotalBg:   Color{226, 232, 240}, // light brand tint
	}
}

// Company is the non-sensitive identification shown in the letterhead.
type Company struct {
	Name    string
	CNPJ    string
	IE      string
	Address string
	Phone   string
	Email   string
}

func (c Company) infoLines() []string {
	var lines []string
	var ids []string
	if c.CNPJ != "" {
		ids = append(ids, "CNPJ: "+c.CNPJ)
	}
	if c.IE != "" {
		ids = append(ids, "IE: "+c.IE)
	}
	if len(ids) > 0 {
		lines = append(lines, strings.Join(ids, "    "))
	}
	if c.Address != "" {
		lines = append(lines, c.Address)
	}
	var contact []string
	if c.Phone != "" {
		contact = append(contact, "Tel: "+c.Phone)
	}
	if c.Email != "" {
		contact = append(contact, c.Email)
	}
	if len(contact) > 0 {
		lines = append(lines, strings.Join(contact, "    "))
	}
	return lines
}

// Column describes one table column.
type Column struct {
	Title  string
	Align  Align
	Weight float64 // relative width; defaults to 1 when zero
}

// Page layout constants (points).
const (
	margin       = 36.0
	bandH        = 66.0 // page-1 letterhead band height
	compactBandH = 22.0 // continuation-page band height
	rowH         = 15.0
	headerRowH   = 17.0
	bodySize     = 8.5
)

// TableReport is the full specification of a paginated, branded table report.
type TableReport struct {
	Theme       Theme
	Company     Company
	Logo        []byte // optional PNG/JPEG; ignored if it fails to decode
	Title       string
	Subtitle    string
	Columns     []Column
	Rows        [][]string
	Totals      []string  // optional; one cell per column, "" leaves a blank
	GeneratedAt time.Time // footer stamp; defaults to now
}

// Render lays the report out across as many A4 pages as needed and returns the
// PDF bytes. It never errors: a bad logo is simply dropped.
func (tr *TableReport) Render() []byte {
	d := New()
	var logo *Image
	if len(tr.Logo) > 0 {
		logo, _ = d.AddImage(tr.Logo)
	}
	th := tr.Theme

	contentW := d.width - 2*margin
	widths := columnWidths(tr.Columns, contentW)
	footTop := d.height - margin + 4

	gen := tr.GeneratedAt
	if gen.IsZero() {
		gen = time.Now()
	}
	note := "Gerado em " + gen.Format("02/01/2006 15:04")

	d.SetFooter(func(p *Page, num, total int) {
		drawFooter(p, th, margin, footTop, d.width-margin, note, num, total)
	})

	page := d.AddPage()
	y := drawLetterhead(page, th, tr.Company, logo, margin, margin, contentW, false)

	// Title block (page 1 only).
	y += 18
	page.Text(margin, y, FontBold, 15, th.Title, tr.Title)
	if tr.Subtitle != "" {
		y += 13
		page.Text(margin, y, FontRegular, 9, th.Muted, tr.Subtitle)
	}
	y += 8

	// Column header band.
	y = drawTableHeader(page, th, tr.Columns, widths, margin, y, contentW)

	rowIndex := 0
	for i, row := range tr.Rows {
		if y+rowH > footTop-rowH { // leave room for footer
			page = d.AddPage()
			top := drawLetterhead(page, th, tr.Company, logo, margin, margin, contentW, true)
			top += 6
			page.Text(margin, top+9, FontBold, 11, th.Title, tr.Title+" (continuação)")
			y = top + 18
			y = drawTableHeader(page, th, tr.Columns, widths, margin, y, contentW)
			rowIndex = 0
		}
		drawDataRow(page, th, tr.Columns, widths, row, margin, y, contentW, rowIndex%2 == 1)
		y += rowH
		rowIndex++
		_ = i
	}

	// Bottom border of the table body.
	page.StrokeLine(margin, y, margin+contentW, y, 0.5, th.Rule)

	// Totals row.
	if len(tr.Totals) > 0 {
		drawTotalsRow(page, th, tr.Columns, widths, tr.Totals, margin, y, contentW)
	}

	return d.Render()
}

// Letterhead draws the branded company band at (x, top) spanning width w and
// returns the y just below it. Shared by every document type so headers look
// identical. When compact is true a slim band (company name only) is drawn, for
// continuation pages.
func (p *Page) Letterhead(th Theme, co Company, logo *Image, x, top, w float64, compact bool) float64 {
	return drawLetterhead(p, th, co, logo, x, top, w, compact)
}

// drawLetterhead renders the company band and returns the y just below it.
func drawLetterhead(p *Page, th Theme, co Company, logo *Image, x, top, w float64, compact bool) float64 {
	h := bandH
	if compact {
		h = compactBandH
	}
	p.FillRect(x, top, w, h, th.Brand)

	if compact {
		p.Text(x+10, top+15, FontBold, 10, th.BrandText, co.Name)
		return top + h
	}

	// Logo on the right, vertically centred in the band.
	if logo != nil {
		lh := h - 16
		lw := logoWidth(logo, lh)
		p.DrawImage(logo, x+w-lw-10, top+8, lw, lh)
	}

	ty := top + 20
	p.Text(x+12, ty, FontBold, 13, th.BrandText, co.Name)
	ty += 14
	for _, ln := range co.infoLines() {
		p.Text(x+12, ty, FontRegular, 8, th.BrandText, ln)
		ty += 10
	}
	return top + h
}

// logoWidth keeps the logo aspect ratio for a target height.
func logoWidth(img *Image, targetH float64) float64 {
	if img.h == 0 {
		return targetH
	}
	w := targetH * float64(img.w) / float64(img.h)
	if max := 130.0; w > max { // cap so a wide logo never crowds the band
		w = max
	}
	return w
}

func drawTableHeader(p *Page, th Theme, cols []Column, widths []float64, x, top, w float64) float64 {
	p.FillRect(x, top, w, headerRowH, th.Brand)
	cx := x
	for i, c := range cols {
		drawCell(p, c.Title, FontBold, bodySize, th.BrandText, c.Align, cx, top+11.5, widths[i])
		cx += widths[i]
	}
	return top + headerRowH
}

func drawDataRow(p *Page, th Theme, cols []Column, widths []float64, row []string, x, top, w float64, zebra bool) {
	if zebra {
		p.FillRect(x, top, w, rowH, th.Zebra)
	}
	cx := x
	for i, c := range cols {
		val := ""
		if i < len(row) {
			val = row[i]
		}
		drawCell(p, val, FontRegular, bodySize, th.Text, c.Align, cx, top+10.5, widths[i])
		cx += widths[i]
	}
}

func drawTotalsRow(p *Page, th Theme, cols []Column, widths []float64, totals []string, x, top, w float64) {
	p.FillRect(x, top, w, rowH+1, th.TotalBg)
	p.StrokeLine(x, top, x+w, top, 1, th.Brand)
	cx := x
	for i, c := range cols {
		val := ""
		if i < len(totals) {
			val = totals[i]
		}
		drawCell(p, val, FontBold, bodySize, th.Text, c.Align, cx, top+11, widths[i])
		cx += widths[i]
	}
}

// drawCell draws one cell's text within [cellX, cellX+width], padded and
// truncated to fit, honouring alignment.
func drawCell(p *Page, s string, font Font, size float64, c Color, align Align, cellX, baseline, width float64) {
	const pad = 4.0
	inner := width - 2*pad
	s = ellipsize(font, size, s, inner)
	switch align {
	case AlignRight:
		p.TextRight(cellX+width-pad, baseline, font, size, c, s)
	case AlignCenter:
		p.TextCenter(cellX+width/2, baseline, font, size, c, s)
	default:
		p.Text(cellX+pad, baseline, font, size, c, s)
	}
}

// ellipsize trims s with a trailing ellipsis until it fits maxW points.
func ellipsize(font Font, size float64, s string, maxW float64) string {
	if TextWidth(font, size, s) <= maxW {
		return s
	}
	r := []rune(s)
	for len(r) > 1 {
		r = r[:len(r)-1]
		if TextWidth(font, size, string(r)+"…") <= maxW {
			return string(r) + "…"
		}
	}
	return "…"
}

// columnWidths scales column weights to the available width.
func columnWidths(cols []Column, total float64) []float64 {
	sum := 0.0
	for _, c := range cols {
		w := c.Weight
		if w <= 0 {
			w = 1
		}
		sum += w
	}
	widths := make([]float64, len(cols))
	for i, c := range cols {
		w := c.Weight
		if w <= 0 {
			w = 1
		}
		widths[i] = total * w / sum
	}
	return widths
}

// Margin is the standard page margin (points) used by the shared layout.
const Margin = margin

// Footer draws the standard footer (thin rule + note on the left, page numbering
// on the right) between x and xRight at the given top. Exported for documents
// that build their own page flow.
func (p *Page) Footer(th Theme, x, top, xRight float64, note string, num, total int) {
	drawFooter(p, th, x, top, xRight, note, num, total)
}

// drawFooter renders the thin top rule, generation note and page numbering.
func drawFooter(p *Page, th Theme, x, top, xRight float64, note string, num, total int) {
	p.StrokeLine(x, top, xRight, top, 0.5, th.Rule)
	p.Text(x, top+10, FontRegular, 7.5, th.Muted, note)
	p.TextRight(xRight, top+10, FontRegular, 7.5, th.Muted,
		"Página "+strconv.Itoa(num)+" de "+strconv.Itoa(total))
}

// ParseHexColor parses "#RRGGBB" / "RRGGBB" into a Color; ok is false on
// malformed input.
func ParseHexColor(s string) (Color, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return Color{}, false
	}
	var rgb [3]uint8
	for i := 0; i < 3; i++ {
		hi, ok1 := hexNibble(s[i*2])
		lo, ok2 := hexNibble(s[i*2+1])
		if !ok1 || !ok2 {
			return Color{}, false
		}
		rgb[i] = hi<<4 | lo
	}
	return Color{R: rgb[0], G: rgb[1], B: rgb[2]}, true
}

func hexNibble(b byte) (uint8, bool) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', true
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, true
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, true
	}
	return 0, false
}
