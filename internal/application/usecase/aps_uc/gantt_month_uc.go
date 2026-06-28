package aps_uc

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
)

// nowFunc is overridable in tests so "today"/lateness is deterministic.
var nowFunc = time.Now

// unsequencedRowID is the synthetic work-center row that collects bars plotted
// from order dates (no APS sequence, hence no work center).
const unsequencedRowID int64 = 0

// ParseGroupBy normalises the group_by query parameter.
func ParseGroupBy(s string) entity.GanttGroupBy {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "order", "production_order", "of":
		return entity.GroupByOrder
	default:
		return entity.GroupByWorkCenter
	}
}

// GetMonthSchedule builds the monthly board and maps it to the JSON response.
func (uc *APSUseCase) GetMonthSchedule(ctx context.Context, year, month int, groupBy entity.GanttGroupBy) (*response.GanttMonthResponse, error) {
	m, err := uc.BuildMonthSchedule(ctx, year, month, groupBy)
	if err != nil {
		return nil, err
	}
	return monthToResponse(m), nil
}

// GetBoard builds an arbitrary-range board (day/week scale) and maps it to JSON.
func (uc *APSUseCase) GetBoard(ctx context.Context, from, to time.Time, scale entity.GanttScale, groupBy entity.GanttGroupBy) (*response.GanttMonthResponse, error) {
	m, err := uc.BuildBoard(ctx, from, to, scale, groupBy)
	if err != nil {
		return nil, err
	}
	return monthToResponse(m), nil
}

// maxBoardDays guards arbitrary ranges (≈13 months) so a bad request can't ask the
// board to materialise thousands of columns.
const maxBoardDays = 372

// ParseScale normalises the scale query parameter.
func ParseScale(s string) entity.GanttScale {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "week", "semana", "w":
		return entity.ScaleWeek
	default:
		return entity.ScaleDay
	}
}

// BuildMonthSchedule is the month-scoped convenience wrapper over BuildBoard: it
// validates year/month, builds the day-scale board for that calendar month and
// stamps Year/Month on the result.
func (uc *APSUseCase) BuildMonthSchedule(ctx context.Context, year, month int, groupBy entity.GanttGroupBy) (*entity.GanttMonth, error) {
	if month < 1 || month > 12 {
		return nil, fmt.Errorf("invalid month %d (want 1-12)", month)
	}
	if year < 1900 || year > 3000 {
		return nil, fmt.Errorf("invalid year %d", year)
	}
	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	to := from.AddDate(0, 1, 0)
	m, err := uc.BuildBoard(ctx, from, to, entity.ScaleDay, groupBy)
	if err != nil {
		return nil, err
	}
	m.Year = year
	m.Month = month
	return m, nil
}

// BuildBoard assembles the schedule board for an arbitrary [from, to) window at the
// requested scale (day or week): calendar backdrop, scheduled + fallback bars
// grouped into rows, finish-start dependencies, the per-resource capacity load and
// a headline summary. It is the single source of truth shared by the JSON endpoint
// and the SVG/PDF export.
func (uc *APSUseCase) BuildBoard(ctx context.Context, from, to time.Time, scale entity.GanttScale, groupBy entity.GanttGroupBy) (*entity.GanttMonth, error) {
	loc := time.Local
	from = truncDay(from.In(loc))
	to = truncDay(to.In(loc))
	if !to.After(from) {
		return nil, fmt.Errorf("invalid range: 'to' (%s) must be after 'from' (%s)",
			to.Format("2006-01-02"), from.Format("2006-01-02"))
	}
	if to.Sub(from) > maxBoardDays*24*time.Hour {
		return nil, fmt.Errorf("range too large: max %d days", maxBoardDays)
	}
	if scale != entity.ScaleWeek {
		scale = entity.ScaleDay
	}
	if groupBy != entity.GroupByOrder {
		groupBy = entity.GroupByWorkCenter
	}
	today := truncDay(nowFunc().In(loc))

	scheduled, err := uc.repo.ListScheduledBars(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("loading scheduled bars: %w", err)
	}
	fallback, err := uc.repo.ListFallbackBars(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("loading fallback bars: %w", err)
	}

	bars := make([]*entity.GanttBar, 0, len(scheduled)+len(fallback))
	bars = append(bars, scheduled...)
	bars = append(bars, fallback...)

	summary := entity.GanttSummary{}
	for _, b := range bars {
		b.IsLate = isLate(b, today)
		b.ColorHex = resolveBarColor(b)
		if b.IsFallback {
			summary.FallbackBars++
		} else {
			summary.SequencedBars++
		}
		if b.IsLate {
			summary.LateBars++
		}
	}
	summary.TotalBars = len(bars)

	m := &entity.GanttMonth{
		RangeFrom:    from,
		RangeTo:      to,
		Scale:        scale,
		GroupBy:      groupBy,
		Days:         uc.buildColumns(ctx, from, to, scale, today),
		Rows:         groupRows(bars, groupBy),
		Dependencies: uc.buildDependencies(ctx, bars, from, to),
		GeneratedAt:  nowFunc(),
	}
	summary.TotalRows = len(m.Rows)

	if groupBy == entity.GroupByWorkCenter {
		load, lerr := uc.repo.ListResourceLoad(ctx, from, to)
		if lerr == nil {
			m.Load = load
			for _, l := range load {
				if l.IsOverloaded {
					summary.OverloadedDays++
				}
			}
		}
	}

	m.Summary = summary
	return m, nil
}

// buildColumns lays out the board's columns at the chosen scale, marking working
// days from the industrial calendar (Saturday/Sunday fallback when a month has no
// calendar) and flagging today.
func (uc *APSUseCase) buildColumns(ctx context.Context, from, to time.Time, scale entity.GanttScale, today time.Time) []entity.GanttDay {
	workday, covered := uc.workdayLookup(ctx, from, to)
	isWork := func(d time.Time) bool {
		if covered[d.Year()*100+int(d.Month())] {
			return workday[d.Format("2006-01-02")]
		}
		return isWeekday(d.Weekday())
	}

	if scale == entity.ScaleWeek {
		cols := make([]entity.GanttDay, 0, int(to.Sub(from).Hours()/(24*7))+1)
		for ws := from; ws.Before(to); ws = ws.AddDate(0, 0, 7) {
			we := ws.AddDate(0, 0, 7)
			if we.After(to) {
				we = to
			}
			anyWork, isToday := false, false
			for d := ws; d.Before(we); d = d.AddDate(0, 0, 1) {
				if isWork(d) {
					anyWork = true
				}
				if d.Equal(today) {
					isToday = true
				}
			}
			_, isoWeek := ws.ISOWeek()
			cols = append(cols, entity.GanttDay{
				Date:      ws,
				End:       we,
				Day:       isoWeek,
				Weekday:   ws.Weekday(),
				IsWorkday: anyWork,
				IsToday:   isToday,
				Label:     ws.Format("02/01"),
			})
		}
		return cols
	}

	cols := make([]entity.GanttDay, 0, int(to.Sub(from).Hours()/24)+1)
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		cols = append(cols, entity.GanttDay{
			Date:      d,
			End:       d.AddDate(0, 0, 1),
			Day:       d.Day(),
			Weekday:   d.Weekday(),
			IsWorkday: isWork(d),
			IsToday:   d.Equal(today),
		})
	}
	return cols
}

// workdayLookup loads the industrial calendar for every month touched by [from, to)
// and returns the set of workday dates plus the set of months the calendar covers
// (a covered month with no entry for a date means that date is a non-working day).
func (uc *APSUseCase) workdayLookup(ctx context.Context, from, to time.Time) (map[string]bool, map[int]bool) {
	workday := map[string]bool{}
	covered := map[int]bool{}
	if uc.cal == nil {
		return workday, covered
	}
	lastDay := to.AddDate(0, 0, -1)
	stop := time.Date(lastDay.Year(), lastDay.Month(), 1, 0, 0, 0, 0, from.Location()).AddDate(0, 1, 0)
	for cur := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location()); cur.Before(stop); cur = cur.AddDate(0, 1, 0) {
		entries, err := uc.cal.ListMonth(ctx, cur.Year(), int(cur.Month()))
		if err != nil || len(entries) == 0 {
			continue
		}
		covered[cur.Year()*100+int(cur.Month())] = true
		for _, e := range entries {
			if e.IsWorkday {
				workday[fmt.Sprintf("%04d-%02d-%02d", e.Year, e.Month, e.Day)] = true
			}
		}
	}
	return workday, covered
}

// buildDependencies returns the finish-start links drawn on the board. Explicit
// edges come from route_operation_network (mapped to on-board sequence ids); orders
// with no explicit edge get a synthesised linear chain from their operation order,
// so the planner always sees the precedence between an order's operations.
func (uc *APSUseCase) buildDependencies(ctx context.Context, bars []*entity.GanttBar, from, to time.Time) []entity.GanttDependency {
	seqToOrder := map[int64]int64{}
	onBoard := map[int64]bool{}
	byOrder := map[int64][]*entity.GanttBar{}
	for _, b := range bars {
		if b.SequenceID == 0 {
			continue // fallback bars are not sequenced, so they have no edges
		}
		seqToOrder[b.SequenceID] = b.ProductionOrderID
		onBoard[b.SequenceID] = true
		byOrder[b.ProductionOrderID] = append(byOrder[b.ProductionOrderID], b)
	}

	deps := []entity.GanttDependency{}
	ordersWithReal := map[int64]bool{}
	if real, err := uc.repo.ListDependencies(ctx, from, to); err == nil {
		for _, d := range real {
			if onBoard[d.FromSequenceID] && onBoard[d.ToSequenceID] {
				deps = append(deps, *d)
				ordersWithReal[seqToOrder[d.FromSequenceID]] = true
			}
		}
	}

	for order, obars := range byOrder {
		if ordersWithReal[order] || len(obars) < 2 {
			continue
		}
		chain := append([]*entity.GanttBar(nil), obars...)
		sort.SliceStable(chain, func(i, j int) bool {
			if chain[i].SequencePosition != chain[j].SequencePosition {
				return chain[i].SequencePosition < chain[j].SequencePosition
			}
			return chain[i].Start.Before(chain[j].Start)
		})
		for i := 1; i < len(chain); i++ {
			deps = append(deps, entity.GanttDependency{
				FromSequenceID: chain[i-1].SequenceID,
				ToSequenceID:   chain[i].SequenceID,
				Implicit:       true,
			})
		}
	}

	sort.SliceStable(deps, func(i, j int) bool {
		if deps[i].FromSequenceID != deps[j].FromSequenceID {
			return deps[i].FromSequenceID < deps[j].FromSequenceID
		}
		return deps[i].ToSequenceID < deps[j].ToSequenceID
	})
	return deps
}

// groupRows lays the bars into timeline lanes, either by work center or by order.
func groupRows(bars []*entity.GanttBar, groupBy entity.GanttGroupBy) []*entity.GanttRow {
	rowByID := map[int64]*entity.GanttRow{}
	var order []int64

	get := func(id int64, mk func() *entity.GanttRow) *entity.GanttRow {
		if r, ok := rowByID[id]; ok {
			return r
		}
		r := mk()
		rowByID[id] = r
		order = append(order, id)
		return r
	}

	for _, b := range bars {
		if groupBy == entity.GroupByOrder {
			r := get(b.ProductionOrderID, func() *entity.GanttRow {
				return &entity.GanttRow{
					Key:      "order:" + strconv.FormatInt(b.ProductionOrderID, 10),
					ID:       b.ProductionOrderID,
					Label:    "OF " + strconv.FormatInt(b.OrderNumber, 10),
					SubLabel: orderSubLabel(b),
				}
			})
			r.Bars = append(r.Bars, b)
			continue
		}
		// work-center grouping
		id := b.WorkCenterID
		r := get(id, func() *entity.GanttRow {
			return &entity.GanttRow{
				Key:   "wc:" + strconv.FormatInt(id, 10),
				ID:    id,
				Label: workCenterLabel(id, b.WorkCenterName),
			}
		})
		r.Bars = append(r.Bars, b)
	}

	rows := make([]*entity.GanttRow, 0, len(order))
	for _, id := range order {
		r := rowByID[id]
		sort.SliceStable(r.Bars, func(i, j int) bool { return r.Bars[i].Start.Before(r.Bars[j].Start) })
		rows = append(rows, r)
	}

	sortRows(rows, groupBy)
	return rows
}

func sortRows(rows []*entity.GanttRow, groupBy entity.GanttGroupBy) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if groupBy == entity.GroupByWorkCenter {
			// Real work centers first (by label), the unsequenced lane last.
			au, bu := a.ID == unsequencedRowID, b.ID == unsequencedRowID
			if au != bu {
				return !au
			}
			if a.Label != b.Label {
				return a.Label < b.Label
			}
			return a.ID < b.ID
		}
		return a.ID < b.ID
	})
}

func orderSubLabel(b *entity.GanttBar) string {
	s := "Item " + strconv.FormatInt(b.ItemCode, 10)
	if b.Mask != "" {
		s += " · " + b.Mask
	}
	return s
}

func workCenterLabel(id int64, name string) string {
	if id == unsequencedRowID {
		return "Sem sequenciamento"
	}
	if name != "" {
		return name
	}
	return "CT " + strconv.FormatInt(id, 10)
}

// ─── derived attributes ───────────────────────────────────────────────────────

// isLate flags a bar that should already be finished but is not: it is not in a
// terminal state and its scheduled end is before the start of today.
func isLate(b *entity.GanttBar, today time.Time) bool {
	if isFinished(b.Status) {
		return false
	}
	return b.End.Before(today)
}

func isFinished(status string) bool {
	switch strings.ToUpper(status) {
	case "DONE", "COMPLETED", "CLOSED":
		return true
	}
	return false
}

// resolveBarColor encodes status and priority into a fill: late bars are red,
// finished bars grey; otherwise the hue comes from the order's priority bucket.
func resolveBarColor(b *entity.GanttBar) string {
	if b.IsLate {
		return "#c0392b" // red — behind schedule
	}
	if isFinished(b.Status) {
		return "#6b7280" // grey — done
	}
	switch priorityBucket(b.Priority) {
	case priorityHigh:
		return "#e67e22" // orange — urgent
	case priorityLow:
		return "#5dade2" // light blue — low
	default:
		return "#2f6fb0" // blue — normal
	}
}

type pbucket int

const (
	priorityMedium pbucket = iota
	priorityHigh
	priorityLow
)

func priorityBucket(p string) pbucket {
	p = strings.ToUpper(strings.TrimSpace(p))
	if p == "" {
		return priorityMedium
	}
	if n, err := strconv.Atoi(p); err == nil {
		switch {
		case n <= 2:
			return priorityHigh
		case n >= 6:
			return priorityLow
		default:
			return priorityMedium
		}
	}
	switch {
	case strings.Contains(p, "ALTA"), strings.Contains(p, "URG"), strings.Contains(p, "HIGH"), strings.Contains(p, "CRIT"):
		return priorityHigh
	case strings.Contains(p, "BAIX"), strings.Contains(p, "LOW"):
		return priorityLow
	default:
		return priorityMedium
	}
}

func truncDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func isWeekday(w time.Weekday) bool {
	return w != time.Saturday && w != time.Sunday
}

// ─── response mapping ─────────────────────────────────────────────────────────

func monthToResponse(m *entity.GanttMonth) *response.GanttMonthResponse {
	out := &response.GanttMonthResponse{
		Year:        m.Year,
		Month:       m.Month,
		Scale:       string(m.Scale),
		GroupBy:     string(m.GroupBy),
		RangeFrom:   m.RangeFrom,
		RangeTo:     m.RangeTo,
		GeneratedAt: m.GeneratedAt,
		Summary: response.GanttSummaryResponse{
			TotalRows:      m.Summary.TotalRows,
			TotalBars:      m.Summary.TotalBars,
			SequencedBars:  m.Summary.SequencedBars,
			FallbackBars:   m.Summary.FallbackBars,
			LateBars:       m.Summary.LateBars,
			OverloadedDays: m.Summary.OverloadedDays,
		},
	}
	for _, d := range m.Days {
		out.Days = append(out.Days, response.GanttDayResponse{
			Date:      d.Date,
			End:       d.End,
			Day:       d.Day,
			Weekday:   int(d.Weekday),
			IsWorkday: d.IsWorkday,
			IsToday:   d.IsToday,
			Label:     d.Label,
		})
	}
	for _, dep := range m.Dependencies {
		out.Dependencies = append(out.Dependencies, response.GanttDependencyResponse{
			FromSequenceID: dep.FromSequenceID,
			ToSequenceID:   dep.ToSequenceID,
			OverlapPct:     dep.OverlapPct,
			Implicit:       dep.Implicit,
		})
	}
	for _, r := range m.Rows {
		row := response.GanttRowResponse{Key: r.Key, ID: r.ID, Label: r.Label, SubLabel: r.SubLabel}
		for _, b := range r.Bars {
			row.Bars = append(row.Bars, response.GanttBarResponse{
				SequenceID:        b.SequenceID,
				ProductionOrderID: b.ProductionOrderID,
				OrderNumber:       b.OrderNumber,
				ItemCode:          b.ItemCode,
				Mask:              b.Mask,
				WorkCenterID:      b.WorkCenterID,
				WorkCenterName:    b.WorkCenterName,
				OperationID:       b.OperationID,
				OperationName:     b.OperationName,
				SequencePosition:  b.SequencePosition,
				Start:             b.Start,
				End:               b.End,
				DurationHours:     b.DurationHours,
				Status:            b.Status,
				Priority:          b.Priority,
				PercentComplete:   b.PercentComplete,
				IsLate:            b.IsLate,
				IsFallback:        b.IsFallback,
				ColorHex:          b.ColorHex,
			})
		}
		out.Rows = append(out.Rows, row)
	}
	for _, l := range m.Load {
		out.Load = append(out.Load, response.GanttLoadResponse{
			WorkCenterID:   l.WorkCenterID,
			Date:           l.Date,
			RequiredHours:  l.RequiredHours,
			AvailableHours: l.AvailableHours,
			LoadPct:        l.LoadPct,
			IsOverloaded:   l.IsOverloaded,
		})
	}
	return out
}
