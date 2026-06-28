package gantt

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
)

// SVG geometry, in logical pixels.
const (
	svgMarginX = 12
	svgLabelW  = 180
	svgColW    = 34
	svgTitleH  = 58
	svgHeaderH = 46
	svgRowH    = 26
	svgLegendH = 44
)

// svgBarBox is a bar's drawn rectangle, used to route dependency connectors.
type svgBarBox struct {
	x1, x2, yMid float64
}

func renderSVG(m *entity.GanttMonth, b Branding) string {
	n := len(m.Days)
	chartW := float64(n * svgColW)
	gridX := float64(svgMarginX + svgLabelW)
	rowsTop := float64(svgTitleH + svgHeaderH)
	height := rowsTop + float64(len(m.Rows))*svgRowH + svgLegendH
	width := gridX + chartW + svgMarginX

	var s strings.Builder
	fmt.Fprintf(&s, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.0f %.0f" font-family="Helvetica,Arial,sans-serif">`, width, height)
	fmt.Fprint(&s, `<rect x="0" y="0" width="100%" height="100%" fill="#ffffff"/>`)

	// Title band.
	brand := b.BrandColorHex
	if brand == "" {
		brand = "#1f3a5f"
	}
	fmt.Fprintf(&s, `<rect x="0" y="0" width="%.0f" height="%d" fill="%s"/>`, width, svgTitleH, brand)
	title := monthTitle(m)
	if b.CompanyName != "" {
		fmt.Fprintf(&s, `<text x="%d" y="24" font-size="14" font-weight="bold" fill="#ffffff">%s</text>`, svgMarginX, esc(b.CompanyName))
	}
	fmt.Fprintf(&s, `<text x="%d" y="%d" font-size="13" fill="#ffffff">%s</text>`, svgMarginX, svgTitleH-16, esc(title))
	if !b.GeneratedAt.IsZero() {
		fmt.Fprintf(&s, `<text x="%.0f" y="20" font-size="9" fill="#cfe0f5" text-anchor="end">Gerado em %s</text>`,
			width-svgMarginX, b.GeneratedAt.Format("02/01/2006 15:04"))
	}

	// Non-working-day bands (day scale only) + today marker, spanning header+rows.
	bandTop := float64(svgTitleH)
	bandBottom := rowsTop + float64(len(m.Rows))*svgRowH
	todayX := -1.0
	for k, d := range m.Days {
		cx := gridX + float64(k*svgColW)
		if !d.IsWorkday && m.Scale != entity.ScaleWeek {
			fmt.Fprintf(&s, `<rect x="%.0f" y="%.0f" width="%d" height="%.0f" fill="#eef1f5"/>`, cx, bandTop, svgColW, bandBottom-bandTop)
		}
		if d.IsToday {
			todayX = cx + float64(svgColW)/2
		}
	}

	// Column header.
	for k, d := range m.Days {
		cx := gridX + float64(k*svgColW)
		top, bottom := colHeaderLabels(m, d)
		fmt.Fprintf(&s, `<text x="%.0f" y="%d" font-size="8" fill="#7a8699" text-anchor="middle">%s</text>`,
			cx+float64(svgColW)/2, svgTitleH+14, esc(top))
		fmt.Fprintf(&s, `<text x="%.0f" y="%d" font-size="11" fill="#2c3e50" text-anchor="middle">%s</text>`,
			cx+float64(svgColW)/2, svgTitleH+34, esc(bottom))
	}
	// Header bottom rule.
	fmt.Fprintf(&s, `<line x1="0" y1="%.0f" x2="%.0f" y2="%.0f" stroke="#c8d0db" stroke-width="1"/>`, rowsTop, width, rowsTop)

	loads := loadIndex(m)
	barGeo := map[int64]svgBarBox{} // sequence id → drawn box, for dependency arrows

	// Rows.
	for i, row := range m.Rows {
		ry := rowsTop + float64(i*svgRowH)
		if i%2 == 1 {
			fmt.Fprintf(&s, `<rect x="0" y="%.0f" width="%.0f" height="%d" fill="#fafbfc"/>`, ry, width, svgRowH)
		}

		// Capacity-load cell tint (work-center grouping only).
		if byDay, ok := loads[row.ID]; ok {
			for k, d := range m.Days {
				if l := byDay[d.Day]; l != nil {
					if tint := loadTintHex(l.LoadPct); tint != "" {
						cx := gridX + float64(k*svgColW)
						fmt.Fprintf(&s, `<rect x="%.0f" y="%.0f" width="%d" height="%d" fill="%s"/>`, cx, ry+1, svgColW, svgRowH-2, tint)
					}
				}
			}
		}

		// Row label.
		label := row.Label
		fmt.Fprintf(&s, `<text x="%d" y="%.0f" font-size="10" font-weight="bold" fill="#2c3e50">%s</text>`,
			svgMarginX, ry+15, esc(clip(label, 26)))
		if row.SubLabel != "" {
			fmt.Fprintf(&s, `<text x="%d" y="%.0f" font-size="8" fill="#7a8699">%s</text>`, svgMarginX, ry+24, esc(clip(row.SubLabel, 30)))
		}

		// Bars.
		for _, bar := range row.Bars {
			x1 := gridX + frac(m, bar.Start)*chartW
			x2 := gridX + frac(m, bar.End)*chartW
			w := x2 - x1
			if w < 3 {
				w = 3
			}
			by := ry + 4
			bh := float64(svgRowH - 8)
			if bar.SequenceID != 0 {
				barGeo[bar.SequenceID] = svgBarBox{x1: x1, x2: x1 + w, yMid: by + bh/2}
			}
			fmt.Fprintf(&s, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="2" fill="%s" stroke="#33415588" stroke-width="0.5"/>`,
				x1, by, w, bh, bar.ColorHex)
			// Progress overlay.
			if bar.PercentComplete > 0 {
				pw := w * bar.PercentComplete / 100
				fmt.Fprintf(&s, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="2" fill="#00000026"/>`, x1, by, pw, bh)
			}
			// Bar label when wide enough.
			if w > 26 {
				fmt.Fprintf(&s, `<text x="%.1f" y="%.1f" font-size="8" fill="#ffffff">%s</text>`,
					x1+3, by+bh-4, esc(clip(barLabel(bar), int(w/5))))
			}
		}
	}

	// Finish-start dependency connectors (drawn over the bars, under the today line).
	for _, dep := range m.Dependencies {
		from, ok1 := barGeo[dep.FromSequenceID]
		to, ok2 := barGeo[dep.ToSequenceID]
		if !ok1 || !ok2 {
			continue
		}
		stroke, dash := "#94a3b8", ""
		if dep.Implicit {
			stroke, dash = "#cbd5e1", ` stroke-dasharray="3 2"`
		}
		midX := from.x2 + 6
		fmt.Fprintf(&s, `<path d="M %.1f %.1f H %.1f V %.1f H %.1f" fill="none" stroke="%s" stroke-width="1"%s/>`,
			from.x2, from.yMid, midX, to.yMid, to.x1, stroke, dash)
		// Arrowhead pointing into the successor's start.
		fmt.Fprintf(&s, `<path d="M %.1f %.1f l -4 -2.5 l 0 5 z" fill="%s"/>`, to.x1, to.yMid, stroke)
	}

	// Today marker on top.
	if todayX >= 0 {
		fmt.Fprintf(&s, `<line x1="%.1f" y1="%.0f" x2="%.1f" y2="%.0f" stroke="#c0392b" stroke-width="1.5" stroke-dasharray="4 3"/>`,
			todayX, float64(svgTitleH), todayX, bandBottom)
		fmt.Fprintf(&s, `<text x="%.1f" y="%.0f" font-size="8" fill="#c0392b" text-anchor="middle">hoje</text>`, todayX, float64(svgTitleH)+8)
	}

	// Legend.
	lx := float64(svgMarginX)
	ly := bandBottom + 26
	for _, it := range legend() {
		fmt.Fprintf(&s, `<rect x="%.0f" y="%.0f" width="12" height="12" rx="2" fill="%s"/>`, lx, ly-10, it.hex)
		fmt.Fprintf(&s, `<text x="%.0f" y="%.0f" font-size="9" fill="#2c3e50">%s</text>`, lx+16, ly, it.label)
		lx += float64(20 + len(it.label)*6 + 18)
	}

	s.WriteString(`</svg>`)
	return s.String()
}

func barLabel(b *entity.GanttBar) string {
	s := "OF " + strconv.FormatInt(b.OrderNumber, 10)
	if b.OperationName != "" {
		s += " · " + b.OperationName
	} else if b.ItemCode != 0 {
		s += " · It." + strconv.FormatInt(b.ItemCode, 10)
	}
	return s
}

func weekdayLetterPT(w int) string {
	// 0=Sunday .. 6=Saturday
	switch w {
	case 0:
		return "D"
	case 1:
		return "S"
	case 2:
		return "T"
	case 3:
		return "Q"
	case 4:
		return "Q"
	case 5:
		return "S"
	default:
		return "S"
	}
}

func clip(s string, max int) string {
	r := []rune(s)
	if max < 1 {
		return ""
	}
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

func esc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
