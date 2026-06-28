package gantt

import (
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
)

// PDF geometry, in points (A4 landscape).
const (
	pdfMargin  = 24.0
	pdfLabelW  = 132.0
	pdfTitleH  = 50.0
	pdfHeaderH = 30.0
	pdfRowH    = 16.0
	pdfLegendH = 26.0
	pdfFooterH = 18.0
)

type pdfGeo struct {
	pageW, pageH float64
	gridX        float64 // left edge of the day grid
	chartW       float64
	rowsTop      float64 // top of the first row
	rowsBottom   float64 // bottom limit for rows on a page
	colW         float64
}

var (
	colGridLine = pdfkit.Color{R: 0xc8, G: 0xd0, B: 0xdb}
	colWeekend  = pdfkit.Color{R: 0xee, G: 0xf1, B: 0xf5}
	colZebra    = pdfkit.Color{R: 0xfa, G: 0xfb, B: 0xfc}
	colMuted    = pdfkit.Color{R: 0x7a, G: 0x86, B: 0x99}
	colInk      = pdfkit.Color{R: 0x2c, G: 0x3e, B: 0x50}
	colToday    = pdfkit.Color{R: 0xc0, G: 0x39, B: 0x2c}
	colLoadHigh = pdfkit.Color{R: 0xfd, G: 0xed, B: 0xec}
	colLoadMed  = pdfkit.Color{R: 0xfe, G: 0xf9, B: 0xe7}
	colLoadLow  = pdfkit.Color{R: 0xea, G: 0xfa, B: 0xf1}

	colDepReal     = pdfkit.Color{R: 0x94, G: 0xa3, B: 0xb8}
	colDepImplicit = pdfkit.Color{R: 0xcb, G: 0xd5, B: 0xe1}
)

func renderPDF(m *entity.GanttMonth, b Branding) []byte {
	doc := pdfkit.NewLandscape()
	pageW, pageH := doc.Size()

	n := len(m.Days)
	if n == 0 {
		n = 1
	}
	gridX := pdfMargin + pdfLabelW
	chartW := pageW - gridX - pdfMargin
	g := pdfGeo{
		pageW:      pageW,
		pageH:      pageH,
		gridX:      gridX,
		chartW:     chartW,
		colW:       chartW / float64(n),
		rowsTop:    pdfTitleH + pdfHeaderH,
		rowsBottom: pageH - pdfMargin - pdfLegendH - pdfFooterH,
	}

	rowsPerPage := int((g.rowsBottom - g.rowsTop) / pdfRowH)
	if rowsPerPage < 1 {
		rowsPerPage = 1
	}

	rows := m.Rows
	if len(rows) == 0 {
		p := doc.AddPage()
		drawBackdrop(p, m, b, g)
		p.Text(g.gridX, g.rowsTop+24, pdfkit.FontOblique, 11, colMuted, "Nenhuma ordem programada neste mês.")
		drawLegend(p, g)
	}
	loads := loadIndex(m)
	for start := 0; start < len(rows); start += rowsPerPage {
		end := start + rowsPerPage
		if end > len(rows) {
			end = len(rows)
		}
		p := doc.AddPage()
		drawBackdrop(p, m, b, g)
		drawRows(p, m, rows[start:end], loads, g)
		drawLegend(p, g)
	}

	gen := b.GeneratedAt
	doc.SetFooter(func(p *pdfkit.Page, pageNum, pageCount int) {
		y := g.pageH - 8
		p.Text(pdfMargin, y, pdfkit.FontRegular, 7, colMuted, "Quadro de Programação de Produção")
		if !gen.IsZero() {
			p.TextCenter(g.pageW/2, y, pdfkit.FontRegular, 7, colMuted, "Gerado em "+gen.Format("02/01/2006 15:04"))
		}
		p.TextRight(g.pageW-pdfMargin, y, pdfkit.FontRegular, 7, colMuted, fmt.Sprintf("Página %d de %d", pageNum, pageCount))
	})

	return doc.Render()
}

// drawBackdrop paints the title band, day header, weekend bands and today marker
// — everything that repeats on every page.
func drawBackdrop(p *pdfkit.Page, m *entity.GanttMonth, b Branding, g pdfGeo) {
	// Title band.
	brand := parseHex(b.BrandColorHex)
	p.FillRect(0, 0, g.pageW, pdfTitleH, brand)
	if b.CompanyName != "" {
		p.Text(pdfMargin, 22, pdfkit.FontBold, 13, pdfkit.White, clip(b.CompanyName, 60))
	}
	p.Text(pdfMargin, pdfTitleH-14, pdfkit.FontRegular, 11, pdfkit.White, monthTitle(m))

	rowsRegionBottom := g.rowsBottom
	// Weekend / non-working-day bands + today marker.
	todayX := -1.0
	for k, d := range m.Days {
		cx := g.gridX + float64(k)*g.colW
		if !d.IsWorkday && m.Scale != entity.ScaleWeek {
			p.FillRect(cx, pdfTitleH, g.colW, rowsRegionBottom-pdfTitleH, colWeekend)
		}
		if d.IsToday {
			todayX = cx + g.colW/2
		}
		// Column header text.
		top, bottom := colHeaderLabels(m, d)
		p.TextCenter(cx+g.colW/2, pdfTitleH+11, pdfkit.FontRegular, 6, colMuted, top)
		p.TextCenter(cx+g.colW/2, pdfTitleH+24, pdfkit.FontBold, 8, colInk, bottom)
	}

	// Header bottom rule.
	p.StrokeLine(0, g.rowsTop, g.pageW, g.rowsTop, 0.6, colGridLine)

	if todayX >= 0 {
		p.StrokeLine(todayX, pdfTitleH, todayX, rowsRegionBottom, 1.0, colToday)
		p.TextCenter(todayX, pdfTitleH+6, pdfkit.FontBold, 6, colToday, "hoje")
	}
}

type pdfBox struct {
	x1, x2, yMid float64
}

func drawRows(p *pdfkit.Page, m *entity.GanttMonth, rows []*entity.GanttRow, loads map[int64]map[int]*entity.GanttResourceLoad, g pdfGeo) {
	geo := map[int64]pdfBox{} // sequence id → drawn box on this page
	for i, row := range rows {
		ry := g.rowsTop + float64(i)*pdfRowH
		if i%2 == 1 {
			p.FillRect(0, ry, g.pageW, pdfRowH, colZebra)
		}

		// Capacity-load cell tint (work-center grouping).
		if byDay, ok := loads[row.ID]; ok {
			for k, d := range m.Days {
				if l := byDay[d.Day]; l != nil {
					if c, ok := loadTintColor(l.LoadPct); ok {
						cx := g.gridX + float64(k)*g.colW
						p.FillRect(cx, ry+1, g.colW, pdfRowH-2, c)
					}
				}
			}
		}

		// Row label.
		p.Text(pdfMargin, ry+10, pdfkit.FontBold, 7.5, colInk, clip(row.Label, 24))
		if row.SubLabel != "" {
			p.Text(pdfMargin, ry+15.5, pdfkit.FontRegular, 6, colMuted, clip(row.SubLabel, 28))
		}

		// Bars.
		for _, bar := range row.Bars {
			x1 := g.gridX + frac(m, bar.Start)*g.chartW
			x2 := g.gridX + frac(m, bar.End)*g.chartW
			w := x2 - x1
			if w < 2 {
				w = 2
			}
			by := ry + 2.5
			bh := pdfRowH - 5
			if bar.SequenceID != 0 {
				geo[bar.SequenceID] = pdfBox{x1: x1, x2: x1 + w, yMid: by + bh/2}
			}
			fill := parseHex(bar.ColorHex)
			p.FillRect(x1, by, w, bh, fill)
			if bar.PercentComplete > 0 {
				p.FillRect(x1, by, w*bar.PercentComplete/100, bh, darken(fill, 0.7))
			}
			p.StrokeRect(x1, by, w, bh, 0.4, colGridLine)
			if w > 24 {
				p.Text(x1+2, by+bh-2, pdfkit.FontRegular, 6, pdfkit.White, clip(barLabel(bar), int(w/4)))
			}
		}
	}

	// Finish-start dependency connectors between bars that landed on this page.
	for _, dep := range m.Dependencies {
		from, ok1 := geo[dep.FromSequenceID]
		to, ok2 := geo[dep.ToSequenceID]
		if !ok1 || !ok2 {
			continue
		}
		col := colDepReal
		if dep.Implicit {
			col = colDepImplicit
		}
		midX := from.x2 + 3
		p.StrokeLine(from.x2, from.yMid, midX, from.yMid, 0.5, col)
		p.StrokeLine(midX, from.yMid, midX, to.yMid, 0.5, col)
		p.StrokeLine(midX, to.yMid, to.x1, to.yMid, 0.5, col)
		// Small arrowhead caret pointing into the successor's start.
		p.StrokeLine(to.x1, to.yMid, to.x1-2.5, to.yMid-1.5, 0.5, col)
		p.StrokeLine(to.x1, to.yMid, to.x1-2.5, to.yMid+1.5, 0.5, col)
	}
}

func drawLegend(p *pdfkit.Page, g pdfGeo) {
	x := pdfMargin
	y := g.rowsBottom + 14
	for _, it := range legend() {
		p.FillRect(x, y-7, 9, 9, parseHex(it.hex))
		p.Text(x+13, y, pdfkit.FontRegular, 7, colInk, it.label)
		x += 13 + float64(len(it.label))*4.2 + 16
	}
}

func loadTintColor(loadPct float64) (pdfkit.Color, bool) {
	switch {
	case loadPct > 100:
		return colLoadHigh, true
	case loadPct >= 80:
		return colLoadMed, true
	case loadPct > 0:
		return colLoadLow, true
	default:
		return pdfkit.Color{}, false
	}
}

func darken(c pdfkit.Color, f float64) pdfkit.Color {
	return pdfkit.Color{R: uint8(float64(c.R) * f), G: uint8(float64(c.G) * f), B: uint8(float64(c.B) * f)}
}
