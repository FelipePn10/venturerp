package request

import (
	"time"

	"github.com/google/uuid"
)

// ─── operations ──────────────────────────────────────────────────────────────

type CreateOperationDTO struct {
	Name                string  `json:"name"`
	Description         *string `json:"description,omitempty"`
	Origin              string  `json:"origin"` // INTERNA | EXTERNA | TERCEIROS
	DefaultWorkCenterID *int64  `json:"default_work_center_id,omitempty"`
	StandardTime        float64 `json:"standard_time"` // legacy flat time (falls back to run_time)
	SetupTime           float64 `json:"setup_time"`    // setup per lot, in time_unit

	// Rich time model (defaults). All in TimeUnit (MIN|HORA|DIA).
	RunTime    float64 `json:"run_time"`     // machine time per run_base_qty
	LaborTime  float64 `json:"labor_time"`   // labor time per run_base_qty (0 ⇒ equals run)
	RunBaseQty float64 `json:"run_base_qty"` // pieces per run cycle (>=1)
	QueueTime  float64 `json:"queue_time"`   // fixed per lot
	WaitTime   float64 `json:"wait_time"`    // fixed per lot
	MoveTime   float64 `json:"move_time"`    // fixed per lot
	CrewSize   float64 `json:"crew_size"`    // simultaneous operators (>=1)
	TimeUnit   string  `json:"time_unit"`    // MIN | HORA | DIA (default HORA)

	// Subcontracting (EXTERNA / TERCEIROS).
	SupplierID           *int64   `json:"supplier_id,omitempty"`
	ServiceItemCode      *int64   `json:"service_item_code,omitempty"`
	CostPerUnit          *float64 `json:"cost_per_unit,omitempty"`
	LeadTimeDays         *int32   `json:"lead_time_days,omitempty"`
	ThirdPartyRemittance string   `json:"third_party_remittance,omitempty"`

	CreatedBy uuid.UUID `json:"created_by"`
}

type UpdateOperationDTO struct {
	ID                  int64   `json:"id"`
	Name                string  `json:"name"`
	Description         *string `json:"description,omitempty"`
	Origin              string  `json:"origin"`
	Situation           string  `json:"situation"` // APROVADA | INATIVA
	DefaultWorkCenterID *int64  `json:"default_work_center_id,omitempty"`
	StandardTime        float64 `json:"standard_time"`
	SetupTime           float64 `json:"setup_time"`

	RunTime    float64 `json:"run_time"`
	LaborTime  float64 `json:"labor_time"`
	RunBaseQty float64 `json:"run_base_qty"`
	QueueTime  float64 `json:"queue_time"`
	WaitTime   float64 `json:"wait_time"`
	MoveTime   float64 `json:"move_time"`
	CrewSize   float64 `json:"crew_size"`
	TimeUnit   string  `json:"time_unit"`

	SupplierID           *int64   `json:"supplier_id,omitempty"`
	ServiceItemCode      *int64   `json:"service_item_code,omitempty"`
	CostPerUnit          *float64 `json:"cost_per_unit,omitempty"`
	LeadTimeDays         *int32   `json:"lead_time_days,omitempty"`
	ThirdPartyRemittance string   `json:"third_party_remittance,omitempty"`
}

// ─── routes ──────────────────────────────────────────────────────────────────

type CreateRouteDTO struct {
	ItemCode    int64      `json:"item_code"`
	Mask        *string    `json:"mask,omitempty"`
	Alternative int16      `json:"alternative"`
	Description *string    `json:"description,omitempty"`
	IsStandard  bool       `json:"is_standard"`
	ValidFrom   *time.Time `json:"valid_from,omitempty"` // nil = valid from the beginning
	ValidTo     *time.Time `json:"valid_to,omitempty"`   // nil = open-ended
	CreatedBy   uuid.UUID  `json:"created_by"`
}

type UpdateRouteDTO struct {
	ID          int64      `json:"id"`
	Description *string    `json:"description,omitempty"`
	Situation   string     `json:"situation"` // APROVADA | INATIVA
	IsStandard  bool       `json:"is_standard"`
	ValidFrom   *time.Time `json:"valid_from,omitempty"`
	ValidTo     *time.Time `json:"valid_to,omitempty"`
}

// ─── route operations ─────────────────────────────────────────────────────────

type AddRouteOperationDTO struct {
	RouteID      int64    `json:"route_id"`
	Sequence     int16    `json:"sequence"`
	OperationID  int64    `json:"operation_id"`
	WorkCenterID *int64   `json:"work_center_id,omitempty"`
	StandardTime *float64 `json:"standard_time,omitempty"`
	SetupTime    *float64 `json:"setup_time,omitempty"`
	// Rich time-model overrides (nil ⇒ inherit from the operation).
	RunTime    *float64 `json:"run_time,omitempty"`
	LaborTime  *float64 `json:"labor_time,omitempty"`
	RunBaseQty *float64 `json:"run_base_qty,omitempty"`
	QueueTime  *float64 `json:"queue_time,omitempty"`
	WaitTime   *float64 `json:"wait_time,omitempty"`
	MoveTime   *float64 `json:"move_time,omitempty"`
	CrewSize   *float64 `json:"crew_size,omitempty"`
	TimeUnit   *string  `json:"time_unit,omitempty"`
	// Subcontracting overrides (nil ⇒ inherit from the operation).
	SupplierID           *int64   `json:"supplier_id,omitempty"`
	ServiceItemCode      *int64   `json:"service_item_code,omitempty"`
	CostPerUnit          *float64 `json:"cost_per_unit,omitempty"`
	LeadTimeDays         *int32   `json:"lead_time_days,omitempty"`
	ThirdPartyRemittance *string  `json:"third_party_remittance,omitempty"`
	Situation            string   `json:"situation"` // APROVADA | INATIVA | FANTASMA
	Notes                *string  `json:"notes,omitempty"`
}

type UpdateRouteOperationDTO struct {
	ID                   int64    `json:"id"`
	WorkCenterID         *int64   `json:"work_center_id,omitempty"`
	StandardTime         *float64 `json:"standard_time,omitempty"`
	SetupTime            *float64 `json:"setup_time,omitempty"`
	RunTime              *float64 `json:"run_time,omitempty"`
	LaborTime            *float64 `json:"labor_time,omitempty"`
	RunBaseQty           *float64 `json:"run_base_qty,omitempty"`
	QueueTime            *float64 `json:"queue_time,omitempty"`
	WaitTime             *float64 `json:"wait_time,omitempty"`
	MoveTime             *float64 `json:"move_time,omitempty"`
	CrewSize             *float64 `json:"crew_size,omitempty"`
	TimeUnit             *string  `json:"time_unit,omitempty"`
	SupplierID           *int64   `json:"supplier_id,omitempty"`
	ServiceItemCode      *int64   `json:"service_item_code,omitempty"`
	CostPerUnit          *float64 `json:"cost_per_unit,omitempty"`
	LeadTimeDays         *int32   `json:"lead_time_days,omitempty"`
	ThirdPartyRemittance *string  `json:"third_party_remittance,omitempty"`
	Situation            string   `json:"situation"`
	Notes                *string  `json:"notes,omitempty"`
}

// ─── alternative resources ────────────────────────────────────────────────────

type AddRouteOpResourceDTO struct {
	RouteOperationID int64   `json:"route_operation_id"`
	WorkCenterID     int64   `json:"work_center_id"`
	Priority         int16   `json:"priority"`    // 1 = most preferred
	TimeFactor       float64 `json:"time_factor"` // scales op time (1.0 = base); default 1
	IsPrimary        bool    `json:"is_primary"`  // when true, becomes the CT used by cost/CRP/lead-time
}

type UpdateRouteOpResourceDTO struct {
	ID         int64   `json:"id"`
	Priority   int16   `json:"priority"`
	TimeFactor float64 `json:"time_factor"`
}

// ─── network ─────────────────────────────────────────────────────────────────

type SetNetworkEdgeDTO struct {
	PredecessorID int64   `json:"predecessor_id"`
	SuccessorID   int64   `json:"successor_id"`
	OverlapPct    float64 `json:"overlap_pct"`
}

type DeleteNetworkEdgeDTO struct {
	PredecessorID int64 `json:"predecessor_id"`
	SuccessorID   int64 `json:"successor_id"`
}
