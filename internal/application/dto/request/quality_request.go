package request

import "github.com/google/uuid"

type CreateInspectionPlanDTO struct {
	ItemCode         int64     `json:"item_code"`
	RouteOperationID *int64    `json:"route_operation_id,omitempty"`
	PointType        string    `json:"point_type"` // RECEBIMENTO | PROCESSO | EXPEDICAO
	Description      string    `json:"description"`
	SampleSize       float64   `json:"sample_size"`
	AcceptanceLevel  float64   `json:"acceptance_level"`
	Instructions     *string   `json:"instructions,omitempty"`
	CreatedBy        uuid.UUID `json:"created_by"`
}

type AddCharacteristicDTO struct {
	PlanID         int64    `json:"plan_id"`
	Name           string   `json:"name"`
	Nominal        *float64 `json:"nominal,omitempty"`
	ToleranceUpper *float64 `json:"tolerance_upper,omitempty"`
	ToleranceLower *float64 `json:"tolerance_lower,omitempty"`
	Unit           *string  `json:"unit,omitempty"`
	IsCritical     bool     `json:"is_critical"`
}

type CreateQualityRecordDTO struct {
	PlanID            int64               `json:"plan_id"`
	ProductionOrderID *int64              `json:"production_order_id,omitempty"`
	Lot               *string             `json:"lot,omitempty"`
	ItemCode          int64               `json:"item_code"`
	InspectedQty      float64             `json:"inspected_qty"`
	ApprovedQty       float64             `json:"approved_qty"`
	RejectedQty       float64             `json:"rejected_qty"`
	Result            string              `json:"result"` // APROVADO | REJEITADO | CONDICIONAL | PENDENTE
	InspectorID       *int64              `json:"inspector_id,omitempty"`
	Notes             *string             `json:"notes,omitempty"`
	CreatedBy         uuid.UUID           `json:"created_by"`
	Measurements      []AddMeasurementDTO `json:"measurements,omitempty"`
}

type AddMeasurementDTO struct {
	CharacteristicID int64   `json:"characteristic_id"`
	MeasuredValue    float64 `json:"measured_value"`
	IsConformant     bool    `json:"is_conformant"`
}

type CreateNCDTO struct {
	QualityRecordID   *int64    `json:"quality_record_id,omitempty"`
	ProductionOrderID *int64    `json:"production_order_id,omitempty"`
	ItemCode          int64     `json:"item_code"`
	Lot               *string   `json:"lot,omitempty"`
	NonConformQty     float64   `json:"nonconform_qty"`
	Description       string    `json:"description"`
	Severity          string    `json:"severity"` // CRITICA | MAIOR | MENOR | OBSERVACAO
	CreatedBy         uuid.UUID `json:"created_by"`
}

type DispositionNCDTO struct {
	Disposition string `json:"disposition"` // SUCATA | RETRABALHO | APROVADO_CONDICIONAL | DEVOLVIDO
	DisposedBy  string `json:"disposed_by"`
}
