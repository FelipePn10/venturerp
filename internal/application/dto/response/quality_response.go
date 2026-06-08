package response

import (
	"time"

	"github.com/google/uuid"
)

type InspectionPlanResponse struct {
	ID               int64                    `json:"id"`
	ItemCode         int64                    `json:"item_code"`
	RouteOperationID *int64                   `json:"route_operation_id,omitempty"`
	PointType        string                   `json:"point_type"`
	Description      string                   `json:"description"`
	SampleSize       float64                  `json:"sample_size"`
	AcceptanceLevel  float64                  `json:"acceptance_level"`
	Instructions     *string                  `json:"instructions,omitempty"`
	IsActive         bool                     `json:"is_active"`
	Characteristics  []CharacteristicResponse `json:"characteristics,omitempty"`
	CreatedAt        time.Time                `json:"created_at"`
	CreatedBy        uuid.UUID                `json:"created_by"`
}

type CharacteristicResponse struct {
	ID             int64    `json:"id"`
	PlanID         int64    `json:"plan_id"`
	Name           string   `json:"name"`
	Nominal        *float64 `json:"nominal,omitempty"`
	ToleranceUpper *float64 `json:"tolerance_upper,omitempty"`
	ToleranceLower *float64 `json:"tolerance_lower,omitempty"`
	Unit           *string  `json:"unit,omitempty"`
	IsCritical     bool     `json:"is_critical"`
}

type QualityRecordResponse struct {
	ID                int64     `json:"id"`
	PlanID            int64     `json:"plan_id"`
	ProductionOrderID *int64    `json:"production_order_id,omitempty"`
	Lot               *string   `json:"lot,omitempty"`
	ItemCode          int64     `json:"item_code"`
	InspectedQty      float64   `json:"inspected_qty"`
	ApprovedQty       float64   `json:"approved_qty"`
	RejectedQty       float64   `json:"rejected_qty"`
	Result            string    `json:"result"`
	InspectorID       *int64    `json:"inspector_id,omitempty"`
	InspectedAt       time.Time `json:"inspected_at"`
	Notes             *string   `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	CreatedBy         uuid.UUID `json:"created_by"`
}

type NonConformanceResponse struct {
	ID                int64      `json:"id"`
	Code              int64      `json:"code"`
	QualityRecordID   *int64     `json:"quality_record_id,omitempty"`
	ProductionOrderID *int64     `json:"production_order_id,omitempty"`
	ItemCode          int64      `json:"item_code"`
	Lot               *string    `json:"lot,omitempty"`
	NonConformQty     float64    `json:"nonconform_qty"`
	Description       string     `json:"description"`
	Severity          string     `json:"severity"`
	RootCause         *string    `json:"root_cause,omitempty"`
	CorrectiveAction  *string    `json:"corrective_action,omitempty"`
	Disposition       *string    `json:"disposition,omitempty"`
	DisposedAt        *time.Time `json:"disposed_at,omitempty"`
	DisposedBy        *uuid.UUID `json:"disposed_by,omitempty"`
	IsOpen            bool       `json:"is_open"`
	CreatedAt         time.Time  `json:"created_at"`
	CreatedBy         uuid.UUID  `json:"created_by"`
}
