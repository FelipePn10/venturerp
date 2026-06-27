package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

// OutlineForPlacement parses a true-shape part's polygon geometry (JSON [{x,y},…])
// and returns its contour rotated by rotationDeg, normalised to the bounding-box
// origin (points in [0,W]×[0,H]) — the shape the cutting-map renderer draws for that
// placement. Returns false for missing/degenerate geometry (the caller then keeps the
// bounding rectangle).
func OutlineForPlacement(geometryJSON string, rotationDeg float64) ([][2]float64, bool) {
	if geometryJSON == "" {
		return nil, false
	}
	var poly []Point
	if err := json.Unmarshal([]byte(geometryJSON), &poly); err != nil || len(poly) < 3 {
		return nil, false
	}
	r := rotatePoly(poly, rotationDeg)
	out := make([][2]float64, len(r))
	for i, p := range r {
		out[i] = [2]float64{p.X, p.Y}
	}
	return out, true
}

// MapBranding is the optional letterhead applied to the PDF cutting map. All
// fields are plain values so the domain stays free of infrastructure deps; the
// caller fills CompanyName/BrandColorHex from the company's fiscal config.
type MapBranding struct {
	CompanyName   string
	BrandColorHex string // #RRGGBB; defaults to corporate navy when empty/invalid
	GeneratedAt   time.Time
}

// MapFormat is a supported cutting-map export format.
type MapFormat string

const (
	MapSVG MapFormat = "svg"
	MapDXF MapFormat = "dxf"
	MapPDF MapFormat = "pdf"
)

// bar1DHeight is the drawn height (mm) of a 1D bar in the map.
const bar1DHeight = 200.0
const mapGap = 120.0 // vertical gap between patterns (mm)

// RenderCutMap draws the plan's patterns as a cutting map in the requested vector
// format (SVG / DXF / PDF). It is pure: given the patterns it returns the file
// bytes and MIME type, with no persistence or HTTP knowledge.
func RenderCutMap(planCode int64, patterns []*entity.CuttingPattern, format MapFormat, b MapBranding) (data []byte, contentType string, err error) {
	switch format {
	case MapSVG:
		return []byte(renderSVG(planCode, patterns)), "image/svg+xml", nil
	case MapDXF:
		return []byte(renderDXF(patterns)), "application/dxf", nil
	case MapPDF:
		return renderPDF(planCode, patterns, b), "application/pdf", nil
	default:
		return nil, "", fmt.Errorf("unsupported map format %q (svg|dxf|pdf)", format)
	}
}

// rect is one drawn shape with an optional label, in map coordinates. When poly is
// set (≥3 points, absolute map coords) the renderers draw that polygon — the real
// true-shape contour — instead of the x/y/w/h rectangle.
type rect struct {
	x, y, w, h float64
	poly       [][2]float64
	label      string
	outer      bool // the stock outline (vs a placed part)
}

// layout flattens the patterns into stacked stock rectangles + part shapes, returning
// them and the overall canvas size. 1D bars are drawn as a strip; true-shape parts
// carrying an Outline are drawn as polygons.
func layout(patterns []*entity.CuttingPattern) (rects []rect, width, height float64) {
	y := mapGap
	for _, p := range patterns {
		sw, sh := p.StockWidthMM, p.StockHeightMM
		is2D := sw > 0 && sh > 0
		if !is2D {
			sw, sh = p.StockLengthMM, bar1DHeight
		}
		if sw <= 0 {
			continue
		}
		rects = append(rects, rect{x: mapGap, y: y, w: sw, h: sh, outer: true,
			label: fmt.Sprintf("Padrão %d  ×%d  (%.0f%%)", p.Sequence, p.RepeatCount, p.UtilizationPct)})
		for _, pl := range p.Placements {
			if is2D {
				r := rect{x: mapGap + pl.PosXMM, y: y + pl.PosYMM, w: pl.WidthMM, h: pl.HeightMM, label: pl.Label}
				if len(pl.Outline) >= 3 { // draw the real contour, offset to its placement
					r.poly = make([][2]float64, len(pl.Outline))
					for i, pt := range pl.Outline {
						r.poly[i] = [2]float64{mapGap + pl.PosXMM + pt[0], y + pl.PosYMM + pt[1]}
					}
				}
				rects = append(rects, r)
			} else {
				rects = append(rects, rect{x: mapGap + pl.OffsetMM, y: y, w: pl.LengthMM, h: sh, label: pl.Label})
			}
		}
		if mapGap+sw > width {
			width = mapGap + sw
		}
		y += sh + mapGap
	}
	return rects, width + mapGap, y
}

func renderSVG(planCode int64, patterns []*entity.CuttingPattern) string {
	rects, w, h := layout(patterns)
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.0f %.0f" font-family="sans-serif">`, w, h)
	fmt.Fprintf(&b, `<text x="%.0f" y="%.0f" font-size="40">Plano de corte #%d</text>`, mapGap, mapGap*0.6, planCode)
	for _, r := range rects {
		fill, stroke, sw := "#f4f7fb", "#1f3a5f", 4.0
		if !r.outer {
			fill, stroke, sw = "#cfe3ff", "#2f6fb0", 2.0
		}
		if len(r.poly) >= 3 {
			var pts strings.Builder
			for _, pt := range r.poly {
				fmt.Fprintf(&pts, "%.2f,%.2f ", pt[0], pt[1])
			}
			fmt.Fprintf(&b, `<polygon points="%s" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
				strings.TrimSpace(pts.String()), fill, stroke, sw)
		} else {
			fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
				r.x, r.y, r.w, r.h, fill, stroke, sw)
		}
		if r.label != "" {
			fs := 28.0
			if r.outer {
				fs = 34.0
			}
			fmt.Fprintf(&b, `<text x="%.2f" y="%.2f" font-size="%.0f" fill="#0a0a0a">%s</text>`,
				r.x+10, r.y+fs+6, fs, svgEscape(r.label))
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func svgEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// renderDXF emits a minimal ASCII DXF (R12 ENTITIES) with a closed polyline per
// rectangle and a TEXT per label — enough for CAM/CAD to read the layout. DXF Y
// grows upward, so we flip Y against the canvas height.
func renderDXF(patterns []*entity.CuttingPattern) string {
	rects, _, h := layout(patterns)
	var b strings.Builder
	b.WriteString("0\nSECTION\n2\nENTITIES\n")
	for _, r := range rects {
		// closed LWPOLYLINE: the real contour when present, else the 4-vertex rect.
		verts := [][2]float64{}
		if len(r.poly) >= 3 {
			for _, pt := range r.poly {
				verts = append(verts, [2]float64{pt[0], h - pt[1]}) // flip Y
			}
		} else {
			y0 := h - (r.y + r.h)
			x0, x1 := r.x, r.x+r.w
			y1 := h - r.y
			verts = [][2]float64{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
		}
		fmt.Fprintf(&b, "0\nLWPOLYLINE\n8\nCUT\n90\n%d\n70\n1\n", len(verts))
		for _, v := range verts {
			fmt.Fprintf(&b, "10\n%.3f\n20\n%.3f\n", v[0], v[1])
		}
		x0, y1 := r.x, h-r.y
		if r.label != "" {
			fmt.Fprintf(&b, "0\nTEXT\n8\nLABEL\n10\n%.3f\n20\n%.3f\n40\n%.1f\n1\n%s\n", x0+10, y1-40, 30.0, dxfText(r.label))
		}
	}
	b.WriteString("0\nENDSEC\n0\nEOF\n")
	return b.String()
}

func dxfText(s string) string { return strings.ReplaceAll(s, "\n", " ") }

// renderPDF writes a single-page vector PDF: a branded header band, the cutting
// map scaled to fit the body, and a footer with the generation stamp. Stroked
// rectangles and Helvetica labels; hand-rolled and dependency-free, matching the
// rest of the export stack.
func renderPDF(planCode int64, patterns []*entity.CuttingPattern, b MapBranding) []byte {
	rects, w, h := layout(patterns)
	const pageW, pageH = 595.0, 842.0 // A4 in points
	const margin = 28.0
	const headerH = 48.0
	const footerH = 26.0

	br, bg, bb := parseHexRGB(b.BrandColorHex)
	bodyTop := margin + headerH + 14 // first usable y (from top) below the header
	bodyBottom := pageH - footerH    // last usable y (PDF space, from bottom)
	bodyH := (pageH - bodyTop) - footerH
	bodyW := pageW - 2*margin

	scale := 1.0
	if w > 0 && h > 0 {
		scale = min(bodyW/w, bodyH/h)
	}
	tx := func(x float64) float64 { return margin + x*scale }
	ty := func(y float64) float64 { return (pageH - bodyTop) - y*scale } // map top at bodyTop

	var c strings.Builder

	// Header band.
	fmt.Fprintf(&c, "%.3f %.3f %.3f rg\n%.2f %.2f %.2f %.2f re f\n",
		br, bg, bb, margin, pageH-margin-headerH, pageW-2*margin, headerH)
	c.WriteString("BT 1 1 1 rg\n")
	if b.CompanyName != "" {
		fmt.Fprintf(&c, "/F2 13 Tf 1 0 0 1 %.2f %.2f Tm (%s) Tj\n", margin+12, pageH-margin-22, pdfText(b.CompanyName))
	}
	fmt.Fprintf(&c, "/F2 11 Tf 1 0 0 1 %.2f %.2f Tm (PLANO DE CORTE #%d) Tj\n", margin+12, pageH-margin-38, planCode)
	fmt.Fprintf(&c, "/F1 8 Tf 1 0 0 1 %.2f %.2f Tm (%d padroes) Tj\n", pageW-margin-90, pageH-margin-22, len(patterns))
	c.WriteString("ET\n")

	// Map rectangles.
	fmt.Fprintf(&c, "%.3f %.3f %.3f RG\n", br, bg, bb)
	for _, r := range rects {
		lw := 1.4
		if !r.outer {
			lw = 0.6
		}
		if len(r.poly) >= 3 {
			fmt.Fprintf(&c, "%.2f w\n", lw)
			for i, pt := range r.poly {
				op := "l"
				if i == 0 {
					op = "m"
				}
				fmt.Fprintf(&c, "%.2f %.2f %s\n", tx(pt[0]), ty(pt[1]), op)
			}
			c.WriteString("h S\n")
		} else {
			fmt.Fprintf(&c, "%.2f w\n%.2f %.2f %.2f %.2f re S\n", lw, tx(r.x), ty(r.y+r.h), r.w*scale, r.h*scale)
		}
	}

	// Part labels (inside each placement, near the top-left).
	c.WriteString("BT /F1 7 Tf 0.13 0.15 0.16 rg\n")
	for _, r := range rects {
		if r.label == "" || r.outer {
			continue
		}
		fmt.Fprintf(&c, "1 0 0 1 %.2f %.2f Tm (%s) Tj\n", tx(r.x)+2, ty(r.y)-9, pdfText(r.label))
	}
	c.WriteString("ET\n")

	// Stock/pattern labels, bold, just above each stock outline so they never
	// collide with the first part's label.
	fmt.Fprintf(&c, "BT /F2 7.5 Tf %.3f %.3f %.3f rg\n", br, bg, bb)
	for _, r := range rects {
		if r.label == "" || !r.outer {
			continue
		}
		fmt.Fprintf(&c, "1 0 0 1 %.2f %.2f Tm (%s) Tj\n", tx(r.x), ty(r.y)+3, pdfText(r.label))
	}
	c.WriteString("ET\n")

	// Footer.
	fmt.Fprintf(&c, "0.78 0.80 0.82 RG\n0.5 w\n%.2f %.2f m %.2f %.2f l S\n",
		margin, bodyBottom, pageW-margin, bodyBottom)
	gen := b.GeneratedAt
	if gen.IsZero() {
		gen = time.Now()
	}
	c.WriteString("BT /F1 7.5 Tf 0.42 0.46 0.49 rg\n")
	fmt.Fprintf(&c, "1 0 0 1 %.2f %.2f Tm (Gerado em %s) Tj\n", margin, bodyBottom-12, pdfText(gen.Format("02/01/2006 15:04")))
	fmt.Fprintf(&c, "1 0 0 1 %.2f %.2f Tm (Plano #%d) Tj\n", pageW-margin-70, bodyBottom-12, planCode)
	c.WriteString("ET\n")

	return assemblePDF(c.String())
}

// parseHexRGB parses "#RRGGBB" into 0..1 RGB components, defaulting to the
// corporate navy (#1B3A5B) on empty or malformed input.
func parseHexRGB(s string) (r, g, b float64) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return 0.106, 0.227, 0.357
	}
	var v [3]int64
	for i := 0; i < 3; i++ {
		n, err := parseHexByte(s[i*2 : i*2+2])
		if err != nil {
			return 0.106, 0.227, 0.357
		}
		v[i] = n
	}
	return float64(v[0]) / 255, float64(v[1]) / 255, float64(v[2]) / 255
}

func parseHexByte(s string) (int64, error) {
	var n int64
	for i := 0; i < len(s); i++ {
		c := s[i]
		var d int64
		switch {
		case c >= '0' && c <= '9':
			d = int64(c - '0')
		case c >= 'a' && c <= 'f':
			d = int64(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = int64(c-'A') + 10
		default:
			return 0, fmt.Errorf("bad hex")
		}
		n = n*16 + d
	}
	return n, nil
}

// pdfText escapes PDF literal syntax and re-encodes the string to single WinAnsi
// bytes so Portuguese accents render correctly under WinAnsiEncoding.
func pdfText(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '(':
			b.WriteString(`\(`)
		case ')':
			b.WriteString(`\)`)
		case '\n', '\r':
			b.WriteByte(' ')
		default:
			b.WriteByte(winAnsiByte(r))
		}
	}
	return b.String()
}

// winAnsiByte maps a rune to its Windows-1252 byte. Latin-1 (0xA0–0xFF) — which
// covers áàâãéêíóôõúç and uppercase — coincides with WinAnsi; other runes fall
// back to '?'.
func winAnsiByte(r rune) byte {
	switch {
	case r < 0x80:
		return byte(r)
	case r >= 0xA0 && r <= 0xFF:
		return byte(r)
	}
	switch r {
	case '–', '—':
		return 0x2D
	case '•':
		return 0x95
	case '“', '”':
		return 0x22
	case '‘', '’':
		return 0x27
	}
	return '?'
}

// assemblePDF wraps a content stream into a minimal valid one-page PDF, tracking
// byte offsets for the xref table.
func assemblePDF(content string) []byte {
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 5 0 R /F2 6 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content)+1, content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>",
	}
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs)+1)
	for i, o := range objs {
		offsets[i+1] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for i := 1; i <= len(objs); i++ {
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objs)+1, xref)
	return []byte(b.String())
}
