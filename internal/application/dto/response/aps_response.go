package response

import "time"

type APSSummaryResponse struct {
	ScheduledOperations int `json:"scheduled_operations"`
	OrdersProcessed     int `json:"orders_processed"`
}

type SequencingExportRowResponse struct {
	EventType         string    `json:"event_type"`
	ProductionOrderID int64     `json:"production_order_id"`
	OrderNumber       int64     `json:"order_number"`
	MachineID         *int64    `json:"machine_id,omitempty"`
	WorkCenterID      *int64    `json:"work_center_id,omitempty"`
	OperationID       *int64    `json:"operation_id,omitempty"`
	EventAt           time.Time `json:"event_at"`
	Quantity          string    `json:"quantity,omitempty"`
	Reason            string    `json:"reason,omitempty"`
}

type SequencingResourceResponse struct {
	ID              int64  `json:"id"`
	Code            int64  `json:"code"`
	Name            string `json:"name"`
	WorkCenterID    int64  `json:"work_center_id"`
	ResourceGroupID *int64 `json:"resource_group_id,omitempty"`
	IsActive        bool   `json:"is_active"`
}
type ResourceGroupResponse struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
}
type MachineCalendarIntervalResponse struct {
	Weekday int    `json:"weekday"`
	Start   string `json:"start"`
	End     string `json:"end"`
}
type MachineCalendarResponse struct {
	ID          int64                             `json:"id"`
	Code        int64                             `json:"code"`
	Description string                            `json:"description"`
	Intervals   []MachineCalendarIntervalResponse `json:"intervals"`
}
type MachineDowntimeResponse struct {
	ID                 int64     `json:"id"`
	MachineID          int64     `json:"machine_id"`
	StartsAt           time.Time `json:"starts_at"`
	EndsAt             time.Time `json:"ends_at"`
	DowntimeType       string    `json:"downtime_type"`
	Reason             string    `json:"reason"`
	MaintenanceOrderID *int64    `json:"maintenance_order_id,omitempty"`
}

type EmployeeContactResponse struct {
	ID          int64  `json:"id"`
	ContactType string `json:"contact_type"`
	Value       string `json:"value"`
	IsPrimary   bool   `json:"is_primary"`
}
type EmployeeFunctionResponse struct {
	ID           int64  `json:"id"`
	FunctionName string `json:"function_name"`
	CostCenterID *int64 `json:"cost_center_id,omitempty"`
	IsSupervisor bool   `json:"is_supervisor"`
	IsManager    bool   `json:"is_manager"`
}
type EmployeeSequencingProfileResponse struct {
	Contacts    []EmployeeContactResponse  `json:"contacts"`
	Functions   []EmployeeFunctionResponse `json:"functions"`
	CreditLimit string                     `json:"credit_limit"`
	ValidUntil  *time.Time                 `json:"valid_until,omitempty"`
}
type ServiceItemResponse struct {
	ID       int64  `json:"id"`
	ItemCode int64  `json:"item_code"`
	Quantity string `json:"quantity"`
	Notes    string `json:"notes,omitempty"`
}
type MachineServiceResponse struct {
	ID                     int64                 `json:"id"`
	ServiceCode            string                `json:"service_code"`
	Description            string                `json:"description"`
	ServiceType            string                `json:"service_type"`
	FrequencyValue         int                   `json:"frequency_value"`
	FrequencyUnit          string                `json:"frequency_unit"`
	MaxTolerance           int                   `json:"max_tolerance"`
	SupplierCode           *int64                `json:"supplier_code,omitempty"`
	ImplementedOn          time.Time             `json:"implemented_on"`
	LastExecutedOn         *time.Time            `json:"last_executed_on,omitempty"`
	Notes                  string                `json:"notes,omitempty"`
	Items                  []ServiceItemResponse `json:"items"`
	ResponsibleEmployeeIDs []int64               `json:"responsible_employee_ids"`
}
type SpecialValueResponse struct {
	FieldID      int64  `json:"field_id"`
	Name         string `json:"name"`
	ValueType    string `json:"value_type"`
	TextValue    string `json:"text_value,omitempty"`
	NumericValue string `json:"numeric_value,omitempty"`
	MaxLength    *int   `json:"max_length,omitempty"`
}
type MachineIndustrialProfileResponse struct {
	UsageDescription                 string                   `json:"usage_description"`
	AcquiredOn                       *time.Time               `json:"acquired_on,omitempty"`
	PreparationTime                  string                   `json:"preparation_time"`
	PreparationTimeUnit              string                   `json:"preparation_time_unit"`
	SupplierCode                     *int64                   `json:"supplier_code,omitempty"`
	Brand                            string                   `json:"brand"`
	IsPreferred                      bool                     `json:"is_preferred"`
	MaintenanceResponsibleEmployeeID *int64                   `json:"maintenance_responsible_employee_id,omitempty"`
	Services                         []MachineServiceResponse `json:"services"`
	SpecialValues                    []SpecialValueResponse   `json:"special_values"`
}

type GanttTaskResponse struct {
	SequenceID        int64     `json:"sequence_id"`
	ProductionOrderID int64     `json:"production_order_id"`
	WorkCenterID      int64     `json:"work_center_id"`
	MachineID         *int64    `json:"machine_id,omitempty"`
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
	MachineID         *int64    `json:"machine_id,omitempty"`
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
