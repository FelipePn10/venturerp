// Package gantt renders the APS monthly production-schedule board (entity.GanttMonth)
// as a printable SVG or PDF. It is the visual counterpart of the JSON endpoint:
// resource/order lanes on the left, the month's days across the top, scheduled and
// fallback bars on the timeline, capacity-load shading, non-working-day bands, a
// "today" marker and a colour legend. PDF output uses the shared pdfkit engine so
// it matches the rest of the ERP's documents.
package gantt

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
)

// Branding is the optional letterhead applied to the board.
type Branding struct {
	CompanyName   string
	BrandColorHex string // #RRGGBB; defaults to corporate navy when empty/invalid
	GeneratedAt   time.Time
}

// Render draws the board in the requested format ("svg" or "pdf"). It is pure:
// given the board it returns the file bytes and MIME type.
func Render(m *entity.GanttMonth, format string, b Branding) ([]byte, string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "svg":
		return []byte(renderSVG(m, b)), "image/svg+xml", nil
	case "pdf":
		return renderPDF(m, b), "application/pdf", nil
	default:
		return nil, "", fmt.Errorf("unsupported gantt format %q (svg|pdf)", format)
	}
}

// ─── shared helpers ───────────────────────────────────────────────────────────

var monthsPT = [...]string{"", "Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho",
	"Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro"}

func monthTitle(m *entity.GanttMonth) string {
	scope := "por centro de trabalho"
	if m.GroupBy == entity.GroupByOrder {
		scope = "por ordem de produção"
	}
	// A month range carries Year/Month; an arbitrary range shows its date span.
	if m.Month >= 1 && m.Month <= 12 && m.Year > 0 {
		return fmt.Sprintf("Programação de Produção — %s/%d (%s)", monthsPT[m.Month], m.Year, scope)
	}
	period := m.RangeFrom.Format("02/01/2006") + " a " + m.RangeTo.AddDate(0, 0, -1).Format("02/01/2006")
	unit := "diário"
	if m.Scale == entity.ScaleWeek {
		unit = "semanal"
	}
	return fmt.Sprintf("Programação de Produção — %s · %s (%s)", period, unit, scope)
}

// colHeaderLabels returns the two header captions for a board column: a small top
// caption and a bold bottom caption, adapted to the board scale.
func colHeaderLabels(m *entity.GanttMonth, d entity.GanttDay) (string, string) {
	if m.Scale == entity.ScaleWeek {
		return "S" + strconv.Itoa(d.Day), d.Label
	}
	return weekdayLetterPT(int(d.Weekday)), strconv.Itoa(d.Day)
}

// frac maps an instant to its [0,1] position across the month window, clamped.
func frac(m *entity.GanttMonth, t time.Time) float64 {
	total := m.RangeTo.Sub(m.RangeFrom).Seconds()
	if total <= 0 {
		return 0
	}
	f := t.Sub(m.RangeFrom).Seconds() / total
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// loadIndex keys the capacity load by work center and day-of-month so a lane can
// look up its own daily load in O(1).
func loadIndex(m *entity.GanttMonth) map[int64]map[int]*entity.GanttResourceLoad {
	idx := map[int64]map[int]*entity.GanttResourceLoad{}
	for _, l := range m.Load {
		byDay := idx[l.WorkCenterID]
		if byDay == nil {
			byDay = map[int]*entity.GanttResourceLoad{}
			idx[l.WorkCenterID] = byDay
		}
		byDay[l.Date.Day()] = l
	}
	return idx
}

// loadTintHex returns a light background tint for a day cell given its load.
func loadTintHex(loadPct float64) string {
	switch {
	case loadPct > 100:
		return "#fdedec" // light red — overloaded
	case loadPct >= 80:
		return "#fef9e7" // light amber — near capacity
	case loadPct > 0:
		return "#eafaf1" // light green — has load, comfortable
	default:
		return ""
	}
}

// parseHex turns "#RRGGBB" into a pdfkit colour, falling back to corporate navy.
func parseHex(s string) pdfkit.Color {
	def := pdfkit.Color{R: 0x1f, G: 0x3a, B: 0x5f}
	s = strings.TrimSpace(s)
	if len(s) == 7 && s[0] == '#' {
		var r, g, b uint8
		if _, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b); err == nil {
			return pdfkit.Color{R: r, G: g, B: b}
		}
	}
	return def
}

// legendItems is the shared colour key for both renderers.
type legendItem struct {
	hex   string
	label string
}

func legend() []legendItem {
	return []legendItem{
		{"#2f6fb0", "Normal"},
		{"#e67e22", "Prioritária"},
		{"#5dade2", "Baixa prioridade"},
		{"#c0392b", "Em atraso"},
		{"#6b7280", "Concluída"},
	}
}
