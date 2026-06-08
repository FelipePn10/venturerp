package request

import "github.com/google/uuid"

// ─── operations ──────────────────────────────────────────────────────────────

type CreateOperationDTO struct {
	Name                string    `json:"name"`
	Description         *string   `json:"description,omitempty"`
	Origin              string    `json:"origin"` // INTERNA | EXTERNA | TERCEIROS
	DefaultWorkCenterID *int64    `json:"default_work_center_id,omitempty"`
	StandardTime        float64   `json:"standard_time"` // hours
	SetupTime           float64   `json:"setup_time"`    // hours
	CreatedBy           uuid.UUID `json:"created_by"`
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
}

// ─── routes ──────────────────────────────────────────────────────────────────

type CreateRouteDTO struct {
	ItemCode    int64     `json:"item_code"`
	Mask        *string   `json:"mask,omitempty"`
	Alternative int16     `json:"alternative"`
	Description *string   `json:"description,omitempty"`
	IsStandard  bool      `json:"is_standard"`
	CreatedBy   uuid.UUID `json:"created_by"`
}

type UpdateRouteDTO struct {
	ID          int64   `json:"id"`
	Description *string `json:"description,omitempty"`
	Situation   string  `json:"situation"` // APROVADA | INATIVA
	IsStandard  bool    `json:"is_standard"`
}

// ─── route operations ─────────────────────────────────────────────────────────

type AddRouteOperationDTO struct {
	RouteID      int64    `json:"route_id"`
	Sequence     int16    `json:"sequence"`
	OperationID  int64    `json:"operation_id"`
	WorkCenterID *int64   `json:"work_center_id,omitempty"`
	StandardTime *float64 `json:"standard_time,omitempty"`
	SetupTime    *float64 `json:"setup_time,omitempty"`
	Situation    string   `json:"situation"` // APROVADA | INATIVA | FANTASMA
	Notes        *string  `json:"notes,omitempty"`
}

type UpdateRouteOperationDTO struct {
	ID           int64    `json:"id"`
	WorkCenterID *int64   `json:"work_center_id,omitempty"`
	StandardTime *float64 `json:"standard_time,omitempty"`
	SetupTime    *float64 `json:"setup_time,omitempty"`
	Situation    string   `json:"situation"`
	Notes        *string  `json:"notes,omitempty"`
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
