package service

import (
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

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
func RenderCutMap(planCode int64, patterns []*entity.CuttingPattern, format MapFormat) (data []byte, contentType string, err error) {
	switch format {
	case MapSVG:
		return []byte(renderSVG(planCode, patterns)), "image/svg+xml", nil
	case MapDXF:
		return []byte(renderDXF(patterns)), "application/dxf", nil
	case MapPDF:
		return renderPDF(planCode, patterns), "application/pdf", nil
	default:
		return nil, "", fmt.Errorf("unsupported map format %q (svg|dxf|pdf)", format)
	}
}

// rect is one drawn rectangle with an optional label, in map coordinates.
type rect struct {
	x, y, w, h float64
	label      string
	outer      bool // the stock outline (vs a placed part)
}

// layout flattens the patterns into stacked stock rectangles + part rectangles,
// returning the rects and the overall canvas size. 1D bars are drawn as a strip.
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
				rects = append(rects, rect{x: mapGap + pl.PosXMM, y: y + pl.PosYMM, w: pl.WidthMM, h: pl.HeightMM, label: pl.Label})
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
		fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s" stroke="%s" stroke-width="%.1f"/>`,
			r.x, r.y, r.w, r.h, fill, stroke, sw)
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
		y0 := h - (r.y + r.h) // flip
		x0, x1 := r.x, r.x+r.w
		y1 := h - r.y
		// closed LWPOLYLINE (90) with 4 vertices
		b.WriteString("0\nLWPOLYLINE\n8\nCUT\n90\n4\n70\n1\n")
		for _, v := range [][2]float64{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}} {
			fmt.Fprintf(&b, "10\n%.3f\n20\n%.3f\n", v[0], v[1])
		}
		if r.label != "" {
			fmt.Fprintf(&b, "0\nTEXT\n8\nLABEL\n10\n%.3f\n20\n%.3f\n40\n%.1f\n1\n%s\n", x0+10, y1-40, 30.0, dxfText(r.label))
		}
	}
	b.WriteString("0\nENDSEC\n0\nEOF\n")
	return b.String()
}

func dxfText(s string) string { return strings.ReplaceAll(s, "\n", " ") }

// renderPDF writes a single-page vector PDF: the map is scaled to fit an A4-ish
// page; rectangles are stroked and labels drawn in Helvetica. Hand-rolled and
// dependency-free, matching the rest of the export stack.
func renderPDF(planCode int64, patterns []*entity.CuttingPattern) []byte {
	rects, w, h := layout(patterns)
	const pageW, pageH = 595.0, 842.0 // A4 in points
	margin := 28.0
	scale := 1.0
	if w > 0 && h > 0 {
		scale = min((pageW-2*margin)/w, (pageH-2*margin)/h)
	}
	tx := func(x float64) float64 { return margin + x*scale }
	ty := func(y float64) float64 { return pageH - margin - y*scale } // PDF Y grows up

	var c strings.Builder
	c.WriteString("0.12 0.23 0.37 RG\n1 w\n")
	for _, r := range rects {
		lw := 1.4
		if !r.outer {
			lw = 0.6
		}
		fmt.Fprintf(&c, "%.2f w\n%.2f %.2f %.2f %.2f re S\n", lw, tx(r.x), ty(r.y+r.h), r.w*scale, r.h*scale)
	}
	c.WriteString("BT /F1 7 Tf 0 0 0 rg\n")
	fmt.Fprintf(&c, "1 0 0 1 %.2f %.2f Tm (Plano de corte #%d) Tj\n", margin, pageH-margin+6, planCode)
	for _, r := range rects {
		if r.label == "" {
			continue
		}
		fmt.Fprintf(&c, "1 0 0 1 %.2f %.2f Tm (%s) Tj\n", tx(r.x)+2, ty(r.y)-9, pdfText(r.label))
	}
	c.WriteString("ET\n")
	return assemblePDF(c.String())
}

func pdfText(s string) string {
	r := strings.NewReplacer("\\", `\\`, "(", `\(`, ")", `\)`, "\n", " ")
	return r.Replace(s)
}

// assemblePDF wraps a content stream into a minimal valid one-page PDF, tracking
// byte offsets for the xref table.
func assemblePDF(content string) []byte {
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content)+1, content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
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
