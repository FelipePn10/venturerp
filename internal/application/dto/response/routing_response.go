package response

import "time"

// OperationTimeBreakdown is the resolved, quantity-aware time model (hours).
type OperationTimeBreakdown struct {
	Setup      float64 `json:"setup_hours"`
	Run        float64 `json:"run_hours"`
	Labor      float64 `json:"labor_hours"`
	RunBaseQty float64 `json:"run_base_qty"`
	Queue      float64 `json:"queue_hours"`
	Wait       float64 `json:"wait_hours"`
	Move       float64 `json:"move_hours"`
	CrewSize   float64 `json:"crew_size"`
}

type OperationResponse struct {
	ID                  int64     `json:"id"`
	Code                int64     `json:"code"`
	Name                string    `json:"name"`
	Description         *string   `json:"description,omitempty"`
	Origin              string    `json:"origin"`
	Situation           string    `json:"situation"`
	DefaultWorkCenterID *int64    `json:"default_work_center_id,omitempty"`
	StandardTime        float64   `json:"standard_time"`
	SetupTime           float64   `json:"setup_time"`
	RunTime             float64   `json:"run_time"`
	LaborTime           float64   `json:"labor_time"`
	RunBaseQty          float64   `json:"run_base_qty"`
	QueueTime           float64   `json:"queue_time"`
	WaitTime            float64   `json:"wait_time"`
	MoveTime            float64   `json:"move_time"`
	CrewSize            float64   `json:"crew_size"`
	TimeUnit            string    `json:"time_unit"`
	SupplierID          *int64    `json:"supplier_id,omitempty"`
	ServiceItemCode     *int64    `json:"service_item_code,omitempty"`
	CostPerUnit         *float64  `json:"cost_per_unit,omitempty"`
	LeadTimeDays        *int32    `json:"lead_time_days,omitempty"`
	IsActive            bool      `json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
}

type ManufacturingRouteResponse struct {
	ID          int64      `json:"id"`
	Code        int64      `json:"code"`
	ItemCode    int64      `json:"item_code"`
	Mask        *string    `json:"mask,omitempty"`
	Alternative int16      `json:"alternative"`
	Description *string    `json:"description,omitempty"`
	Situation   string     `json:"situation"`
	IsStandard  bool       `json:"is_standard"`
	ValidFrom   *time.Time `json:"valid_from,omitempty"`
	ValidTo     *time.Time `json:"valid_to,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}

type RouteOperationResponse struct {
	ID               int64    `json:"id"`
	RouteID          int64    `json:"route_id"`
	Sequence         int16    `json:"sequence"`
	OperationID      int64    `json:"operation_id"`
	OperationName    string   `json:"operation_name"`
	WorkCenterID     *int64   `json:"work_center_id,omitempty"`
	WorkCenterName   string   `json:"work_center_name,omitempty"`
	StandardTime     *float64 `json:"standard_time,omitempty"`
	SetupTime        *float64 `json:"setup_time,omitempty"`
	EffectiveStdTime float64  `json:"effective_std_time"`
	EffectiveSetup   float64  `json:"effective_setup"`
	// EffTime is the resolved, quantity-aware time model in hours (setup/run/labor/queue/wait/move).
	EffTime OperationTimeBreakdown `json:"eff_time"`
	// Subcontracting overrides (nil ⇒ inherit from the operation).
	SupplierID      *int64   `json:"supplier_id,omitempty"`
	ServiceItemCode *int64   `json:"service_item_code,omitempty"`
	CostPerUnit     *float64 `json:"cost_per_unit,omitempty"`
	LeadTimeDays    *int32   `json:"lead_time_days,omitempty"`
	Situation       string   `json:"situation"`
	Notes           *string  `json:"notes,omitempty"`
}

type RouteOpResourceResponse struct {
	ID               int64   `json:"id"`
	RouteOperationID int64   `json:"route_operation_id"`
	WorkCenterID     int64   `json:"work_center_id"`
	WorkCenterName   string  `json:"work_center_name,omitempty"`
	Priority         int16   `json:"priority"`
	TimeFactor       float64 `json:"time_factor"`
	IsPrimary        bool    `json:"is_primary"`
}

type NetworkEdgeResponse struct {
	ID            int64   `json:"id"`
	PredecessorID int64   `json:"predecessor_id"`
	SuccessorID   int64   `json:"successor_id"`
	OverlapPct    float64 `json:"overlap_pct"`
}

type RouteDetailResponse struct {
	Route      ManufacturingRouteResponse `json:"route"`
	Operations []RouteOperationResponse   `json:"operations"`
	Network    []NetworkEdgeResponse      `json:"network"`
	Resources  []RouteOpResourceResponse  `json:"resources"`
}

type RouteLeadTimeResponse struct {
	RouteID      int64   `json:"route_id"`
	TotalHours   float64 `json:"total_hours"`
	CriticalPath []int64 `json:"critical_path"` // route_operation IDs
}
