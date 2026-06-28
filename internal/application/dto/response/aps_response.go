package response

import "time"

type APSSummaryResponse struct {
	ScheduledOperations int `json:"scheduled_operations"`
	OrdersProcessed     int `json:"orders_processed"`
}

type GanttTaskResponse struct {
	SequenceID        int64     `json:"sequence_id"`
	ProductionOrderID int64     `json:"production_order_id"`
	WorkCenterID      int64     `json:"work_center_id"`
	SequencePosition  int       `json:"sequence_position"`
	ScheduledStart    time.Time `json:"scheduled_start"`
	ScheduledEnd      time.Time `json:"scheduled_end"`
	Status            string    `json:"status"`
	DurationHours     float64   `json:"duration_hours"`
}

// ─── monthly schedule board (Gantt) ───────────────────────────────────────────

type GanttMonthResponse struct {
	Year         int                       `json:"year"`
	Month        int                       `json:"month"`
	Scale        string                    `json:"scale"`
	GroupBy      string                    `json:"group_by"`
	RangeFrom    time.Time                 `json:"range_from"`
	RangeTo      time.Time                 `json:"range_to"`
	GeneratedAt  time.Time                 `json:"generated_at"`
	Days         []GanttDayResponse        `json:"days"`
	Rows         []GanttRowResponse        `json:"rows"`
	Dependencies []GanttDependencyResponse `json:"dependencies,omitempty"`
	Load         []GanttLoadResponse       `json:"load,omitempty"`
	Summary      GanttSummaryResponse      `json:"summary"`
}

type GanttDayResponse struct {
	Date      time.Time `json:"date"`
	End       time.Time `json:"end"`
	Day       int       `json:"day"`
	Weekday   int       `json:"weekday"` // 0=Sunday .. 6=Saturday
	IsWorkday bool      `json:"is_workday"`
	IsToday   bool      `json:"is_today"`
	Label     string    `json:"label,omitempty"`
}

type GanttDependencyResponse struct {
	FromSequenceID int64   `json:"from_sequence_id"`
	ToSequenceID   int64   `json:"to_sequence_id"`
	OverlapPct     float64 `json:"overlap_pct"`
	Implicit       bool    `json:"implicit"`
}

type GanttRowResponse struct {
	Key      string             `json:"key"`
	ID       int64              `json:"id"`
	Label    string             `json:"label"`
	SubLabel string             `json:"sub_label,omitempty"`
	Bars     []GanttBarResponse `json:"bars"`
}

type GanttBarResponse struct {
	SequenceID        int64     `json:"sequence_id"`
	ProductionOrderID int64     `json:"production_order_id"`
	OrderNumber       int64     `json:"order_number"`
	ItemCode          int64     `json:"item_code"`
	Mask              string    `json:"mask,omitempty"`
	WorkCenterID      int64     `json:"work_center_id"`
	WorkCenterName    string    `json:"work_center_name,omitempty"`
	OperationID       *int64    `json:"operation_id,omitempty"`
	OperationName     string    `json:"operation_name,omitempty"`
	SequencePosition  int       `json:"sequence_position"`
	Start             time.Time `json:"start"`
	End               time.Time `json:"end"`
	DurationHours     float64   `json:"duration_hours"`
	Status            string    `json:"status"`
	Priority          string    `json:"priority,omitempty"`
	PercentComplete   float64   `json:"percent_complete"`
	IsLate            bool      `json:"is_late"`
	IsFallback        bool      `json:"is_fallback"`
	ColorHex          string    `json:"color_hex"`
}

type GanttLoadResponse struct {
	WorkCenterID   int64     `json:"work_center_id"`
	Date           time.Time `json:"date"`
	RequiredHours  float64   `json:"required_hours"`
	AvailableHours float64   `json:"available_hours"`
	LoadPct        float64   `json:"load_pct"`
	IsOverloaded   bool      `json:"is_overloaded"`
}

// ─── reschedule (manual move) ─────────────────────────────────────────────────

// RescheduleResultResponse reports a manual board move: where the dragged bar
// landed, which dependent bars were pushed to keep the finish-start chain valid,
// and any day on which a touched work center is now booked beyond its capacity.
type RescheduleResultResponse struct {
	Moved          RescheduledBarResponse    `json:"moved"`
	Shifted        []RescheduledBarResponse  `json:"shifted"`
	CascadeApplied bool                      `json:"cascade_applied"`
	Warnings       []CapacityWarningResponse `json:"warnings,omitempty"`
}

type RescheduledBarResponse struct {
	SequenceID        int64     `json:"sequence_id"`
	ProductionOrderID int64     `json:"production_order_id"`
	WorkCenterID      int64     `json:"work_center_id"`
	ScheduledStart    time.Time `json:"scheduled_start"`
	ScheduledEnd      time.Time `json:"scheduled_end"`
	DurationHours     float64   `json:"duration_hours"`
}

type CapacityWarningResponse struct {
	WorkCenterID   int64     `json:"work_center_id"`
	Date           time.Time `json:"date"`
	ScheduledHours float64   `json:"scheduled_hours"`
	AvailableHours float64   `json:"available_hours"`
	OverByHours    float64   `json:"over_by_hours"`
}

type GanttSummaryResponse struct {
	TotalRows      int `json:"total_rows"`
	TotalBars      int `json:"total_bars"`
	SequencedBars  int `json:"sequenced_bars"`
	FallbackBars   int `json:"fallback_bars"`
	LateBars       int `json:"late_bars"`
	OverloadedDays int `json:"overloaded_days"`
}
