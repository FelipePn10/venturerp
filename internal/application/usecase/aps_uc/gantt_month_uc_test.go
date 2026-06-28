package aps_uc

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	apsentity "github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
	apsrepo "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
	calentity "github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/entity"
)

// ─── fakes ────────────────────────────────────────────────────────────────────

type fakeAPSRepo struct {
	scheduled []*apsentity.GanttBar
	fallback  []*apsentity.GanttBar
	load      []*apsentity.GanttResourceLoad
	deps      []*apsentity.GanttDependency           // real board edges
	orderDeps map[int64][]*apsentity.GanttDependency // per-order edges (cascade)
	seqs      map[int64]*apsentity.ProductionSequence
	capacity  map[int64]float64
	updated   []int64 // ids passed to UpdateSequence, in call order
}

func (f *fakeAPSRepo) UpsertSequence(context.Context, *apsentity.ProductionSequence) (*apsentity.ProductionSequence, error) {
	return nil, nil
}
func (f *fakeAPSRepo) GetSequence(_ context.Context, id int64) (*apsentity.ProductionSequence, error) {
	s, ok := f.seqs[id]
	if !ok {
		return nil, fmt.Errorf("sequence %d not found", id)
	}
	cp := *s
	return &cp, nil
}
func (f *fakeAPSRepo) UpdateSequence(_ context.Context, seq *apsentity.ProductionSequence) (*apsentity.ProductionSequence, error) {
	if f.seqs == nil {
		f.seqs = map[int64]*apsentity.ProductionSequence{}
	}
	cp := *seq
	f.seqs[seq.ID] = &cp
	f.updated = append(f.updated, seq.ID)
	out := cp
	return &out, nil
}
func (f *fakeAPSRepo) ListByOrder(_ context.Context, orderID int64) ([]*apsentity.ProductionSequence, error) {
	var out []*apsentity.ProductionSequence
	for _, s := range f.seqs {
		if s.ProductionOrderID == orderID {
			cp := *s
			out = append(out, &cp)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].SequencePosition < out[j].SequencePosition })
	return out, nil
}
func (f *fakeAPSRepo) ListByWorkCenter(_ context.Context, wc int64, from, to time.Time) ([]*apsentity.ProductionSequence, error) {
	var out []*apsentity.ProductionSequence
	for _, s := range f.seqs {
		if s.WorkCenterID == wc && !s.ScheduledStart.Before(from) && !s.ScheduledEnd.After(to) {
			cp := *s
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (f *fakeAPSRepo) DeleteByOrder(context.Context, int64) error { return nil }
func (f *fakeAPSRepo) GetOpenProductionOrders(context.Context) ([]apsrepo.OrderRow, error) {
	return nil, nil
}
func (f *fakeAPSRepo) GetOrderOperations(context.Context, int64) ([]apsrepo.OpRow, error) {
	return nil, nil
}
func (f *fakeAPSRepo) GetWorkCenterCapacity(_ context.Context, wc int64) (float64, error) {
	if v, ok := f.capacity[wc]; ok {
		return v, nil
	}
	return 8, nil
}
func (f *fakeAPSRepo) ListScheduledBars(context.Context, time.Time, time.Time) ([]*apsentity.GanttBar, error) {
	return f.scheduled, nil
}
func (f *fakeAPSRepo) ListFallbackBars(context.Context, time.Time, time.Time) ([]*apsentity.GanttBar, error) {
	return f.fallback, nil
}
func (f *fakeAPSRepo) ListResourceLoad(context.Context, time.Time, time.Time) ([]*apsentity.GanttResourceLoad, error) {
	return f.load, nil
}
func (f *fakeAPSRepo) ListDependencies(context.Context, time.Time, time.Time) ([]*apsentity.GanttDependency, error) {
	return f.deps, nil
}
func (f *fakeAPSRepo) ListOrderDependencies(_ context.Context, orderID int64) ([]*apsentity.GanttDependency, error) {
	return f.orderDeps[orderID], nil
}

type fakeCalendar struct {
	entries []*calentity.IndustrialCalendar
}

func (c *fakeCalendar) CreateDay(context.Context, *calentity.IndustrialCalendar) (*calentity.IndustrialCalendar, error) {
	return nil, nil
}
func (c *fakeCalendar) GetDay(context.Context, int, int, int) (*calentity.IndustrialCalendar, error) {
	return nil, nil
}
func (c *fakeCalendar) GetWorkdaysInMonth(context.Context, int, int) ([]*calentity.IndustrialCalendar, error) {
	return nil, nil
}
func (c *fakeCalendar) IsWorkday(context.Context, int, int, int) (bool, error) { return true, nil }
func (c *fakeCalendar) GetNextWorkday(context.Context, int, int, int) (time.Time, error) {
	return time.Time{}, nil
}
func (c *fakeCalendar) ListMonth(context.Context, int, int) ([]*calentity.IndustrialCalendar, error) {
	return c.entries, nil
}
func (c *fakeCalendar) DeleteDay(context.Context, int, int, int) error { return nil }
func (c *fakeCalendar) SubtractWorkdays(context.Context, time.Time, int) (time.Time, error) {
	return time.Time{}, nil
}

// ─── colour / lateness unit tests ─────────────────────────────────────────────

func TestPriorityBucket(t *testing.T) {
	cases := map[string]pbucket{
		"":        priorityMedium,
		"1":       priorityHigh,
		"2":       priorityHigh,
		"3":       priorityMedium,
		"7":       priorityLow,
		"ALTA":    priorityHigh,
		"URGENTE": priorityHigh,
		"baixa":   priorityLow,
		"normal":  priorityMedium,
	}
	for in, want := range cases {
		if got := priorityBucket(in); got != want {
			t.Errorf("priorityBucket(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestResolveBarColor(t *testing.T) {
	late := &apsentity.GanttBar{IsLate: true, Status: "SCHEDULED", Priority: "1"}
	if got := resolveBarColor(late); got != "#c0392b" {
		t.Errorf("late bar should be red, got %s", got)
	}
	done := &apsentity.GanttBar{Status: "DONE", Priority: "1"}
	if got := resolveBarColor(done); got != "#6b7280" {
		t.Errorf("done bar should be grey, got %s", got)
	}
	urgent := &apsentity.GanttBar{Status: "SCHEDULED", Priority: "1"}
	if got := resolveBarColor(urgent); got != "#e67e22" {
		t.Errorf("urgent bar should be orange, got %s", got)
	}
	normal := &apsentity.GanttBar{Status: "SCHEDULED", Priority: "4"}
	if got := resolveBarColor(normal); got != "#2f6fb0" {
		t.Errorf("normal bar should be blue, got %s", got)
	}
}

func TestIsLate(t *testing.T) {
	today := time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local)
	pastUnfinished := &apsentity.GanttBar{Status: "SCHEDULED", End: today.Add(-48 * time.Hour)}
	if !isLate(pastUnfinished, today) {
		t.Error("a scheduled bar ending before today must be late")
	}
	pastFinished := &apsentity.GanttBar{Status: "DONE", End: today.Add(-48 * time.Hour)}
	if isLate(pastFinished, today) {
		t.Error("a finished bar is never late")
	}
	future := &apsentity.GanttBar{Status: "SCHEDULED", End: today.Add(48 * time.Hour)}
	if isLate(future, today) {
		t.Error("a bar ending in the future is not late")
	}
}

// ─── board assembly tests ─────────────────────────────────────────────────────

func mkBar(order, wc int64, wcName string, start time.Time, dur time.Duration, status, prio string, fallback bool) *apsentity.GanttBar {
	return &apsentity.GanttBar{
		ProductionOrderID: order,
		OrderNumber:       order,
		ItemCode:          order * 10,
		WorkCenterID:      wc,
		WorkCenterName:    wcName,
		Start:             start,
		End:               start.Add(dur),
		Status:            status,
		Priority:          prio,
		IsFallback:        fallback,
	}
}

func TestBuildMonthSchedule_WorkCenterGrouping(t *testing.T) {
	orig := nowFunc
	nowFunc = func() time.Time { return time.Date(2026, 6, 15, 9, 0, 0, 0, time.Local) }
	defer func() { nowFunc = orig }()

	s1 := time.Date(2026, 6, 3, 8, 0, 0, 0, time.Local)
	s2 := time.Date(2026, 6, 10, 8, 0, 0, 0, time.Local)
	repo := &fakeAPSRepo{
		scheduled: []*apsentity.GanttBar{
			mkBar(101, 7, "Corte", s1, 4*time.Hour, "SCHEDULED", "1", false),
			mkBar(102, 7, "Corte", s2, 4*time.Hour, "CONFIRMED", "5", false),
		},
		fallback: []*apsentity.GanttBar{
			mkBar(200, 0, "", s1, 24*time.Hour, "OPEN", "", true),
		},
		load: []*apsentity.GanttResourceLoad{
			{WorkCenterID: 7, Date: time.Date(2026, 6, 3, 0, 0, 0, 0, time.Local), RequiredHours: 12, AvailableHours: 8, LoadPct: 150, IsOverloaded: true},
		},
	}
	uc := New(repo)

	m, err := uc.BuildMonthSchedule(context.Background(), 2026, 6, apsentity.GroupByWorkCenter)
	if err != nil {
		t.Fatalf("BuildMonthSchedule: %v", err)
	}

	if len(m.Days) != 30 {
		t.Errorf("June has 30 days, got %d", len(m.Days))
	}
	// Without a calendar, weekend fallback applies. 2026-06-06 is a Saturday.
	if m.Days[5].IsWorkday {
		t.Errorf("2026-06-06 (Sat) should be a non-working day")
	}
	if !m.Days[14].IsToday {
		t.Errorf("day 15 should be flagged as today")
	}

	// Two rows: work center 7 (Corte) and the unsequenced lane (0), in that order.
	if len(m.Rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(m.Rows))
	}
	if m.Rows[0].ID != 7 || m.Rows[0].Label != "Corte" {
		t.Errorf("first row should be work center 7 (Corte), got id=%d label=%q", m.Rows[0].ID, m.Rows[0].Label)
	}
	if m.Rows[1].ID != unsequencedRowID || m.Rows[1].Label != "Sem sequenciamento" {
		t.Errorf("unsequenced lane should sort last, got id=%d label=%q", m.Rows[1].ID, m.Rows[1].Label)
	}

	if m.Summary.TotalBars != 3 || m.Summary.SequencedBars != 2 || m.Summary.FallbackBars != 1 {
		t.Errorf("summary mismatch: %+v", m.Summary)
	}
	if m.Summary.OverloadedDays != 1 {
		t.Errorf("want 1 overloaded day, got %d", m.Summary.OverloadedDays)
	}
	if len(m.Load) != 1 {
		t.Errorf("work-center grouping should carry load, got %d entries", len(m.Load))
	}
}

func TestBuildMonthSchedule_OrderGroupingNoLoad(t *testing.T) {
	orig := nowFunc
	nowFunc = func() time.Time { return time.Date(2026, 6, 15, 9, 0, 0, 0, time.Local) }
	defer func() { nowFunc = orig }()

	s1 := time.Date(2026, 6, 3, 8, 0, 0, 0, time.Local)
	repo := &fakeAPSRepo{
		scheduled: []*apsentity.GanttBar{
			mkBar(101, 7, "Corte", s1, 4*time.Hour, "SCHEDULED", "1", false),
			mkBar(101, 8, "Solda", s1.Add(8*time.Hour), 4*time.Hour, "SCHEDULED", "1", false),
		},
		load: []*apsentity.GanttResourceLoad{{WorkCenterID: 7, Date: s1, LoadPct: 50}},
	}
	uc := New(repo)

	m, err := uc.BuildMonthSchedule(context.Background(), 2026, 6, apsentity.GroupByOrder)
	if err != nil {
		t.Fatalf("BuildMonthSchedule: %v", err)
	}
	if len(m.Rows) != 1 {
		t.Fatalf("both bars belong to OF 101 → 1 row, got %d", len(m.Rows))
	}
	if m.Rows[0].Label != "OF 101" {
		t.Errorf("row label = %q, want OF 101", m.Rows[0].Label)
	}
	if len(m.Rows[0].Bars) != 2 {
		t.Errorf("OF 101 row should hold 2 bars, got %d", len(m.Rows[0].Bars))
	}
	if m.Load != nil {
		t.Errorf("order grouping must not carry resource load")
	}
}

func TestBuildMonthSchedule_CalendarWorkdays(t *testing.T) {
	orig := nowFunc
	nowFunc = func() time.Time { return time.Date(2026, 6, 1, 9, 0, 0, 0, time.Local) }
	defer func() { nowFunc = orig }()

	// Calendar says only day 1 is a workday and day 2 is a holiday (non-workday).
	cal := &fakeCalendar{entries: []*calentity.IndustrialCalendar{
		{Year: 2026, Month: 6, Day: 1, IsWorkday: true},
		{Year: 2026, Month: 6, Day: 2, IsWorkday: false},
	}}
	uc := New(&fakeAPSRepo{}).WithCalendar(cal)

	m, err := uc.BuildMonthSchedule(context.Background(), 2026, 6, apsentity.GroupByWorkCenter)
	if err != nil {
		t.Fatalf("BuildMonthSchedule: %v", err)
	}
	if !m.Days[0].IsWorkday {
		t.Errorf("day 1 should be a workday per calendar")
	}
	if m.Days[1].IsWorkday {
		t.Errorf("day 2 should be a holiday per calendar")
	}
	// Day 3 has no calendar entry but the calendar is non-empty, so it is treated
	// as a non-working day (not in the workday set).
	if m.Days[2].IsWorkday {
		t.Errorf("day 3 absent from a populated calendar should be non-working")
	}
}

func TestBuildMonthSchedule_InvalidMonth(t *testing.T) {
	uc := New(&fakeAPSRepo{})
	if _, err := uc.BuildMonthSchedule(context.Background(), 2026, 13, apsentity.GroupByWorkCenter); err == nil {
		t.Error("month 13 must be rejected")
	}
}

// ─── shared helpers for dependency / scale / reschedule tests ──────────────────

func dt(y, mo, d, h int) time.Time {
	return time.Date(y, time.Month(mo), d, h, 0, 0, 0, time.Local)
}

func mkSeqBar(seqID, order int64, pos int, wc int64, start time.Time, dur time.Duration) *apsentity.GanttBar {
	return &apsentity.GanttBar{
		SequenceID: seqID, ProductionOrderID: order, OrderNumber: order, ItemCode: order * 10,
		WorkCenterID: wc, SequencePosition: pos,
		Start: start, End: start.Add(dur), Status: "SCHEDULED", Priority: "5",
	}
}

// ─── dependency tests ─────────────────────────────────────────────────────────

func TestBuildDependencies_ImplicitChain(t *testing.T) {
	s := dt(2026, 6, 3, 8)
	repo := &fakeAPSRepo{scheduled: []*apsentity.GanttBar{
		mkSeqBar(1, 101, 1, 7, s, 4*time.Hour),
		mkSeqBar(2, 101, 2, 8, s.Add(8*time.Hour), 4*time.Hour),
	}}
	m, err := New(repo).BuildMonthSchedule(context.Background(), 2026, 6, apsentity.GroupByWorkCenter)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(m.Dependencies) != 1 {
		t.Fatalf("want 1 synthesised edge, got %d", len(m.Dependencies))
	}
	d := m.Dependencies[0]
	if d.FromSequenceID != 1 || d.ToSequenceID != 2 || !d.Implicit {
		t.Errorf("want implicit 1→2, got %+v", d)
	}
}

func TestBuildDependencies_ExplicitSuppressesImplicit(t *testing.T) {
	s := dt(2026, 6, 3, 8)
	repo := &fakeAPSRepo{
		scheduled: []*apsentity.GanttBar{
			mkSeqBar(1, 101, 1, 7, s, 4*time.Hour),
			mkSeqBar(2, 101, 2, 8, s.Add(8*time.Hour), 4*time.Hour),
			mkSeqBar(3, 101, 3, 9, s.Add(16*time.Hour), 4*time.Hour),
		},
		deps: []*apsentity.GanttDependency{{FromSequenceID: 1, ToSequenceID: 3, OverlapPct: 25}},
	}
	m, err := New(repo).BuildMonthSchedule(context.Background(), 2026, 6, apsentity.GroupByOrder)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	// The order has an explicit edge, so no linear chain is synthesised: exactly the
	// one real 1→3 edge survives (1→2→3 implicit must NOT appear).
	if len(m.Dependencies) != 1 {
		t.Fatalf("want only the explicit edge, got %d: %+v", len(m.Dependencies), m.Dependencies)
	}
	if d := m.Dependencies[0]; d.Implicit || d.FromSequenceID != 1 || d.ToSequenceID != 3 || d.OverlapPct != 25 {
		t.Errorf("want explicit 1→3 overlap 25, got %+v", d)
	}
}

// ─── scale / range tests ──────────────────────────────────────────────────────

func TestBuildBoard_WeekScale(t *testing.T) {
	orig := nowFunc
	nowFunc = func() time.Time { return dt(2026, 6, 10, 9) }
	defer func() { nowFunc = orig }()

	from := dt(2026, 6, 1, 0)
	to := dt(2026, 6, 29, 0) // 28 days → 4 whole week columns
	m, err := New(&fakeAPSRepo{}).BuildBoard(context.Background(), from, to, apsentity.ScaleWeek, apsentity.GroupByWorkCenter)
	if err != nil {
		t.Fatalf("build board: %v", err)
	}
	if m.Scale != apsentity.ScaleWeek {
		t.Errorf("scale = %q, want week", m.Scale)
	}
	if len(m.Days) != 4 {
		t.Fatalf("28 days at week scale → 4 columns, got %d", len(m.Days))
	}
	if m.Days[0].Label == "" || m.Days[0].End.IsZero() {
		t.Errorf("week column must carry a label and an end: %+v", m.Days[0])
	}
	// Today (10/06) falls in the second week column (08/06–14/06).
	if !m.Days[1].IsToday {
		t.Errorf("second week column should contain today")
	}
}

func TestBuildBoard_InvalidRange(t *testing.T) {
	from := dt(2026, 6, 10, 0)
	if _, err := New(&fakeAPSRepo{}).BuildBoard(context.Background(), from, from, apsentity.ScaleDay, apsentity.GroupByWorkCenter); err == nil {
		t.Error("empty range (to == from) must be rejected")
	}
}
