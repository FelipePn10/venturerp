// Package pdfkit is a small, dependency-free PDF builder tuned for professional
// business documents: proportional Helvetica with real metrics, colour fills,
// strokes, embedded raster logos (PNG/JPEG) and a top-left coordinate system so
// report layout code reads naturally. It is the shared rendering core behind
// every PDF the ERP issues (generic report exports, romaneio, DANFE, …) so they
// look consistent — header band, zebra rows, totals and a paginated footer.
package pdfkit

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// A4 portrait geometry, in points.
const (
	A4Width  = 595.28
	A4Height = 841.89
)

// Color is an 8-bit-per-channel RGB colour.
type Color struct{ R, G, B uint8 }

func (c Color) op(stroke bool) string {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	verb := "rg"
	if stroke {
		verb = "RG"
	}
	return fmt.Sprintf("%s %s %s %s", f3(r), f3(g), f3(b), verb)
}

// Common palette helpers.
var (
	White = Color{255, 255, 255}
	Black = Color{0, 0, 0}
)

// Doc is a document under construction. Pages share the same width/height.
type Doc struct {
	width, height float64
	pages         []*Page
	images        []*Image // embedded XObjects, in registration order
	footer        func(p *Page, pageNum, pageCount int)
}

// New starts an A4 portrait document.
func New() *Doc { return &Doc{width: A4Width, height: A4Height} }

// Size reports the page dimensions in points.
func (d *Doc) Size() (w, h float64) { return d.width, d.height }

// SetFooter registers a callback invoked once per page at render time, after the
// total page count is known — ideal for "Página X de Y".
func (d *Doc) SetFooter(fn func(p *Page, pageNum, pageCount int)) { d.footer = fn }

// AddPage appends a blank page and returns it for drawing.
func (d *Doc) AddPage() *Page {
	p := &Page{doc: d}
	d.pages = append(d.pages, p)
	return p
}

// Page accumulates content operators. Coordinates are top-left based: y grows
// downward from the top edge, which matches how documents are laid out.
type Page struct {
	doc *Doc
	buf bytes.Buffer
}

// W and H expose the page dimensions.
func (p *Page) W() float64 { return p.doc.width }
func (p *Page) H() float64 { return p.doc.height }

// y converts a top-left y into PDF's bottom-left space.
func (p *Page) y(top float64) float64 { return p.doc.height - top }

// Text draws s with its baseline at (x, baselineTop), left-aligned.
func (p *Page) Text(x, baselineTop float64, font Font, size float64, c Color, s string) {
	fmt.Fprintf(&p.buf, "%s\nBT /%s %s Tf 1 0 0 1 %s %s Tm (%s) Tj ET\n",
		c.op(false), font.resAlias(), f2(size), f2(x), f2(p.y(baselineTop)), pdfString(s))
}

// TextRight draws s so it ends at xRight.
func (p *Page) TextRight(xRight, baselineTop float64, font Font, size float64, c Color, s string) {
	p.Text(xRight-TextWidth(font, size, s), baselineTop, font, size, c, s)
}

// TextCenter draws s centred on xCenter.
func (p *Page) TextCenter(xCenter, baselineTop float64, font Font, size float64, c Color, s string) {
	p.Text(xCenter-TextWidth(font, size, s)/2, baselineTop, font, size, c, s)
}

// FillRect fills the rectangle whose top-left corner is (x, top).
func (p *Page) FillRect(x, top, w, h float64, c Color) {
	fmt.Fprintf(&p.buf, "%s\n%s %s %s %s re f\n",
		c.op(false), f2(x), f2(p.y(top)-h), f2(w), f2(h))
}

// StrokeLine strokes a line between two top-left points.
func (p *Page) StrokeLine(x1, top1, x2, top2, lineW float64, c Color) {
	fmt.Fprintf(&p.buf, "%s\n%s w %s %s m %s %s l S\n",
		c.op(true), f2(lineW), f2(x1), f2(p.y(top1)), f2(x2), f2(p.y(top2)))
}

// StrokeRect strokes the outline of a rectangle with top-left corner (x, top).
func (p *Page) StrokeRect(x, top, w, h, lineW float64, c Color) {
	fmt.Fprintf(&p.buf, "%s\n%s w %s %s %s %s re S\n",
		c.op(true), f2(lineW), f2(x), f2(p.y(top)-h), f2(w), f2(h))
}

// DrawImage places a previously added image (by handle) in the box whose
// top-left corner is (x, top).
func (p *Page) DrawImage(img *Image, x, top, w, h float64) {
	if img == nil {
		return
	}
	fmt.Fprintf(&p.buf, "q %s 0 0 %s %s %s cm /%s Do Q\n",
		f2(w), f2(h), f2(x), f2(p.y(top)-h), img.alias)
}

// Render assembles the final PDF bytes.
func (d *Doc) Render() []byte {
	// Stamp footers now that the total page count is final.
	if d.footer != nil {
		for i, pg := range d.pages {
			d.footer(pg, i+1, len(d.pages))
		}
	}

	var buf bytes.Buffer
	var offsets []int
	buf.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")

	addObj := func(body string) int {
		offsets = append(offsets, buf.Len())
		num := len(offsets)
		buf.WriteString(strconv.Itoa(num) + " 0 obj\n")
		buf.WriteString(body)
		buf.WriteString("\nendobj\n")
		return num
	}
	addStream := func(dict, stream string) int {
		offsets = append(offsets, buf.Len())
		num := len(offsets)
		buf.WriteString(strconv.Itoa(num) + " 0 obj\n")
		buf.WriteString(dict + "\nstream\n")
		buf.WriteString(stream)
		buf.WriteString("\nendstream\nendobj\n")
		return num
	}
	addBinStream := func(dict string, data []byte) int {
		offsets = append(offsets, buf.Len())
		num := len(offsets)
		buf.WriteString(strconv.Itoa(num) + " 0 obj\n")
		buf.WriteString(dict + "\nstream\n")
		buf.Write(data)
		buf.WriteString("\nendstream\nendobj\n")
		return num
	}

	// Reserve object numbers: 1 Catalog, 2 Pages, 3-5 fonts.
	const catalog, pagesObj = 1, 2
	_ = catalog

	// 1 Catalog
	addObj("<< /Type /Catalog /Pages 2 0 R >>")

	// 2 Pages — kids filled after we know page object numbers. We emit a
	// placeholder length now and patch via a two-pass approach: compute kids
	// first by reserving page object numbers.
	// Fonts occupy 3,4,5; images follow; then pages + contents.
	fontBase := 3
	imageBase := fontBase + 3
	pageBase := imageBase + len(d.images)

	kids := make([]string, len(d.pages))
	for i := range d.pages {
		kids[i] = strconv.Itoa(pageBase+i*2) + " 0 R"
	}
	addObj(fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>",
		len(d.pages), strings.Join(kids, " ")))

	// 3-5 Fonts.
	addObj("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	addObj("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")
	addObj("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Oblique /Encoding /WinAnsiEncoding >>")

	// Images.
	xobjEntries := ""
	for _, img := range d.images {
		dict := fmt.Sprintf(
			"<< /Type /XObject /Subtype /Image /Width %d /Height %d "+
				"/ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /%s /Length %d >>",
			img.w, img.h, img.filter, len(img.data))
		addBinStream(dict, img.data)
		xobjEntries += fmt.Sprintf("/%s %d 0 R ", img.alias, imageBase+imgIndex(d.images, img))
	}

	resources := "<< /Font << /F1 3 0 R /F2 4 0 R /F3 5 0 R >>"
	if xobjEntries != "" {
		resources += " /XObject << " + xobjEntries + ">>"
	}
	resources += " >>"

	mediaBox := fmt.Sprintf("[0 0 %s %s]", f2(d.width), f2(d.height))
	for i, pg := range d.pages {
		contentNum := pageBase + i*2 + 1
		addObj(fmt.Sprintf(
			"<< /Type /Page /Parent %d 0 R /MediaBox %s /Resources %s /Contents %d 0 R >>",
			pagesObj, mediaBox, resources, contentNum))
		stream := pg.buf.String()
		addStream(fmt.Sprintf("<< /Length %d >>", len(stream)), stream)
	}

	// Cross-reference table.
	xrefStart := buf.Len()
	n := len(offsets) + 1
	buf.WriteString("xref\n")
	fmt.Fprintf(&buf, "0 %d\n", n)
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	buf.WriteString("trailer\n")
	fmt.Fprintf(&buf, "<< /Size %d /Root 1 0 R >>\n", n)
	buf.WriteString("startxref\n")
	buf.WriteString(strconv.Itoa(xrefStart))
	buf.WriteString("\n%%EOF\n")
	return buf.Bytes()
}

func imgIndex(imgs []*Image, target *Image) int {
	for i, im := range imgs {
		if im == target {
			return i
		}
	}
	return 0
}

// f2/f3 format floats with 2/3 decimals without exponent.
func f2(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
func f3(v float64) string { return strconv.FormatFloat(v, 'f', 3, 64) }
