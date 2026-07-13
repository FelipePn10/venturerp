package entity

import "time"

type SequenceStatus string

const (
	StatusScheduled SequenceStatus = "SCHEDULED"
	StatusConfirmed SequenceStatus = "CONFIRMED"
	StatusDone      SequenceStatus = "DONE"
)

type ProductionSequence struct {
	ID                int64
	ProductionOrderID int64
	OperationID       *int64
	WorkCenterID      int64
	MachineID         *int64
	SequencePosition  int
	ScheduledStart    time.Time
	ScheduledEnd      time.Time
	Status            SequenceStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// GanttTask is the projection returned by the Gantt endpoint.
type GanttTask struct {
	SequenceID        int64
	ProductionOrderID int64
	WorkCenterID      int64
	MachineID         *int64
	SequencePosition  int
	ScheduledStart    time.Time
	ScheduledEnd      time.Time
	Status            SequenceStatus
	DurationHours     float64
}

// ─── Monthly schedule board (Gantt) ───────────────────────────────────────────
//
// GanttMonth is the aggregate behind the monthly production-schedule board: a
// finite set of timeline rows (grouped either by work-center or by order), the
// bars scheduled within the month, the calendar backdrop (workdays / today) and
// the per-resource capacity load used to draw the load histogram. It is pure
// data — the use case fills it and the export renderer draws it.

// GanttGroupBy selects what each timeline row represents.
type GanttGroupBy string

const (
	GroupByWorkCenter GanttGroupBy = "work_center"
	GroupByOrder      GanttGroupBy = "order"
)

// GanttScale is the time resolution of the board's columns: one column per day
// (the month view) or one per ISO week (the zoomed-out view for longer ranges).
type GanttScale string

const (
	ScaleDay  GanttScale = "day"
	ScaleWeek GanttScale = "week"
)

// GanttBar is one scheduled block drawn on the timeline.
type GanttBar struct {
	SequenceID        int64 // production_sequences.id (0 for fallback bars)
	ProductionOrderID int64
	OrderNumber       int64
	ItemCode          int64
	Mask              string
	WorkCenterID      int64 // machine_types.id; 0 when unknown (fallback)
	WorkCenterName    string
	OperationID       *int64
	OperationName     string
	SequencePosition  int
	Start             time.Time
	End               time.Time
	DurationHours     float64
	Status            string // sequence status, or order status for fallback bars
	Priority          string
	PercentComplete   float64 // 0..100
	IsLate            bool    // behind schedule / past due and not finished
	IsFallback        bool    // plotted from the order's dates, not APS-sequenced
	ColorHex          string  // resolved fill, by status/priority/lateness
}

// GanttRow is a single horizontal lane of the board.
type GanttRow struct {
	Key      string // stable id, e.g. "wc:12" or "order:34"
	ID       int64
	Label    string
	SubLabel string
	Bars     []*GanttBar
}

// GanttDay is one column of the board backdrop. At day scale it is a calendar day
// (Day = day-of-month); at week scale it is an ISO week spanning [Date, End) and
// Label carries the column caption (e.g. "Sem 27").
type GanttDay struct {
	Date      time.Time
	End       time.Time // exclusive column end (== Date+1d at day scale)
	Day       int       // day-of-month (day scale) or ISO week number (week scale)
	Weekday   time.Weekday
	IsWorkday bool
	IsToday   bool
	Label     string // explicit caption; when empty the renderer falls back to Day
}

// GanttDependency is a finish-start link between two bars: the successor must not
// start before the predecessor finishes (minus the predecessor's overlap window).
// Real links come from route_operation_network; when an order has none, the board
// synthesises a linear chain from the operations' sequence positions (Implicit).
type GanttDependency struct {
	FromSequenceID int64   // predecessor production_sequences.id
	ToSequenceID   int64   // successor production_sequences.id
	OverlapPct     float64 // 0..100; how much the successor may overlap the predecessor
	Implicit       bool    // synthesised from sequence order, not an explicit edge
}

// GanttResourceLoad is the CRP load for one work-center on one day.
type GanttResourceLoad struct {
	WorkCenterID   int64
	Date           time.Time
	RequiredHours  float64
	AvailableHours float64
	LoadPct        float64
	IsOverloaded   bool
}

// GanttSummary is the headline tally of the board.
type GanttSummary struct {
	TotalRows      int
	TotalBars      int
	SequencedBars  int
	FallbackBars   int
	LateBars       int
	OverloadedDays int
}

// GanttMonth is the full board. Despite the name it serves both the month view and
// arbitrary [RangeFrom, RangeTo) ranges at day or week scale (Year/Month are 0 for
// non-month ranges).
type GanttMonth struct {
	Year         int
	Month        int
	RangeFrom    time.Time // inclusive (range start, local midnight)
	RangeTo      time.Time // exclusive (range end)
	Scale        GanttScale
	GroupBy      GanttGroupBy
	Days         []GanttDay
	Rows         []*GanttRow
	Dependencies []GanttDependency
	Load         []*GanttResourceLoad // populated for the work-center grouping
	Summary      GanttSummary
	GeneratedAt  time.Time
}
